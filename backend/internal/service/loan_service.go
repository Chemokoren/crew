package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

var (
	ErrLowCreditScore = errors.New("credit score is too low for loan approval")
	ErrInvalidStatus  = errors.New("invalid loan status transition")
)

type LoanService interface {
	ApplyForLoan(ctx context.Context, crewMemberID uuid.UUID, amountCents int64, tenureDays int) (*models.LoanApplication, error)
	ApproveLoan(ctx context.Context, loanID uuid.UUID, lenderID uuid.UUID, approvedAmountCents int64, interestRate float64) (*models.LoanApplication, error)
	DisburseLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	RejectLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	RepayLoan(ctx context.Context, loanID uuid.UUID, amountCents int64) (*models.LoanApplication, error)
	MarkDefaulted(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	GetLoan(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error)
	ListLoans(ctx context.Context, filter repository.LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error)
	GetOverdueLoans(ctx context.Context) ([]models.LoanApplication, error)
}

type loanService struct {
	loanRepo   repository.LoanApplicationRepository
	creditRepo repository.CreditScoreRepository
	walletRepo repository.WalletRepository
	txMgr      *database.TxManager
}

func NewLoanService(
	loanRepo repository.LoanApplicationRepository,
	creditRepo repository.CreditScoreRepository,
	walletRepo repository.WalletRepository,
	txMgr *database.TxManager,
) LoanService {
	return &loanService{
		loanRepo:   loanRepo,
		creditRepo: creditRepo,
		walletRepo: walletRepo,
		txMgr:      txMgr,
	}
}

func (s *loanService) ApplyForLoan(ctx context.Context, crewMemberID uuid.UUID, amountCents int64, tenureDays int) (*models.LoanApplication, error) {
	// Optional: Check credit score before allowing application
	score, err := s.creditRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err == nil && score != nil {
		if score.Score < 400 {
			// Require at least 400 points to apply
			return nil, ErrLowCreditScore
		}
	}

	loan := &models.LoanApplication{
		CrewMemberID:         crewMemberID,
		AmountRequestedCents: amountCents,
		TenureDays:           tenureDays,
		Currency:             "KES",
		Status:               models.LoanApplied,
	}

	if err := s.loanRepo.Create(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

func (s *loanService) ApproveLoan(ctx context.Context, loanID uuid.UUID, lenderID uuid.UUID, approvedAmountCents int64, interestRate float64) (*models.LoanApplication, error) {
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	if loan.Status != models.LoanApplied {
		return nil, ErrInvalidStatus
	}

	loan.Status = models.LoanApproved
	loan.LenderID = lenderID
	loan.AmountApprovedCents = approvedAmountCents
	loan.InterestRate = interestRate

	if err := s.loanRepo.Update(ctx, loan); err != nil {
		return nil, err
	}

	return loan, nil
}

func (s *loanService) DisburseLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error) {
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	if loan.Status != models.LoanApproved {
		return nil, ErrInvalidStatus
	}

	// Wrap wallet credit + loan status update in a single transaction
	// to prevent inconsistent state (credit without status change → double disbursement)
	disburseFn := func(txCtx context.Context) error {
		wallet, err := s.walletRepo.GetByCrewMemberID(txCtx, loan.CrewMemberID)
		if err != nil {
			return fmt.Errorf("get wallet for disbursement: %w", err)
		}

		idempotencyKey := "LOAN_DISBURSE_" + loan.ID.String()
		_, err = s.walletRepo.CreditWallet(txCtx, wallet.ID, wallet.Version, loan.AmountApprovedCents,
			models.TxCatLoan, idempotencyKey, loan.ID.String(), "Loan Disbursement")
		if err != nil {
			return fmt.Errorf("credit wallet for disbursement: %w", err)
		}

		now := time.Now()
		dueAt := now.AddDate(0, 0, loan.TenureDays)
		loan.Status = models.LoanDisbursed
		loan.DisbursedAt = &now
		loan.DueAt = &dueAt

		if err := s.loanRepo.Update(txCtx, loan); err != nil {
			return fmt.Errorf("update loan status: %w", err)
		}

		return nil
	}

	if s.txMgr != nil {
		if err := s.txMgr.RunInTx(ctx, disburseFn); err != nil {
			return nil, err
		}
	} else {
		// Fallback for tests without a real DB
		if err := disburseFn(ctx); err != nil {
			return nil, err
		}
	}

	return loan, nil
}

func (s *loanService) RejectLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error) {
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	if loan.Status != models.LoanApplied {
		return nil, ErrInvalidStatus
	}

	loan.Status = models.LoanRejected
	if err := s.loanRepo.Update(ctx, loan); err != nil {
		return nil, err
	}

	return loan, nil
}

func (s *loanService) GetLoan(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error) {
	return s.loanRepo.GetByID(ctx, id)
}

func (s *loanService) ListLoans(ctx context.Context, filter repository.LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error) {
	return s.loanRepo.List(ctx, filter, page, perPage)
}

// RepayLoan processes a loan repayment. Debits the crew member's wallet and
// tracks whether repayment was on-time or late relative to due_at.
func (s *loanService) RepayLoan(ctx context.Context, loanID uuid.UUID, amountCents int64) (*models.LoanApplication, error) {
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	if loan.Status != models.LoanDisbursed && loan.Status != models.LoanRepaying {
		return nil, ErrInvalidStatus
	}

	repayFn := func(txCtx context.Context) error {
		// 1. Debit the crew member's wallet
		wallet, err := s.walletRepo.GetByCrewMemberID(txCtx, loan.CrewMemberID)
		if err != nil {
			return fmt.Errorf("get wallet for repayment: %w", err)
		}

		idempotencyKey := fmt.Sprintf("LOAN_REPAY_%s_%d", loan.ID.String(), loan.TotalRepaidCents+amountCents)
		_, err = s.walletRepo.DebitWallet(txCtx, wallet.ID, wallet.Version, amountCents,
			models.TxCatLoan, idempotencyKey, loan.ID.String(), "Loan Repayment")
		if err != nil {
			return fmt.Errorf("debit wallet for repayment: %w", err)
		}

		// 2. Track repayment amount
		loan.TotalRepaidCents += amountCents

		// 3. Calculate total owed (principal + interest)
		totalOwedCents := loan.AmountApprovedCents +
			int64(float64(loan.AmountApprovedCents)*loan.InterestRate)

		// 4. Check if fully repaid
		if loan.TotalRepaidCents >= totalOwedCents {
			now := time.Now()
			loan.Status = models.LoanCompleted
			loan.RepaidAt = &now

			// Calculate days past due (0 if on-time, positive if late)
			if loan.DueAt != nil && now.After(*loan.DueAt) {
				loan.DaysPastDue = int(now.Sub(*loan.DueAt).Hours() / 24)
			}
		} else {
			// Partial repayment — mark as REPAYING
			loan.Status = models.LoanRepaying

			// Track current days past due even for partial payments
			if loan.DueAt != nil && time.Now().After(*loan.DueAt) {
				loan.DaysPastDue = int(time.Now().Sub(*loan.DueAt).Hours() / 24)
			}
		}

		if err := s.loanRepo.Update(txCtx, loan); err != nil {
			return fmt.Errorf("update loan after repayment: %w", err)
		}

		return nil
	}

	if s.txMgr != nil {
		if err := s.txMgr.RunInTx(ctx, repayFn); err != nil {
			return nil, err
		}
	} else {
		if err := repayFn(ctx); err != nil {
			return nil, err
		}
	}

	return loan, nil
}

// MarkDefaulted marks a loan as defaulted and records the days past due.
// Called by the LoanDefaultDetectorJob when a loan is overdue beyond the grace period.
func (s *loanService) MarkDefaulted(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error) {
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	if loan.Status != models.LoanDisbursed && loan.Status != models.LoanRepaying {
		return nil, ErrInvalidStatus
	}

	loan.Status = models.LoanDefaulted
	if loan.DueAt != nil {
		loan.DaysPastDue = int(time.Now().Sub(*loan.DueAt).Hours() / 24)
	}

	if err := s.loanRepo.Update(ctx, loan); err != nil {
		return nil, err
	}

	return loan, nil
}

// GetOverdueLoans returns all disbursed/repaying loans past their due date.
func (s *loanService) GetOverdueLoans(ctx context.Context) ([]models.LoanApplication, error) {
	now := time.Now()

	// Get DISBURSED loans past due
	disbursed, _, err := s.loanRepo.List(ctx, repository.LoanApplicationFilter{
		Status: string(models.LoanDisbursed),
	}, 1, 10000)
	if err != nil {
		return nil, err
	}

	// Get REPAYING loans past due
	repaying, _, err := s.loanRepo.List(ctx, repository.LoanApplicationFilter{
		Status: string(models.LoanRepaying),
	}, 1, 10000)
	if err != nil {
		return nil, err
	}

	var overdue []models.LoanApplication
	for _, l := range append(disbursed, repaying...) {
		if l.DueAt != nil && now.After(*l.DueAt) {
			overdue = append(overdue, l)
		}
	}

	return overdue, nil
}
