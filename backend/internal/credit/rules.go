package credit

import (
	"context"
	"fmt"
	"math"
)

// RulesScorer implements the Scorer interface using a deterministic,
// multi-factor rules engine. This is the V2 scorer — designed to be
// eventually replaced or ensembled with an ML model in V3.
//
// Score Range: 300 – 850 (matches FICO scale)
//
//	Base:              300 points
//	Work History:      25% → max 138 pts
//	Income Stability:  20% → max 110 pts
//	Payment History:   30% → max 165 pts
//	Account Health:    15% → max  82 pts
//	Platform Tenure:   10% → max  55 pts
//	                         --------
//	Total Variable:            550 pts
//	Maximum Score:             850 pts
type RulesScorer struct{}

// NewRulesScorer creates a new rule-based scorer.
func NewRulesScorer() *RulesScorer {
	return &RulesScorer{}
}

func (s *RulesScorer) Version() string { return "rules-v2.1" }

func (s *RulesScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	if fv == nil {
		return nil, fmt.Errorf("nil feature vector")
	}

	var factors []ScoreFactor
	totalPoints := 300 // Base score

	// --- A. Work History (25% — max 138 pts) ---
	workFactors, workPts := s.scoreWorkHistory(fv)
	factors = append(factors, workFactors...)
	totalPoints += workPts

	// --- B. Income Stability (20% — max 110 pts) ---
	incomeFactors, incomePts := s.scoreIncome(fv)
	factors = append(factors, incomeFactors...)
	totalPoints += incomePts

	// --- C. Payment History (30% — max 165 pts) ---
	paymentFactors, paymentPts := s.scorePaymentHistory(fv)
	factors = append(factors, paymentFactors...)
	totalPoints += paymentPts

	// --- D. Account Health (15% — max 82 pts) ---
	healthFactors, healthPts := s.scoreAccountHealth(fv)
	factors = append(factors, healthFactors...)
	totalPoints += healthPts

	// --- E. Platform Tenure (10% — max 55 pts) ---
	tenureFactors, tenurePts := s.scoreTenure(fv)
	factors = append(factors, tenureFactors...)
	totalPoints += tenurePts

	// Clamp to [300, 850]
	if totalPoints > 850 {
		totalPoints = 850
	}
	if totalPoints < 300 {
		totalPoints = 300
	}

	suggestions := s.generateSuggestions(fv, totalPoints)

	return &ScoreResult{
		Score:        totalPoints,
		Grade:        ScoreGrade(totalPoints),
		Factors:      factors,
		Suggestions:  suggestions,
		ModelVersion: s.Version(),
		ComputedAt:   fv.ComputedAt,
		Features:     fv,
	}, nil
}

// --- A. Work History (max 138 pts) ---

func (s *RulesScorer) scoreWorkHistory(fv *FeatureVector) ([]ScoreFactor, int) {
	total := 0

	// A1. Completed shifts (max 60)
	shiftPts := minI(fv.CompletedShifts30d*4, 60)
	total += shiftPts
	f1 := ScoreFactor{
		Category:    "WORK_HISTORY",
		Name:        "Completed Shifts (30d)",
		Points:      shiftPts,
		MaxPoints:   60,
		Percentage:  pct(shiftPts, 60),
		Description: fmt.Sprintf("%d shifts completed in last 30 days", fv.CompletedShifts30d),
		Impact:      impact(shiftPts, 60),
	}

	// A2. Shift consistency (max 30)
	consistPts := int(math.Min(fv.ShiftConsistency, 1.0) * 30)
	total += consistPts
	f2 := ScoreFactor{
		Category:    "WORK_HISTORY",
		Name:        "Shift Consistency",
		Points:      consistPts,
		MaxPoints:   30,
		Percentage:  pct(consistPts, 30),
		Description: fmt.Sprintf("Consistency ratio: %.1f%%", fv.ShiftConsistency*100),
		Impact:      impact(consistPts, 30),
	}

	// A3. Low cancellation rate (max 28)
	cancelPts := int(28.0 * (1 - fv.CancellationRate))
	total += cancelPts
	f3 := ScoreFactor{
		Category:    "WORK_HISTORY",
		Name:        "Reliability (Low Cancellations)",
		Points:      cancelPts,
		MaxPoints:   28,
		Percentage:  pct(cancelPts, 28),
		Description: fmt.Sprintf("Cancellation rate: %.0f%%", fv.CancellationRate*100),
		Impact:      impact(cancelPts, 28),
	}

	// A4. Active days ratio (max 20)
	activePts := int(fv.ActiveDaysRatio * 20)
	total += activePts
	f4 := ScoreFactor{
		Category:    "WORK_HISTORY",
		Name:        "Active Days",
		Points:      activePts,
		MaxPoints:   20,
		Percentage:  pct(activePts, 20),
		Description: fmt.Sprintf("Active %.0f%% of last 30 days", fv.ActiveDaysRatio*100),
		Impact:      impact(activePts, 20),
	}

	return []ScoreFactor{f1, f2, f3, f4}, total
}

// --- B. Income Stability (max 110 pts) ---

func (s *RulesScorer) scoreIncome(fv *FeatureVector) ([]ScoreFactor, int) {
	total := 0

	// B1. Total verified earnings (max 40)
	earnPts := minI(int(fv.TotalEarnings30dKES/2500*40), 40)
	total += earnPts
	f1 := ScoreFactor{
		Category:    "INCOME",
		Name:        "Total Earnings (30d)",
		Points:      earnPts,
		MaxPoints:   40,
		Percentage:  pct(earnPts, 40),
		Description: fmt.Sprintf("KES %.0f earned in last 30 days", fv.TotalEarnings30dKES),
		Impact:      impact(earnPts, 40),
	}

	// B2. Income trend (max 30)
	var trendPts int
	switch fv.IncomeTrend {
	case "GROWING":
		trendPts = 30
	case "STABLE":
		trendPts = 20
	case "DECLINING":
		trendPts = 5
	default:
		trendPts = 10
	}
	total += trendPts
	f2 := ScoreFactor{
		Category:    "INCOME",
		Name:        "Income Trend",
		Points:      trendPts,
		MaxPoints:   30,
		Percentage:  pct(trendPts, 30),
		Description: fmt.Sprintf("Income trend: %s (%.0f%% MoM)", fv.IncomeTrend, fv.IncomeTrendRatio*100),
		Impact:      impact(trendPts, 30),
	}

	// B3. Earning diversity (max 20)
	divPts := minI(fv.EarningTypeDiversity*10, 20)
	total += divPts
	f3 := ScoreFactor{
		Category:    "INCOME",
		Name:        "Earning Diversity",
		Points:      divPts,
		MaxPoints:   20,
		Percentage:  pct(divPts, 20),
		Description: fmt.Sprintf("%d earning types (FIXED, COMMISSION, HYBRID)", fv.EarningTypeDiversity),
		Impact:      impact(divPts, 20),
	}

	// B4. Withdrawal-to-earning ratio (max 20)
	var wdPts int
	if fv.WithdrawalToEarningRate < 0.5 {
		wdPts = 20 // Saves more than half
	} else if fv.WithdrawalToEarningRate < 0.8 {
		wdPts = 10
	} else {
		wdPts = 0
	}
	total += wdPts
	f4 := ScoreFactor{
		Category:    "INCOME",
		Name:        "Savings Discipline",
		Points:      wdPts,
		MaxPoints:   20,
		Percentage:  pct(wdPts, 20),
		Description: fmt.Sprintf("Withdrawing %.0f%% of earnings", fv.WithdrawalToEarningRate*100),
		Impact:      impact(wdPts, 20),
	}

	return []ScoreFactor{f1, f2, f3, f4}, total
}

// --- C. Payment History (max 165 pts) ---

func (s *RulesScorer) scorePaymentHistory(fv *FeatureVector) ([]ScoreFactor, int) {
	total := 0

	// C1. Loan repayment record (max 80)
	var repayPts int
	if fv.TotalLoansCompleted+fv.TotalLoansDefaulted == 0 {
		repayPts = 20 // No history — neutral but not zero
	} else {
		repayPts = int(fv.OnTimeRepaymentRate * 80)
	}
	total += repayPts
	f1 := ScoreFactor{
		Category:    "PAYMENT_HISTORY",
		Name:        "Loan Repayment Record",
		Points:      repayPts,
		MaxPoints:   80,
		Percentage:  pct(repayPts, 80),
		Description: fmt.Sprintf("%d completed, %d defaulted (%.0f%% on-time)", fv.TotalLoansCompleted, fv.TotalLoansDefaulted, fv.OnTimeRepaymentRate*100),
		Impact:      impact(repayPts, 80),
	}

	// C2. Insurance (max 25)
	insPts := minI(fv.ActiveInsurancePolicies*13, 25)
	total += insPts
	f2 := ScoreFactor{
		Category:    "PAYMENT_HISTORY",
		Name:        "Active Insurance Policies",
		Points:      insPts,
		MaxPoints:   25,
		Percentage:  pct(insPts, 25),
		Description: fmt.Sprintf("%d active policies", fv.ActiveInsurancePolicies),
		Impact:      impact(insPts, 25),
	}

	// C3. PIN security (max 10)
	pinPts := 0
	if fv.HasPINSet {
		pinPts = 10
	}
	total += pinPts
	f3 := ScoreFactor{
		Category:    "PAYMENT_HISTORY",
		Name:        "Transaction PIN Security",
		Points:      pinPts,
		MaxPoints:   10,
		Percentage:  pct(pinPts, 10),
		Description: boolDesc(fv.HasPINSet, "PIN set", "No PIN set"),
		Impact:      impact(pinPts, 10),
	}

	// C4. Clean record — no negative incidents (max 50)
	negativePts := 50
	if fv.TotalLoansDefaulted > 0 {
		negativePts -= minI(fv.TotalLoansDefaulted*25, 50)
	}
	if negativePts < 0 {
		negativePts = 0
	}
	total += negativePts
	f4 := ScoreFactor{
		Category:    "PAYMENT_HISTORY",
		Name:        "Clean Record",
		Points:      negativePts,
		MaxPoints:   50,
		Percentage:  pct(negativePts, 50),
		Description: fmt.Sprintf("Defaults: %d", fv.TotalLoansDefaulted),
		Impact:      impact(negativePts, 50),
	}

	return []ScoreFactor{f1, f2, f3, f4}, total
}

// --- D. Account Health (max 82 pts) ---

func (s *RulesScorer) scoreAccountHealth(fv *FeatureVector) ([]ScoreFactor, int) {
	total := 0

	// D1. Balance trend (max 30)
	var balPts int
	switch fv.BalanceTrend {
	case "GROWING":
		balPts = 30
	case "STABLE":
		balPts = 15
	default:
		balPts = 5
	}
	total += balPts
	f1 := ScoreFactor{
		Category:    "ACCOUNT_HEALTH",
		Name:        "Balance Trend",
		Points:      balPts,
		MaxPoints:   30,
		Percentage:  pct(balPts, 30),
		Description: fmt.Sprintf("Trend: %s (KES %.0f current)", fv.BalanceTrend, fv.CurrentBalanceKES),
		Impact:      impact(balPts, 30),
	}

	// D2. KYC verification (max 22)
	var kycPts int
	switch fv.KYCStatus {
	case "VERIFIED":
		kycPts = 22
	case "PENDING":
		kycPts = 10
	default:
		kycPts = 0
	}
	total += kycPts
	f2 := ScoreFactor{
		Category:    "ACCOUNT_HEALTH",
		Name:        "KYC Verification",
		Points:      kycPts,
		MaxPoints:   22,
		Percentage:  pct(kycPts, 22),
		Description: fmt.Sprintf("KYC status: %s", fv.KYCStatus),
		Impact:      impact(kycPts, 22),
	}

	// D3. Average balance (max 30)
	avgBalPts := minI(int(fv.AvgBalance30dKES/1000), 30)
	total += avgBalPts
	f3 := ScoreFactor{
		Category:    "ACCOUNT_HEALTH",
		Name:        "Average Balance (30d)",
		Points:      avgBalPts,
		MaxPoints:   30,
		Percentage:  pct(avgBalPts, 30),
		Description: fmt.Sprintf("Avg KES %.0f", fv.AvgBalance30dKES),
		Impact:      impact(avgBalPts, 30),
	}

	return []ScoreFactor{f1, f2, f3}, total
}

// --- E. Platform Tenure (max 55 pts) ---

func (s *RulesScorer) scoreTenure(fv *FeatureVector) ([]ScoreFactor, int) {
	total := 0

	// E1. Account age (max 30)
	months := fv.AccountAgeDays / 30
	agePts := minI(months*3, 30)
	total += agePts
	f1 := ScoreFactor{
		Category:    "PLATFORM_TENURE",
		Name:        "Account Age",
		Points:      agePts,
		MaxPoints:   30,
		Percentage:  pct(agePts, 30),
		Description: fmt.Sprintf("%d months on platform", months),
		Impact:      impact(agePts, 30),
	}

	// E2. First shift to now (max 15)
	shiftMonths := fv.FirstShiftAgeDays / 30
	shiftAgePts := minI(shiftMonths*3, 15)
	total += shiftAgePts
	f2 := ScoreFactor{
		Category:    "PLATFORM_TENURE",
		Name:        "Work History Length",
		Points:      shiftAgePts,
		MaxPoints:   15,
		Percentage:  pct(shiftAgePts, 15),
		Description: fmt.Sprintf("%d months since first shift", shiftMonths),
		Impact:      impact(shiftAgePts, 15),
	}

	// E3. Recency (max 10)
	var recencyPts int
	if fv.DaysSinceLastShift <= 3 {
		recencyPts = 10
	} else if fv.DaysSinceLastShift <= 7 {
		recencyPts = 7
	} else if fv.DaysSinceLastShift <= 14 {
		recencyPts = 4
	} else {
		recencyPts = 0
	}
	total += recencyPts
	f3 := ScoreFactor{
		Category:    "PLATFORM_TENURE",
		Name:        "Recent Activity",
		Points:      recencyPts,
		MaxPoints:   10,
		Percentage:  pct(recencyPts, 10),
		Description: fmt.Sprintf("%d days since last shift", fv.DaysSinceLastShift),
		Impact:      impact(recencyPts, 10),
	}

	return []ScoreFactor{f1, f2, f3}, total
}

// --- Suggestions ---

func (s *RulesScorer) generateSuggestions(fv *FeatureVector, score int) []string {
	var suggestions []string

	if fv.CompletedShifts30d < 10 {
		suggestions = append(suggestions, "Complete more shifts to boost your work history score")
	}
	if fv.TotalEarnings30dKES < 10000 {
		suggestions = append(suggestions, "Increase your monthly earnings to improve your income score")
	}
	if !fv.HasPINSet {
		suggestions = append(suggestions, "Set a transaction PIN to improve your security score")
	}
	if fv.KYCStatus != "VERIFIED" {
		suggestions = append(suggestions, "Complete KYC verification for a higher account health score")
	}
	if fv.ActiveInsurancePolicies == 0 {
		suggestions = append(suggestions, "Get an insurance policy to demonstrate financial responsibility")
	}
	if fv.WithdrawalToEarningRate > 0.8 {
		suggestions = append(suggestions, "Save more of your earnings — withdrawing less improves your score")
	}
	if fv.CancellationRate > 0.2 {
		suggestions = append(suggestions, "Reduce shift cancellations to improve reliability")
	}
	if fv.DaysSinceLastShift > 7 {
		suggestions = append(suggestions, "Stay active — recent work activity boosts your score")
	}

	return suggestions
}

// --- Helpers ---

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pct(pts, max int) float64 {
	if max == 0 {
		return 0
	}
	return float64(pts) / float64(max)
}

func impact(pts, max int) string {
	ratio := pct(pts, max)
	if ratio >= 0.7 {
		return "POSITIVE"
	}
	if ratio >= 0.3 {
		return "NEUTRAL"
	}
	return "NEGATIVE"
}

func boolDesc(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}
