package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/identity"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

// CrewService manages crew member business logic.
type CrewService struct {
	crewRepo       repository.CrewRepository
	membershipRepo repository.MembershipRepository
	userRepo       repository.UserRepository
	notifSvc       *NotificationService
	idp            identity.Provider
	logger         *slog.Logger
}

// NewCrewService creates a new CrewService.
func NewCrewService(crewRepo repository.CrewRepository, membershipRepo repository.MembershipRepository, idp identity.Provider, logger *slog.Logger) *CrewService {
	return &CrewService{crewRepo: crewRepo, membershipRepo: membershipRepo, idp: idp, logger: logger}
}

// WithUserRepo injects user repository for employee auto-registration.
func (s *CrewService) WithUserRepo(repo repository.UserRepository) { s.userRepo = repo }

// WithNotificationSvc injects notification service for welcome SMS.
func (s *CrewService) WithNotificationSvc(svc *NotificationService) { s.notifSvc = svc }

// CreateCrewInput holds the data for creating a crew member.
type CreateCrewInput struct {
	NationalID     string          `json:"national_id" validate:"required"`
	Phone          string          `json:"phone,omitempty"`
	FirstName      string          `json:"first_name" validate:"required"`
	LastName       string          `json:"last_name" validate:"required"`
	Role           models.CrewRole `json:"role" validate:"required"`
	JobTypeID      *uuid.UUID      `json:"job_type_id,omitempty"`
	JobTitle       string          `json:"job_title,omitempty"`
	OrganizationID *uuid.UUID      `json:"organization_id,omitempty"` // Auto-populated from JWT claims
}

// LookupResult holds the outcome of a national ID lookup.
type LookupResult struct {
	Found      bool                `json:"found"`
	CrewMember *models.CrewMember  `json:"crew_member,omitempty"`
	Linked     bool                `json:"linked"`     // Already linked to this org
}

// LookupByNationalID checks if an employee is already registered by their national ID.
// If orgID is provided, also checks if they are already linked to that organization.
func (s *CrewService) LookupByNationalID(ctx context.Context, nationalID string, orgID *uuid.UUID) (*LookupResult, error) {
	crew, err := s.crewRepo.GetByNationalID(ctx, nationalID)
	if err != nil {
		// Not found — new employee
		return &LookupResult{Found: false}, nil
	}

	result := &LookupResult{
		Found:      true,
		CrewMember: crew,
	}

	// Check if already linked to the requesting org
	if orgID != nil && s.membershipRepo != nil {
		membership, err := s.membershipRepo.GetActive(ctx, crew.ID, *orgID)
		if err == nil && membership != nil {
			result.Linked = true
		}
	}

	return result, nil
}

// CreateCrewMember creates a new crew member with an auto-generated crew ID.
// If the employee already exists (by national ID), it links them to the org instead.
// For new employees, it also creates a User account and sends a welcome SMS.
func (s *CrewService) CreateCrewMember(ctx context.Context, input CreateCrewInput) (*models.CrewMember, error) {
	// Step 1: Check if this employee already exists by national ID
	existing, _ := s.crewRepo.GetByNationalID(ctx, input.NationalID)
	if existing != nil {
		// Employee exists — just link them to this org
		if input.OrganizationID != nil {
			if err := s.linkToOrg(ctx, existing, *input.OrganizationID); err != nil {
				return nil, fmt.Errorf("link existing employee to org: %w", err)
			}
		}
		s.logger.Info("existing employee linked to org",
			slog.String("crew_id", existing.CrewID),
			slog.String("name", existing.FullName()),
		)
		return existing, nil
	}

	// Step 2: Create new crew member
	crewID, err := s.crewRepo.NextCrewID(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate crew id: %w", err)
	}

	crew := &models.CrewMember{
		CrewID:     crewID,
		NationalID: input.NationalID,
		FirstName:  input.FirstName,
		LastName:   input.LastName,
		Role:       input.Role,
		JobTypeID:  input.JobTypeID,
		JobTitle:   input.JobTitle,
		KYCStatus:  models.KYCPending,
		IsActive:   true,
	}

	if err := s.crewRepo.Create(ctx, crew); err != nil {
		return nil, fmt.Errorf("create crew member: %w", err)
	}

	// Step 3: Link to org
	if input.OrganizationID != nil {
		if err := s.linkToOrg(ctx, crew, *input.OrganizationID); err != nil {
			s.logger.Warn("failed to create org membership for new crew member",
				slog.String("crew_id", crew.CrewID),
				slog.Any("err", err),
			)
		}
	}

	// Step 4: Create User account + send welcome SMS
	s.registerEmployeeAccount(ctx, crew, input.Phone, input.NationalID)

	s.logger.Info("crew member created",
		slog.String("crew_id", crew.CrewID),
		slog.String("name", crew.FullName()),
	)

	return crew, nil
}

// linkToOrg creates an org membership if one doesn't already exist.
func (s *CrewService) linkToOrg(ctx context.Context, crew *models.CrewMember, orgID uuid.UUID) error {
	// Check if already linked
	if s.membershipRepo != nil {
		existing, err := s.membershipRepo.GetActive(ctx, crew.ID, orgID)
		if err == nil && existing != nil {
			s.logger.Info("employee already linked to org",
				slog.String("crew_id", crew.CrewID),
				slog.String("org_id", orgID.String()),
			)
			return nil // Already linked
		}
	}

	membership := &models.CrewSACCOMembership{
		CrewMemberID:   crew.ID,
		OrganizationID: orgID,
		RoleInOrg:      models.OrgRoleMember,
		JoinedAt:       time.Now(),
		IsActive:       true,
	}
	return s.membershipRepo.Create(ctx, membership)
}

// registerEmployeeAccount creates a User account for the crew member if one doesn't exist.
// Default username = national ID, default password = national ID.
// Sends a welcome SMS with login details.
func (s *CrewService) registerEmployeeAccount(ctx context.Context, crew *models.CrewMember, phone, nationalID string) {
	if s.userRepo == nil || phone == "" {
		return
	}

	// Check if user already exists for this crew member
	existingUser, _ := s.userRepo.GetByCrewMemberID(ctx, crew.ID)
	if existingUser != nil {
		return // Already has an account
	}

	// Check if phone is already taken
	existingByPhone, _ := s.userRepo.GetByPhone(ctx, phone)
	if existingByPhone != nil {
		s.logger.Warn("phone already registered to another user, skipping account creation",
			slog.String("phone", phone),
			slog.String("crew_id", crew.CrewID),
		)
		return
	}

	// Hash default password (national ID)
	hash, err := bcrypt.GenerateFromPassword([]byte(nationalID), 12)
	if err != nil {
		s.logger.Error("failed to hash default password", slog.Any("err", err))
		return
	}

	user := &models.User{
		Phone:        phone,
		PasswordHash: string(hash),
		SystemRole:   types.RoleEmployee,
		CrewMemberID: &crew.ID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create employee user account",
			slog.String("crew_id", crew.CrewID),
			slog.Any("err", err),
		)
		return
	}

	s.logger.Info("employee user account created",
		slog.String("crew_id", crew.CrewID),
		slog.String("phone", phone),
	)

	// Send welcome SMS (async — don't block the API response)
	if s.notifSvc != nil {
		msg := fmt.Sprintf(
			"Welcome to Crew! Your account has been created. "+
				"Username: %s, Default Password: %s. "+
				"Please change your password on first login.",
			nationalID, nationalID,
		)
		go func() {
			// Use a detached context so the SMS isn't cancelled when the HTTP request ends
			smsCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.notifSvc.SendSMSToPhone(smsCtx, phone, msg); err != nil {
				s.logger.Warn("failed to send welcome SMS",
					slog.String("phone", phone),
					slog.Any("err", err),
				)
			} else {
				s.logger.Info("welcome SMS sent",
					slog.String("phone", phone),
					slog.String("crew_id", crew.CrewID),
				)
			}
		}()
	}
}

// ResendCredentials looks up the user account for a crew member and resends their default credentials.
func (s *CrewService) ResendCredentials(ctx context.Context, id uuid.UUID) error {
	if s.userRepo == nil || s.notifSvc == nil {
		return fmt.Errorf("user repository or notification service not configured")
	}

	crew, err := s.crewRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get crew member: %w", err)
	}

	user, err := s.userRepo.GetByCrewMemberID(ctx, crew.ID)
	if err != nil {
		return fmt.Errorf("get user for crew member: %w", err)
	}

	if user.Phone == "" {
		return fmt.Errorf("user has no phone number registered")
	}

	msg := fmt.Sprintf(
		"Welcome to Crew! Your account has been created. "+
			"Username: %s, Default Password: %s. "+
			"Please change your password on first login.",
		crew.NationalID, crew.NationalID,
	)

	// Send synchronously here so the handler can return an error if it fails
	if err := s.notifSvc.SendSMSToPhone(ctx, user.Phone, msg); err != nil {
		s.logger.Error("failed to resend welcome SMS",
			slog.String("phone", user.Phone),
			slog.Any("err", err),
		)
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	s.logger.Info("credentials resent successfully",
		slog.String("crew_id", crew.CrewID),
		slog.String("phone", user.Phone),
	)

	return nil
}

// GetCrewMember retrieves a crew member by UUID.
func (s *CrewService) GetCrewMember(ctx context.Context, id uuid.UUID) (*models.CrewMember, error) {
	return s.crewRepo.GetByID(ctx, id)
}

// UpdateKYCInput holds the data for updating KYC status.
type UpdateKYCInput struct {
	CrewMemberID uuid.UUID
	Status       models.KYCStatus `json:"kyc_status" validate:"required,oneof=PENDING VERIFIED REJECTED"`
	SerialNumber string
	Reason       string // Reason for status change (e.g. why unverified)
}

// UpdateKYCStatus changes a crew member's KYC status.
func (s *CrewService) UpdateKYCStatus(ctx context.Context, input UpdateKYCInput) (*models.CrewMember, error) {
	crew, err := s.crewRepo.GetByID(ctx, input.CrewMemberID)
	if err != nil {
		return nil, err
	}

	crew.KYCStatus = input.Status
	if input.Status == models.KYCVerified {
		// Automatic IPRS verification if IDP is configured
		if s.idp != nil && input.SerialNumber != "" {
			details, err := s.idp.VerifyCitizen(ctx, identity.VerifyRequest{
				IDNumber:     crew.NationalID,
				SerialNumber: input.SerialNumber,
			})
			if err != nil {
				return nil, fmt.Errorf("iprs verification failed: %w", err)
			}
			if !details.Verified {
				return nil, fmt.Errorf("iprs verification failed: invalid credentials")
			}
		}

		now := time.Now()
		crew.KYCVerifiedAt = &now
	} else {
		// Clear verification timestamp when unverifying or rejecting
		crew.KYCVerifiedAt = nil
	}

	if err := s.crewRepo.Update(ctx, crew); err != nil {
		return nil, fmt.Errorf("update kyc: %w", err)
	}

	s.logger.Info("crew KYC updated",
		slog.String("crew_id", crew.CrewID),
		slog.String("kyc_status", string(crew.KYCStatus)),
		slog.String("reason", input.Reason),
	)

	return crew, nil
}

// ListCrewMembers lists crew members with optional filters.
func (s *CrewService) ListCrewMembers(ctx context.Context, filter repository.CrewFilter, page, perPage int) ([]models.CrewMember, int64, error) {
	return s.crewRepo.List(ctx, filter, page, perPage)
}

// DeactivateCrewMember soft-deactivates a crew member.
func (s *CrewService) DeactivateCrewMember(ctx context.Context, id uuid.UUID) error {
	crew, err := s.crewRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	crew.IsActive = false
	if err := s.crewRepo.Update(ctx, crew); err != nil {
		return fmt.Errorf("deactivate crew: %w", err)
	}

	s.logger.Info("crew member deactivated", slog.String("crew_id", crew.CrewID))
	return nil
}

// VerifyNationalID validates a crew member's national ID via IPRS.
func (s *CrewService) VerifyNationalID(ctx context.Context, id uuid.UUID, serialNumber string) (*models.CrewMember, error) {
	if s.idp == nil {
		return nil, fmt.Errorf("identity provider not configured")
	}

	crew, err := s.crewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	details, err := s.idp.VerifyCitizen(ctx, identity.VerifyRequest{
		IDNumber:     crew.NationalID,
		SerialNumber: serialNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("iprs verify: %w", err)
	}

	if details.Verified {
		crew.KYCStatus = models.KYCVerified
		now := time.Now()
		crew.KYCVerifiedAt = &now
		if err := s.crewRepo.Update(ctx, crew); err != nil {
			return nil, fmt.Errorf("update kyc status: %w", err)
		}
	}

	return crew, nil
}

// GetByNationalID searches for a crew member by national ID.
func (s *CrewService) GetByNationalID(ctx context.Context, nationalID string) (*models.CrewMember, error) {
	return s.crewRepo.GetByNationalID(ctx, nationalID)
}

// BulkImportResult holds the outcome of a bulk import operation.
type BulkImportResult struct {
	Imported int                    `json:"imported"`
	Errors   []repository.BulkError `json:"errors,omitempty"`
}

// BulkImport creates multiple crew members in a single operation.
// For each member, it checks if they already exist (by national ID) and links them
// to the org. For new members, it creates the crew record, user account, and sends SMS.
func (s *CrewService) BulkImport(ctx context.Context, inputs []CreateCrewInput) (*BulkImportResult, error) {
	var newCrewMembers []models.CrewMember
	var newInputs []CreateCrewInput // parallel array for phone lookup
	var linkedCount int

	// Capture the org ID from the first input (all share the same org from JWT)
	var orgID *uuid.UUID
	if len(inputs) > 0 && inputs[0].OrganizationID != nil {
		orgID = inputs[0].OrganizationID
	}

	// Phase 1: Check each input — link existing or queue for bulk creation
	for _, input := range inputs {
		existing, _ := s.crewRepo.GetByNationalID(ctx, input.NationalID)
		if existing != nil {
			// Already registered — just link to org
			if orgID != nil {
				if err := s.linkToOrg(ctx, existing, *orgID); err != nil {
					s.logger.Warn("bulk import: failed to link existing employee",
						slog.String("crew_id", existing.CrewID),
						slog.Any("err", err),
					)
				}
			}
			linkedCount++
			continue
		}

		crewID, err := s.crewRepo.NextCrewID(ctx)
		if err != nil {
			return nil, fmt.Errorf("generate crew id: %w", err)
		}

		newCrewMembers = append(newCrewMembers, models.CrewMember{
			CrewID:     crewID,
			NationalID: input.NationalID,
			FirstName:  input.FirstName,
			LastName:   input.LastName,
			Role:       input.Role,
			JobTypeID:  input.JobTypeID,
			JobTitle:   input.JobTitle,
			KYCStatus:  models.KYCPending,
			IsActive:   true,
		})
		newInputs = append(newInputs, input)
	}

	// Phase 2: Bulk create new members
	var bulkErrors []repository.BulkError
	if len(newCrewMembers) > 0 {
		var err error
		bulkErrors, err = s.crewRepo.BulkCreate(ctx, newCrewMembers)
		if err != nil {
			return nil, fmt.Errorf("bulk create: %w", err)
		}
	}

	// Build a set of failed indices for quick lookup
	failedIndices := make(map[int]bool, len(bulkErrors))
	for _, be := range bulkErrors {
		failedIndices[be.Index] = true
	}

	// Phase 3: Link to org + create user accounts for successfully imported members
	for i := range newCrewMembers {
		if failedIndices[i] {
			continue
		}
		if orgID != nil {
			if err := s.linkToOrg(ctx, &newCrewMembers[i], *orgID); err != nil {
				s.logger.Warn("bulk import: failed to create org membership",
					slog.String("crew_id", newCrewMembers[i].CrewID),
					slog.Any("err", err),
				)
			}
		}
		// Create user account + send welcome SMS
		s.registerEmployeeAccount(ctx, &newCrewMembers[i], newInputs[i].Phone, newInputs[i].NationalID)
	}

	imported := len(newCrewMembers) - len(bulkErrors) + linkedCount
	s.logger.Info("bulk import complete",
		slog.Int("total", len(inputs)),
		slog.Int("new_created", len(newCrewMembers)-len(bulkErrors)),
		slog.Int("existing_linked", linkedCount),
		slog.Int("errors", len(bulkErrors)),
	)

	return &BulkImportResult{
		Imported: imported,
		Errors:   bulkErrors,
	}, nil
}

