package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/credit"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

var (
	ErrLowCreditScore    = errors.New("credit score is too low for loan approval")
	ErrInvalidStatus     = errors.New("invalid loan status transition")
	ErrAmountExceedsTier = errors.New("requested amount exceeds your loan tier limit")
	ErrTenureExceedsTier = errors.New("requested tenure exceeds your loan tier limit")
	ErrLoanCooldown      = errors.New("you must wait before applying for another loan")
	ErrActiveLoan        = errors.New("you already have an active loan")
	ErrActiveLoanInCat   = errors.New("you already have an active loan in this category")
	ErrExposureLimit     = errors.New("total outstanding would exceed your exposure limit")
	ErrCategoryDisabled  = errors.New("this loan category is not currently available")
	ErrInvalidCategory   = errors.New("unrecognized loan category")
)

type LoanService interface {
	ApplyForLoan(ctx context.Context, crewMemberID uuid.UUID, amountCents int64, tenureDays int, category models.LoanCategory, purpose string) (*models.LoanApplication, error)
	ApproveLoan(ctx context.Context, loanID uuid.UUID, lenderID uuid.UUID, approvedAmountCents int64, interestRate float64) (*models.LoanApplication, error)
	DisburseLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	RejectLoan(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	RepayLoan(ctx context.Context, loanID uuid.UUID, amountCents int64) (*models.LoanApplication, error)
	MarkDefaulted(ctx context.Context, loanID uuid.UUID) (*models.LoanApplication, error)
	GetLoan(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error)
	ListLoans(ctx context.Context, filter repository.LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error)
	GetOverdueLoans(ctx context.Context) ([]models.LoanApplication, error)
	GetLoanTier(ctx context.Context, crewMemberID uuid.UUID) (*credit.LoanTier, int, error)
	GetLoanPolicy() *models.LoanPolicyConfig
}

type loanService struct {
	loanRepo   repository.LoanApplicationRepository
	creditRepo repository.CreditScoreRepository
	walletRepo repository.WalletRepository
	txMgr      *database.TxManager
	policy     *models.LoanPolicyConfig
}

func NewLoanService(
	loanRepo repository.LoanApplicationRepository,
	creditRepo repository.CreditScoreRepository,
	walletRepo repository.WalletRepository,
	txMgr *database.TxManager,
	opts ...LoanServiceOption,
) LoanService {
	svc := &loanService{
		loanRepo:   loanRepo,
		creditRepo: creditRepo,
		walletRepo: walletRepo,
		txMgr:      txMgr,
		policy:     models.DefaultLoanPolicy(),
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// LoanServiceOption is a functional option for configuring LoanService.
type LoanServiceOption func(*loanService)

// WithLoanPolicy sets the concurrent loan policy configuration.
func WithLoanPolicy(policy *models.LoanPolicyConfig) LoanServiceOption {
	return func(s *loanService) {
		if policy != nil {
			s.policy = policy
		}
	}
}

// GetLoanPolicy exposes the current lending policy for USSD/API introspection.
func (s *loanService) GetLoanPolicy() *models.LoanPolicyConfig {
	return s.policy
}

// isActiveLoanStatus returns true if the loan status is considered "active".
func isActiveLoanStatus(status models.LoanStatus) bool {
	return status == models.LoanApplied || status == models.LoanApproved ||
		status == models.LoanDisbursed || status == models.LoanRepaying
}

func (s *loanService) ApplyForLoan(ctx context.Context, crewMemberID uuid.UUID, amountCents int64, tenureDays int, category models.LoanCategory, purpose string) (*models.LoanApplication, error) {
	// 0. Validate category
	if category == "" {
		category = models.LoanCatPersonal
	}
	if !category.IsValid() {
		return nil, ErrInvalidCategory
	}
	if !s.policy.IsCategoryEnabled(category) {
		return nil, fmt.Errorf("%w: %s", ErrCategoryDisabled, category.Label())
	}

	// 1. Get credit score and resolve loan tier
	score, err := s.creditRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err != nil || score == nil {
		return nil, ErrLowCreditScore
	}

	tier := credit.GetTierForScore(score.Score)
	if tier == nil {
		return nil, ErrLowCreditScore
	}

	// 2. Enforce tier limits
	if amountCents > tier.MaxLoanCents {
		return nil, fmt.Errorf("%w: max KES %.0f for your %s tier",
			ErrAmountExceedsTier, tier.FormatMaxLoanKES(), tier.Grade)
	}

	if tenureDays > tier.MaxTenureDays {
		return nil, fmt.Errorf("%w: max %d days for your %s tier",
			ErrTenureExceedsTier, tier.MaxTenureDays, tier.Grade)
	}

	// 3. Enforce concurrency policy
	loans, _, _ := s.loanRepo.List(ctx, repository.LoanApplicationFilter{
		CrewMemberID: &crewMemberID,
	}, 1, 50) // Fetch all recent loans for policy checks

	if err := s.checkConcurrencyPolicy(loans, category, amountCents, tier); err != nil {
		return nil, err
	}

	// 4. Check cooldown between completed loans (per-category in PER_CATEGORY mode)
	for _, l := range loans {
		if tier.CooldownDays > 0 && l.Status == models.LoanCompleted && l.RepaidAt != nil {
			// In PER_CATEGORY mode, cooldown only applies within the same category
			if s.policy.ConcurrencyPolicy == models.PolicyPerCategory && l.Category != category {
				continue
			}
			cooldownEnd := l.RepaidAt.AddDate(0, 0, tier.CooldownDays)
			if time.Now().Before(cooldownEnd) {
				return nil, fmt.Errorf("%w: next eligible on %s",
					ErrLoanCooldown, cooldownEnd.Format("2006-01-02"))
			}
		}
	}

	// 5. Create loan with tier-assigned interest rate
	loan := &models.LoanApplication{
		CrewMemberID:         crewMemberID,
		Category:             category,
		Purpose:              purpose,
		AmountRequestedCents: amountCents,
		InterestRate:         tier.InterestRate,
		TenureDays:           tenureDays,
		Currency:             "KES",
		Status:               models.LoanApplied,
	}

	if err := s.loanRepo.Create(ctx, loan); err != nil {
		return nil, err
	}
	return loan, nil
}

// checkConcurrencyPolicy enforces the system's loan concurrency rules.
//
// Policy behaviors:
//
//	SINGLE       → Block if ANY active loan exists (conservative, default)
//	PER_CATEGORY → Block only if active loan exists in the SAME category
//	              (allows Personal + Emergency simultaneously)
//	AGGREGATE    → Block if total active count exceeds MaxConcurrentLoans
//	              OR if total outstanding principal + new amount > exposure limit
func (s *loanService) checkConcurrencyPolicy(
	loans []models.LoanApplication,
	category models.LoanCategory,
	newAmountCents int64,
	tier *credit.LoanTier,
) error {
	var activeCount int
	var totalOutstandingCents int64

	for _, l := range loans {
		if !isActiveLoanStatus(l.Status) {
			continue
		}
		activeCount++

		// Track outstanding principal for aggregate exposure check
		outstandingCents := l.AmountRequestedCents
		if l.AmountApprovedCents > 0 {
			outstandingCents = l.AmountApprovedCents
		}
		totalOutstandingCents += outstandingCents - l.TotalRepaidCents

		// Per-policy blocking
		switch s.policy.ConcurrencyPolicy {
		case models.PolicySingle:
			return ErrActiveLoan

		case models.PolicyPerCategory:
			if l.Category == category {
				return fmt.Errorf("%w: you have an active %s loan",
					ErrActiveLoanInCat, category.Label())
			}

		case models.PolicyAggregate:
			// Checked after the loop (need total counts)
		}
	}

	// Aggregate-specific checks
	if s.policy.ConcurrencyPolicy == models.PolicyAggregate {
		// Check max concurrent count
		if activeCount >= s.policy.MaxConcurrentLoans {
			return fmt.Errorf("%w: maximum %d concurrent loans allowed",
				ErrActiveLoan, s.policy.MaxConcurrentLoans)
		}

		// Check absolute exposure limit
		if s.policy.MaxAggregateExposureCents > 0 {
			if totalOutstandingCents+newAmountCents > s.policy.MaxAggregateExposureCents {
				return fmt.Errorf("%w: total outstanding would be KES %.0f (max KES %.0f)",
					ErrExposureLimit,
					float64(totalOutstandingCents+newAmountCents)/100,
					float64(s.policy.MaxAggregateExposureCents)/100)
			}
		}

		// Check tier-based exposure multiplier
		if s.policy.AggregateExposureMultiplier > 0 && tier != nil {
			maxExposure := int64(float64(tier.MaxLoanCents) * s.policy.AggregateExposureMultiplier)
			if totalOutstandingCents+newAmountCents > maxExposure {
				return fmt.Errorf("%w: total outstanding would be KES %.0f (max KES %.0f for your %s tier)",
					ErrExposureLimit,
					float64(totalOutstandingCents+newAmountCents)/100,
					float64(maxExposure)/100,
					tier.Grade)
			}
		}
	}

	return nil
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
	loan.LenderID = &lenderID
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

// GetLoanTier returns the loan tier and credit score for a crew member.
// Returns (tier, score, nil) on success, or (nil, score, ErrLowCreditScore) if ineligible.
func (s *loanService) GetLoanTier(ctx context.Context, crewMemberID uuid.UUID) (*credit.LoanTier, int, error) {
	scoreRecord, err := s.creditRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err != nil || scoreRecord == nil {
		return nil, 0, ErrLowCreditScore
	}

	tier := credit.GetTierForScore(scoreRecord.Score)
	if tier == nil {
		return nil, scoreRecord.Score, ErrLowCreditScore
	}

	return tier, scoreRecord.Score, nil
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
