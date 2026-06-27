package urlparser

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// blockedV4CIDRs are special-use IPv4 ranges the stdlib net.IP predicates do not
// cover but that must never be reachable: "this host on this network"
// (0.0.0.0/8, which on Linux routes to localhost), CGNAT / Tailscale
// (100.64.0.0/10 — tailnet peers live here), and reserved Class E including the
// limited-broadcast address 255.255.255.255 (240.0.0.0/4).
var blockedV4CIDRs = []*net.IPNet{
	mustCIDR("0.0.0.0/8"),
	mustCIDR("100.64.0.0/10"),
	mustCIDR("240.0.0.0/4"),
}

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// isBlockedIP reports whether ip is anything other than a routable public
// address. It blocks loopback, RFC1918 private + unique-local (fc00::/7),
// link-local (incl. 169.254.0.0/16 cloud metadata and fe80::/10), CGNAT,
// multicast, the unspecified address, and the special-use IPv4 ranges in
// blockedV4CIDRs. IPv6 transition forms (6to4, NAT64) are decoded to their
// embedded IPv4 and re-checked so an internal IPv4 cannot be smuggled inside
// an IPv6 literal.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if v4 := embeddedIPv4(ip); v4 != nil {
		return isBlockedIP(v4)
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

	if v4 := ip.To4(); v4 != nil {
		for _, blocked := range blockedV4CIDRs {
			if blocked.Contains(v4) {
				return true
			}
		}
	}
	return false
}

// embeddedIPv4 returns the IPv4 address embedded in an IPv6 transition address
// (6to4 2002::/16 or NAT64 well-known prefix 64:ff9b::/96), or nil if ip is not
// one of those forms. The ordinary IPv4-mapped form (::ffff:a.b.c.d) is left to
// the stdlib predicates, which already decode it via To4().
func embeddedIPv4(ip net.IP) net.IP {
	if len(ip) != net.IPv6len {
		return nil
	}
	// 6to4 (2002:AABB:CCDD::/48): IPv4 AA.BB.CC.DD is in bytes 2..5.
	if ip[0] == 0x20 && ip[1] == 0x02 {
		return net.IPv4(ip[2], ip[3], ip[4], ip[5])
	}
	// NAT64 well-known prefix (64:ff9b::/96): IPv4 is in the last 4 bytes.
	if ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b &&
		ip[4] == 0 && ip[5] == 0 && ip[6] == 0 && ip[7] == 0 &&
		ip[8] == 0 && ip[9] == 0 && ip[10] == 0 && ip[11] == 0 {
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	}
	return nil
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
