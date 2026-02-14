package storage

import (
	"fmt"
	"github.com/H3nSte1n/recipe/pkg/config"
)

func NewFileStore(cfg *config.Config) (FileStore, error) {
	switch cfg.Storage.Type {
	case "local":
		return NewLocalFileStore(
			cfg.Storage.LocalPath,
			cfg.Storage.BaseURL,
		)
	case "s3":
		return nil, fmt.Errorf("s3 storage not implemented")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
