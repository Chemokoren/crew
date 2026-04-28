package credit

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// FeatureComputer computes a FeatureVector from raw database records.
// This is the "Feature Engineering" layer.
type FeatureComputer struct {
	earningRepo    repository.EarningRepository
	assignmentRepo repository.AssignmentRepository
	walletRepo     repository.WalletRepository
	loanRepo       repository.LoanApplicationRepository
	insuranceRepo  repository.InsurancePolicyRepository
	crewRepo       repository.CrewRepository
	userRepo       repository.UserRepository
	snapshotRepo   repository.WalletSnapshotRepository
	negativeRepo   repository.NegativeEventRepository
	logger         *slog.Logger
}

// NewFeatureComputer creates a new FeatureComputer.
func NewFeatureComputer(
	earningRepo repository.EarningRepository,
	assignmentRepo repository.AssignmentRepository,
	walletRepo repository.WalletRepository,
	loanRepo repository.LoanApplicationRepository,
	insuranceRepo repository.InsurancePolicyRepository,
	crewRepo repository.CrewRepository,
	userRepo repository.UserRepository,
	snapshotRepo repository.WalletSnapshotRepository,
	negativeRepo repository.NegativeEventRepository,
	logger *slog.Logger,
) *FeatureComputer {
	return &FeatureComputer{
		earningRepo:    earningRepo,
		assignmentRepo: assignmentRepo,
		walletRepo:     walletRepo,
		loanRepo:       loanRepo,
		insuranceRepo:  insuranceRepo,
		crewRepo:       crewRepo,
		userRepo:       userRepo,
		snapshotRepo:   snapshotRepo,
		negativeRepo:   negativeRepo,
		logger:         logger,
	}
}

// Compute builds a full FeatureVector for a crew member by querying all data sources.
func (fc *FeatureComputer) Compute(ctx context.Context, crewMemberID uuid.UUID) (*FeatureVector, error) {
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	ninetyDaysAgo := now.AddDate(0, 0, -90)
	sixtyDaysAgo := now.AddDate(0, 0, -60)

	fv := &FeatureVector{
		CrewMemberID: crewMemberID,
		ComputedAt:   now,
	}

	// --- Work History ---
	fc.computeWorkHistory(ctx, fv, crewMemberID, thirtyDaysAgo, ninetyDaysAgo, sixtyDaysAgo, now)

	// --- Income ---
	fc.computeIncome(ctx, fv, crewMemberID, thirtyDaysAgo, ninetyDaysAgo, sixtyDaysAgo, now)

	// --- Payment History ---
	fc.computePaymentHistory(ctx, fv, crewMemberID)

	// --- Account Health ---
	fc.computeAccountHealth(ctx, fv, crewMemberID)

	// --- Platform Tenure ---
	fc.computeTenure(ctx, fv, crewMemberID, now)

	return fv, nil
}

func (fc *FeatureComputer) computeWorkHistory(
	ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID,
	thirtyDaysAgo, ninetyDaysAgo, sixtyDaysAgo, now time.Time,
) {
	// 30-day assignments
	assignments30, _, err := fc.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &thirtyDaysAgo,
	}, 1, 10000)
	if err != nil {
		fc.logger.Warn("credit: failed to fetch 30d assignments", slog.String("error", err.Error()))
	}

	completed30, cancelled30 := 0, 0
	activeDays := make(map[string]bool)
	var lastShiftDate time.Time
	for _, a := range assignments30 {
		if a.Status == models.AssignmentCompleted {
			completed30++
			activeDays[a.ShiftDate.Format("2006-01-02")] = true
			if a.ShiftDate.After(lastShiftDate) {
				lastShiftDate = a.ShiftDate
			}
		} else if a.Status == models.AssignmentCancelled {
			cancelled30++
		}
	}
	fv.CompletedShifts30d = completed30
	fv.CancelledShifts30d = cancelled30
	fv.ActiveDaysRatio = float64(len(activeDays)) / 30.0
	if fv.ActiveDaysRatio > 1.0 {
		fv.ActiveDaysRatio = 1.0
	}

	total30 := completed30 + cancelled30
	if total30 > 0 {
		fv.CancellationRate = float64(cancelled30) / float64(total30)
	}

	if !lastShiftDate.IsZero() {
		fv.DaysSinceLastShift = int(now.Sub(lastShiftDate).Hours() / 24)
	} else {
		fv.DaysSinceLastShift = 999
	}

	// 90-day assignments for trend
	assignments90, _, err := fc.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &ninetyDaysAgo,
	}, 1, 10000)
	if err == nil {
		completed90 := 0
		for _, a := range assignments90 {
			if a.Status == models.AssignmentCompleted {
				completed90++
			}
		}
		fv.CompletedShifts90d = completed90
	}

	// Shift consistency: this month vs previous month
	assignmentsPrev, _, err := fc.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &sixtyDaysAgo,
		DateTo:       &thirtyDaysAgo,
	}, 1, 10000)
	if err == nil {
		completedPrev := 0
		for _, a := range assignmentsPrev {
			if a.Status == models.AssignmentCompleted {
				completedPrev++
			}
		}
		if completedPrev > 0 {
			fv.ShiftConsistency = math.Min(float64(completed30)/float64(completedPrev), 1.5)
		} else if completed30 > 0 {
			fv.ShiftConsistency = 1.0 // First month of work
		}
	}
}

func (fc *FeatureComputer) computeIncome(
	ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID,
	thirtyDaysAgo, ninetyDaysAgo, sixtyDaysAgo, now time.Time,
) {
	// 30-day earnings
	isVerified := true
	earnings30, _, err := fc.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &thirtyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	if err != nil {
		fc.logger.Warn("credit: failed to fetch 30d earnings", slog.String("error", err.Error()))
	}

	var total30Cents int64
	earningTypes := make(map[string]bool)
	for _, e := range earnings30 {
		total30Cents += e.AmountCents
		earningTypes[string(e.EarningType)] = true
	}
	fv.TotalEarnings30dKES = float64(total30Cents) / 100
	fv.EarningTypeDiversity = len(earningTypes)

	activeDays := fv.ActiveDaysRatio * 30
	if activeDays > 0 {
		fv.AvgDailyEarningsKES = fv.TotalEarnings30dKES / activeDays
	}

	// 90-day earnings
	earnings90, _, err := fc.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &ninetyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	if err == nil {
		var total90Cents int64
		for _, e := range earnings90 {
			total90Cents += e.AmountCents
		}
		fv.TotalEarnings90dKES = float64(total90Cents) / 100
	}

	// Income trend: compare last 30d vs previous 30d
	earningsPrev, _, err := fc.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &sixtyDaysAgo,
		DateTo:       &thirtyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	if err == nil {
		var prevCents int64
		for _, e := range earningsPrev {
			prevCents += e.AmountCents
		}
		if prevCents > 0 {
			ratio := float64(total30Cents) / float64(prevCents)
			fv.IncomeTrendRatio = ratio
			if ratio > 1.1 {
				fv.IncomeTrend = "GROWING"
			} else if ratio > 0.9 {
				fv.IncomeTrend = "STABLE"
			} else {
				fv.IncomeTrend = "DECLINING"
			}
		} else if total30Cents > 0 {
			fv.IncomeTrend = "GROWING"
			fv.IncomeTrendRatio = 2.0
		} else {
			fv.IncomeTrend = "STABLE"
			fv.IncomeTrendRatio = 1.0
		}
	}

	// Withdrawal-to-earning ratio
	if fv.TotalEarnings30dKES > 0 {
		wallet, err := fc.walletRepo.GetByCrewMemberID(ctx, crewMemberID)
		if err == nil {
			withdrawals, _, _ := fc.walletRepo.GetTransactions(ctx, wallet.ID, repository.TxFilter{
				Category: "WITHDRAWAL",
				DateFrom: &thirtyDaysAgo,
			}, 1, 10000)
			var totalWithdrawnCents int64
			for _, tx := range withdrawals {
				totalWithdrawnCents += tx.AmountCents
			}
			fv.WithdrawalToEarningRate = float64(totalWithdrawnCents) / float64(total30Cents)
		}
	}
}

func (fc *FeatureComputer) computePaymentHistory(ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID) {
	// Loan history
	loans, _, err := fc.loanRepo.List(ctx, repository.LoanApplicationFilter{
		CrewMemberID: &crewMemberID,
	}, 1, 1000)
	if err != nil {
		fc.logger.Warn("credit: failed to fetch loans", slog.String("error", err.Error()))
	}

	completed, defaulted, onTime := 0, 0, 0
	hasActive := false
	for _, l := range loans {
		switch l.Status {
		case models.LoanCompleted:
			completed++
			// Use actual due-date tracking: repaid_at vs due_at
			if l.WasRepaidOnTime() {
				onTime++
			}
			// If repaid_at or due_at is nil (legacy data), count as on-time
			if l.RepaidAt == nil || l.DueAt == nil {
				onTime++
			}
		case models.LoanDefaulted:
			defaulted++
		case models.LoanDisbursed, models.LoanRepaying, models.LoanApproved, models.LoanApplied:
			hasActive = true
		}
	}
	fv.TotalLoansCompleted = completed
	fv.TotalLoansDefaulted = defaulted
	fv.HasActiveLoan = hasActive
	if completed+defaulted > 0 {
		fv.OnTimeRepaymentRate = float64(onTime) / float64(completed+defaulted)
		// Clamp to [0, 1] — legacy double-counting could exceed 1.0
		if fv.OnTimeRepaymentRate > 1.0 {
			fv.OnTimeRepaymentRate = 1.0
		}
	} else {
		fv.OnTimeRepaymentRate = 0.5 // Neutral — no loan history
	}

	// Insurance policies
	policies, _, err := fc.insuranceRepo.List(ctx, repository.InsurancePolicyFilter{
		CrewMemberID: &crewMemberID,
		Status:       "ACTIVE",
	}, 1, 100)
	if err == nil {
		fv.ActiveInsurancePolicies = len(policies)
	}

	// PIN status
	user, err := fc.userRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err == nil && user != nil {
		fv.HasPINSet = user.PINHash != ""
	}

	// Negative events
	if fc.negativeRepo != nil {
		fc.computeNegativeEvents(ctx, fv, crewMemberID)
	}
}

func (fc *FeatureComputer) computeAccountHealth(ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID) {
	wallet, err := fc.walletRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err != nil {
		return
	}
	fv.CurrentBalanceKES = float64(wallet.BalanceCents) / 100
	fv.IsActive = wallet.IsActive

	// Compute true 30-day average balance from daily snapshots.
	// Falls back to current balance if no snapshots exist yet (new deployment).
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	avgCents, err := fc.snapshotRepo.GetAvgBalance(ctx, crewMemberID, thirtyDaysAgo, time.Now())
	if err != nil || avgCents == 0 {
		// No snapshots yet — use current balance as fallback
		fv.AvgBalance30dKES = fv.CurrentBalanceKES
	} else {
		fv.AvgBalance30dKES = float64(avgCents) / 100
	}

	// Balance trend: compare first-half vs second-half of 30-day snapshots
	snapshots, err := fc.snapshotRepo.GetSnapshots(ctx, crewMemberID, thirtyDaysAgo, time.Now())
	if err == nil && len(snapshots) >= 4 {
		// Split snapshots into first half and second half
		mid := len(snapshots) / 2
		var firstHalfSum, secondHalfSum int64
		for _, s := range snapshots[:mid] {
			firstHalfSum += s.BalanceCents
		}
		for _, s := range snapshots[mid:] {
			secondHalfSum += s.BalanceCents
		}
		firstHalfAvg := float64(firstHalfSum) / float64(mid)
		secondHalfAvg := float64(secondHalfSum) / float64(len(snapshots)-mid)

		if secondHalfAvg > firstHalfAvg*1.1 {
			fv.BalanceTrend = "GROWING"
		} else if secondHalfAvg > firstHalfAvg*0.9 {
			fv.BalanceTrend = "STABLE"
		} else {
			fv.BalanceTrend = "DECLINING"
		}
	} else {
		// Not enough snapshots — use savings-rate heuristic as fallback
		if wallet.TotalCreditedCents > 0 {
			savingsRate := float64(wallet.BalanceCents) / float64(wallet.TotalCreditedCents)
			if savingsRate > 0.3 {
				fv.BalanceTrend = "GROWING"
			} else if savingsRate > 0.1 {
				fv.BalanceTrend = "STABLE"
			} else {
				fv.BalanceTrend = "DECLINING"
			}
		} else {
			fv.BalanceTrend = "STABLE"
		}
	}

	// KYC status
	crew, err := fc.crewRepo.GetByID(ctx, crewMemberID)
	if err == nil {
		fv.KYCStatus = string(crew.KYCStatus)
	}
}

func (fc *FeatureComputer) computeTenure(ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID, now time.Time) {
	crew, err := fc.crewRepo.GetByID(ctx, crewMemberID)
	if err != nil {
		return
	}
	fv.AccountAgeDays = int(now.Sub(crew.CreatedAt).Hours() / 24)

	// First shift age
	veryOld := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	assignments, _, err := fc.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &veryOld,
	}, 1, 1)
	if err == nil && len(assignments) > 0 {
		fv.FirstShiftAgeDays = int(now.Sub(assignments[0].ShiftDate).Hours() / 24)
	}
}

func (fc *FeatureComputer) computeNegativeEvents(ctx context.Context, fv *FeatureVector, crewMemberID uuid.UUID) {
	total, err := fc.negativeRepo.CountUnresolved(ctx, crewMemberID)
	if err != nil {
		return
	}
	fv.UnresolvedNegativeEvents = int(total)

	fraud, _ := fc.negativeRepo.CountByType(ctx, crewMemberID, "FRAUD_FLAG")
	fv.FraudFlags = int(fraud)

	disputes, _ := fc.negativeRepo.CountByType(ctx, crewMemberID, "DISPUTE")
	fv.Disputes = int(disputes)

	locks, _ := fc.negativeRepo.CountByType(ctx, crewMemberID, "ACCOUNT_LOCK")
	fv.AccountLocks = int(locks)
}
