package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	AppName string `mapstructure:"APP_NAME"`
	AppEnv  string `mapstructure:"APP_ENV"`
	AppPort string `mapstructure:"APP_PORT"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`

	JWTSecret        string        `mapstructure:"JWT_SECRET"`
	JWTDuration      time.Duration `mapstructure:"JWT_DURATION"`
	JWTExpirationHrs time.Duration `mapstructure:"JWT_EXPIRATION_HOURS"`

	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     string `mapstructure:"SMTP_PORT"`
	SMTPUser     string `mapstructure:"SMTP_USER"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`

	LogLevel string `mapstructure:"LOG_LEVEL"`
}

func LoadConfig(path string) (config Config, err error) {
	// Initialize Viper
	v := viper.New()

	// Set the path to look for the .env file
	v.SetConfigFile(fmt.Sprintf("%s/.env", path))
	v.SetConfigType("env") // or v.SetConfigType("env")

	// Enable VIPER to read Environment Variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			return config, fmt.Errorf("config file not found: %v", err)
		}
		return config, fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal config
	err = v.Unmarshal(&config)
	if err != nil {
		return config, fmt.Errorf("unable to decode config into struct: %v", err)
	}

	return config, nil
}
