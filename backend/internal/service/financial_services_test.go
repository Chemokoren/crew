package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func TestCreditService(t *testing.T) {
	creditRepo := mock.NewCreditScoreRepo()
	
	// Create fully mocked env
	mockEarningRepo := &mockEarningRepo{} // From financial_test.go
	mockAssignmentRepo := mock.NewAssignmentRepo()

	svc := NewCreditService(creditRepo, mockEarningRepo, mockAssignmentRepo)
	ctx := context.Background()
	crewID := uuid.New()

	// 1. Calculate Score (Empty State)
	score, err := svc.CalculateScore(ctx, crewID)
	if err != nil {
		t.Fatalf("CalculateScore: %v", err)
	}

	if score.Score != 300 {
		t.Errorf("Expected base score 300, got %d", score.Score)
	}

	// 2. Fetch Score
	fetched, err := svc.GetScore(ctx, crewID)
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if fetched == nil || fetched.Score != 300 {
		t.Errorf("GetScore returned wrong score")
	}
}

func TestLoanService(t *testing.T) {
	loanRepo := mock.NewLoanApplicationRepo()
	creditRepo := mock.NewCreditScoreRepo()
	walletRepo := mock.NewWalletRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditSvc := NewAuditService(mock.NewAuditRepo(), logger)
	crewRepo := mock.NewCrewRepo()
	walletSvc := NewWalletService(walletRepo, crewRepo, auditSvc, logger)

	svc := NewLoanService(loanRepo, creditRepo, walletRepo, nil) // nil txMgr for unit tests
	ctx := context.Background()

	// Pre-create crew member and wallet
	crew := &models.CrewMember{
		ID:        uuid.New(),
		CrewID:    "CRW-12345",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleDriver,
		KYCStatus: models.KYCVerified,
		IsActive:  true,
	}
	crewRepo.Create(ctx, crew)
	walletSvc.GetOrCreateWallet(ctx, crew.ID)

	// Pre-create credit score to allow loan application (> 400 points)
	creditScore := &models.CreditScore{
		CrewMemberID: crew.ID,
		Score:        500,
	}
	creditRepo.Upsert(ctx, creditScore)

	// 1. Apply For Loan
	loan, err := svc.ApplyForLoan(ctx, crew.ID, 100000, 30) // 1000 KES, 30 days
	if err != nil {
		t.Fatalf("ApplyForLoan: %v", err)
	}
	if loan.Status != models.LoanApplied {
		t.Errorf("Expected APPLIED, got %v", loan.Status)
	}

	// 2. Approve Loan
	lenderID := uuid.New()
	loan, err = svc.ApproveLoan(ctx, loan.ID, lenderID, 100000, 0.05) // 5% interest
	if err != nil {
		t.Fatalf("ApproveLoan: %v", err)
	}
	if loan.Status != models.LoanApproved {
		t.Errorf("Expected APPROVED, got %v", loan.Status)
	}

	// 3. Disburse Loan
	loan, err = svc.DisburseLoan(ctx, loan.ID)
	if err != nil {
		t.Fatalf("DisburseLoan: %v", err)
	}
	if loan.Status != models.LoanDisbursed {
		t.Errorf("Expected DISBURSED, got %v", loan.Status)
	}

	// Check Wallet Balance
	wallet, _ := walletSvc.GetBalance(ctx, crew.ID)
	if wallet.BalanceCents != 100000 {
		t.Errorf("Expected wallet balance 100000, got %d", wallet.BalanceCents)
	}
}

func TestInsuranceService(t *testing.T) {
	repo := mock.NewInsurancePolicyRepo()
	svc := NewInsuranceService(repo, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	ctx := context.Background()
	crewID := uuid.New()

	// 1. Create Policy
	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	policy, err := svc.CreatePolicy(ctx, crewID, "NHIF", "HEALTH", "MONTHLY", 50000, startDate, endDate)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if policy.Status != models.PolicyActive {
		t.Errorf("Expected ACTIVE, got %v", policy.Status)
	}

	// 2. Mark Lapsed
	err = svc.MarkPolicyLapsed(ctx, policy.ID)
	if err != nil {
		t.Fatalf("MarkPolicyLapsed: %v", err)
	}

	fetched, _ := svc.GetPolicy(ctx, policy.ID)
	if fetched.Status != models.PolicyLapsed {
		t.Errorf("Expected LAPSED, got %v", fetched.Status)
	}
}
