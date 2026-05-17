package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
)

// SystemAnnouncementRepo is the GORM implementation of repository.SystemAnnouncementRepository.
type SystemAnnouncementRepo struct {
	db *gorm.DB
}

func NewSystemAnnouncementRepo(db *gorm.DB) *SystemAnnouncementRepo {
	return &SystemAnnouncementRepo{db: db}
}

func (r *SystemAnnouncementRepo) Create(ctx context.Context, a *models.SystemAnnouncement) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *SystemAnnouncementRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.SystemAnnouncement, error) {
	var a models.SystemAnnouncement
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SystemAnnouncementRepo) Update(ctx context.Context, a *models.SystemAnnouncement) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *SystemAnnouncementRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.SystemAnnouncement{}, "id = ?", id).Error
}

func (r *SystemAnnouncementRepo) ListAll(ctx context.Context, page, perPage int) ([]models.SystemAnnouncement, int64, error) {
	var announcements []models.SystemAnnouncement
	var total int64

	r.db.WithContext(ctx).Model(&models.SystemAnnouncement{}).Count(&total)
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&announcements).Error
	return announcements, total, err
}

func (r *SystemAnnouncementRepo) ListActive(ctx context.Context) ([]models.SystemAnnouncement, error) {
	var announcements []models.SystemAnnouncement
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", true, now, now).
		Order("severity DESC, created_at DESC").
		Find(&announcements).Error
	return announcements, err
}
