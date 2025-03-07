package domain

import (
	"time"
)

type Profile struct {
	ID         string    `json:"id" gorm:"primaryKey;type:uuid"`
	UserID     string    `json:"user_id" gorm:"type:uuid;not null"`
	Bio        string    `json:"bio" gorm:"type:text"`
	Location   string    `json:"location" gorm:"type:varchar(255)"`
	AvatarURL  string    `json:"avatar_url" gorm:"type:varchar(255)"`
	WebsiteURL string    `json:"website_url" gorm:"type:varchar(255)"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
	User       *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// DTOs
type CreateProfileRequest struct {
	Bio        string `json:"bio"`
	Location   string `json:"location"`
	WebsiteURL string `json:"website_url"`
}

type UpdateProfileRequest struct {
	Bio        *string `json:"bio,omitempty"`
	Location   *string `json:"location,omitempty"`
	WebsiteURL *string `json:"website_url,omitempty"`
}
