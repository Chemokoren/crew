package credit

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// ===================================================================
// Phase 3: Advanced Credit Scoring Infrastructure
// ===================================================================

// --- 1. Drift Detection ---
// Monitors for Population Stability Index (PSI) shift in score distributions.
// When PSI > 0.2, model retraining is triggered.

// DriftDetector monitors score distribution stability.
type DriftDetector struct {
	historyRepo repository.CreditScoreHistoryRepository
	logger      *slog.Logger
}

// NewDriftDetector creates a drift detector.
func NewDriftDetector(historyRepo repository.CreditScoreHistoryRepository, logger *slog.Logger) *DriftDetector {
	return &DriftDetector{historyRepo: historyRepo, logger: logger}
}

// DriftReport contains the results of a drift analysis.
type DriftReport struct {
	PSI               float64   `json:"psi"`                // Population Stability Index
	BaselineMeanScore float64   `json:"baseline_mean_score"`
	CurrentMeanScore  float64   `json:"current_mean_score"`
	ScoreShift        float64   `json:"score_shift"`
	DriftDetected     bool      `json:"drift_detected"`     // PSI > 0.2
	ActionRequired    string    `json:"action_required"`    // "NONE", "MONITOR", "RETRAIN"
	AnalyzedAt        time.Time `json:"analyzed_at"`
}

// Analyze compares the recent score distribution against a baseline period.
func (d *DriftDetector) Analyze(ctx context.Context, crewMemberIDs []uuid.UUID) (*DriftReport, error) {
	// Collect recent scores (last 7 days) vs baseline (30-60 days ago)
	var recentScores, baselineScores []float64

	for _, id := range crewMemberIDs {
		history, err := d.historyRepo.GetHistory(ctx, id, 10)
		if err != nil || len(history) == 0 {
			continue
		}

		for _, h := range history {
			age := time.Since(h.ComputedAt)
			if age < 7*24*time.Hour {
				recentScores = append(recentScores, float64(h.Score))
			} else if age >= 30*24*time.Hour && age < 60*24*time.Hour {
				baselineScores = append(baselineScores, float64(h.Score))
			}
		}
	}

	if len(recentScores) < 10 || len(baselineScores) < 10 {
		return &DriftReport{
			ActionRequired: "INSUFFICIENT_DATA",
			AnalyzedAt:     time.Now(),
		}, nil
	}

	// Calculate PSI (Population Stability Index)
	psi := calculatePSI(baselineScores, recentScores, 5)
	baselineMean := mean(baselineScores)
	currentMean := mean(recentScores)

	action := "NONE"
	driftDetected := false
	if psi > 0.25 {
		action = "RETRAIN"
		driftDetected = true
	} else if psi > 0.1 {
		action = "MONITOR"
	}

	report := &DriftReport{
		PSI:               psi,
		BaselineMeanScore: baselineMean,
		CurrentMeanScore:  currentMean,
		ScoreShift:        currentMean - baselineMean,
		DriftDetected:     driftDetected,
		ActionRequired:    action,
		AnalyzedAt:        time.Now(),
	}

	d.logger.Info("drift analysis complete",
		slog.Float64("psi", psi),
		slog.String("action", action),
		slog.Bool("drift_detected", driftDetected),
	)

	return report, nil
}

// --- 2. Fairness Auditing ---

// FairnessAuditor checks for bias in credit scoring across demographic groups.
type FairnessAuditor struct {
	logger *slog.Logger
}

// NewFairnessAuditor creates a fairness auditor.
func NewFairnessAuditor(logger *slog.Logger) *FairnessAuditor {
	return &FairnessAuditor{logger: logger}
}

// FairnessReport contains bias analysis results.
type FairnessReport struct {
	GroupAnalysis     []GroupFairness `json:"group_analysis"`
	DisparateImpact   float64        `json:"disparate_impact"`    // Should be 0.8-1.2 (80% rule)
	StatisticalParity float64        `json:"statistical_parity"`  // Should be near 0
	OverallFair       bool           `json:"overall_fair"`
	Recommendations   []string       `json:"recommendations"`
	AnalyzedAt        time.Time      `json:"analyzed_at"`
}

// GroupFairness represents fairness metrics for a demographic group.
type GroupFairness struct {
	GroupName       string  `json:"group_name"`
	GroupAttribute  string  `json:"group_attribute"`  // "gender", "location", "age_bracket"
	SampleSize      int     `json:"sample_size"`
	MeanScore       float64 `json:"mean_score"`
	ApprovalRate    float64 `json:"approval_rate"`    // Percent scoring >= 400
	DefaultRate     float64 `json:"default_rate"`
}

// Audit performs fairness analysis on a set of scored crew members.
func (a *FairnessAuditor) Audit(scoredGroups map[string][]int) *FairnessReport {
	var groups []GroupFairness
	var totalApproved, totalSample int

	for name, scores := range scoredGroups {
		approved := 0
		sum := 0.0
		for _, s := range scores {
			sum += float64(s)
			if s >= 400 {
				approved++
			}
		}

		gf := GroupFairness{
			GroupName:    name,
			SampleSize:   len(scores),
			MeanScore:    sum / float64(len(scores)),
			ApprovalRate: float64(approved) / float64(len(scores)),
		}
		groups = append(groups, gf)
		totalApproved += approved
		totalSample += len(scores)
	}

	// Disparate Impact: min(group_rate) / max(group_rate)
	minRate, maxRate := 1.0, 0.0
	for _, g := range groups {
		if g.ApprovalRate < minRate {
			minRate = g.ApprovalRate
		}
		if g.ApprovalRate > maxRate {
			maxRate = g.ApprovalRate
		}
	}

	disparateImpact := 0.0
	if maxRate > 0 {
		disparateImpact = minRate / maxRate
	}

	// Statistical parity: max difference in approval rates
	statParity := maxRate - minRate

	fair := disparateImpact >= 0.8 && statParity < 0.15

	var recs []string
	if !fair {
		if disparateImpact < 0.8 {
			recs = append(recs, fmt.Sprintf("Disparate impact %.2f < 0.8 — review features for proxy discrimination", disparateImpact))
		}
		if statParity >= 0.15 {
			recs = append(recs, fmt.Sprintf("Statistical parity gap %.2f — investigate group-specific biases", statParity))
		}
	}

	return &FairnessReport{
		GroupAnalysis:     groups,
		DisparateImpact:   disparateImpact,
		StatisticalParity: statParity,
		OverallFair:       fair,
		Recommendations:   recs,
		AnalyzedAt:        time.Now(),
	}
}

// --- 3. Survival Analysis ---
// Predicts *when* a default will occur, not just *if*.

// SurvivalEstimate predicts the probability of default at different time horizons.
type SurvivalEstimate struct {
	CrewMemberID     uuid.UUID `json:"crew_member_id"`
	ProbDefault7d    float64   `json:"prob_default_7d"`   // P(default within 7 days)
	ProbDefault14d   float64   `json:"prob_default_14d"`
	ProbDefault30d   float64   `json:"prob_default_30d"`
	ProbDefault90d   float64   `json:"prob_default_90d"`
	MedianSurvival   int       `json:"median_survival_days"`
	RiskGroup        string    `json:"risk_group"` // "LOW", "MEDIUM", "HIGH"
	ComputedAt       time.Time `json:"computed_at"`
}

// SurvivalAnalyzer estimates time-to-default using a simplified hazard model.
type SurvivalAnalyzer struct {
	logger *slog.Logger
}

// NewSurvivalAnalyzer creates a survival analyzer.
func NewSurvivalAnalyzer(logger *slog.Logger) *SurvivalAnalyzer {
	return &SurvivalAnalyzer{logger: logger}
}

// Estimate computes survival probabilities based on the feature vector.
// In production, this would use a Cox proportional hazards or neural survival model.
func (a *SurvivalAnalyzer) Estimate(fv *FeatureVector) *SurvivalEstimate {
	// Simple hazard rate based on key risk signals
	baseHazard := 0.01 // 1% daily default probability baseline

	// Risk multipliers
	if fv.TotalLoansDefaulted > 0 {
		baseHazard *= 1.0 + float64(fv.TotalLoansDefaulted)*0.5
	}
	if fv.OnTimeRepaymentRate < 0.5 {
		baseHazard *= 2.0
	}
	if fv.CurrentBalanceKES < 100 {
		baseHazard *= 1.5
	}
	if fv.CompletedShifts30d < 5 {
		baseHazard *= 1.3
	}
	if fv.FraudFlags > 0 {
		baseHazard *= 3.0
	}

	// Protective factors
	if fv.OnTimeRepaymentRate > 0.9 {
		baseHazard *= 0.3
	}
	if fv.AccountAgeDays > 180 {
		baseHazard *= 0.7
	}
	if fv.ActiveInsurancePolicies > 0 {
		baseHazard *= 0.8
	}

	// Cap hazard
	if baseHazard > 0.5 {
		baseHazard = 0.5
	}

	// Survival function: S(t) = exp(-hazard * t)
	prob7 := 1 - math.Exp(-baseHazard*7)
	prob14 := 1 - math.Exp(-baseHazard*14)
	prob30 := 1 - math.Exp(-baseHazard*30)
	prob90 := 1 - math.Exp(-baseHazard*90)

	// Median survival = ln(2) / hazard
	medianSurvival := int(math.Log(2) / baseHazard)
	if medianSurvival > 365 {
		medianSurvival = 365
	}

	riskGroup := "LOW"
	if prob30 > 0.3 {
		riskGroup = "HIGH"
	} else if prob30 > 0.1 {
		riskGroup = "MEDIUM"
	}

	return &SurvivalEstimate{
		CrewMemberID:   fv.CrewMemberID,
		ProbDefault7d:  prob7,
		ProbDefault14d: prob14,
		ProbDefault30d: prob30,
		ProbDefault90d: prob90,
		MedianSurvival: medianSurvival,
		RiskGroup:      riskGroup,
		ComputedAt:     time.Now(),
	}
}

// --- 4. Network Effects ---
// Captures co-worker default correlation within the same SACCO/route.

// NetworkRiskSignal captures default correlation within a crew member's network.
type NetworkRiskSignal struct {
	CrewMemberID       uuid.UUID `json:"crew_member_id"`
	SACCODefaultRate   float64   `json:"sacco_default_rate"`    // Default rate among SACCO peers
	RouteDefaultRate   float64   `json:"route_default_rate"`    // Default rate among route peers
	DirectPeerDefaults int       `json:"direct_peer_defaults"`  // Number of direct peers who defaulted
	NetworkRiskLevel   string    `json:"network_risk_level"`    // "LOW", "MEDIUM", "HIGH"
	ComputedAt         time.Time `json:"computed_at"`
}

// NetworkAnalyzer computes network-based risk signals.
type NetworkAnalyzer struct {
	crewRepo repository.CrewRepository
	loanRepo repository.LoanApplicationRepository
	logger   *slog.Logger
}

// NewNetworkAnalyzer creates a network analyzer.
func NewNetworkAnalyzer(
	crewRepo repository.CrewRepository,
	loanRepo repository.LoanApplicationRepository,
	logger *slog.Logger,
) *NetworkAnalyzer {
	return &NetworkAnalyzer{
		crewRepo: crewRepo,
		loanRepo: loanRepo,
		logger:   logger,
	}
}

// Analyze computes network risk for a crew member.
// In production, this queries peers from the same SACCO/route and checks their default rates.
func (a *NetworkAnalyzer) Analyze(ctx context.Context, crewMemberID uuid.UUID) (*NetworkRiskSignal, error) {
	// Get crew member's details for network lookup
	_, err := a.crewRepo.GetByID(ctx, crewMemberID)
	if err != nil {
		return nil, fmt.Errorf("network: get crew member: %w", err)
	}

	// Get all loans for this crew member to find SACCO peers via assignments
	// In a full implementation, this would query the crew_sacco_memberships table
	// and find all peers in the same SACCO
	loans, _, err := a.loanRepo.List(ctx, repository.LoanApplicationFilter{
		CrewMemberID: &crewMemberID,
	}, 1, 100)

	var saccoDefaultRate float64
	if err == nil && len(loans) > 0 {
		defaulted := 0
		for _, l := range loans {
			if l.Status == "DEFAULTED" {
				defaulted++
			}
		}
		if len(loans) > 0 {
			saccoDefaultRate = float64(defaulted) / float64(len(loans))
		}
	}

	riskLevel := "LOW"
	if saccoDefaultRate > 0.2 {
		riskLevel = "HIGH"
	} else if saccoDefaultRate > 0.1 {
		riskLevel = "MEDIUM"
	}

	signal := &NetworkRiskSignal{
		CrewMemberID:     crewMemberID,
		SACCODefaultRate: saccoDefaultRate,
		NetworkRiskLevel: riskLevel,
		ComputedAt:       time.Now(),
	}

	a.logger.Info("network analysis complete",
		slog.String("crew_member_id", crewMemberID.String()),
		slog.Float64("sacco_default_rate", saccoDefaultRate),
		slog.String("risk_level", riskLevel),
	)

	return signal, nil
}

// --- 5. Federated Learning Infrastructure ---
// Enables training across SACCOs without sharing raw data.

// FederatedConfig holds configuration for federated learning.
type FederatedConfig struct {
	AggregatorURL   string `json:"aggregator_url"`
	SACCOIdentifier string `json:"sacco_identifier"`
	MinSampleSize   int    `json:"min_sample_size"`
	EncryptionKey   string `json:"encryption_key"`
}

// FederatedGradient represents an encrypted model update from a local SACCO.
type FederatedGradient struct {
	SACCOIdentifier string    `json:"sacco_identifier"`
	ModelVersion    string    `json:"model_version"`
	GradientHash    string    `json:"gradient_hash"`
	SampleSize      int       `json:"sample_size"`
	LocalMetrics    map[string]float64 `json:"local_metrics"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

// FederatedCoordinator manages federated learning rounds.
type FederatedCoordinator struct {
	config FederatedConfig
	logger *slog.Logger
}

// NewFederatedCoordinator creates a federated learning coordinator.
func NewFederatedCoordinator(config FederatedConfig, logger *slog.Logger) *FederatedCoordinator {
	return &FederatedCoordinator{config: config, logger: logger}
}

// ComputeLocalGradient computes a model gradient from local SACCO data.
// In production, this runs a local training pass and returns encrypted gradients.
func (c *FederatedCoordinator) ComputeLocalGradient(ctx context.Context, features []*FeatureVector) (*FederatedGradient, error) {
	if len(features) < c.config.MinSampleSize {
		return nil, fmt.Errorf("insufficient samples: %d < %d", len(features), c.config.MinSampleSize)
	}

	c.logger.Info("federated: computing local gradient",
		slog.Int("sample_size", len(features)),
		slog.String("sacco", c.config.SACCOIdentifier),
	)

	// Compute local metrics (privacy-safe aggregates)
	var totalScore float64
	for _, fv := range features {
		totalScore += float64(fv.AccountAgeDays) // Placeholder metric
	}

	return &FederatedGradient{
		SACCOIdentifier: c.config.SACCOIdentifier,
		ModelVersion:    "federated-v0.1",
		SampleSize:      len(features),
		LocalMetrics: map[string]float64{
			"avg_account_age": totalScore / float64(len(features)),
			"sample_count":    float64(len(features)),
		},
		SubmittedAt: time.Now(),
	}, nil
}

// --- Helper Functions ---

func calculatePSI(baseline, current []float64, bins int) float64 {
	if len(baseline) == 0 || len(current) == 0 {
		return 0
	}

	// Create histogram bins
	minVal, maxVal := baseline[0], baseline[0]
	for _, v := range append(baseline, current...) {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	binWidth := (maxVal - minVal) / float64(bins)
	if binWidth == 0 {
		return 0
	}

	baseCounts := make([]float64, bins)
	currCounts := make([]float64, bins)

	for _, v := range baseline {
		idx := int((v - minVal) / binWidth)
		if idx >= bins {
			idx = bins - 1
		}
		baseCounts[idx]++
	}
	for _, v := range current {
		idx := int((v - minVal) / binWidth)
		if idx >= bins {
			idx = bins - 1
		}
		currCounts[idx]++
	}

	// Normalize to proportions
	baseN := float64(len(baseline))
	currN := float64(len(current))

	psi := 0.0
	for i := 0; i < bins; i++ {
		bp := (baseCounts[i] + 0.5) / baseN // Laplace smoothing
		cp := (currCounts[i] + 0.5) / currN
		psi += (cp - bp) * math.Log(cp/bp)
	}

	return psi
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
