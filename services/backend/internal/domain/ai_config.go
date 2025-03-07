package domain

import (
	"encoding/json"
	"time"
)

type UserAIConfig struct {
	ID        string          `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    string          `json:"user_id" gorm:"type:uuid;not null"`
	AIModelID string          `json:"ai_model_id" gorm:"type:uuid;not null"`
	APIKey    string          `json:"-" gorm:"not null"` // Hide in JSON responses
	IsDefault bool            `json:"is_default" gorm:"default:false"`
	Settings  json.RawMessage `json:"settings" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	User      *User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	AIModel   *AIModel        `json:"ai_model,omitempty" gorm:"foreignKey:AIModelID"`
}

type AIModel struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name         string    `json:"name" gorm:"not null"`
	Provider     string    `json:"provider" gorm:"not null"`
	ModelVersion string    `json:"model_version" gorm:"not null"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// DTOs
type CreateUserAIConfigRequest struct {
	AIModelID string          `json:"ai_model_id" validate:"required"`
	APIKey    string          `json:"api_key" validate:"required"`
	IsDefault bool            `json:"is_default"`
	Settings  json.RawMessage `json:"settings"`
}

type UpdateUserAIConfigRequest struct {
	APIKey    *string         `json:"api_key,omitempty"`
	IsDefault *bool           `json:"is_default,omitempty"`
	Settings  json.RawMessage `json:"settings,omitempty"`
}
