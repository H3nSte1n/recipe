package urlparser

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// cgnatRange is RFC 6598 Carrier-Grade NAT space (100.64.0.0/10). Go's
// net.IP.IsPrivate does not cover it, but Tailscale assigns tailnet peers
// addresses in this range, so it must be blocked to stop an authenticated
// member from reaching other tailnet nodes via URL import.
var cgnatRange = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// isBlockedIP reports whether ip is anything other than a routable public
// address. It blocks loopback, RFC1918 private + unique-local (fc00::/7),
// link-local (incl. 169.254.0.0/16 cloud metadata and fe80::/10), CGNAT,
// multicast, and the unspecified address.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && cgnatRange.Contains(v4) {
		return true
	}
	return false
}

// validatePublicURL parses rawURL, requires an http/https scheme, resolves the
// hostname, and rejects the URL if any resolved address is non-public. Rejecting
// on *any* blocked address defeats DNS round-robin that mixes a public record
// with an internal one. It returns the parsed URL for reuse by the caller.
func validatePublicURL(ctx context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("URL has no host")
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host %q resolved to no addresses", host)
	}

	for _, ipAddr := range ips {
		if isBlockedIP(ipAddr.IP) {
			return nil, fmt.Errorf("host %q resolves to a non-public address (%s)", host, ipAddr.IP)
		}
	}

	return parsed, nil
}

// safeDialContext is the authoritative SSRF guard: it runs on every dial,
// including each redirect hop, re-resolving the host and rejecting any
// non-public address, then pins the connection to a validated IP. Pinning
// closes the TOCTOU/DNS-rebinding window between validation and connect.
// HTTPS is unaffected: the Transport derives SNI and certificate verification
// from the URL hostname, not from the dialed IP.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address %q: %w", addr, err)
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("host %q resolved to no addresses", host)
	}

	for _, ipAddr := range ips {
		if isBlockedIP(ipAddr.IP) {
			return nil, fmt.Errorf("blocked dial to non-public address %s (host %q)", ipAddr.IP, host)
		}
	}

	dialer := &net.Dialer{}
	var lastErr error
	for _, ipAddr := range ips {
		// Dial the validated IP directly so the connection cannot be rebound to a
		// different address than the one we checked.
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
