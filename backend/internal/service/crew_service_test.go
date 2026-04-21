package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kibsoft/amy-mis/internal/external/identity"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

// MockIPRSProvider simulates the identity.Provider interface.
type MockIPRSProvider struct {
	ShouldFail bool
}

func (m *MockIPRSProvider) Name() string { return "mock_iprs" }

func (m *MockIPRSProvider) VerifyCitizen(ctx context.Context, req identity.VerifyRequest) (*identity.CitizenDetails, error) {
	if m.ShouldFail {
		return nil, os.ErrNotExist
	}
	return &identity.CitizenDetails{
		Provider:     "mock_iprs",
		IDNumber:     req.IDNumber,
		SerialNumber: req.SerialNumber,
		FirstName:    "John",
		LastName:     "Doe",
		Verified:     true,
	}, nil
}

func TestCrewService_CreateCrewMember(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewCrewService(repo, nil, logger)

	input := service.CreateCrewInput{
		NationalID: "12345678",
		FirstName:  "Jane",
		LastName:   "Doe",
		Role:       models.RoleDriver,
	}

	crew, err := svc.CreateCrewMember(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if crew.NationalID != "12345678" {
		t.Errorf("expected national id 12345678, got %s", crew.NationalID)
	}
	if crew.KYCStatus != models.KYCPending {
		t.Errorf("expected pending kyc status")
	}
}

func TestCrewService_VerifyNationalID(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockIDP := &MockIPRSProvider{ShouldFail: false}
	svc := service.NewCrewService(repo, mockIDP, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "87654321",
		FirstName:  "John",
		LastName:   "Doe",
		Role:       models.RoleConductor,
	})

	verifiedCrew, err := svc.VerifyNationalID(context.Background(), crew.ID, "SERIAL_123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if verifiedCrew.KYCStatus != models.KYCVerified {
		t.Errorf("expected kyc verified, got %s", verifiedCrew.KYCStatus)
	}
	if verifiedCrew.KYCVerifiedAt == nil {
		t.Errorf("expected kyc verified at to be set")
	}
}

func TestCrewService_VerifyNationalID_Failure(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockIDP := &MockIPRSProvider{ShouldFail: true}
	svc := service.NewCrewService(repo, mockIDP, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "87654321",
		FirstName:  "John",
		LastName:   "Doe",
		Role:       models.RoleConductor,
	})

	_, err := svc.VerifyNationalID(context.Background(), crew.ID, "SERIAL_123")
	if err == nil {
		t.Fatalf("expected error due to provider failure")
	}
}

func TestCrewService_DeactivateCrewMember(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := service.NewCrewService(repo, nil, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "111",
		Role:       models.RoleDriver,
	})

	err := svc.DeactivateCrewMember(context.Background(), crew.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	fetched, _ := svc.GetCrewMember(context.Background(), crew.ID)
	if fetched.IsActive {
		t.Errorf("expected crew to be inactive")
	}
}

func TestCrewService_UpdateKYCStatus_WithIPRS(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockIDP := &MockIPRSProvider{ShouldFail: false}
	svc := service.NewCrewService(repo, mockIDP, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "87654321",
		FirstName:  "John",
		LastName:   "Doe",
		Role:       models.RoleConductor,
	})

	updated, err := svc.UpdateKYCStatus(context.Background(), service.UpdateKYCInput{
		CrewMemberID: crew.ID,
		Status:       models.KYCVerified,
		SerialNumber: "SERIAL_123",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.KYCStatus != models.KYCVerified {
		t.Errorf("expected kyc verified, got %s", updated.KYCStatus)
	}
	if updated.KYCVerifiedAt == nil {
		t.Errorf("expected kyc verified at to be set")
	}
}
