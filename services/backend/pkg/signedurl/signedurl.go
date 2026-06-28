// Package signedurl mints and verifies short-lived HMAC signatures for local
// upload URLs. Because the frontend renders images in <img> tags, which cannot
// send an Authorization header, access is gated by a signature carried in the
// URL query string rather than by the JWT. The signature is minted only for
// callers who could already read the owning recipe, and expires after a TTL.
package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// DefaultTTL is a generous validity window for minted upload links. It is long
// because the frontend embeds the signed URL in an <img> and cannot transparently
// refresh it; recipe visibility is already enforced at the recipe endpoint, so
// the link's main job is to be unguessable and eventually expire.
const DefaultTTL = 24 * time.Hour

type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) *Signer {
	return &Signer{secret: []byte(secret), ttl: ttl}
}

// Sign appends exp and sig query parameters to a local uploads URL. The
// signature covers the URL's filename and expiry. URLs that are empty,
// unparseable, or do not point at the local /uploads/ path (e.g. an external
// image URL on an imported recipe) are returned unchanged.
func (s *Signer) Sign(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if !strings.Contains(u.Path, "/uploads/") {
		return rawURL
	}
	exp := time.Now().Add(s.ttl).Unix()
	sig := s.compute(path.Base(u.Path), exp)

	q := u.Query()
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return u.String()
}

// Verify checks that sig is a valid, unexpired signature for filename.
func (s *Signer) Verify(filename, expStr, sig string) error {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid expiry")
	}
	if time.Now().Unix() >= exp {
		return fmt.Errorf("signature expired")
	}
	expected := s.compute(filename, exp)
	// constant-time comparison
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (s *Signer) compute(filename string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "%s|%d", filename, exp)
	return hex.EncodeToString(mac.Sum(nil))
}
