package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// SACCOService handles SACCO business logic.
type SACCOService struct {
	saccoRepo      repository.SACCORepository
	membershipRepo repository.MembershipRepository
	floatRepo      repository.SACCOFloatRepository
	logger         *slog.Logger
}

func NewSACCOService(
	saccoRepo repository.SACCORepository,
	membershipRepo repository.MembershipRepository,
	floatRepo repository.SACCOFloatRepository,
	logger *slog.Logger,
) *SACCOService {
	return &SACCOService{
		saccoRepo:      saccoRepo,
		membershipRepo: membershipRepo,
		floatRepo:      floatRepo,
		logger:         logger,
	}
}

// --- SACCO CRUD ---

type CreateSACCOInput struct {
	Name               string `json:"name" binding:"required"`
	RegistrationNumber string `json:"registration_number" binding:"required"`
	County             string `json:"county" binding:"required"`
	SubCounty          string `json:"sub_county"`
	ContactPhone       string `json:"contact_phone" binding:"required"`
	ContactEmail       string `json:"contact_email"`
}

func (s *SACCOService) CreateSACCO(ctx context.Context, input CreateSACCOInput) (*models.SACCO, error) {
	sacco := &models.SACCO{
		Name:               input.Name,
		RegistrationNumber: input.RegistrationNumber,
		County:             input.County,
		SubCounty:          input.SubCounty,
		ContactPhone:       input.ContactPhone,
		ContactEmail:       input.ContactEmail,
		Currency:           "KES",
		IsActive:           true,
	}

	if err := s.saccoRepo.Create(ctx, sacco); err != nil {
		return nil, fmt.Errorf("create sacco: %w", err)
	}

	s.logger.Info("SACCO created", slog.String("id", sacco.ID.String()), slog.String("name", sacco.Name))
	return sacco, nil
}

func (s *SACCOService) GetSACCO(ctx context.Context, id uuid.UUID) (*models.SACCO, error) {
	return s.saccoRepo.GetByID(ctx, id)
}

type UpdateSACCOInput struct {
	Name         *string `json:"name"`
	County       *string `json:"county"`
	SubCounty    *string `json:"sub_county"`
	ContactPhone *string `json:"contact_phone"`
	ContactEmail *string `json:"contact_email"`
}

func (s *SACCOService) UpdateSACCO(ctx context.Context, id uuid.UUID, input UpdateSACCOInput) (*models.SACCO, error) {
	sacco, err := s.saccoRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		sacco.Name = *input.Name
	}
	if input.County != nil {
		sacco.County = *input.County
	}
	if input.SubCounty != nil {
		sacco.SubCounty = *input.SubCounty
	}
	if input.ContactPhone != nil {
		sacco.ContactPhone = *input.ContactPhone
	}
	if input.ContactEmail != nil {
		sacco.ContactEmail = *input.ContactEmail
	}

	if err := s.saccoRepo.Update(ctx, sacco); err != nil {
		return nil, fmt.Errorf("update sacco: %w", err)
	}
	return sacco, nil
}

func (s *SACCOService) DeleteSACCO(ctx context.Context, id uuid.UUID) error {
	return s.saccoRepo.Delete(ctx, id)
}

func (s *SACCOService) ListSACCOs(ctx context.Context, page, perPage int, search string) ([]models.SACCO, int64, error) {
	return s.saccoRepo.List(ctx, page, perPage, search)
}

// --- Membership ---

type AddMemberInput struct {
	CrewMemberID uuid.UUID       `json:"crew_member_id" binding:"required"`
	SaccoID      uuid.UUID       `json:"sacco_id" binding:"required"`
	Role         models.SACCORole `json:"role_in_sacco"`
}

func (s *SACCOService) AddMember(ctx context.Context, input AddMemberInput) (*models.CrewSACCOMembership, error) {
	// Check if already a member
	existing, err := s.membershipRepo.GetActive(ctx, input.CrewMemberID, input.SaccoID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("crew member is already an active member of this SACCO")
	}

	role := input.Role
	if role == "" {
		role = models.SACCORoleMember
	}

	m := &models.CrewSACCOMembership{
		CrewMemberID: input.CrewMemberID,
		SaccoID:      input.SaccoID,
		RoleInSacco:  role,
		IsActive:     true,
	}

	if err := s.membershipRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	s.logger.Info("member added to SACCO",
		slog.String("crew_member_id", input.CrewMemberID.String()),
		slog.String("sacco_id", input.SaccoID.String()),
	)
	return m, nil
}

func (s *SACCOService) RemoveMember(ctx context.Context, membershipID uuid.UUID) error {
	m, err := s.membershipRepo.GetByID(ctx, membershipID)
	if err != nil {
		return err
	}
	m.IsActive = false
	return s.membershipRepo.Update(ctx, m)
}

func (s *SACCOService) ListMembers(ctx context.Context, saccoID uuid.UUID, page, perPage int) ([]models.CrewSACCOMembership, int64, error) {
	return s.membershipRepo.ListBySACCO(ctx, saccoID, page, perPage)
}

func (s *SACCOService) GetCrewMemberships(ctx context.Context, crewMemberID uuid.UUID) ([]models.CrewSACCOMembership, error) {
	return s.membershipRepo.ListByCrewMember(ctx, crewMemberID)
}

// --- Float ---

func (s *SACCOService) GetFloat(ctx context.Context, saccoID uuid.UUID) (*models.SACCOFloat, error) {
	return s.floatRepo.GetOrCreate(ctx, saccoID)
}

type FloatOperationInput struct {
	SaccoID        uuid.UUID `json:"sacco_id" binding:"required"`
	AmountCents    int64     `json:"amount_cents" binding:"required,min=1"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Reference      string    `json:"reference"`
}

func (s *SACCOService) CreditFloat(ctx context.Context, input FloatOperationInput) (*models.SACCOFloatTransaction, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, input.SaccoID)
	if err != nil {
		return nil, err
	}
	return s.floatRepo.CreditFloat(ctx, sf.ID, sf.Version, input.AmountCents, input.IdempotencyKey, input.Reference)
}

func (s *SACCOService) DebitFloat(ctx context.Context, input FloatOperationInput) (*models.SACCOFloatTransaction, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, input.SaccoID)
	if err != nil {
		return nil, err
	}
	return s.floatRepo.DebitFloat(ctx, sf.ID, sf.Version, input.AmountCents, input.IdempotencyKey, input.Reference)
}

func (s *SACCOService) ListFloatTransactions(ctx context.Context, saccoID uuid.UUID, filter repository.SACCOFloatFilter, page, perPage int) ([]models.SACCOFloatTransaction, int64, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, saccoID)
	if err != nil {
		return nil, 0, err
	}
	return s.floatRepo.GetTransactions(ctx, sf.ID, filter, page, perPage)
}
