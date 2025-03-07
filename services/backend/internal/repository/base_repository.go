package repository

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	GetDB() *gorm.DB
	WithTransaction(ctx context.Context, fn TransactionFunc) error
}

type TransactionFunc func(Repository) error

type baseRepository struct {
	db *gorm.DB
}

func (r *baseRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *baseRepository) WithTransaction(ctx context.Context, fn TransactionFunc) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create a new repository with the transaction
		repo := &baseRepository{db: tx}
		return fn(repo)
	})
}
