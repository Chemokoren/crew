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
	"github.com/kibsoft/amy-mis/pkg/errs"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and user registration.
type AuthService struct {
	userRepo  repository.UserRepository
	crewRepo  repository.CrewRepository
	orgRepo   repository.OrganizationRepository
	jwt       *jwt.Manager
	txMgr     *database.TxManager
	tenantSvc *TenantService
	notifSvc  *NotificationService
	logger    *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	crewRepo repository.CrewRepository,
	jwtManager *jwt.Manager,
	txMgr *database.TxManager,
	logger *slog.Logger,
	opts ...AuthServiceOption,
) *AuthService {
	s := &AuthService{
		userRepo: userRepo,
		crewRepo: crewRepo,
		jwt:      jwtManager,
		txMgr:    txMgr,
		logger:   logger,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AuthServiceOption configures optional dependencies for AuthService.
type AuthServiceOption func(*AuthService)

// WithOrgRepo sets the organization repository (needed for SACCO_ADMIN registration).
func WithOrgRepo(repo repository.OrganizationRepository) AuthServiceOption {
	return func(s *AuthService) { s.orgRepo = repo }
}

// WithTenantSvc sets the tenant service (needed for industry bootstrap on registration).
func WithTenantSvc(svc *TenantService) AuthServiceOption {
	return func(s *AuthService) { s.tenantSvc = svc }
}

// WithNotifSvc sets the notification service (sends welcome SMS on registration).
func WithNotifSvc(svc *NotificationService) AuthServiceOption {
	return func(s *AuthService) { s.notifSvc = svc }
}

// RegisterInput holds the data required to register a new user.
type RegisterInput struct {
	Phone    string           `json:"phone" validate:"required,e164"`
	Email    string           `json:"email" validate:"omitempty,email"`
	Password string           `json:"password" validate:"required,min=8"`
	Role     types.SystemRole `json:"role" validate:"required"`

	// Employee profile fields (optional at registration — role/NationalID set via profile/KYC)
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	JobTypeID *uuid.UUID `json:"job_type_id,omitempty"`

	// Organization fields (required when role == EMPLOYER)
	OrganizationName   string              `json:"organization_name"`
	OrganizationRegNo  string              `json:"organization_reg_no"`
	OrganizationCounty string              `json:"organization_county"`
	OrganizationPhone  string              `json:"organization_phone"`
	IndustryType       models.IndustryType `json:"industry_type"`
}

// RegisterResult holds the output of a successful registration.
type RegisterResult struct {
	User       *models.User       `json:"user"`
	CrewMember *models.CrewMember `json:"crew_member,omitempty"`
	Tokens     *jwt.TokenPair     `json:"tokens"`
}

// Register creates a new user account and optionally a crew member profile.
// For SACCO_ADMIN role, it also creates an Organization and bootstraps industry defaults.
// The entire operation runs inside a database transaction to prevent orphan records.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// 1. Validate role
	if !input.Role.IsValid() {
		return nil, fmt.Errorf("%w: invalid system role %q", ErrValidation, input.Role)
	}

	// 1b. Validate organization fields for EMPLOYER
	if input.Role == types.RoleEmployer {
		if input.OrganizationName == "" {
			return nil, fmt.Errorf("%w: organization name is required for Employer", ErrValidation)
		}
		if input.OrganizationRegNo == "" {
			return nil, fmt.Errorf("%w: organization registration number is required", ErrValidation)
		}
		if input.OrganizationCounty == "" {
			return nil, fmt.Errorf("%w: organization county is required", ErrValidation)
		}
		if input.OrganizationPhone == "" {
			return nil, fmt.Errorf("%w: organization contact phone is required", ErrValidation)
		}
		if input.IndustryType == "" {
			input.IndustryType = models.IndustryGeneral
		}
		if s.orgRepo == nil {
			return nil, fmt.Errorf("%w: organization registration is not configured", ErrValidation)
		}
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
	var org *models.Organization

	// 5. Execute crew/org + user creation inside a single transaction.
	// If txMgr is nil (e.g., in tests with mock repos), run without transaction.
	createFn := func(txCtx context.Context) error {
		// If EMPLOYER, create the Organization first
		if input.Role == types.RoleEmployer && s.orgRepo != nil {
			tmpl := models.GetIndustryTemplate(input.IndustryType)
			org = &models.Organization{
				Name:               input.OrganizationName,
				RegistrationNumber: input.OrganizationRegNo,
				County:             input.OrganizationCounty,
				ContactPhone:       input.OrganizationPhone,
				ContactEmail:       input.Email,
				Currency:           "KES",
				IsActive:           true,
				IndustryType:       input.IndustryType,
				OrganizationType:   tmpl.OrgType,
				DisplayName:        input.OrganizationName,
			}
			// Set UI labels from template
			if tmpl.UILabels != nil {
				cfg := &models.TenantConfig{UILabels: tmpl.UILabels}
				_ = org.SetTenantConfig(cfg)
			}

			if err := s.orgRepo.Create(txCtx, org); err != nil {
				return fmt.Errorf("create organization: %w", err)
			}
			user.OrganizationID = &org.ID
		}

		// If EMPLOYEE role, create crew member profile
		// CrewRole defaults to OTHER — user sets their actual role on their profile
		// NationalID is collected later during the KYC/verification flow
		if input.Role == types.RoleEmployee {
			crewID, err := s.crewRepo.NextCrewID(txCtx)
			if err != nil {
				return fmt.Errorf("generate crew id: %w", err)
			}

			crewMember = &models.CrewMember{
				CrewID:    crewID,
				FirstName: input.FirstName,
				LastName:  input.LastName,
				Role:      models.RoleOther,
				JobTypeID: input.JobTypeID,
				KYCStatus: models.KYCPending,
				IsActive:  true,
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

	// 6. Bootstrap industry defaults (outside transaction — non-fatal if it fails)
	if org != nil && s.tenantSvc != nil {
		if _, err := s.tenantSvc.BootstrapIndustry(ctx, org.ID, input.IndustryType); err != nil {
			s.logger.Warn("failed to bootstrap industry for new org",
				slog.String("org_id", org.ID.String()),
				slog.String("industry", string(input.IndustryType)),
				slog.String("error", err.Error()),
			)
		}
	}

	// 7. Generate tokens (outside transaction — not a DB operation)
	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	s.logger.Info("user registered",
		slog.String("user_id", user.ID.String()),
		slog.String("phone", user.Phone),
		slog.String("role", string(user.SystemRole)),
	)

	// 8. Send welcome SMS (async — don't block response)
	if s.notifSvc != nil && user.Phone != "" {
		phone := user.Phone
		go func() {
			smsCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			msg := fmt.Sprintf(
				"Welcome to Crew! Your account has been created. "+
					"Login with your phone number %s. "+
					"Please keep your credentials safe.",
				phone,
			)
			if err := s.notifSvc.SendSMSToPhone(smsCtx, phone, msg); err != nil {
				s.logger.Warn("failed to send registration welcome SMS",
					slog.String("phone", phone),
					slog.Any("err", err),
				)
			}
		}()
	}

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
	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.OrganizationID)
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

	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.SystemRole, user.CrewMemberID, user.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokens, nil
}

// GetUserByID retrieves a user by their UUID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetUserByPhone retrieves a user by their phone number.
// Used by the USSD gateway to identify registered users.
func (s *AuthService) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return nil, ErrNotFound
	}
	return user, nil
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

// GetEnrichedProfile returns the user plus their crew member profile, KYC restrictions, and verification mode.
func (s *AuthService) GetEnrichedProfile(ctx context.Context, userID uuid.UUID) (*models.User, *models.CrewMember, []string, string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, nil, "", err
	}

	var crew *models.CrewMember
	if user.CrewMemberID != nil {
		crew, _ = s.crewRepo.GetByID(ctx, *user.CrewMemberID)
	}

	// Resolve KYC restrictions and verification mode from tenant config
	var restrictions []string
	kycMode := models.KYCModeUpload // default
	if crew != nil && user.OrganizationID != nil && s.orgRepo != nil {
		org, err := s.orgRepo.GetByID(ctx, *user.OrganizationID)
		if err == nil {
			cfg, err := org.GetTenantConfig()
			if err == nil {
				kycMode = cfg.ResolvedKYCMode()
				if cfg.KYCRequired && crew.KYCStatus != models.KYCVerified {
					restrictions = cfg.KYCRestrictedActions
					if len(restrictions) == 0 {
						restrictions = models.DefaultKYCRestrictedActions()
					}
				}
			}
		}
	}

	return user, crew, restrictions, kycMode, nil
}

// UpdateProfileInput holds the fields a user can update on their own profile.
type UpdateProfileInput struct {
	UserID    uuid.UUID
	Role      *models.CrewRole
	JobTitle  *string
	FirstName *string
	LastName  *string
}

// UpdateProfile updates the current user's crew member profile (job/specialization).
func (s *AuthService) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*models.CrewMember, error) {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	if user.CrewMemberID == nil {
		return nil, fmt.Errorf("%w: no crew profile linked to this account", ErrValidation)
	}

	crew, err := s.crewRepo.GetByID(ctx, *user.CrewMemberID)
	if err != nil {
		return nil, fmt.Errorf("load crew profile: %w", err)
	}

	// Apply partial updates
	if input.Role != nil {
		crew.Role = *input.Role
	}
	if input.JobTitle != nil {
		crew.JobTitle = *input.JobTitle
	}
	if input.FirstName != nil {
		crew.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		crew.LastName = *input.LastName
	}

	if err := s.crewRepo.Update(ctx, crew); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	s.logger.Info("profile updated",
		slog.String("user_id", input.UserID.String()),
		slog.String("crew_id", crew.CrewID),
	)

	return crew, nil
}

// InitiateKYCInput holds the data for a user-initiated KYC verification.
type InitiateKYCInput struct {
	UserID       uuid.UUID
	NationalID   string
	SerialNumber string
}

// InitiateKYC lets a user submit their National ID for IPRS verification.
// This updates the crew member's national ID and triggers verification.
func (s *AuthService) InitiateKYC(ctx context.Context, input InitiateKYCInput) (*models.CrewMember, error) {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	if user.CrewMemberID == nil {
		return nil, fmt.Errorf("%w: no crew profile linked to this account", ErrValidation)
	}

	crew, err := s.crewRepo.GetByID(ctx, *user.CrewMemberID)
	if err != nil {
		return nil, fmt.Errorf("load crew profile: %w", err)
	}

	// Store the national ID on the crew member record
	crew.NationalID = input.NationalID

	// Mark as pending while verification runs
	crew.KYCStatus = models.KYCPending
	crew.KYCVerifiedAt = nil

	if err := s.crewRepo.Update(ctx, crew); err != nil {
		return nil, fmt.Errorf("save national id: %w", err)
	}

	s.logger.Info("KYC initiated",
		slog.String("user_id", input.UserID.String()),
		slog.String("crew_id", crew.CrewID),
	)

	return crew, nil
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

// ListUsers returns a paginated list of users for admin management.
// The search parameter filters by phone or email (ILIKE).
func (s *AuthService) ListUsers(ctx context.Context, page, perPage int, search string) ([]models.User, int64, error) {
	return s.userRepo.List(ctx, page, perPage, search)
}

// SetPIN sets or updates the transaction PIN for a user identified by phone.
func (s *AuthService) SetPIN(ctx context.Context, phone, pin string) error {
	if len(pin) < 4 || len(pin) > 6 {
		return fmt.Errorf("%w: PIN must be 4-6 digits", errs.ErrValidation)
	}

	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash PIN: %w", err)
	}

	user.PINHash = string(hash)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("save PIN: %w", err)
	}

	s.logger.Info("transaction PIN set", slog.String("phone", phone))
	return nil
}

// VerifyPIN checks the provided PIN against the stored hash for a user identified by phone.
func (s *AuthService) VerifyPIN(ctx context.Context, phone, pin string) error {
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		return err
	}

	if user.PINHash == "" {
		return fmt.Errorf("%w: no PIN set for this account", errs.ErrValidation)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PINHash), []byte(pin)); err != nil {
		return errs.ErrInvalidCredentials
	}

	return nil
}
