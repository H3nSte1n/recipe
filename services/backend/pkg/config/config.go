package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Frontend FrontendConfig `mapstructure:"frontend"`
	DB       DBConfig       `mapstructure:"db"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Storage  StorageConfig  `mapstructure:"storage"`
	AI       AIConfig       `mapstructure:"ai"`
	Security SecurityConfig `mapstructure:"security"`
	CORS     CORSConfig     `mapstructure:"cors"`
	LogLevel string         `mapstructure:"log_level"`
}

type SecurityConfig struct {
	// EncryptionKey is the application-layer secret used to encrypt sensitive
	// columns at rest (e.g. user AI API keys). Inject via SECURITY_ENCRYPTION_KEY.
	EncryptionKey string `mapstructure:"encryption_key"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port string `mapstructure:"port"`
}

type FrontendConfig struct {
	Url string `mapstructure:"url"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type JWTConfig struct {
	Secret        string        `mapstructure:"secret"`
	Duration      time.Duration `mapstructure:"duration"`
	ExpirationHrs time.Duration `mapstructure:"expiration_hours"`
	// Issuer and Audience are set as the "iss"/"aud" claims on issued tokens
	// and are required to match on every token the auth middleware accepts.
	// Fall back to defaultJWTIssuer/defaultJWTAudience when unset.
	Issuer   string `mapstructure:"issuer"`
	Audience string `mapstructure:"audience"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type StorageConfig struct {
	Type      string    `mapstructure:"type"`
	LocalPath string    `mapstructure:"local_path"`
	BaseURL   string    `mapstructure:"base_url"`
	AWS       AWSConfig `mapstructure:"aws"`
}

type AIConfig struct {
	OpenAIAPIKey    string `mapstructure:"openai_api_key"`
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
}

type AWSConfig struct {
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

func LoadConfig(env string) (config Config, err error) {
	v := viper.New()

	v.SetConfigName(fmt.Sprintf("env.%s", env))
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind secrets to environment variables so they can be injected at
	// deploy time (e.g. DB_PASSWORD, JWT_SECRET, SECURITY_ENCRYPTION_KEY) and
	// override the committed YAML placeholders. viper's AutomaticEnv alone does not
	// reliably override fields populated via Unmarshal, so each key is bound here.
	secretKeys := []string{
		"db.password",
		"jwt.secret",
		"smtp.password",
		"ai.openai_api_key",
		"ai.anthropic_api_key",
		"storage.aws.access_key_id",
		"storage.aws.secret_access_key",
		"security.encryption_key",
	}
	for _, key := range secretKeys {
		_ = v.BindEnv(key)
	}

	// Same reasoning as secretKeys above, for non-secret fields we still want
	// overridable via env var (e.g. JWT_ISSUER, JWT_AUDIENCE) in deployments
	// that don't want to bake them into the committed YAML.
	overridableKeys := []string{
		"jwt.issuer",
		"jwt.audience",
	}
	for _, key := range overridableKeys {
		_ = v.BindEnv(key)
	}

	if err := v.ReadInConfig(); err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	if strings.TrimSpace(config.JWT.Issuer) == "" {
		config.JWT.Issuer = defaultJWTIssuer
	}
	if strings.TrimSpace(config.JWT.Audience) == "" {
		config.JWT.Audience = defaultJWTAudience
	}

	return config, nil
}

// defaultJWTIssuer/defaultJWTAudience are used when jwt.issuer/jwt.audience
// are not set via config or env var. Kept as simple constants per project
// convention (config-driven where a pattern already exists, constants
// otherwise) rather than a new configuration subsystem.
const (
	defaultJWTIssuer   = "recipe-app"
	defaultJWTAudience = "recipe-app-api"
)

const minJWTSecretBytes = 32

// knownWeakJWTSecrets are placeholder values shipped in sample configs. Using
// one in a real deployment means the secret was never set, so it is rejected.
var knownWeakJWTSecrets = map[string]bool{
	"your-super-secret-key-here": true,
	"change_me":                  true,
	"change-me":                  true,
	"changeme":                   true,
	"secret":                     true,
}

// Validate enforces fail-closed checks that must hold before the server boots.
// It currently guards the JWT signing secret: a weak or default secret lets an
// attacker forge tokens for any user, so an empty, placeholder, or too-short
// secret aborts startup. Inject a strong secret via the JWT_SECRET env var.
func (c *Config) Validate() error {
	// Validate the trimmed secret so a whitespace-padded value can't pass the
	// length check on padding alone.
	secret := strings.TrimSpace(c.JWT.Secret)
	if secret == "" {
		return fmt.Errorf("jwt.secret is not set; inject a strong secret via JWT_SECRET")
	}
	if knownWeakJWTSecrets[strings.ToLower(secret)] {
		return fmt.Errorf("jwt.secret is a known placeholder value; set a strong random secret (>= %d bytes) via JWT_SECRET", minJWTSecretBytes)
	}
	if len(secret) < minJWTSecretBytes {
		return fmt.Errorf("jwt.secret must be at least %d bytes, got %d; inject a strong secret via JWT_SECRET", minJWTSecretBytes, len(secret))
	}
	return nil
}
