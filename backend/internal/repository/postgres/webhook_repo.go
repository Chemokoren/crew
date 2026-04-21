package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// WebhookEventRepo is the GORM implementation of repository.WebhookEventRepository.
type WebhookEventRepo struct {
	db *gorm.DB
}

func NewWebhookEventRepo(db *gorm.DB) *WebhookEventRepo {
	return &WebhookEventRepo{db: db}
}

func (r *WebhookEventRepo) Create(ctx context.Context, event *models.WebhookEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create webhook event: %w", err)
	}
	return nil
}

func (r *WebhookEventRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEvent, error) {
	var event models.WebhookEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get webhook event: %w", err)
	}
	return &event, nil
}

func (r *WebhookEventRepo) GetByExternalRef(ctx context.Context, source models.WebhookSource, ref string) (*models.WebhookEvent, error) {
	var event models.WebhookEvent
	if err := r.db.WithContext(ctx).Where("source = ? AND external_ref = ?", source, ref).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get webhook by ref: %w", err)
	}
	return &event, nil
}

func (r *WebhookEventRepo) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.WebhookEvent{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_processed": true, "processed_at": now})
	if result.Error != nil {
		return fmt.Errorf("mark webhook processed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *WebhookEventRepo) ListUnprocessed(ctx context.Context, source models.WebhookSource, limit int) ([]models.WebhookEvent, error) {
	var events []models.WebhookEvent
	query := r.db.WithContext(ctx).Where("is_processed = false")
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if err := query.Order("created_at ASC").Limit(limit).Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list unprocessed webhooks: %w", err)
	}
	return events, nil
}
