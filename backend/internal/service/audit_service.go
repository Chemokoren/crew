package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// AuditService records all CUD operations for compliance.
type AuditService struct {
	auditRepo repository.AuditLogRepository
	logger    *slog.Logger
}

func NewAuditService(auditRepo repository.AuditLogRepository, logger *slog.Logger) *AuditService {
	return &AuditService{auditRepo: auditRepo, logger: logger}
}

// Log records a single audit event.
func (s *AuditService) Log(ctx context.Context, userID uuid.UUID, action, resource string, resourceID *uuid.UUID, oldValue, newValue interface{}, ipAddress, userAgent string) {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	entry := &models.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		OldValue:   oldJSON,
		NewValue:   newJSON,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	if err := s.auditRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log", slog.String("error", err.Error()))
		return
	}
}

func (s *AuditService) ListLogs(ctx context.Context, resource string, resourceID *uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, resource, resourceID, page, perPage)
}
