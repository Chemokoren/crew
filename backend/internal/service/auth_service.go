package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
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
	logger   *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	crewRepo repository.CrewRepository,
	jwtManager *jwt.Manager,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		crewRepo: crewRepo,
		jwt:      jwtManager,
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

	// 5. If CREW role, create crew member profile
	var crewMember *models.CrewMember
	if input.Role == types.RoleCrewUser {
		crewID, err := s.crewRepo.NextCrewID(ctx)
		if err != nil {
			return nil, fmt.Errorf("generate crew id: %w", err)
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

		if err := s.crewRepo.Create(ctx, crewMember); err != nil {
			return nil, fmt.Errorf("create crew member: %w", err)
		}

		user.CrewMemberID = &crewMember.ID
	}

	// 6. Create user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 7. Generate tokens
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
