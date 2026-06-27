package config

import (
	"os"
	"path/filepath"
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
