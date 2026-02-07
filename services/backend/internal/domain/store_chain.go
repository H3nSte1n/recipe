package domain

import "time"

type StoreChain struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name      string         `json:"name" gorm:"not null"`
	Country   string         `json:"country" gorm:"not null"`
	Layout    []StoreSection `json:"layout" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

type StoreSection struct {
	Order      int        `json:"order"`
	Name       string     `json:"name"`
	Categories []Category `json:"categories"`
}
