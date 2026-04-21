package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// CrewRepo is the GORM implementation of repository.CrewRepository.
type CrewRepo struct {
	db *gorm.DB
}

// NewCrewRepo creates a new CrewRepo.
func NewCrewRepo(db *gorm.DB) *CrewRepo {
	return &CrewRepo{db: db}
}

// getDB returns the transaction from context if present, otherwise the default DB.
func (r *CrewRepo) getDB(ctx context.Context) *gorm.DB {
	if tx := database.ExtractTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *CrewRepo) Create(ctx context.Context, crew *models.CrewMember) error {
	if err := r.getDB(ctx).Create(crew).Error; err != nil {
		return fmt.Errorf("create crew member: %w", err)
	}
	return nil
}

func (r *CrewRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.CrewMember, error) {
	var crew models.CrewMember
	if err := r.getDB(ctx).Where("id = ?", id).First(&crew).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get crew by id: %w", err)
	}
	return &crew, nil
}

func (r *CrewRepo) GetByCrewID(ctx context.Context, crewID string) (*models.CrewMember, error) {
	var crew models.CrewMember
	if err := r.getDB(ctx).Where("crew_id = ?", crewID).First(&crew).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get crew by crew_id: %w", err)
	}
	return &crew, nil
}

func (r *CrewRepo) Update(ctx context.Context, crew *models.CrewMember) error {
	if err := r.getDB(ctx).Save(crew).Error; err != nil {
		return fmt.Errorf("update crew member: %w", err)
	}
	return nil
}

func (r *CrewRepo) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete via GORM's DeletedAt
	if err := r.getDB(ctx).Where("id = ?", id).Delete(&models.CrewMember{}).Error; err != nil {
		return fmt.Errorf("delete crew member: %w", err)
	}
	return nil
}

func (r *CrewRepo) List(ctx context.Context, filter repository.CrewFilter, page, perPage int) ([]models.CrewMember, int64, error) {
	var members []models.CrewMember
	var total int64

	query := r.getDB(ctx).Model(&models.CrewMember{})

	if filter.SaccoID != nil {
		query = query.Where("id IN (SELECT crew_member_id FROM crew_sacco_memberships WHERE sacco_id = ? AND is_active = true)", *filter.SaccoID)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.KYCStatus != "" {
		query = query.Where("kyc_status = ?", filter.KYCStatus)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("(first_name ILIKE ? OR last_name ILIKE ? OR crew_id ILIKE ?)", search, search, search)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&members).Error; err != nil {
		return nil, 0, fmt.Errorf("list crew members: %w", err)
	}

	return members, total, nil
}

// NextCrewID generates the next human-readable crew ID (CRW-00001).
func (r *CrewRepo) NextCrewID(ctx context.Context) (string, error) {
	var nextVal int64
	if err := r.getDB(ctx).Raw("SELECT nextval('crew_id_seq')").Scan(&nextVal).Error; err != nil {
		return "", fmt.Errorf("get next crew_id_seq: %w", err)
	}
	return fmt.Sprintf("CRW-%05d", nextVal), nil
}
