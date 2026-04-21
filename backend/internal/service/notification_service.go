package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/sms"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// NotificationService handles notification dispatch and management.
type NotificationService struct {
	notifRepo repository.NotificationRepository
	userRepo  repository.UserRepository
	smsMgr    *sms.Manager
	logger    *slog.Logger
}

func NewNotificationService(
	notifRepo repository.NotificationRepository,
	userRepo  repository.UserRepository,
	smsMgr    *sms.Manager,
	logger    *slog.Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo: notifRepo,
		userRepo:  userRepo,
		smsMgr:    smsMgr,
		logger:    logger,
	}
}

// SendToCrewMember looks up the user for a crew member and dispatches a notification.
func (s *NotificationService) SendToCrewMember(ctx context.Context, crewMemberID uuid.UUID, channel models.NotificationChannel, title, body string) (*models.Notification, error) {
	user, err := s.userRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user for crew member: %w", err)
	}
	return s.SendNotification(ctx, user.ID, channel, title, body)
}

// SendNotification creates and dispatches a notification to a user.
func (s *NotificationService) SendNotification(ctx context.Context, userID uuid.UUID, channel models.NotificationChannel, title, body string) (*models.Notification, error) {
	now := time.Now()
	status := models.NotifPending

	if channel == models.ChannelInApp {
		status = models.NotifSent
	}

	n := &models.Notification{
		UserID:  userID,
		Channel: channel,
		Title:   title,
		Body:    body,
		Status:  status,
	}
	
	if err := s.notifRepo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	if channel == models.ChannelSMS {
		if s.smsMgr == nil {
			s.logger.Warn("SMS channel requested but SMS manager is nil", slog.String("user_id", userID.String()))
			n.Status = models.NotifFailed
		} else {
			user, err := s.userRepo.GetByID(ctx, userID)
			if err != nil {
				s.logger.Error("failed to get user for SMS", slog.String("error", err.Error()))
				n.Status = models.NotifFailed
			} else if user.Phone == "" {
				s.logger.Error("user has no phone number", slog.String("user_id", userID.String()))
				n.Status = models.NotifFailed
			} else {
				res, err := s.smsMgr.Send(ctx, user.Phone, body)
				if err != nil || !res.Success {
					s.logger.Error("failed to send SMS", slog.Any("error", err), slog.Any("result", res))
					n.Status = models.NotifFailed
				} else {
					n.Status = models.NotifSent
					n.SentAt = &now
				}
			}
		}
		// Update status based on delivery attempt
		if err := s.notifRepo.Update(ctx, n); err != nil {
			s.logger.Error("failed to update notification status", slog.String("error", err.Error()))
		}
	} else if channel == models.ChannelInApp {
		n.SentAt = &now
		if err := s.notifRepo.Update(ctx, n); err != nil {
			s.logger.Error("failed to update notification status", slog.String("error", err.Error()))
		}
	}

	s.logger.Info("notification processed", slog.String("user_id", userID.String()), slog.String("channel", string(channel)), slog.String("status", string(n.Status)))
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
