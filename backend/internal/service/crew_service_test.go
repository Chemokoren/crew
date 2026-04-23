package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/kibsoft/amy-mis/internal/external/identity"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
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

// --- Graceful Degradation Tests (nil IDP / IPRS disabled) ---

func TestCrewService_VerifyNationalID_NilProvider(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// IDP is nil — simulates IDENTITY_IPRS_ENABLED=false
	svc := service.NewCrewService(repo, nil, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "99998888",
		FirstName:  "Grace",
		LastName:   "Wanjiku",
		Role:       models.RoleDriver,
	})

	_, err := svc.VerifyNationalID(context.Background(), crew.ID, "SERIAL_789")
	if err == nil {
		t.Fatal("expected error when identity provider is nil")
	}
	// Should return a clear error message, not a panic
	expected := "identity provider not configured"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestCrewService_UpdateKYCStatus_NilProvider_SkipsIPRS(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// IDP is nil — system should still allow KYC status updates
	svc := service.NewCrewService(repo, nil, logger)

	crew, _ := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "77776666",
		FirstName:  "Peter",
		LastName:   "Otieno",
		Role:       models.RoleConductor,
	})

	// UpdateKYCStatus with serial number but nil IDP should still succeed
	// because the IPRS check is gated behind `if s.idp != nil`
	updated, err := svc.UpdateKYCStatus(context.Background(), service.UpdateKYCInput{
		CrewMemberID: crew.ID,
		Status:       models.KYCVerified,
		SerialNumber: "SERIAL_456",
	})

	if err != nil {
		t.Fatalf("expected no error when IDP is nil, got %v", err)
	}
	if updated.KYCStatus != models.KYCVerified {
		t.Errorf("expected kyc verified, got %s", updated.KYCStatus)
	}
	if updated.KYCVerifiedAt == nil {
		t.Errorf("expected kyc verified at to be set even without IPRS")
	}
}

func TestCrewService_CRUD_WorksWithoutIPRS(t *testing.T) {
	repo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// All CRUD operations should work perfectly without IPRS
	svc := service.NewCrewService(repo, nil, logger)

	// Create
	crew, err := svc.CreateCrewMember(context.Background(), service.CreateCrewInput{
		NationalID: "55554444",
		FirstName:  "Mary",
		LastName:   "Njeri",
		Role:       models.RoleDriver,
	})
	if err != nil {
		t.Fatalf("create without IDP: %v", err)
	}

	// Get
	fetched, err := svc.GetCrewMember(context.Background(), crew.ID)
	if err != nil {
		t.Fatalf("get without IDP: %v", err)
	}
	if fetched.NationalID != "55554444" {
		t.Errorf("NationalID = %q, want 55554444", fetched.NationalID)
	}

	// List
	members, total, err := svc.ListCrewMembers(context.Background(), repository.CrewFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("list without IDP: %v", err)
	}
	if total == 0 || len(members) == 0 {
		t.Error("expected at least one crew member in list")
	}

	// Deactivate
	err = svc.DeactivateCrewMember(context.Background(), crew.ID)
	if err != nil {
		t.Fatalf("deactivate without IDP: %v", err)
	}

	deactivated, _ := svc.GetCrewMember(context.Background(), crew.ID)
	if deactivated.IsActive {
		t.Error("expected crew to be inactive")
	}

	// Search by national ID
	found, err := svc.GetByNationalID(context.Background(), "55554444")
	if err != nil {
		t.Fatalf("search by NID without IDP: %v", err)
	}
	if found.FirstName != "Mary" {
		t.Errorf("FirstName = %q, want Mary", found.FirstName)
	}
}
