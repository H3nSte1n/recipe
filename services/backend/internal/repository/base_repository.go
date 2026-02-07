package repository

import (
	"context"
	"gorm.io/gorm"
)

type Repository[T any] interface {
	GetDB() *gorm.DB
	withDB(db *gorm.DB) Repository[T]
	WithTransaction(ctx context.Context, fn TransactionFunc[T]) error
}

type TransactionFunc[T any] func(Repository[T]) error

type BaseRepository[T any] struct {
	db *gorm.DB
}

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
