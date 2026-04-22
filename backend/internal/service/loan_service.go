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
	GetLoan(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error)
	ListLoans(ctx context.Context, filter repository.LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error)
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
