package signedurl

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseSig(t *testing.T, signed string) (filename, exp, sig string) {
	t.Helper()
	u, err := url.Parse(signed)
	require.NoError(t, err)
	return u.Path[len("/uploads/"):], u.Query().Get("exp"), u.Query().Get("sig")
}

func TestSigner_SignVerifyRoundTrip(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	signed := s.Sign("http://localhost:8080/uploads/abc.png")

	filename, exp, sig := parseSig(t, signed)
	assert.Equal(t, "abc.png", filename)
	assert.NoError(t, s.Verify(filename, exp, sig))
}

func TestSigner_RejectsTamperedFilename(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	signed := s.Sign("http://localhost:8080/uploads/abc.png")
	_, exp, sig := parseSig(t, signed)

	assert.Error(t, s.Verify("evil.png", exp, sig))
}

func TestSigner_RejectsTamperedSig(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	signed := s.Sign("http://localhost:8080/uploads/abc.png")
	filename, exp, _ := parseSig(t, signed)

	assert.Error(t, s.Verify(filename, exp, "deadbeef"))
}

func TestSigner_RejectsExpired(t *testing.T) {
	s := NewSigner("test-secret", -time.Minute) // already expired
	signed := s.Sign("http://localhost:8080/uploads/abc.png")
	filename, exp, sig := parseSig(t, signed)

	assert.Error(t, s.Verify(filename, exp, sig))
}

func TestSigner_RejectsWrongSecret(t *testing.T) {
	a := NewSigner("secret-a", time.Hour)
	b := NewSigner("secret-b", time.Hour)
	signed := a.Sign("http://localhost:8080/uploads/abc.png")
	filename, exp, sig := parseSig(t, signed)

	assert.Error(t, b.Verify(filename, exp, sig))
}

func TestSigner_RejectsMalformedExp(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	assert.Error(t, s.Verify("abc.png", "not-a-number", "abcd"))
}

func TestSigner_EmptyURLUnchanged(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	assert.Equal(t, "", s.Sign(""))
}
