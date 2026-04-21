package identity

import (
	"context"
	"fmt"
	"testing"

	"log/slog"
	"os"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

type mockIdentityProvider struct {
	name      string
	verifyErr error
}

func (m *mockIdentityProvider) Name() string { return m.name }

func (m *mockIdentityProvider) VerifyCitizen(ctx context.Context, req VerifyRequest) (*CitizenDetails, error) {
	if m.verifyErr != nil {
		return nil, m.verifyErr
	}
	return &CitizenDetails{
		Provider:    m.name,
		IDNumber:    req.IDNumber,
		FirstName:   "Jane",
		LastName:    "Kamau",
		Gender:      "Female",
		DateOfBirth: "1990-05-15",
		Verified:    true,
	}, nil
}

func TestManagerVerifyCitizenPrimary(t *testing.T) {
	primary := &mockIdentityProvider{name: "iprs"}
	mgr := NewManager(testLogger(), primary)

	result, err := mgr.VerifyCitizen(context.Background(), VerifyRequest{IDNumber: "12345678"})
	if err != nil {
		t.Fatalf("VerifyCitizen: %v", err)
	}
	if !result.Verified {
		t.Error("should be verified")
	}
	if result.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want Jane", result.FirstName)
	}
	if result.Provider != "iprs" {
		t.Errorf("Provider = %q, want iprs", result.Provider)
	}
}

func TestManagerVerifyCitizenFallback(t *testing.T) {
	primary := &mockIdentityProvider{name: "iprs", verifyErr: fmt.Errorf("IPRS down")}
	fallback := &mockIdentityProvider{name: "alternative-kyc"}
	mgr := NewManager(testLogger(), primary, fallback)

	result, err := mgr.VerifyCitizen(context.Background(), VerifyRequest{IDNumber: "12345678"})
	if err != nil {
		t.Fatalf("should fallback: %v", err)
	}
	if result.Provider != "alternative-kyc" {
		t.Errorf("Provider = %q, want alternative-kyc", result.Provider)
	}
}

func TestManagerAllProvidersFail(t *testing.T) {
	p1 := &mockIdentityProvider{name: "p1", verifyErr: fmt.Errorf("fail1")}
	p2 := &mockIdentityProvider{name: "p2", verifyErr: fmt.Errorf("fail2")}
	mgr := NewManager(testLogger(), p1, p2)

	_, err := mgr.VerifyCitizen(context.Background(), VerifyRequest{IDNumber: "12345678"})
	if err == nil {
		t.Error("should fail when all providers fail")
	}
}
