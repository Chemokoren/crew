package credit

import (
	"context"
	"fmt"
	"math"
)

// DefaultWeights defines the default credit scoring factor weights.
// These can be overridden per tenant via TenantConfig.CreditScoringWeights.
var DefaultWeights = map[string]float64{
	"WORK_HISTORY":    0.25,
	"INCOME":          0.20,
	"PAYMENT_HISTORY": 0.30,
	"ACCOUNT_HEALTH":  0.15,
	"PLATFORM_TENURE": 0.10,
}

// IndustryDefaultWeights provides recommended weight profiles per industry.
// Tenants can further customize these via TenantConfig.
var IndustryDefaultWeights = map[string]map[string]float64{
	"TRANSPORT": {
		"WORK_HISTORY":    0.30, // Daily revenue regularity is paramount
		"INCOME":          0.20,
		"PAYMENT_HISTORY": 0.25,
		"ACCOUNT_HEALTH":  0.15,
		"PLATFORM_TENURE": 0.10,
	},
	"CONSTRUCTION": {
		"WORK_HISTORY":    0.25, // Weekly hours consistency matters
		"INCOME":          0.20,
		"PAYMENT_HISTORY": 0.25,
		"ACCOUNT_HEALTH":  0.15,
		"PLATFORM_TENURE": 0.15, // Tenure with multiple sites is valued
	},
	"HEALTH": {
		"WORK_HISTORY":    0.25, // Monthly visit completion
		"INCOME":          0.15,
		"PAYMENT_HISTORY": 0.30,
		"ACCOUNT_HEALTH":  0.15,
		"PLATFORM_TENURE": 0.15, // Long-term commitment valued
	},
	"LOGISTICS": {
		"WORK_HISTORY":    0.30,
		"INCOME":          0.20,
		"PAYMENT_HISTORY": 0.25,
		"ACCOUNT_HEALTH":  0.15,
		"PLATFORM_TENURE": 0.10,
	},
	"AGRICULTURE": {
		"WORK_HISTORY":    0.20,
		"INCOME":          0.25, // Seasonal income matters most
		"PAYMENT_HISTORY": 0.25,
		"ACCOUNT_HEALTH":  0.20, // Savings discipline crucial for seasonal workers
		"PLATFORM_TENURE": 0.10,
	},
}

// WeightedScorer wraps RulesScorer with configurable category weights.
// Implements E2 (industry-weighted scoring) by applying per-tenant weight overrides.
type WeightedScorer struct {
	base    *RulesScorer
	weights map[string]float64
}

// NewWeightedScorer creates a scorer with custom weights.
// Pass nil weights to use DefaultWeights.
func NewWeightedScorer(weights map[string]float64) *WeightedScorer {
	w := DefaultWeights
	if weights != nil {
		w = weights
	}
	return &WeightedScorer{base: NewRulesScorer(), weights: w}
}

// NewWeightedScorerForIndustry creates a scorer tuned for a specific industry.
// Tenant-level overrides take precedence over industry defaults.
func NewWeightedScorerForIndustry(industry string, tenantOverrides map[string]float64) *WeightedScorer {
	weights := DefaultWeights
	if iw, ok := IndustryDefaultWeights[industry]; ok {
		weights = iw
	}
	// Tenant overrides take precedence
	if tenantOverrides != nil {
		for k, v := range tenantOverrides {
			weights[k] = v
		}
	}
	return &WeightedScorer{base: NewRulesScorer(), weights: weights}
}

func (s *WeightedScorer) Version() string { return "weighted-v1.0" }

func (s *WeightedScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	if fv == nil {
		return nil, fmt.Errorf("nil feature vector")
	}

	// Compute raw category scores using the base rules scorer
	workFactors, workRaw := s.base.scoreWorkHistory(fv)

	// Add cross-industry bonus factors
	crossIndustryFactors := s.scoreIndustryFactors(fv)
	workFactors = append(workFactors, crossIndustryFactors...)
	var crossPts int
	for _, f := range crossIndustryFactors {
		crossPts += f.Points
	}

	incomeFactors, incomeRaw := s.base.scoreIncome(fv)
	paymentFactors, paymentRaw := s.base.scorePaymentHistory(fv)
	healthFactors, healthRaw := s.base.scoreAccountHealth(fv)
	tenureFactors, tenureRaw := s.base.scoreTenure(fv)

	// Max points per category (from base scorer)
	maxWork := 138
	maxIncome := 110
	maxPayment := 165
	maxHealth := 82
	maxTenure := 55

	// Normalize each category to [0, 1] then apply weights
	normWork := float64(workRaw+crossPts) / float64(maxWork+30) // Extended max for cross-industry
	normIncome := float64(incomeRaw) / float64(maxIncome)
	normPayment := float64(paymentRaw) / float64(maxPayment)
	normHealth := float64(healthRaw) / float64(maxHealth)
	normTenure := float64(tenureRaw) / float64(maxTenure)

	// Cap normalizations at 1.0
	normWork = math.Min(normWork, 1.0)
	normIncome = math.Min(normIncome, 1.0)
	normPayment = math.Min(normPayment, 1.0)
	normHealth = math.Min(normHealth, 1.0)
	normTenure = math.Min(normTenure, 1.0)

	// Weighted sum → 550 variable points
	variableMax := 550.0
	weighted := (normWork*s.weight("WORK_HISTORY") +
		normIncome*s.weight("INCOME") +
		normPayment*s.weight("PAYMENT_HISTORY") +
		normHealth*s.weight("ACCOUNT_HEALTH") +
		normTenure*s.weight("PLATFORM_TENURE")) * variableMax

	totalPoints := 300 + int(weighted) // Base 300

	// CRB bonus (unchanged — additive)
	var crbFactors []ScoreFactor
	if fv.CRBScoreAvailable {
		cf, crbPts := s.base.scoreCRB(fv)
		crbFactors = cf
		totalPoints += crbPts
	}

	// Clamp
	if totalPoints > 850 {
		totalPoints = 850
	}
	if totalPoints < 300 {
		totalPoints = 300
	}

	var allFactors []ScoreFactor
	allFactors = append(allFactors, workFactors...)
	allFactors = append(allFactors, incomeFactors...)
	allFactors = append(allFactors, paymentFactors...)
	allFactors = append(allFactors, healthFactors...)
	allFactors = append(allFactors, tenureFactors...)
	allFactors = append(allFactors, crbFactors...)

	suggestions := s.base.generateSuggestions(fv, totalPoints)

	return &ScoreResult{
		Score:        totalPoints,
		Grade:        ScoreGrade(totalPoints),
		Factors:      allFactors,
		Suggestions:  suggestions,
		ModelVersion: s.Version(),
		ComputedAt:   fv.ComputedAt,
		Features:     fv,
	}, nil
}

// scoreIndustryFactors computes bonus points for cross-industry credit signals.
func (s *WeightedScorer) scoreIndustryFactors(fv *FeatureVector) []ScoreFactor {
	var factors []ScoreFactor

	// Cross-org tenure bonus (max 15 pts) — E3
	orgPts := minI(fv.OrgCount*5, 15)
	factors = append(factors, ScoreFactor{
		Category:    "CROSS_INDUSTRY",
		Name:        "Multi-Organization Experience",
		Points:      orgPts,
		MaxPoints:   15,
		Percentage:  pct(orgPts, 15),
		Description: fmt.Sprintf("Worked with %d organizations", fv.OrgCount),
		Impact:      impact(orgPts, 15),
	})

	// Hours consistency bonus (max 15 pts)
	hoursPts := int(fv.HoursConsistency30d * 15)
	factors = append(factors, ScoreFactor{
		Category:    "CROSS_INDUSTRY",
		Name:        "Hours Consistency",
		Points:      hoursPts,
		MaxPoints:   15,
		Percentage:  pct(hoursPts, 15),
		Description: fmt.Sprintf("Consistency score: %.0f%%", fv.HoursConsistency30d*100),
		Impact:      impact(hoursPts, 15),
	})

	return factors
}

func (s *WeightedScorer) weight(category string) float64 {
	if w, ok := s.weights[category]; ok {
		return w
	}
	return DefaultWeights[category]
}
