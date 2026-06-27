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

	if err := v.ReadInConfig(); err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return config, nil
}
