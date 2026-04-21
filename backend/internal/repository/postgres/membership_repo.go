package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// MembershipRepo is the GORM implementation of repository.MembershipRepository.
type MembershipRepo struct {
	db *gorm.DB
}

func NewMembershipRepo(db *gorm.DB) *MembershipRepo {
	return &MembershipRepo{db: db}
}

func (r *MembershipRepo) Create(ctx context.Context, m *models.CrewSACCOMembership) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

func (r *MembershipRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.CrewSACCOMembership, error) {
	var m models.CrewSACCOMembership
	if err := r.db.WithContext(ctx).Preload("CrewMember").Preload("Sacco").Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get membership: %w", err)
	}
	return &m, nil
}

func (r *MembershipRepo) Update(ctx context.Context, m *models.CrewSACCOMembership) error {
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update membership: %w", err)
	}
	return nil
}

func (r *MembershipRepo) ListBySACCO(ctx context.Context, saccoID uuid.UUID, page, perPage int) ([]models.CrewSACCOMembership, int64, error) {
	var members []models.CrewSACCOMembership
	var total int64

	query := r.db.WithContext(ctx).Model(&models.CrewSACCOMembership{}).Where("sacco_id = ? AND is_active = true", saccoID)
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Preload("CrewMember").Offset(offset).Limit(perPage).Order("joined_at DESC").Find(&members).Error; err != nil {
		return nil, 0, fmt.Errorf("list memberships by sacco: %w", err)
	}
	return members, total, nil
}

func (r *MembershipRepo) ListByCrewMember(ctx context.Context, crewMemberID uuid.UUID) ([]models.CrewSACCOMembership, error) {
	var members []models.CrewSACCOMembership
	if err := r.db.WithContext(ctx).Preload("Sacco").Where("crew_member_id = ? AND is_active = true", crewMemberID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("list memberships by crew: %w", err)
	}
	return members, nil
}

func (r *MembershipRepo) GetActive(ctx context.Context, crewMemberID, saccoID uuid.UUID) (*models.CrewSACCOMembership, error) {
	var m models.CrewSACCOMembership
	if err := r.db.WithContext(ctx).Where("crew_member_id = ? AND sacco_id = ? AND is_active = true", crewMemberID, saccoID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get active membership: %w", err)
	}
	return &m, nil
}
