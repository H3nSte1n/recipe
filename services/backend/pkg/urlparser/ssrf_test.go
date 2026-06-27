package urlparser

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},             // loopback
		{"::1", true},                   // loopback v6
		{"10.0.0.5", true},              // RFC1918
		{"172.16.0.1", true},            // RFC1918
		{"192.168.1.1", true},           // RFC1918
		{"169.254.169.254", true},       // link-local (cloud metadata)
		{"fe80::1", true},               // link-local v6
		{"fc00::1", true},               // unique-local v6
		{"100.64.0.1", true},            // CGNAT / Tailscale
		{"100.127.255.255", true},       // CGNAT upper edge
		{"224.0.0.1", true},             // multicast
		{"0.0.0.0", true},               // unspecified
		{"8.8.8.8", false},              // public
		{"1.1.1.1", false},              // public
		{"99.255.255.255", false},       // just below CGNAT
		{"100.128.0.1", false},          // just above CGNAT
		{"2606:4700:4700::1111", false}, // public v6 (Cloudflare)
	}

	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		require.NotNil(t, ip, "parse %s", c.ip)
		assert.Equal(t, c.blocked, isBlockedIP(ip), "ip %s", c.ip)
	}
}

func TestValidatePublicURL_RejectsNonHTTPScheme(t *testing.T) {
	for _, raw := range []string{
		"ftp://example.com/file",
		"file:///etc/passwd",
		"gopher://example.com",
		"javascript:alert(1)",
	} {
		_, err := validatePublicURL(context.Background(), raw)
		assert.Error(t, err, "scheme should be rejected: %s", raw)
	}
}

func TestValidatePublicURL_RejectsInternalIPLiterals(t *testing.T) {
	// IP-literal hosts resolve to themselves with no DNS, so these are offline.
	for _, raw := range []string{
		"http://127.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/",
		"http://192.168.0.1/",
		"http://[::1]/",
		"http://100.64.0.1/",
	} {
		_, err := validatePublicURL(context.Background(), raw)
		assert.Error(t, err, "internal host should be rejected: %s", raw)
	}
}

func TestValidatePublicURL_AllowsPublicIPLiteral(t *testing.T) {
	parsed, err := validatePublicURL(context.Background(), "http://8.8.8.8/")
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", parsed.Hostname())
}

func TestSafeDialContext_BlocksInternalAddresses(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1:80",
		"169.254.169.254:80",
		"10.0.0.1:443",
		"100.64.0.1:80",
	} {
		conn, err := safeDialContext(context.Background(), "tcp", addr)
		assert.Error(t, err, "dial should be blocked: %s", addr)
		assert.Nil(t, conn)
	}
}

func TestDefaultRedirectPolicy(t *testing.T) {
	mkReq := func(rawURL string) *http.Request {
		u, _ := url.Parse(rawURL)
		return &http.Request{URL: u}
	}

	// Normal http redirect, short chain → allowed.
	assert.NoError(t, defaultRedirectPolicy(mkReq("http://example.com/a"), nil))

	// Non-http scheme hop → rejected.
	assert.Error(t, defaultRedirectPolicy(mkReq("file:///etc/passwd"), nil))

	// Too many redirects → rejected.
	via := make([]*http.Request, 10)
	assert.Error(t, defaultRedirectPolicy(mkReq("http://example.com/loop"), via))
}
