package credit

// LoanTier defines the loan parameters for a credit score grade.
// Tiers are used for graduated lending — users start with low limits
// and earn higher tiers through successful repayment.
type LoanTier struct {
	Grade           string  `json:"grade"`            // "EXCELLENT", "GOOD", "FAIR", "POOR"
	MinScore        int     `json:"min_score"`        // Minimum score for this tier
	MaxLoanCents    int64   `json:"max_loan_cents"`   // Maximum loan amount in cents
	InterestRate    float64 `json:"interest_rate"`     // Annual interest rate (decimal)
	MaxTenureDays   int     `json:"max_tenure_days"`  // Maximum repayment period
	CooldownDays    int     `json:"cooldown_days"`     // Days between loans
	Description     string  `json:"description"`      // Human-readable label
}

// DefaultTiers returns the standard graduated lending tiers.
//
// Design rationale:
//   - POOR (400-499):  Starter loans — KES 1,000 max, 7 days, 15% rate
//     The purpose is to establish a repayment track record. Low risk to lender.
//   - FAIR (500-649):  Growth loans — KES 5,000 max, 14 days, 12% rate
//     Proven reliability. Higher limits, longer tenure.
//   - GOOD (650-749):  Standard loans — KES 20,000 max, 30 days, 8% rate
//     Consistent earner with clean repayment history.
//   - EXCELLENT (750+): Premium loans — KES 50,000 max, 30 days, 5% rate
//     Top-tier crew member. Best rates, highest limits.
//
// Interest rates are competitive with Kenya's digital lenders:
//   - M-Shwari: ~7.5% per month
//   - Tala: ~15% for 30-day loans
//   - Branch: ~14% for 30-day loans
//   - CrewPay: 5-15% graduated by trust level
func DefaultTiers() []LoanTier {
	return []LoanTier{
		{
			Grade:         "EXCELLENT",
			MinScore:      750,
			MaxLoanCents:  5_000_000, // KES 50,000
			InterestRate:  0.05,
			MaxTenureDays: 30,
			CooldownDays:  0,
			Description:   "Premium — KES 50,000 max at 5%",
		},
		{
			Grade:         "GOOD",
			MinScore:      650,
			MaxLoanCents:  2_000_000, // KES 20,000
			InterestRate:  0.08,
			MaxTenureDays: 30,
			CooldownDays:  3,
			Description:   "Standard — KES 20,000 max at 8%",
		},
		{
			Grade:         "FAIR",
			MinScore:      500,
			MaxLoanCents:  500_000, // KES 5,000
			InterestRate:  0.12,
			MaxTenureDays: 14,
			CooldownDays:  7,
			Description:   "Growth — KES 5,000 max at 12%",
		},
		{
			Grade:         "POOR",
			MinScore:      400,
			MaxLoanCents:  100_000, // KES 1,000
			InterestRate:  0.15,
			MaxTenureDays: 7,
			CooldownDays:  14,
			Description:   "Starter — KES 1,000 max at 15%",
		},
	}
}

// GetTierForScore returns the best tier a crew member qualifies for.
// Returns nil if the score is below the minimum threshold (400).
func GetTierForScore(score int) *LoanTier {
	tiers := DefaultTiers()
	for i := range tiers {
		if score >= tiers[i].MinScore {
			return &tiers[i]
		}
	}
	return nil // Below 400 — not eligible
}

// FormatMaxLoanKES returns the max loan in KES (human-readable).
func (t *LoanTier) FormatMaxLoanKES() float64 {
	return float64(t.MaxLoanCents) / 100
}

// FormatInterestPercent returns the interest rate as a percentage string.
func (t *LoanTier) FormatInterestPercent() float64 {
	return t.InterestRate * 100
}
