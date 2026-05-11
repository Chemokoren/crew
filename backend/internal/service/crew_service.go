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
)

// CrewService manages crew member business logic.
type CrewService struct {
	crewRepo       repository.CrewRepository
	membershipRepo repository.MembershipRepository
	idp            identity.Provider
	logger         *slog.Logger
}

// NewCrewService creates a new CrewService.
func NewCrewService(crewRepo repository.CrewRepository, membershipRepo repository.MembershipRepository, idp identity.Provider, logger *slog.Logger) *CrewService {
	return &CrewService{crewRepo: crewRepo, membershipRepo: membershipRepo, idp: idp, logger: logger}
}

// CreateCrewInput holds the data for creating a crew member.
type CreateCrewInput struct {
	NationalID     string          `json:"national_id" validate:"required"`
	FirstName      string          `json:"first_name" validate:"required"`
	LastName       string          `json:"last_name" validate:"required"`
	Role           models.CrewRole `json:"role" validate:"required"`
	JobTypeID      *uuid.UUID      `json:"job_type_id,omitempty"`
	JobTitle       string          `json:"job_title,omitempty"`
	OrganizationID *uuid.UUID      `json:"organization_id,omitempty"` // Auto-populated from JWT claims
}

// CreateCrewMember creates a new crew member with an auto-generated crew ID.
func (s *CrewService) CreateCrewMember(ctx context.Context, input CreateCrewInput) (*models.CrewMember, error) {
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

	// Auto-create org membership if an org ID is provided (from JWT claims)
	if input.OrganizationID != nil {
		membership := &models.CrewSACCOMembership{
			CrewMemberID:   crew.ID,
			OrganizationID: *input.OrganizationID,
			RoleInOrg:      models.OrgRoleMember,
			JoinedAt:       time.Now(),
			IsActive:       true,
		}
		if err := s.membershipRepo.Create(ctx, membership); err != nil {
			s.logger.Warn("failed to create org membership for new crew member",
				slog.String("crew_id", crew.CrewID),
				slog.String("org_id", input.OrganizationID.String()),
				slog.Any("err", err),
			)
		}
	}

	s.logger.Info("crew member created",
		slog.String("crew_id", crew.CrewID),
		slog.String("name", crew.FullName()),
	)

	return crew, nil
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
func (s *CrewService) BulkImport(ctx context.Context, inputs []CreateCrewInput) (*BulkImportResult, error) {
	var crewMembers []models.CrewMember
	// Capture the org ID from the first input (all share the same org from JWT)
	var orgID *uuid.UUID
	if len(inputs) > 0 && inputs[0].OrganizationID != nil {
		orgID = inputs[0].OrganizationID
	}

	for _, input := range inputs {
		crewID, err := s.crewRepo.NextCrewID(ctx)
		if err != nil {
			return nil, fmt.Errorf("generate crew id: %w", err)
		}

		crewMembers = append(crewMembers, models.CrewMember{
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
	}

	bulkErrors, err := s.crewRepo.BulkCreate(ctx, crewMembers)
	if err != nil {
		return nil, fmt.Errorf("bulk create: %w", err)
	}

	// Build a set of failed indices for quick lookup
	failedIndices := make(map[int]bool, len(bulkErrors))
	for _, be := range bulkErrors {
		failedIndices[be.Index] = true
	}

	// Create org memberships for each successfully imported member
	if orgID != nil && s.membershipRepo != nil {
		for i, cm := range crewMembers {
			if failedIndices[i] {
				continue
			}
			membership := &models.CrewSACCOMembership{
				CrewMemberID:   cm.ID,
				OrganizationID: *orgID,
				RoleInOrg:      models.OrgRoleMember,
				JoinedAt:       time.Now(),
				IsActive:       true,
			}
			if err := s.membershipRepo.Create(ctx, membership); err != nil {
				s.logger.Warn("bulk import: failed to create org membership",
					slog.String("crew_id", cm.CrewID),
					slog.Any("err", err),
				)
			}
		}
	}

	imported := len(crewMembers) - len(bulkErrors)
	s.logger.Info("bulk import complete",
		slog.Int("total", len(crewMembers)),
		slog.Int("imported", imported),
		slog.Int("errors", len(bulkErrors)),
	)

	return &BulkImportResult{
		Imported: imported,
		Errors:   bulkErrors,
	}, nil
}

