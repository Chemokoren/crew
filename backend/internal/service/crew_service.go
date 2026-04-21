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
	crewRepo repository.CrewRepository
	idp      identity.Provider
	logger   *slog.Logger
}

// NewCrewService creates a new CrewService.
func NewCrewService(crewRepo repository.CrewRepository, idp identity.Provider, logger *slog.Logger) *CrewService {
	return &CrewService{crewRepo: crewRepo, idp: idp, logger: logger}
}

// CreateCrewInput holds the data for creating a crew member.
type CreateCrewInput struct {
	NationalID string          `json:"national_id" validate:"required"`
	FirstName  string          `json:"first_name" validate:"required"`
	LastName   string          `json:"last_name" validate:"required"`
	Role       models.CrewRole `json:"role" validate:"required,oneof=DRIVER CONDUCTOR RIDER OTHER"`
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
		KYCStatus:  models.KYCPending,
		IsActive:   true,
	}

	if err := s.crewRepo.Create(ctx, crew); err != nil {
		return nil, fmt.Errorf("create crew member: %w", err)
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
	}

	if err := s.crewRepo.Update(ctx, crew); err != nil {
		return nil, fmt.Errorf("update kyc: %w", err)
	}

	s.logger.Info("crew KYC updated",
		slog.String("crew_id", crew.CrewID),
		slog.String("kyc_status", string(crew.KYCStatus)),
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
