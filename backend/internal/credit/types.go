// Package credit provides a pluggable, explainable credit scoring engine.
//
// Architecture:
//
//	┌─────────────┐    ┌──────────────┐    ┌──────────────┐
//	│ Feature      │ →  │ Scorer       │ →  │ Explanation  │
//	│ Engineering  │    │ (pluggable)  │    │ Engine       │
//	└─────────────┘    └──────────────┘    └──────────────┘
//
// The Scorer interface allows swapping between:
//   - RulesScorer   (V2 — deterministic, multi-factor)
//   - MLScorer      (V3 — calls external ML service)
//   - HybridScorer  (ensemble of rules + ML)
package credit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// --- Feature Vector ---

// FeatureVector holds all computed signals for a crew member.
// This is the "Feature Store" — the single source of truth for scoring.
type FeatureVector struct {
	CrewMemberID uuid.UUID `json:"crew_member_id"`
	ComputedAt   time.Time `json:"computed_at"`

	// --- Work History ---
	CompletedShifts30d  int     `json:"completed_shifts_30d"`
	CompletedShifts90d  int     `json:"completed_shifts_90d"`
	CancelledShifts30d  int     `json:"cancelled_shifts_30d"`
	CancellationRate    float64 `json:"cancellation_rate"`    // 0.0 – 1.0
	ActiveDaysRatio     float64 `json:"active_days_ratio"`    // days_worked / 30
	ShiftConsistency    float64 `json:"shift_consistency"`    // this_month / last_month (capped 1.0)

	// --- Income ---
	TotalEarnings30dKES     float64 `json:"total_earnings_30d_kes"`
	TotalEarnings90dKES     float64 `json:"total_earnings_90d_kes"`
	AvgDailyEarningsKES     float64 `json:"avg_daily_earnings_kes"`
	IncomeTrend             string  `json:"income_trend"` // "GROWING", "STABLE", "DECLINING"
	IncomeTrendRatio        float64 `json:"income_trend_ratio"`
	EarningTypeDiversity    int     `json:"earning_type_diversity"` // number of distinct types
	WithdrawalToEarningRate float64 `json:"withdrawal_to_earning_rate"`

	// --- Payment History ---
	TotalLoansCompleted  int     `json:"total_loans_completed"`
	TotalLoansDefaulted  int     `json:"total_loans_defaulted"`
	OnTimeRepaymentRate  float64 `json:"on_time_repayment_rate"` // 0.0 – 1.0
	HasActiveLoan        bool    `json:"has_active_loan"`
	ActiveInsurancePolicies int  `json:"active_insurance_policies"`
	HasPINSet            bool    `json:"has_pin_set"`

	// --- Account Health ---
	CurrentBalanceKES    float64 `json:"current_balance_kes"`
	AvgBalance30dKES     float64 `json:"avg_balance_30d_kes"`
	BalanceTrend         string  `json:"balance_trend"` // "GROWING", "STABLE", "DECLINING"
	KYCStatus            string  `json:"kyc_status"`    // "VERIFIED", "PENDING", "NONE"
	IsActive             bool    `json:"is_active"`

	// --- Platform Tenure ---
	AccountAgeDays       int     `json:"account_age_days"`
	FirstShiftAgeDays    int     `json:"first_shift_age_days"`
	DaysSinceLastShift   int     `json:"days_since_last_shift"`

	// --- External Credit (CRB) ---
	CRBScoreAvailable    bool    `json:"crb_score_available"`
	CRBScore             int     `json:"crb_score"`
	CRBTotalLoans        int     `json:"crb_total_loans"`
	CRBDefaultedLoans    int     `json:"crb_defaulted_loans"`
	CRBExposureKES       float64 `json:"crb_exposure_kes"`

	// --- Negative Signals ---
	UnresolvedNegativeEvents int `json:"unresolved_negative_events"`
	FraudFlags               int `json:"fraud_flags"`
	Disputes                 int `json:"disputes"`
	AccountLocks             int `json:"account_locks"`
}

// --- Scorer Interface ---

// ScoreResult holds the computed score and its breakdown.
type ScoreResult struct {
	Score        int              `json:"score"`         // 300–850
	Grade        string           `json:"grade"`         // "EXCELLENT", "GOOD", "FAIR", "POOR", "VERY_POOR"
	Factors      []ScoreFactor    `json:"factors"`       // Individual factor contributions
	Suggestions  []string         `json:"suggestions"`   // Actionable tips to improve
	ModelVersion string           `json:"model_version"` // e.g., "rules-v2.1", "ml-v3.0"
	ComputedAt   time.Time        `json:"computed_at"`
	Features     *FeatureVector   `json:"features,omitempty"` // Raw features (for debugging/explainability)
}

// ScoreFactor represents one component of the score with its contribution.
type ScoreFactor struct {
	Category    string  `json:"category"`    // "WORK_HISTORY", "INCOME", etc.
	Name        string  `json:"name"`        // Human-readable name
	Points      int     `json:"points"`      // Points contributed
	MaxPoints   int     `json:"max_points"`  // Maximum possible for this factor
	Percentage  float64 `json:"percentage"`  // Points / MaxPoints
	Description string  `json:"description"` // Explanation
	Impact      string  `json:"impact"`      // "POSITIVE", "NEUTRAL", "NEGATIVE"
}

// Scorer computes a credit score from a feature vector.
// Implement this interface to add new scoring models.
type Scorer interface {
	// Score computes a credit score from features.
	Score(ctx context.Context, features *FeatureVector) (*ScoreResult, error)

	// Version returns the model version identifier.
	Version() string
}

// ScoreGrade converts a numeric score to a letter grade.
func ScoreGrade(score int) string {
	switch {
	case score >= 750:
		return "EXCELLENT"
	case score >= 650:
		return "GOOD"
	case score >= 500:
		return "FAIR"
	case score >= 400:
		return "POOR"
	default:
		return "VERY_POOR"
	}
}
