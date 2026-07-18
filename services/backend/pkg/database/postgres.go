package database

import (
	"fmt"
	"github.com/H3nSte1n/recipe/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewPostgresConnection(config *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		config.DB.Host,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
		config.DB.Port,
		config.DB.SSLMode,
	)

	// GORM's default logger logs slow-query warnings with full SQL, including
	// literal query args like plaintext user emails. Silence it in production;
	// keep the default (Warn) elsewhere for local debugging.
	logLevel := gormlogger.Warn
	if config.App.Env == "production" {
		logLevel = gormlogger.Silent
	}
	gormLogger := gormlogger.Default.LogMode(logLevel)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}
