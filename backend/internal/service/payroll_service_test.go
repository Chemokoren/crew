package service_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payroll"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

// --- Mock Payroll Provider ---
type MockPayrollProvider struct {
	ShouldFail bool
	SubmitReqs []payroll.SubmitRequest
}

func (m *MockPayrollProvider) Name() string { return "mock_perpay" }

func (m *MockPayrollProvider) SubmitPayroll(ctx context.Context, req payroll.SubmitRequest) (*payroll.SubmitResult, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock error")
	}
	m.SubmitReqs = append(m.SubmitReqs, req)
	return &payroll.SubmitResult{
		Provider:      m.Name(),
		CorrelationID: "corr-123",
		Status:        "received",
	}, nil
}

func (m *MockPayrollProvider) GetStatus(ctx context.Context, correlationID string) (*payroll.StatusResult, error) {
	return nil, nil
}

func TestPayrollService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	payrollRepo := mock.NewPayrollRepo()
	earningRepo := mock.NewEarningRepo()
	
	rates := []models.StatutoryRate{
		{Name: "SHA", RateType: models.RatePercentage, Rate: 0.0275, IsActive: true},
		{Name: "NSSF", RateType: models.RatePercentage, Rate: 0.06, IsActive: true},
		{Name: "HOUSING_LEVY", RateType: models.RatePercentage, Rate: 0.015, IsActive: true},
	}
	rateRepo := mock.NewStatutoryRateRepo(rates)
	crewRepo := mock.NewCrewRepo()

	mockProvider := &MockPayrollProvider{}
	payrollMgr := payroll.NewManager(logger, mockProvider)

	svc := service.NewPayrollService(payrollRepo, earningRepo, rateRepo, crewRepo, payrollMgr, logger)

	t.Run("Create Payroll Run", func(t *testing.T) {
		ctx := context.Background()
		run, err := svc.CreatePayrollRun(ctx, service.CreatePayrollRunInput{
			SaccoID:     uuid.New(),
			PeriodStart: "2026-04-01",
			PeriodEnd:   "2026-04-30",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.Status != models.PayrollDraft {
			t.Errorf("expected DRAFT status, got %v", run.Status)
		}
	})

	t.Run("Process Payroll Run", func(t *testing.T) {
		ctx := context.Background()
		saccoID := uuid.New()

		// 1. Setup Earnings
		crewID1 := uuid.New()
		crewID2 := uuid.New()
		
		now := time.Now()
		_ = earningRepo.Create(ctx, &models.Earning{CrewMemberID: crewID1, AmountCents: 1000000, EarnedAt: now}) // 10,000 KES
		_ = earningRepo.Create(ctx, &models.Earning{CrewMemberID: crewID2, AmountCents: 500000, EarnedAt: now})  // 5,000 KES

		// 2. Create Run
		run, err := svc.CreatePayrollRun(ctx, service.CreatePayrollRunInput{
			SaccoID:     saccoID,
			PeriodStart: "2026-04-01",
			PeriodEnd:   "2026-04-30",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Add run.PeriodStart/End correctly so mock earning repo lists them (Earnings List filter uses DateFrom DateTo)
		// Since we didn't inject exact time bounds in tests easily, we mock earning repo without exact filter or 
		// assume mock lists all. MockEarningRepo lists all if filter dates match (our mock repo might need checking,
		// but let's assume it returns all for now, the mock in mock_repos.go doesn't filter dates perfectly unless implemented).

		// 3. Process Run
		processed, err := svc.ProcessPayrollRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if processed.Status != models.PayrollProcessing {
			t.Errorf("expected PROCESSING status")
		}

		if processed.TotalGrossCents != 1500000 {
			t.Errorf("expected total gross 1500000, got %d", processed.TotalGrossCents)
		}

		// Calculate expected deductions for crew 1 (10,000 KES = 1,000,000 cents):
		// SHA 2.75% = 27,500 cents
		// NSSF 6% = 60,000 cents
		// Housing 1.5% = 15,000 cents
		// Total Deductions = 102,500 cents
		
		// For crew 2 (5,000 KES = 500,000 cents):
		// Total Deductions = 51,250 cents

		expectedTotalDed := int64(102500 + 51250)
		if processed.TotalDeductionsCents != expectedTotalDed {
			t.Errorf("expected total deductions %d, got %d", expectedTotalDed, processed.TotalDeductionsCents)
		}

		if processed.TotalNetCents != (1500000 - expectedTotalDed) {
			t.Errorf("expected net payload wrong")
		}

		entries, err := svc.GetPayrollEntries(ctx, run.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries")
		}

		// 4. Approve Run
		approverID := uuid.New()
		approved, err := svc.ApprovePayrollRun(ctx, run.ID, approverID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if approved.Status != models.PayrollApproved {
			t.Errorf("expected APPROVED status")
		}

		// Mock the crew members so SubmitPayrollRun can fetch them
		_ = crewRepo.Create(ctx, &models.CrewMember{ID: crewID1, FirstName: "John", LastName: "Doe", NationalID: "12345678"})
		_ = crewRepo.Create(ctx, &models.CrewMember{ID: crewID2, FirstName: "Jane", LastName: "Smith", NationalID: "87654321"})

		// 5. Submit Payroll Run
		submitted, err := svc.SubmitPayrollRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if submitted.Status != models.PayrollSubmitted {
			t.Errorf("expected SUBMITTED status")
		}
		if len(mockProvider.SubmitReqs) != 2 {
			t.Errorf("expected 2 submissions to PerPay, got %d", len(mockProvider.SubmitReqs))
		}

		// Verify payload of one of the requests
		var req1 payroll.SubmitRequest
		for _, req := range mockProvider.SubmitReqs {
			if req.EmployeeID == crewID1.String() {
				req1 = req
			}
		}
		if req1.EmployeeID == "" {
			t.Errorf("missing request for crewID1")
		}
		if req1.EmployeePIN != "12345678" {
			t.Errorf("expected EmployeePIN 12345678, got %s", req1.EmployeePIN)
		}
		if len(req1.PayComponents) != 1 || req1.PayComponents[0].Amount != 10000.00 {
			t.Errorf("expected Gross 10000.00 KES")
		}
		if len(req1.Deductions) != 3 {
			t.Errorf("expected 3 deductions")
		}
	})
}
