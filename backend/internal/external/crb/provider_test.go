package crb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Provider Interface Compliance ---

func TestTransUnionProvider_Name(t *testing.T) {
	p := NewTransUnionProvider(TransUnionConfig{}, testLogger())
	if got := p.Name(); got != "transunion" {
		t.Errorf("Name() = %q, want transunion", got)
	}
}

func TestMetropolProvider_Name(t *testing.T) {
	p := NewMetropolProvider(MetropolConfig{}, testLogger())
	if got := p.Name(); got != "metropol" {
		t.Errorf("Name() = %q, want metropol", got)
	}
}

func TestTransUnion_GetCreditReport_InvalidURL(t *testing.T) {
	p := NewTransUnionProvider(TransUnionConfig{BaseURL: "http://invalid.localhost:9999"}, testLogger())
	_, err := p.GetCreditReport(context.Background(), "12345678")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestMetropol_GetCreditReport_InvalidURL(t *testing.T) {
	p := NewMetropolProvider(MetropolConfig{BaseURL: "http://invalid.localhost:9999"}, testLogger())
	_, err := p.GetCreditReport(context.Background(), "12345678")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestTransUnion_SubmitLoanData(t *testing.T) {
	p := NewTransUnionProvider(TransUnionConfig{}, testLogger())
	err := p.SubmitLoanData(context.Background(), LoanReportData{
		LoanID: uuid.New(), Status: "PERFORMING",
	})
	if err != nil {
		t.Errorf("stub submit should succeed: %v", err)
	}
}

func TestMetropol_SubmitLoanData(t *testing.T) {
	p := NewMetropolProvider(MetropolConfig{}, testLogger())
	err := p.SubmitLoanData(context.Background(), LoanReportData{
		LoanID: uuid.New(), Status: "DEFAULTED",
	})
	if err != nil {
		t.Errorf("stub submit should succeed: %v", err)
	}
}

// --- Manager Tests ---

func TestManager_NoProviders(t *testing.T) {
	mgr := NewManager(testLogger())
	_, err := mgr.GetCreditReport(context.Background(), "12345678")
	if err == nil {
		t.Error("expected error with no providers")
	}
}

func TestManager_AllProvidersFail(t *testing.T) {
	p1 := NewTransUnionProvider(TransUnionConfig{BaseURL: "http://bad1.localhost:9999"}, testLogger())
	p2 := NewMetropolProvider(MetropolConfig{BaseURL: "http://bad2.localhost:9999"}, testLogger())
	mgr := NewManager(testLogger(), p1, p2)

	_, err := mgr.GetCreditReport(context.Background(), "12345678")
	if err == nil {
		t.Error("expected error when all providers fail")
	}
}

func TestManager_SubmitLoanData_BestEffort(t *testing.T) {
	p1 := NewTransUnionProvider(TransUnionConfig{}, testLogger())
	p2 := NewMetropolProvider(MetropolConfig{}, testLogger())
	mgr := NewManager(testLogger(), p1, p2)

	// Should not panic even with stub providers
	mgr.SubmitLoanData(context.Background(), LoanReportData{
		LoanID: uuid.New(), Status: "PERFORMING",
	})
}

// --- CreditReport Struct Tests ---

func TestCreditReport_Fields(t *testing.T) {
	report := CreditReport{
		NationalID:     "12345678",
		CRBScore:       720,
		TotalLoans:     5,
		DefaultedLoans: 1,
		ProviderName:   "transunion",
	}

	if report.CRBScore < 0 || report.CRBScore > 900 {
		t.Errorf("CRB score %d out of range", report.CRBScore)
	}
	if report.DefaultedLoans > report.TotalLoans {
		t.Error("defaulted > total loans")
	}
}
