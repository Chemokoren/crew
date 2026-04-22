package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and user registration.
type AuthService struct {
	userRepo repository.UserRepository
	crewRepo repository.CrewRepository
	jwt      *jwt.Manager
	txMgr    *database.TxManager
	logger   *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	crewRepo repository.CrewRepository,
	jwtManager *jwt.Manager,
	txMgr *database.TxManager,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		crewRepo: crewRepo,
		jwt:      jwtManager,
		txMgr:    txMgr,
		logger:   logger,
	}
}

// RegisterInput holds the data required to register a new user.
type RegisterInput struct {
	Phone    string           `json:"phone" validate:"required,e164"`
	Email    string           `json:"email" validate:"omitempty,email"`
	Password string           `json:"password" validate:"required,min=8"`
	Role     types.SystemRole `json:"role" validate:"required"`

	// Optional: create a crew profile at the same time
	FirstName  string          `json:"first_name" validate:"required_if=Role CREW"`
	LastName   string          `json:"last_name" validate:"required_if=Role CREW"`
	NationalID string          `json:"national_id" validate:"required_if=Role CREW"`
	CrewRole   models.CrewRole `json:"crew_role" validate:"required_if=Role CREW,omitempty,oneof=DRIVER CONDUCTOR RIDER OTHER"`
}

// RegisterResult holds the output of a successful registration.
type RegisterResult struct {
	User       *models.User       `json:"user"`
	CrewMember *models.CrewMember `json:"crew_member,omitempty"`
	Tokens     *jwt.TokenPair     `json:"tokens"`
}

// Register creates a new user account and optionally a crew member profile.
// The entire operation runs inside a database transaction to prevent orphan records.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// 1. Validate role
	if !input.Role.IsValid() {
		return nil, fmt.Errorf("%w: invalid system role %q", ErrValidation, input.Role)
	}

	// 2. Check phone uniqueness
	existing, err := s.userRepo.GetByPhone(ctx, input.Phone)
	if err != nil && err != ErrNotFound {
		return nil, fmt.Errorf("check phone: %w", err)
	}
	if existing != nil {
		return nil, ErrPhoneAlreadyExists
	}

	// 3. Hash password (cost 12 — balance of security and performance)
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 4. Build user
	user := &models.User{
		Phone:        input.Phone,
		Email:        input.Email,
		PasswordHash: string(hash),
		SystemRole:   input.Role,
		IsActive:     true,
	}

	var crewMember *models.CrewMember

	// 5. Execute crew + user creation inside a single transaction.
	// If txMgr is nil (e.g., in tests with mock repos), run without transaction.
	createFn := func(txCtx context.Context) error {
		// If CREW role, create crew member profile first
		if input.Role == types.RoleCrewUser {
			crewID, err := s.crewRepo.NextCrewID(txCtx)
			if err != nil {
				return fmt.Errorf("generate crew id: %w", err)
			}

			crewMember = &models.CrewMember{
				CrewID:     crewID,
				NationalID: input.NationalID,
				FirstName:  input.FirstName,
				LastName:   input.LastName,
				Role:       input.CrewRole,
				KYCStatus:  models.KYCPending,
				IsActive:   true,
			}

			if err := s.crewRepo.Create(txCtx, crewMember); err != nil {
				return fmt.Errorf("create crew member: %w", err)
			}

			user.CrewMemberID = &crewMember.ID
		}

		// Create user
		if err := s.userRepo.Create(txCtx, user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		return nil
	}

	if s.txMgr != nil {
		if err := s.txMgr.RunInTx(ctx, createFn); err != nil {
			return nil, err
		}
	} else {
		// No transaction manager (test/mock mode) — run directly
		if err := createFn(ctx); err != nil {
			return nil, err
		}
	}

	// 6. Generate tokens (outside transaction — not a DB operation)
	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.SaccoID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	s.logger.Info("user registered",
		slog.String("user_id", user.ID.String()),
		slog.String("phone", user.Phone),
		slog.String("role", string(user.SystemRole)),
	)

	return &RegisterResult{
		User:       user,
		CrewMember: crewMember,
		Tokens:     tokens,
	}, nil
}

// LoginInput holds credentials for authentication.
type LoginInput struct {
	Phone    string `json:"phone" validate:"required,e164"`
	Password string `json:"password" validate:"required"`
}

// LoginResult holds the output of a successful login.
type LoginResult struct {
	User   *models.User   `json:"user"`
	Tokens *jwt.TokenPair `json:"tokens"`
}

// Login authenticates a user by phone and password.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	// 1. Look up user
	user, err := s.userRepo.GetByPhone(ctx, input.Phone)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 2. Check account status
	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	// 3. Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 4. Update last login
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Warn("failed to update last_login_at", slog.String("user_id", user.ID.String()))
		// Non-fatal — continue login
	}

	// 5. Generate tokens
	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.SaccoID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	s.logger.Info("user logged in",
		slog.String("user_id", user.ID.String()),
		slog.String("phone", user.Phone),
	)

	return &LoginResult{
		User:   user,
		Tokens: tokens,
	}, nil
}

// RefreshToken generates a new token pair from a valid refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.jwt.VerifyToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.SaccoID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, nil
}

// GetUserByID retrieves a user by their UUID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// DisableAccount deactivates a user account (admin only).
func (s *AuthService) DisableAccount(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.IsActive = false
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("disable account: %w", err)
	}
	s.logger.Info("account disabled", slog.String("user_id", userID.String()))
	return nil
}

// EnableAccount re-activates a disabled user account (admin only).
func (s *AuthService) EnableAccount(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.IsActive = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("enable account: %w", err)
	}
	s.logger.Info("account enabled", slog.String("user_id", userID.String()))
	return nil
}

// AdminResetPassword resets a user's password (admin-initiated, no old password required).
func (s *AuthService) AdminResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	s.logger.Info("password reset by admin", slog.String("user_id", userID.String()))
	return nil
}

// ChangePassword allows an authenticated user to change their own password.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	s.logger.Info("password changed", slog.String("user_id", userID.String()))
	return nil
}

// SystemStats holds system-wide statistics.
type SystemStats struct {
	TotalUsers  int64 `json:"total_users"`
	ActiveUsers int64 `json:"active_users"`
	TotalCrew   int64 `json:"total_crew"`
}

// GetSystemStats returns system-wide statistics for the admin dashboard.
func (s *AuthService) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	total, active, err := s.userRepo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	crewCount, err := s.crewRepo.Count(ctx)
	if err != nil {
		crewCount = 0 // non-fatal
	}
	return &SystemStats{
		TotalUsers:  total,
		ActiveUsers: active,
		TotalCrew:   crewCount,
	}, nil
}

