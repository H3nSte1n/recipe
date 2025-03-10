// internal/repository/base_repository.go
package repository

import (
	"context"
	"gorm.io/gorm"
)

// Generic Repository interface
type Repository[T any] interface {
	GetDB() *gorm.DB
	withDB(db *gorm.DB) Repository[T]
	WithTransaction(ctx context.Context, fn TransactionFunc[T]) error
}

// Generic TransactionFunc
type TransactionFunc[T any] func(Repository[T]) error

// Generic base repository
type BaseRepository[T any] struct {
	db *gorm.DB
}

// Constructor for BaseRepository
func NewBaseRepository[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

func (r *BaseRepository[T]) GetDB() *gorm.DB {
	return r.db
}

func (r *BaseRepository[T]) withDB(db *gorm.DB) Repository[T] {
	return &BaseRepository[T]{db: db}
}

func (r *BaseRepository[T]) WithTransaction(ctx context.Context, fn TransactionFunc[T]) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := r.withDB(tx)
		return fn(repo)
	})
}
