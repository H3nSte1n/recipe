package domain

import "time"

type PasswordResetToken struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    string    `json:"user_id" gorm:"type:uuid"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenRevocation records, per user, the newest time at which all JWTs
// issued before that time must be rejected. Written on password reset and
// account deletion; read by the auth middleware on every authenticated
// request.
type TokenRevocation struct {
	UserID    string    `json:"user_id" gorm:"primaryKey;type:uuid"`
	RevokedAt time.Time `json:"revoked_at" gorm:"not null"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}
