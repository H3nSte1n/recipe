package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleYAML = `
db:
  password: file_password
jwt:
  secret: file_secret
security:
  encryption_key: file_encryption_key
`

// writeTempConfig writes env.<env>.yaml into a temp dir and chdirs into it so
// LoadConfig (which reads from ".") finds it. t.Chdir restores the cwd on cleanup.
func writeTempConfig(t *testing.T, env, contents string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "env."+env+".yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	t.Chdir(dir)
}

func TestLoadConfig_ReadsFile(t *testing.T) {
	writeTempConfig(t, "test", sampleYAML)

	cfg, err := LoadConfig("test")
	require.NoError(t, err)
	assert.Equal(t, "file_password", cfg.DB.Password)
	assert.Equal(t, "file_secret", cfg.JWT.Secret)
	assert.Equal(t, "file_encryption_key", cfg.Security.EncryptionKey)
}

func TestLoadConfig_EnvOverridesSecrets(t *testing.T) {
	writeTempConfig(t, "test", sampleYAML)

	t.Setenv("DB_PASSWORD", "env_password")
	t.Setenv("SECURITY_ENCRYPTION_KEY", "env_encryption_key")

	cfg, err := LoadConfig("test")
	require.NoError(t, err)

	// Bound secret keys must be overridable from the environment.
	assert.Equal(t, "env_password", cfg.DB.Password)
	assert.Equal(t, "env_encryption_key", cfg.Security.EncryptionKey)
	// Unset secrets keep their file value.
	assert.Equal(t, "file_secret", cfg.JWT.Secret)
}

func TestConfig_Validate_JWTSecret(t *testing.T) {
	strong := "a-sufficiently-long-random-jwt-secret-value-1234"
	require.GreaterOrEqual(t, len(strong), minJWTSecretBytes)

	cases := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", "    ", true},
		{"known placeholder", "your-super-secret-key-here", true},
		{"placeholder case-insensitive", "CHANGE_ME", true},
		{"too short", "short-secret", true},
		{"exactly 31 bytes", strings.Repeat("a", 31), true},
		{"exactly 32 bytes", strings.Repeat("a", 32), false},
		{"strong secret", strong, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := Config{JWT: JWTConfig{Secret: c.secret}}
			err := cfg.Validate()
			if c.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
