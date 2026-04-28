package credit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Rules Scorer Tests ---

func makeBaselineFeatures() *FeatureVector {
	return &FeatureVector{
		CrewMemberID:        uuid.New(),
		ComputedAt:          time.Now(),
		CompletedShifts30d:  15,
		CompletedShifts90d:  40,
		CancelledShifts30d:  1,
		CancellationRate:    0.06,
		ActiveDaysRatio:     0.5,
		ShiftConsistency:    0.9,
		TotalEarnings30dKES: 30000,
		TotalEarnings90dKES: 85000,
		AvgDailyEarningsKES: 1000,
		IncomeTrend:         "STABLE",
		IncomeTrendRatio:    1.0,
		EarningTypeDiversity: 2,
		WithdrawalToEarningRate: 0.4,
		TotalLoansCompleted: 3,
		TotalLoansDefaulted: 0,
		OnTimeRepaymentRate: 1.0,
		HasActiveLoan:       false,
		ActiveInsurancePolicies: 1,
		HasPINSet:           true,
		CurrentBalanceKES:   5000,
		AvgBalance30dKES:    4500,
		BalanceTrend:        "STABLE",
		KYCStatus:           "VERIFIED",
		IsActive:            true,
		AccountAgeDays:      180,
		FirstShiftAgeDays:   170,
		DaysSinceLastShift:  2,
	}
}

func TestRulesScorer_BaselineScore(t *testing.T) {
	scorer := NewRulesScorer()
	result, err := scorer.Score(context.Background(), makeBaselineFeatures())
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}
	if result.Score < 300 || result.Score > 850 {
		t.Errorf("score %d out of [300,850] range", result.Score)
	}
	if result.Grade == "" {
		t.Error("grade is empty")
	}
	if len(result.Factors) == 0 {
		t.Error("no factors returned")
	}
	if result.ModelVersion != "rules-v2.1" {
		t.Errorf("model version = %q, want rules-v2.1", result.ModelVersion)
	}
}

func TestRulesScorer_NilFeatureVector(t *testing.T) {
	scorer := NewRulesScorer()
	_, err := scorer.Score(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil feature vector")
	}
}

func TestRulesScorer_MinimumScore(t *testing.T) {
	scorer := NewRulesScorer()
	// Worst possible features — zero activity, many negatives
	fv := &FeatureVector{
		CrewMemberID:        uuid.New(),
		ComputedAt:          time.Now(),
		CancellationRate:    1.0,
		TotalLoansDefaulted: 5,
		FraudFlags:          3,
		AccountLocks:        2,
		Disputes:            3,
	}
	result, err := scorer.Score(context.Background(), fv)
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}
	// Worst-case should be at or near the floor (300).
	// Some neutral factors (e.g., clean record=0) still contribute minor points.
	if result.Score < 300 {
		t.Errorf("worst-case score = %d, below floor 300", result.Score)
	}
	if result.Score > 400 {
		t.Errorf("worst-case score = %d, should be POOR or VERY_POOR", result.Score)
	}
	if result.Grade != "VERY_POOR" && result.Grade != "POOR" {
		t.Errorf("worst-case grade = %q, want VERY_POOR or POOR", result.Grade)
	}
}

func TestRulesScorer_MaximumScore(t *testing.T) {
	scorer := NewRulesScorer()
	// Best possible features
	fv := &FeatureVector{
		CrewMemberID:        uuid.New(),
		ComputedAt:          time.Now(),
		CompletedShifts30d:  30,
		CompletedShifts90d:  90,
		ShiftConsistency:    1.0,
		ActiveDaysRatio:     1.0,
		TotalEarnings30dKES: 100000,
		TotalEarnings90dKES: 300000,
		AvgDailyEarningsKES: 3333,
		IncomeTrend:         "GROWING",
		IncomeTrendRatio:    1.5,
		EarningTypeDiversity: 5,
		WithdrawalToEarningRate: 0.2,
		TotalLoansCompleted: 10,
		OnTimeRepaymentRate: 1.0,
		ActiveInsurancePolicies: 2,
		HasPINSet:           true,
		CurrentBalanceKES:   50000,
		AvgBalance30dKES:    45000,
		BalanceTrend:        "GROWING",
		KYCStatus:           "VERIFIED",
		IsActive:            true,
		AccountAgeDays:      400,
		FirstShiftAgeDays:   390,
		DaysSinceLastShift:  0,
	}
	result, err := scorer.Score(context.Background(), fv)
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}
	if result.Score != 850 {
		t.Errorf("best-case score = %d, want 850 (ceiling)", result.Score)
	}
}

func TestRulesScorer_DefaultsPenalize(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	clean := makeBaselineFeatures()
	cleanResult, _ := scorer.Score(ctx, clean)

	defaulted := makeBaselineFeatures()
	defaulted.TotalLoansDefaulted = 2
	defaultedResult, _ := scorer.Score(ctx, defaulted)

	if defaultedResult.Score >= cleanResult.Score {
		t.Errorf("defaulted score %d >= clean score %d", defaultedResult.Score, cleanResult.Score)
	}
}

func TestRulesScorer_FraudPenalizes(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	clean := makeBaselineFeatures()
	cleanResult, _ := scorer.Score(ctx, clean)

	fraud := makeBaselineFeatures()
	fraud.FraudFlags = 2
	fraudResult, _ := scorer.Score(ctx, fraud)

	if fraudResult.Score >= cleanResult.Score {
		t.Errorf("fraud score %d >= clean score %d", fraudResult.Score, cleanResult.Score)
	}
}

func TestRulesScorer_CRBBonusApplied(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	noCRB := makeBaselineFeatures()
	noCRBResult, _ := scorer.Score(ctx, noCRB)

	withCRB := makeBaselineFeatures()
	withCRB.CRBScoreAvailable = true
	withCRB.CRBScore = 700
	withCRB.CRBTotalLoans = 5
	withCRBResult, _ := scorer.Score(ctx, withCRB)

	if withCRBResult.Score <= noCRBResult.Score {
		t.Errorf("CRB bonus not applied: with=%d, without=%d", withCRBResult.Score, noCRBResult.Score)
	}
}

func TestRulesScorer_CRBDefaultsPenalize(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	goodCRB := makeBaselineFeatures()
	goodCRB.CRBScoreAvailable = true
	goodCRB.CRBScore = 700
	goodCRBResult, _ := scorer.Score(ctx, goodCRB)

	badCRB := makeBaselineFeatures()
	badCRB.CRBScoreAvailable = true
	badCRB.CRBScore = 700
	badCRB.CRBDefaultedLoans = 3
	badCRBResult, _ := scorer.Score(ctx, badCRB)

	if badCRBResult.Score >= goodCRBResult.Score {
		t.Errorf("CRB defaults not penalizing: bad=%d >= good=%d", badCRBResult.Score, goodCRBResult.Score)
	}
}

func TestRulesScorer_FactorCategories(t *testing.T) {
	scorer := NewRulesScorer()
	result, _ := scorer.Score(context.Background(), makeBaselineFeatures())

	categories := make(map[string]int)
	for _, f := range result.Factors {
		categories[f.Category]++
	}

	required := []string{"WORK_HISTORY", "INCOME", "PAYMENT_HISTORY", "ACCOUNT_HEALTH", "PLATFORM_TENURE"}
	for _, cat := range required {
		if categories[cat] == 0 {
			t.Errorf("missing factor category: %s", cat)
		}
	}
}

func TestRulesScorer_FactorPointsNonNegative(t *testing.T) {
	scorer := NewRulesScorer()
	result, _ := scorer.Score(context.Background(), makeBaselineFeatures())

	for _, f := range result.Factors {
		if f.Category != "PAYMENT_HISTORY" && f.Category != "EXTERNAL_CREDIT" && f.Points < 0 {
			t.Errorf("factor %s/%s has negative points %d", f.Category, f.Name, f.Points)
		}
		if f.MaxPoints < 0 {
			t.Errorf("factor %s/%s has negative max %d", f.Category, f.Name, f.MaxPoints)
		}
	}
}

func TestRulesScorer_Suggestions(t *testing.T) {
	scorer := NewRulesScorer()

	// Low-score user should get suggestions
	fv := &FeatureVector{
		CrewMemberID: uuid.New(),
		ComputedAt:   time.Now(),
		CompletedShifts30d: 2,
		KYCStatus: "PENDING",
	}
	result, _ := scorer.Score(context.Background(), fv)
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for low-score user")
	}
}

func TestRulesScorer_Deterministic(t *testing.T) {
	scorer := NewRulesScorer()
	fv := makeBaselineFeatures()
	ctx := context.Background()

	r1, _ := scorer.Score(ctx, fv)
	r2, _ := scorer.Score(ctx, fv)

	if r1.Score != r2.Score {
		t.Errorf("non-deterministic: %d != %d", r1.Score, r2.Score)
	}
}

func TestRulesScorer_IncomeGrowthBoosts(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	stable := makeBaselineFeatures()
	stable.IncomeTrend = "STABLE"
	stable.IncomeTrendRatio = 1.0
	stableResult, _ := scorer.Score(ctx, stable)

	growing := makeBaselineFeatures()
	growing.IncomeTrend = "GROWING"
	growing.IncomeTrendRatio = 1.5
	growingResult, _ := scorer.Score(ctx, growing)

	if growingResult.Score <= stableResult.Score {
		t.Errorf("growing income %d should beat stable %d", growingResult.Score, stableResult.Score)
	}
}

func TestRulesScorer_KYCVerifiedBoosts(t *testing.T) {
	scorer := NewRulesScorer()
	ctx := context.Background()

	verified := makeBaselineFeatures()
	verified.KYCStatus = "VERIFIED"
	vResult, _ := scorer.Score(ctx, verified)

	pending := makeBaselineFeatures()
	pending.KYCStatus = "PENDING"
	pResult, _ := scorer.Score(ctx, pending)

	if vResult.Score <= pResult.Score {
		t.Errorf("verified KYC %d should beat pending %d", vResult.Score, pResult.Score)
	}
}
