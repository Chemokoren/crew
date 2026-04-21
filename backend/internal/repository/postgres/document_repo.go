package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// DocumentRepo is the GORM implementation of repository.DocumentRepository.
type DocumentRepo struct {
	db *gorm.DB
}

func NewDocumentRepo(db *gorm.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) Create(ctx context.Context, doc *models.Document) error {
	if err := r.db.WithContext(ctx).Create(doc).Error; err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	return nil
}

func (r *DocumentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get document: %w", err)
	}
	return &doc, nil
}

func (r *DocumentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Document{}).Error; err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

func (r *DocumentRepo) List(ctx context.Context, filter repository.DocumentFilter, page, perPage int) ([]models.Document, int64, error) {
	var docs []models.Document
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Document{})
	if filter.CrewMemberID != nil {
		query = query.Where("crew_member_id = ?", *filter.CrewMemberID)
	}
	if filter.SaccoID != nil {
		query = query.Where("sacco_id = ?", *filter.SaccoID)
	}
	if filter.VehicleID != nil {
		query = query.Where("vehicle_id = ?", *filter.VehicleID)
	}
	if filter.DocumentType != "" {
		query = query.Where("document_type = ?", filter.DocumentType)
	}
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&docs).Error; err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}
	return docs, total, nil
}
