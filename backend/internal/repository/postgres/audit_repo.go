package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
)

// AuditLogRepo is the GORM implementation of repository.AuditLogRepository.
type AuditLogRepo struct {
	db *gorm.DB
}

func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, log *models.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *AuditLogRepo) List(ctx context.Context, action, resource string, resourceID, userID *uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if action != "" {
		search := strings.ToLower(action)
		switch search {
		case "create":
			search = "created"
		case "update":
			search = "updated"
		case "delete":
			search = "deleted"
		case "approve":
			search = "approved"
		case "reject":
			search = "rejected"
		case "export":
			search = "exported"
		}
		query = query.Where("action LIKE ?", "%"+search+"%")
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if resourceID != nil {
		query = query.Where("resource_id = ?", *resourceID)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, total, nil
}

func (r *AuditLogRepo) ListByUserID(ctx context.Context, userID uuid.UUID, action string, page, perPage int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("user_id = ? OR resource_id = ?", userID, userID)
	if action != "" {
		search := strings.ToLower(action)
		switch search {
		case "create":
			search = "created"
		case "update":
			search = "updated"
		case "delete":
			search = "deleted"
		case "approve":
			search = "approved"
		case "reject":
			search = "rejected"
		case "export":
			search = "exported"
		}
		query = query.Where("action LIKE ?", "%"+search+"%")
	}
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs by user: %w", err)
	}
	return logs, total, nil
}

