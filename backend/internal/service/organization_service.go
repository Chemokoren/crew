package service

import (
	"time"
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// OrganizationService handles SACCO business logic.
type OrganizationService struct {
	saccoRepo      repository.OrganizationRepository
	membershipRepo repository.MembershipRepository
	floatRepo      repository.OrganizationFloatRepository
	auditSvc       *AuditService
	logger         *slog.Logger
}

func NewOrganizationService(
	saccoRepo repository.OrganizationRepository,
	membershipRepo repository.MembershipRepository,
	floatRepo repository.OrganizationFloatRepository,
	auditSvc *AuditService,
	logger *slog.Logger,
) *OrganizationService {
	return &OrganizationService{
		saccoRepo:      saccoRepo,
		membershipRepo: membershipRepo,
		floatRepo:      floatRepo,
		auditSvc:       auditSvc,
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

func (s *OrganizationService) CreateSACCO(ctx context.Context, input CreateSACCOInput) (*models.SACCO, error) {
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

func (s *OrganizationService) GetSACCO(ctx context.Context, id uuid.UUID) (*models.SACCO, error) {
	return s.saccoRepo.GetByID(ctx, id)
}

type UpdateSACCOInput struct {
	Name            *string              `json:"name"`
	County          *string              `json:"county"`
	SubCounty       *string              `json:"sub_county"`
	ContactPhone    *string              `json:"contact_phone"`
	ContactEmail    *string              `json:"contact_email"`
	IndustryType    *models.IndustryType `json:"industry_type"`
	DefaultLanguage *string              `json:"default_language"`
	DisplayName     *string              `json:"display_name"`
}

func (s *OrganizationService) UpdateSACCO(ctx context.Context, id uuid.UUID, input UpdateSACCOInput) (*models.SACCO, error) {
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
	if input.IndustryType != nil {
		sacco.IndustryType = *input.IndustryType
		// Auto-set org type from industry template
		tmpl := models.GetIndustryTemplate(*input.IndustryType)
		sacco.OrganizationType = tmpl.OrgType
		// Set UI labels in tenant config
		if tmpl.UILabels != nil {
			cfg, _ := sacco.GetTenantConfig()
			if cfg == nil {
				cfg = &models.TenantConfig{}
			}
			cfg.UILabels = tmpl.UILabels
			_ = sacco.SetTenantConfig(cfg)
		}
	}
	if input.DefaultLanguage != nil {
		sacco.DefaultLanguage = *input.DefaultLanguage
	}
	if input.DisplayName != nil {
		sacco.DisplayName = *input.DisplayName
	}

	if err := s.saccoRepo.Update(ctx, sacco); err != nil {
		return nil, fmt.Errorf("update sacco: %w", err)
	}
	return sacco, nil
}

func (s *OrganizationService) DeleteSACCO(ctx context.Context, id uuid.UUID) error {
	return s.saccoRepo.Delete(ctx, id)
}

func (s *OrganizationService) ListSACCOs(ctx context.Context, page, perPage int, search string) ([]models.SACCO, int64, error) {
	return s.saccoRepo.List(ctx, page, perPage, search)
}

// --- Membership ---

type AddMemberInput struct {
	CrewMemberID uuid.UUID        `json:"crew_member_id" binding:"required"`
	OrganizationID      uuid.UUID        `json:"sacco_id"` // Set by handler from URL
	Role         models.SACCORole `json:"role_in_sacco"`
	JoinedAt     string           `json:"joined_at"`
}

func (s *OrganizationService) AddMember(ctx context.Context, input AddMemberInput) (*models.CrewSACCOMembership, error) {
	// Check if already a member
	existing, err := s.membershipRepo.GetActive(ctx, input.CrewMemberID, input.OrganizationID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("crew member is already an active member of this SACCO")
	}

	role := input.Role
	if role == "" {
		role = models.SACCORoleMember
	}

	m := &models.CrewSACCOMembership{
		CrewMemberID: input.CrewMemberID,
		OrganizationID:      input.OrganizationID,
		RoleInOrg:  role,
		IsActive:     true,
	}

	if input.JoinedAt != "" {
		if t, err := time.Parse(time.RFC3339, input.JoinedAt); err == nil {
			m.JoinedAt = t
		} else {
			m.JoinedAt = time.Now()
		}
	} else {
		m.JoinedAt = time.Now()
	}

	if err := s.membershipRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	s.logger.Info("member added to SACCO",
		slog.String("crew_member_id", input.CrewMemberID.String()),
		slog.String("sacco_id", input.OrganizationID.String()),
	)
	return m, nil
}

type UpdateMemberInput struct {
	Role     models.SACCORole `json:"role_in_sacco" binding:"required"`
	JoinedAt string           `json:"joined_at"`
}

func (s *OrganizationService) UpdateMember(ctx context.Context, membershipID uuid.UUID, input UpdateMemberInput) (*models.CrewSACCOMembership, error) {
	membership, err := s.membershipRepo.GetByID(ctx, membershipID)
	if err != nil {
		return nil, fmt.Errorf("membership not found: %w", err)
	}

	membership.RoleInOrg = input.Role
	membership.UpdatedAt = time.Now()
	
	if input.JoinedAt != "" {
		if t, err := time.Parse(time.RFC3339, input.JoinedAt); err == nil {
			membership.JoinedAt = t
		}
	}

	if err := s.membershipRepo.Update(ctx, membership); err != nil {
		return nil, fmt.Errorf("failed to update membership: %w", err)
	}
	return membership, nil
}

func (s *OrganizationService) RemoveMember(ctx context.Context, membershipID uuid.UUID) error {
	m, err := s.membershipRepo.GetByID(ctx, membershipID)
	if err != nil {
		return err
	}
	m.IsActive = false
	return s.membershipRepo.Update(ctx, m)
}

func (s *OrganizationService) ListMembers(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]models.CrewSACCOMembership, int64, error) {
	return s.membershipRepo.ListByOrganization(ctx, orgID, page, perPage)
}

func (s *OrganizationService) GetCrewMemberships(ctx context.Context, crewMemberID uuid.UUID) ([]models.CrewSACCOMembership, error) {
	return s.membershipRepo.ListByCrewMember(ctx, crewMemberID)
}

// --- Float ---

func (s *OrganizationService) GetFloat(ctx context.Context, orgID uuid.UUID) (*models.OrganizationFloat, error) {
	return s.floatRepo.GetOrCreate(ctx, orgID)
}

type FloatOperationInput struct {
	OrganizationID        uuid.UUID `json:"sacco_id"`
	AmountCents    int64     `json:"amount_cents" binding:"required,min=1"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Reference      string    `json:"reference"`
}

func (s *OrganizationService) CreditFloat(ctx context.Context, input FloatOperationInput) (*models.OrganizationFloatTransaction, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	tx, err := s.floatRepo.CreditFloat(ctx, sf.ID, sf.Version, input.AmountCents, input.IdempotencyKey, input.Reference)
	if err == nil {
		s.auditSvc.Log(ctx, nil, "CREDIT_FLOAT", "sacco_float", &sf.ID, nil, tx, "", "")
	}
	return tx, err
}

func (s *OrganizationService) DebitFloat(ctx context.Context, input FloatOperationInput) (*models.OrganizationFloatTransaction, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	tx, err := s.floatRepo.DebitFloat(ctx, sf.ID, sf.Version, input.AmountCents, input.IdempotencyKey, input.Reference)
	if err == nil {
		s.auditSvc.Log(ctx, nil, "DEBIT_FLOAT", "sacco_float", &sf.ID, nil, tx, "", "")
	}
	return tx, err
}

func (s *OrganizationService) ListFloatTransactions(ctx context.Context, orgID uuid.UUID, filter repository.OrganizationFloatFilter, page, perPage int) ([]models.OrganizationFloatTransaction, int64, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, orgID)
	if err != nil {
		return nil, 0, err
	}
	return s.floatRepo.GetTransactions(ctx, sf.ID, filter, page, perPage)
}

// CreatePendingTopUp creates a PENDING float transaction without changing the
// balance. Used for mobile money STK push flows where the balance is credited
// only after the payment callback confirms success.
func (s *OrganizationService) CreatePendingTopUp(ctx context.Context, input FloatOperationInput) (*models.OrganizationFloatTransaction, error) {
	sf, err := s.floatRepo.GetOrCreate(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	tx, err := s.floatRepo.CreatePendingTransaction(ctx, sf.ID, input.AmountCents, input.IdempotencyKey, input.Reference)
	if err == nil {
		s.auditSvc.Log(ctx, nil, "INITIATE_TOPUP", "sacco_float", &sf.ID, nil, tx, "", "")
	}
	return tx, err
}

// ConfirmPendingTopUp atomically credits the float balance and marks the
// pending transaction as COMPLETED. Called from the webhook handler when
// the payment provider confirms a successful STK push payment.
func (s *OrganizationService) ConfirmPendingTopUp(ctx context.Context, txID uuid.UUID) (*models.OrganizationFloatTransaction, error) {
	return s.floatRepo.ConfirmPendingTransaction(ctx, txID)
}

// FailPendingTopUp marks a pending float transaction as FAILED.
func (s *OrganizationService) FailPendingTopUp(ctx context.Context, txID uuid.UUID, reason string) error {
	return s.floatRepo.FailPendingTransaction(ctx, txID, reason)
}

// GetFloatTxByIdempotencyKey finds a float transaction by its idempotency key.
func (s *OrganizationService) GetFloatTxByIdempotencyKey(ctx context.Context, key string) (*models.OrganizationFloatTransaction, error) {
	return s.floatRepo.GetByIdempotencyKey(ctx, key)
}

// UpdatePendingRef appends reference info to a pending float transaction.
func (s *OrganizationService) UpdatePendingRef(ctx context.Context, txID uuid.UUID, refSuffix string) error {
	return s.floatRepo.AppendReference(ctx, txID, refSuffix)
}
