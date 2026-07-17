package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// staleLimiterTTL controls how long an idle per-IP limiter is kept around before it's evicted.
// Without this the limiter map would grow forever as new IPs show up, since single-instance
// in-memory rate limiting (by design, per the deployment's single-app-instance topology) has no
// other place to expire entries.
const staleLimiterTTL = 10 * time.Minute

// ipRateLimiter tracks one rate.Limiter per client IP for a single endpoint. It is deliberately
// in-memory and per-instance: this app runs as a single backend instance behind nginx, so there
// is no need for a shared store (e.g. Redis) to make limits consistent across instances.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	burst    int
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	l := &ipRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		burst:    burst,
	}
	go l.evictStaleLoop()
	return l
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.limiters[ip]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter.Allow()
}

// evictStaleLoop periodically drops limiters for IPs that haven't made a request in a while, so
// long-running processes don't accumulate one entry per client IP ever seen.
func (l *ipRateLimiter) evictStaleLoop() {
	ticker := time.NewTicker(staleLimiterTTL)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-staleLimiterTTL)
		l.mu.Lock()
		for ip, entry := range l.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit returns Gin middleware that limits requests to r per second (with burst allowed
// immediately) per client IP, as reported by c.ClientIP(). Each call to RateLimit creates its own
// independent limiter set, so applying it to several routes rate-limits each route separately.
//
// c.ClientIP() only reflects the real client once Gin's trusted-proxy list is configured
// correctly (see router.trustedProxiesFromEnv) — otherwise X-Forwarded-For is attacker-controlled
// and this limiter keys on whatever IP the client claims.
func RateLimit(r rate.Limit, burst int) gin.HandlerFunc {
	limiter := newIPRateLimiter(r, burst)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
			c.Abort()
			return
		}
		c.Next()
	}
}
