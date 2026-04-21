package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// NotificationService handles notification dispatch and management.
type NotificationService struct {
	notifRepo repository.NotificationRepository
	logger    *slog.Logger
}

func NewNotificationService(
	notifRepo repository.NotificationRepository,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{notifRepo: notifRepo, logger: logger}
}

// SendNotification creates and dispatches a notification to a user.
func (s *NotificationService) SendNotification(ctx context.Context, userID uuid.UUID, channel models.NotificationChannel, title, body string) (*models.Notification, error) {
	now := time.Now()
	n := &models.Notification{
		UserID:  userID,
		Channel: channel,
		Title:   title,
		Body:    body,
		Status:  models.NotifSent,
		SentAt:  &now,
	}
	if err := s.notifRepo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	s.logger.Info("notification sent", slog.String("user_id", userID.String()), slog.String("channel", string(channel)))
	return n, nil
}

// SendFromTemplate renders a template and sends a notification.
func (s *NotificationService) SendFromTemplate(ctx context.Context, userID uuid.UUID, eventName string, vars map[string]string) (*models.Notification, error) {
	tmpl, err := s.notifRepo.GetTemplate(ctx, eventName)
	if err != nil {
		return nil, fmt.Errorf("template %s: %w", eventName, err)
	}
	title := renderTemplate(tmpl.TitleTemplate, vars)
	body := renderTemplate(tmpl.BodyTemplate, vars)
	return s.SendNotification(ctx, userID, tmpl.Channel, title, body)
}

func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, filter repository.NotificationFilter, page, perPage int) ([]models.Notification, int64, error) {
	return s.notifRepo.ListByUser(ctx, userID, filter, page, perPage)
}

func (s *NotificationService) MarkRead(ctx context.Context, id uuid.UUID) error {
	return s.notifRepo.MarkRead(ctx, id)
}

func renderTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
