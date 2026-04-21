package payroll

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

type mockPayrollProvider struct {
	name      string
	submitErr error
	statusErr error
}

func (m *mockPayrollProvider) Name() string { return m.name }

func (m *mockPayrollProvider) SubmitPayroll(ctx context.Context, req SubmitRequest) (*SubmitResult, error) {
	if m.submitErr != nil {
		return nil, m.submitErr
	}
	return &SubmitResult{
		Provider:      m.name,
		CorrelationID: "corr-001",
		Status:        "received",
		StatusURL:     "/payroll/v1/requests/corr-001/status",
	}, nil
}

func (m *mockPayrollProvider) GetStatus(ctx context.Context, correlationID string) (*StatusResult, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return &StatusResult{
		Provider:      m.name,
		CorrelationID: correlationID,
		Status:        "completed",
		GrossPay:      75000.00,
		NetPay:        62500.00,
		Deductions:    12500.00,
	}, nil
}

func TestManagerSubmitPayroll(t *testing.T) {
	primary := &mockPayrollProvider{name: "perpay"}
	mgr := NewManager(testLogger(), primary)

	result, err := mgr.SubmitPayroll(context.Background(), SubmitRequest{
		EmployeeID: "EMP-001", FullName: "Jane Doe",
	})
	if err != nil {
		t.Fatalf("SubmitPayroll: %v", err)
	}
	if result.CorrelationID != "corr-001" {
		t.Errorf("CorrelationID = %q, want corr-001", result.CorrelationID)
	}
	if result.Status != "received" {
		t.Errorf("Status = %q, want received", result.Status)
	}
}

func TestManagerGetStatus(t *testing.T) {
	primary := &mockPayrollProvider{name: "perpay"}
	mgr := NewManager(testLogger(), primary)

	result, err := mgr.GetStatus(context.Background(), "corr-001")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if result.GrossPay != 75000.00 {
		t.Errorf("GrossPay = %.2f, want 75000.00", result.GrossPay)
	}
	if result.NetPay != 62500.00 {
		t.Errorf("NetPay = %.2f, want 62500.00", result.NetPay)
	}
}

func TestManagerSetPrimary(t *testing.T) {
	p1 := &mockPayrollProvider{name: "perpay"}
	p2 := &mockPayrollProvider{name: "alternative"}
	mgr := NewManager(testLogger(), p1, p2)

	if err := mgr.SetPrimary("alternative"); err != nil {
		t.Fatalf("SetPrimary: %v", err)
	}

	result, _ := mgr.SubmitPayroll(context.Background(), SubmitRequest{})
	if result.Provider != "alternative" {
		t.Errorf("after SetPrimary, Provider = %q, want alternative", result.Provider)
	}
}

func TestManagerSetPrimaryNotFound(t *testing.T) {
	mgr := NewManager(testLogger(), &mockPayrollProvider{name: "perpay"})
	if err := mgr.SetPrimary("nonexistent"); err == nil {
		t.Error("SetPrimary should fail for unknown provider")
	}
}

func TestManagerSubmitPayrollFailure(t *testing.T) {
	primary := &mockPayrollProvider{name: "perpay", submitErr: fmt.Errorf("network error")}
	mgr := NewManager(testLogger(), primary)

	_, err := mgr.SubmitPayroll(context.Background(), SubmitRequest{})
	if err == nil {
		t.Error("SubmitPayroll should fail when provider fails")
	}
}
