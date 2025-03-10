package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig     `mapstructure:"app"`
	DB       DBConfig      `mapstructure:"db"`
	JWT      JWTConfig     `mapstructure:"jwt"`
	SMTP     SMTPConfig    `mapstructure:"smtp"`
	Storage  StorageConfig `mapstructure:"storage"`
	AI       AIConfig      `mapstructure:"ai"`
	LogLevel string        `mapstructure:"log_level"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port string `mapstructure:"port"`
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

	// Change to yaml config
	v.SetConfigName(fmt.Sprintf("env.%s", env))
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Enable env variable override
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := v.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return config, nil
}
