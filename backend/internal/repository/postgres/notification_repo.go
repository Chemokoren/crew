package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// NotificationRepo is the GORM implementation of repository.NotificationRepository.
type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, n *models.Notification) error {
	if err := r.db.WithContext(ctx).Create(n).Error; err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var n models.Notification
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get notification: %w", err)
	}
	return &n, nil
}

func (r *NotificationRepo) Update(ctx context.Context, n *models.Notification) error {
	if err := r.db.WithContext(ctx).Save(n).Error; err != nil {
		return fmt.Errorf("update notification: %w", err)
	}
	return nil
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter repository.NotificationFilter, page, perPage int) ([]models.Notification, int64, error) {
	var notifs []models.Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ?", userID)
	if filter.Channel != "" {
		query = query.Where("channel = ?", filter.Channel)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&notifs).Error; err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	return notifs, total, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Notification{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": models.NotifRead, "read_at": now})
	if result.Error != nil {
		return fmt.Errorf("mark notification read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *NotificationRepo) GetTemplate(ctx context.Context, eventName string) (*models.NotificationTemplate, error) {
	var t models.NotificationTemplate
	if err := r.db.WithContext(ctx).Where("event_name = ? AND is_active = true", eventName).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &t, nil
}

func (r *NotificationRepo) CreateTemplate(ctx context.Context, t *models.NotificationTemplate) error {
	if err := r.db.WithContext(ctx).Create(t).Error; err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}
