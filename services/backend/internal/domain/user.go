package domain

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

type User struct {
	ID              string     `json:"id" gorm:"primaryKey;type:uuid"`
	Email           string     `json:"email" gorm:"unique;not null"`
	PasswordHash    string     `json:"-" gorm:"not null"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// FailedLoginAttempts and LockedUntil back the account-lockout check in
	// UserService.Login: consecutive bad passwords increment the counter, and
	// hitting the threshold sets a cooldown expiry. Never serialized to
	// clients — that would leak lockout state to an unauthenticated caller.
	FailedLoginAttempts int        `json:"-" gorm:"column:failed_login_attempts;not null;default:0"`
	LockedUntil         *time.Time `json:"-" gorm:"column:locked_until"`
}

// IsEmailVerified reports whether the user has completed email verification.
func (u *User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// UserSummary is a non-PII projection of a user used by list endpoints. It omits
// email and timestamps so an authenticated member cannot enumerate everyone's
// email address.
type UserSummary struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Helper method to hash password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// Helper method to check password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
