package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/email"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"time"
)

type userRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]domain.User, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	CreateProfile(ctx context.Context, profile *domain.Profile) error
	CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, tokenID string) error
	SetLoginLockoutState(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error
	ResetLoginLockout(ctx context.Context, userID string) error
	SetTokenRevocation(ctx context.Context, userID string, revokedAt time.Time) error
	CreateVerificationToken(ctx context.Context, token *domain.EmailVerificationToken) error
	GetVerificationTokenByToken(ctx context.Context, token string) (*domain.EmailVerificationToken, error)
	MarkVerificationTokenUsed(ctx context.Context, tokenID string) error
	GetLatestVerificationToken(ctx context.Context, userID string) (*domain.EmailVerificationToken, error)
	MarkEmailVerified(ctx context.Context, userID string) error
	IsEmailVerified(ctx context.Context, userID string) (bool, error)
	WithTypedTransaction(ctx context.Context, fn func(repository.UserRepository) error) error
}

const (
	// maxFailedLoginAttempts is how many consecutive bad passwords are tolerated before an
	// account is locked out. 5 balances usability (typos happen) against slowing down
	// credential-stuffing/brute-force attempts.
	maxFailedLoginAttempts = 5
	// accountLockDuration is how long an account stays locked once maxFailedLoginAttempts is
	// reached. 15 minutes is long enough to make automated brute-forcing impractical while
	// short enough that a legitimate user isn't locked out indefinitely.
	accountLockDuration = 15 * time.Minute
)

type UserService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	ValidateToken(token string) (*jwt.Token, error)
	ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]domain.UserSummary, error)
	VerifyEmail(ctx context.Context, req *domain.VerifyEmailRequest) error
	ResendVerification(ctx context.Context, req *domain.ResendVerificationRequest) error
	IsEmailVerified(ctx context.Context, userID string) (bool, error)
}

// verificationTokenTTL is how long a freshly issued email-verification token
// stays valid.
const verificationTokenTTL = 24 * time.Hour

// resendVerificationCooldown is the minimum time a user must wait between
// verification-email resend requests. Enforced at the service layer (via the
// timestamp on the most recent token) rather than dedicated rate-limiting
// middleware/infra, since that infra doesn't exist yet in this codebase.
const resendVerificationCooldown = 60 * time.Second

type userService struct {
	userRepo     userRepository
	jwtSecret    []byte
	jwtDuration  time.Duration
	jwtIssuer    string
	jwtAudience  string
	emailService email.EmailService
	logger       *zap.Logger
}

func NewUserService(userRepo userRepository, jwtSecret string, config config.Config, emailService email.EmailService, logger *zap.Logger) UserService {
	return &userService{
		userRepo:     userRepo,
		jwtSecret:    []byte(jwtSecret),
		jwtDuration:  config.JWT.Duration,
		jwtIssuer:    config.JWT.Issuer,
		jwtAudience:  config.JWT.Audience,
		emailService: emailService,
		logger:       logger,
	}
}

func (s *userService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !apperrors.IsNotFound(err) {
		return nil, err
	}
	if existingUser != nil {
		return nil, apperrors.New("email already registered")
	}

	var user *domain.User
	err = s.userRepo.WithTypedTransaction(ctx, func(txRepo repository.UserRepository) error {
		hashedPassword, err := domain.HashPassword(req.Password)
		if err != nil {
			return err
		}

		user = &domain.User{
			ID:           uuid.New().String(),
			Email:        req.Email,
			PasswordHash: hashedPassword,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
		}

		if err := txRepo.Create(ctx, user); err != nil {
			return err
		}

		profile := &domain.Profile{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			Bio:       fmt.Sprintf("Hello, I'm %s %s", user.FirstName, user.LastName),
			Location:  "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		return txRepo.CreateProfile(ctx, profile)
	})

	if err != nil {
		return nil, err
	}

	// Issue a verification token and email it. Sending is best-effort: a mail
	// outage or unconfigured SMTP provider must not block account creation
	// (unlike ForgotPassword, where the user is actively waiting on the
	// email). The token is already persisted, so ResendVerification can
	// recover from a failed send.
	tokenString, err := s.createVerificationToken(ctx, user.ID)
	if err != nil {
		s.logger.Warn("failed to create email verification token", zap.String("user_id", user.ID), zap.Error(err))
	} else if err := s.emailService.SendVerificationEmail(user.Email, tokenString); err != nil {
		s.logger.Warn("failed to send verification email", zap.String("user_id", user.ID), zap.Error(err))
	}

	return user, nil
}

// createVerificationToken generates a random token, persists it, and returns
// the plaintext so the caller can email it.
func (s *userService) createVerificationToken(ctx context.Context, userID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	tokenString := hex.EncodeToString(tokenBytes)

	verificationToken := &domain.EmailVerificationToken{
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(verificationTokenTTL),
	}

	if err := s.userRepo.CreateVerificationToken(ctx, verificationToken); err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.New("invalid credentials")
	}

	if user.LockedUntil != nil {
		if time.Now().Before(*user.LockedUntil) {
			return nil, apperrors.ErrAccountLocked
		}
		// Lockout window has expired: clear it so the account gets a fresh set of
		// attempts instead of re-locking on the very next bad password.
		if err := s.userRepo.ResetLoginLockout(ctx, user.ID); err != nil {
			s.logger.Warn("failed to clear expired login lockout", zap.Error(err))
		}
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
	}

	if !domain.CheckPasswordHash(req.Password, user.PasswordHash) {
		s.recordFailedLogin(ctx, user)
		return nil, apperrors.New("invalid credentials")
	}

	if user.FailedLoginAttempts > 0 {
		if err := s.userRepo.ResetLoginLockout(ctx, user.ID); err != nil {
			s.logger.Warn("failed to reset login lockout state after successful login", zap.Error(err))
		}
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

// recordFailedLogin increments the account's failed-login counter and, once
// maxFailedLoginAttempts is reached, sets a locked-until cooldown. Persistence failures are
// logged but not surfaced to the caller — Login already returns "invalid credentials" for the
// bad password itself, and a lockout-bookkeeping error shouldn't turn into a 500 on top of that.
func (s *userService) recordFailedLogin(ctx context.Context, user *domain.User) {
	attempts := user.FailedLoginAttempts + 1

	var lockedUntil *time.Time
	if attempts >= maxFailedLoginAttempts {
		until := time.Now().Add(accountLockDuration)
		lockedUntil = &until
	}

	if err := s.userRepo.SetLoginLockoutState(ctx, user.ID, attempts, lockedUntil); err != nil {
		s.logger.Warn("failed to record failed login attempt", zap.Error(err))
	}
}

func (s *userService) ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if !apperrors.IsNotFound(err) {
			s.logger.Warn("failed to lookup user for password reset", zap.Error(err))
		}
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	tokenString := hex.EncodeToString(tokenBytes)

	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.userRepo.CreateResetToken(ctx, resetToken); err != nil {
		return err
	}

	return s.emailService.SendPasswordResetEmail(user.Email, tokenString)
}

func (s *userService) ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error {
	resetToken, err := s.userRepo.GetResetTokenByToken(ctx, req.Token)
	if err != nil {
		return apperrors.New("invalid token")
	}

	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return apperrors.New("token expired or already used")
	}

	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return err
	}

	hashedPassword, err := domain.HashPassword(req.Password)
	if err != nil {
		return err
	}

	return s.userRepo.WithTypedTransaction(ctx, func(txRepo repository.UserRepository) error {
		if err := txRepo.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
			return err
		}
		if err := txRepo.MarkResetTokenUsed(ctx, resetToken.ID); err != nil {
			return err
		}
		// Invalidate any JWT issued before this moment so a token obtained
		// prior to the reset (e.g. by an attacker) can't keep working.
		return txRepo.SetTokenRevocation(ctx, user.ID, time.Now())
	})
}

// VerifyEmail redeems a verification token and marks the owning user as
// verified. Tokens are single-use and expire after verificationTokenTTL.
func (s *userService) VerifyEmail(ctx context.Context, req *domain.VerifyEmailRequest) error {
	verificationToken, err := s.userRepo.GetVerificationTokenByToken(ctx, req.Token)
	if err != nil {
		return apperrors.New("invalid token")
	}

	if verificationToken.Used || time.Now().After(verificationToken.ExpiresAt) {
		return apperrors.New("token expired or already used")
	}

	if err := s.userRepo.MarkEmailVerified(ctx, verificationToken.UserID); err != nil {
		return err
	}

	return s.userRepo.MarkVerificationTokenUsed(ctx, verificationToken.ID)
}

// ResendVerification issues a fresh verification token for an unverified
// user, subject to a per-user cooldown so a single resend button can't be
// used to spam the mailbox. Like ForgotPassword, the "no user" and
// "already verified" cases return nil silently to avoid confirming whether
// an email is registered. The cooldown case also returns nil (rather than an
// error) so it can't be used to distinguish "registered + unverified" from
// those cases by response shape; the request is simply a no-op.
func (s *userService) ResendVerification(ctx context.Context, req *domain.ResendVerificationRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if !apperrors.IsNotFound(err) {
			s.logger.Warn("failed to lookup user for resend verification", zap.Error(err))
		}
		return nil
	}

	if user.IsEmailVerified() {
		return nil
	}

	latest, err := s.userRepo.GetLatestVerificationToken(ctx, user.ID)
	if err == nil && latest != nil && time.Since(latest.CreatedAt) < resendVerificationCooldown {
		// Cooldown hit: no-op, not an error. Returning nil here (same as the
		// "no user" / "already verified" branches above) keeps the response
		// from leaking which of those three cases occurred.
		return nil
	}

	tokenString, err := s.createVerificationToken(ctx, user.ID)
	if err != nil {
		return err
	}

	return s.emailService.SendVerificationEmail(user.Email, tokenString)
}

// IsEmailVerified is used by the RequireVerified middleware to gate
// state-changing requests from users who haven't confirmed their email yet.
func (s *userService) IsEmailVerified(ctx context.Context, userID string) (bool, error) {
	return s.userRepo.IsEmailVerified(ctx, userID)
}

func (s *userService) generateToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"iss":     s.jwtIssuer,
		"aud":     s.jwtAudience,
		"iat":     now.Unix(),
		"nbf":     now.Unix(),
		"exp":     now.Add(s.jwtDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken is unused by any request path today (AuthRequired middleware
// has its own equivalent validator, since it also needs to check revocation)
// but is kept hardened the same way — iss/aud/nbf checked, same as the
// middleware — so it can't become a weaker bypass if it's ever wired up.
func (s *userService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer(s.jwtIssuer), jwt.WithAudience(s.jwtAudience))
}

func (s *userService) Delete(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if apperrors.IsNotFound(err) {
			return apperrors.ErrNotFound.Wrap("user not found")
		}
		return err
	}
	return s.userRepo.WithTypedTransaction(ctx, func(txRepo repository.UserRepository) error {
		// Revoke first: token_revocations has no FK to users, so it survives
		// the delete and keeps rejecting any JWT issued before this moment.
		if err := txRepo.SetTokenRevocation(ctx, user.ID, time.Now()); err != nil {
			return err
		}
		return txRepo.Delete(ctx, user.ID)
	})
}

func (s *userService) ListAll(ctx context.Context) ([]domain.UserSummary, error) {
	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Project to a non-PII summary so the list endpoint cannot be used to
	// enumerate every member's email address.
	summaries := make([]domain.UserSummary, len(users))
	for i, u := range users {
		summaries[i] = domain.UserSummary{
			ID:        u.ID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		}
	}
	return summaries, nil
}
