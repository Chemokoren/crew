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
	prefRepo := mock.NewNotificationPreferenceRepo()
	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

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
	prefRepo := mock.NewNotificationPreferenceRepo()
	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

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

	prefRepo := mock.NewNotificationPreferenceRepo()
	svc := service.NewNotificationService(repo, prefRepo, userRepo, smsMgr, logger)

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

// --- Preference-Gated Notification Tests ---

func TestNotificationService_SMSOptOut_SuppressesNotification(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockProvider := &MockSMSProvider{}
	smsMgr := sms.NewManager(logger, mockProvider)

	svc := service.NewNotificationService(repo, prefRepo, userRepo, smsMgr, logger)

	user := &models.User{ID: uuid.New(), Phone: "+254712345678"}
	userRepo.Create(context.Background(), user)

	// Opt out of SMS
	prefRepo.Upsert(context.Background(), &models.NotificationPreference{
		UserID:     user.ID,
		SMSOptIn:   false,
		PushOptIn:  true,
		InAppOptIn: true,
	})

	n, err := svc.SendNotification(context.Background(), user.ID, models.ChannelSMS, "Alert", "Suppressed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should return nil notification (silently skipped)
	if n != nil {
		t.Errorf("expected nil notification when user opted out of SMS, got status=%s", n.Status)
	}
	if mockProvider.SentCount != 0 {
		t.Errorf("expected 0 SMS sent when opted out, got %d", mockProvider.SentCount)
	}
}

func TestNotificationService_PushOptOut_SuppressesNotification(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

	userID := uuid.New()

	// Opt out of Push
	prefRepo.Upsert(context.Background(), &models.NotificationPreference{
		UserID:     userID,
		SMSOptIn:   true,
		PushOptIn:  false,
		InAppOptIn: true,
	})

	n, err := svc.SendNotification(context.Background(), userID, models.ChannelPush, "Alert", "Suppressed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != nil {
		t.Errorf("expected nil notification when user opted out of Push, got status=%s", n.Status)
	}
}

func TestNotificationService_InAppOptOut_SuppressesNotification(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

	userID := uuid.New()

	// Opt out of InApp
	prefRepo.Upsert(context.Background(), &models.NotificationPreference{
		UserID:     userID,
		SMSOptIn:   true,
		PushOptIn:  true,
		InAppOptIn: false,
	})

	n, err := svc.SendNotification(context.Background(), userID, models.ChannelInApp, "Alert", "Suppressed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != nil {
		t.Errorf("expected nil notification when user opted out of InApp, got status=%s", n.Status)
	}
}

func TestNotificationService_NoPrefsAllowsDelivery(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

	// No preferences stored — should default to allowing delivery
	userID := uuid.New()
	n, err := svc.SendNotification(context.Background(), userID, models.ChannelPush, "Default Allowed", "Should be delivered")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n == nil {
		t.Fatal("expected notification to be created when no preferences exist")
	}
	if n.Title != "Default Allowed" {
		t.Errorf("Title = %q, want 'Default Allowed'", n.Title)
	}
}

func TestNotificationService_GetPreferences_DefaultsWhenNone(t *testing.T) {
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewNotificationService(mock.NewNotificationRepo(), prefRepo, mock.NewUserRepo(), nil, logger)

	userID := uuid.New()
	prefs, err := svc.GetPreferences(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if prefs == nil {
		t.Fatal("expected default preferences to be returned")
	}
	if !prefs.SMSOptIn {
		t.Error("default SMSOptIn should be true")
	}
	if !prefs.PushOptIn {
		t.Error("default PushOptIn should be true")
	}
	if !prefs.InAppOptIn {
		t.Error("default InAppOptIn should be true")
	}
	if prefs.MarketingOptIn {
		t.Error("default MarketingOptIn should be false")
	}
}

func TestNotificationService_UpdateAndGetPreferences(t *testing.T) {
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewNotificationService(mock.NewNotificationRepo(), prefRepo, mock.NewUserRepo(), nil, logger)

	userID := uuid.New()

	// Update preferences
	err := svc.UpdatePreferences(context.Background(), &models.NotificationPreference{
		UserID:         userID,
		SMSOptIn:       false,
		PushOptIn:      true,
		InAppOptIn:     true,
		MarketingOptIn: true,
	})
	if err != nil {
		t.Fatalf("update preferences: %v", err)
	}

	// Fetch and verify
	prefs, err := svc.GetPreferences(context.Background(), userID)
	if err != nil {
		t.Fatalf("get preferences: %v", err)
	}
	if prefs.SMSOptIn {
		t.Error("SMSOptIn should be false after update")
	}
	if !prefs.MarketingOptIn {
		t.Error("MarketingOptIn should be true after update")
	}
}

func TestNotificationService_SMSFailure_UpdatesStatus(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockProvider := &MockSMSProvider{ShouldFail: true}
	smsMgr := sms.NewManager(logger, mockProvider)

	svc := service.NewNotificationService(repo, prefRepo, userRepo, smsMgr, logger)

	user := &models.User{ID: uuid.New(), Phone: "+254712345678"}
	userRepo.Create(context.Background(), user)

	n, err := svc.SendNotification(context.Background(), user.ID, models.ChannelSMS, "Fail", "Will fail")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.Status != models.NotifFailed {
		t.Errorf("expected status FAILED on SMS delivery failure, got %s", n.Status)
	}
}

func TestNotificationService_InAppNotification_SentImmediately(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

	userID := uuid.New()
	n, err := svc.SendNotification(context.Background(), userID, models.ChannelInApp, "InApp", "Immediate")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.Status != models.NotifSent {
		t.Errorf("expected IN_APP status SENT, got %s", n.Status)
	}
	if n.SentAt == nil {
		t.Error("expected SentAt to be set for IN_APP notification")
	}
}

func TestNotificationService_NilSMSManager_FailsGracefully(t *testing.T) {
	repo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	prefRepo := mock.NewNotificationPreferenceRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// SMS manager is nil — simulates SMS provider disabled
	svc := service.NewNotificationService(repo, prefRepo, userRepo, nil, logger)

	user := &models.User{ID: uuid.New(), Phone: "+254712345678"}
	userRepo.Create(context.Background(), user)

	n, err := svc.SendNotification(context.Background(), user.ID, models.ChannelSMS, "No SMS", "Should fail gracefully")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.Status != models.NotifFailed {
		t.Errorf("expected FAILED when SMS manager is nil, got %s", n.Status)
	}
}
