package service_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/sms"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

func TestNotificationService_DispatchAndList(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewNotificationService(repo, userRepo, nil, logger)

	userID := uuid.New()

	_, err := svc.SendNotification(context.Background(), userID, models.NotificationChannel("PUSH"), "Test Title", "Test Message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	notifs, total, err := svc.ListNotifications(context.Background(), userID, repository.NotificationFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 notification, got %d", total)
	}
	if notifs[0].Title != "Test Title" {
		t.Errorf("expected Test Title, got %s", notifs[0].Title)
	}
}

func TestNotificationService_MarkRead(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewNotificationService(repo, userRepo, nil, logger)

	userID := uuid.New()
	svc.SendNotification(context.Background(), userID, models.NotificationChannel("PUSH"), "Title", "Message")

	notifs, _, _ := svc.ListNotifications(context.Background(), userID, repository.NotificationFilter{}, 1, 10)
	notifID := notifs[0].ID

	err := svc.MarkRead(context.Background(), notifID)
	if err != nil {
		t.Fatalf("expected no error on mark read, got %v", err)
	}

	// Fetch again to check status
	n, err := repo.GetByID(context.Background(), notifID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.Status != models.NotifRead {
		t.Errorf("expected status read, got %s", n.Status)
	}
}

// --- Mock SMS Provider ---

type MockSMSProvider struct {
	ShouldFail bool
	SentCount  int
	LastPhone  string
	LastMsg    string
}

func (m *MockSMSProvider) Name() string { return "mock_sms" }

func (m *MockSMSProvider) Send(ctx context.Context, phone, message string) (*sms.SendResult, error) {
	if m.ShouldFail {
		return &sms.SendResult{Success: false, Error: "mock error"}, fmt.Errorf("mock error")
	}
	m.SentCount++
	m.LastPhone = phone
	m.LastMsg = message
	return &sms.SendResult{
		Provider:  m.Name(),
		MessageID: "msg-123",
		Success:   true,
	}, nil
}

func (m *MockSMSProvider) SendBulk(ctx context.Context, phones []string, message string) ([]sms.SendResult, error) {
	return nil, nil
}

func TestNotificationService_SendSMS(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockProvider := &MockSMSProvider{}
	smsMgr := sms.NewManager(logger, mockProvider)

	svc := service.NewNotificationService(repo, userRepo, smsMgr, logger)

	// Create user with phone
	user := &models.User{
		ID:    uuid.New(),
		Phone: "+254712345678",
	}
	userRepo.Create(context.Background(), user)

	n, err := svc.SendNotification(context.Background(), user.ID, models.ChannelSMS, "Alert", "Your assignment is ready")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if n.Status != models.NotifSent {
		t.Errorf("expected status sent, got %s", n.Status)
	}

	if mockProvider.SentCount != 1 {
		t.Errorf("expected 1 SMS sent, got %d", mockProvider.SentCount)
	}
	if mockProvider.LastPhone != "+254712345678" {
		t.Errorf("expected +254712345678, got %s", mockProvider.LastPhone)
	}
}
