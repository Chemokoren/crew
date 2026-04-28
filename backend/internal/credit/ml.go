package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

// --- ML Scorer Infrastructure ---

// MLConfig holds configuration for the ML scoring service.
type MLConfig struct {
	Endpoint     string        `json:"endpoint"`      // e.g., "http://ml-service:8080/predict"
	Timeout      time.Duration `json:"timeout"`
	ModelName    string        `json:"model_name"`
	ModelVersion string        `json:"model_version"`
}

// MLScorerV3 calls an external ML model for credit scoring.
// Designed for future deployment with TensorFlow Serving, ONNX Runtime, or custom Python service.
type MLScorerV3 struct {
	config MLConfig
	logger *slog.Logger
}

// NewMLScorerV3 creates an ML-based scorer.
func NewMLScorerV3(config MLConfig, logger *slog.Logger) *MLScorerV3 {
	return &MLScorerV3{config: config, logger: logger}
}

func (s *MLScorerV3) Version() string {
	return fmt.Sprintf("ml-%s-%s", s.config.ModelName, s.config.ModelVersion)
}

func (s *MLScorerV3) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	// TODO: When ML model is trained, this will HTTP POST features to the serving endpoint.
	// For now, return a stub that mirrors the rules scorer range.
	s.logger.Warn("ML scorer invoked but no model deployed — returning stub score")

	return &ScoreResult{
		Score:        500,
		Grade:        ScoreGrade(500),
		ModelVersion: s.Version(),
		ComputedAt:   time.Now(),
		Features:     fv,
		Suggestions:  []string{"ML model not yet deployed — using stub"},
	}, nil
}

// --- Hybrid Scorer (Ensemble) ---

// HybridScorerConfig defines the ensemble weights.
type HybridScorerConfig struct {
	RulesWeight float64 `json:"rules_weight"` // 0.0–1.0, default 0.7
	MLWeight    float64 `json:"ml_weight"`    // 0.0–1.0, default 0.3
}

// HybridScorer combines RulesScorer and MLScorer via weighted ensemble.
// Used during A/B testing and gradual ML model rollout.
type HybridScorer struct {
	rules  *RulesScorer
	ml     *MLScorerV3
	config HybridScorerConfig
	logger *slog.Logger
}

// NewHybridScorer creates an ensemble scorer.
func NewHybridScorer(rules *RulesScorer, ml *MLScorerV3, config HybridScorerConfig, logger *slog.Logger) *HybridScorer {
	if config.RulesWeight+config.MLWeight == 0 {
		config.RulesWeight = 0.7
		config.MLWeight = 0.3
	}
	return &HybridScorer{rules: rules, ml: ml, config: config, logger: logger}
}

func (s *HybridScorer) Version() string {
	return fmt.Sprintf("hybrid-v3.0(rules=%.0f%%,ml=%.0f%%)",
		s.config.RulesWeight*100, s.config.MLWeight*100)
}

func (s *HybridScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	rulesResult, err := s.rules.Score(ctx, fv)
	if err != nil {
		return nil, fmt.Errorf("hybrid: rules scorer failed: %w", err)
	}

	mlResult, err := s.ml.Score(ctx, fv)
	if err != nil {
		s.logger.Warn("hybrid: ML scorer failed, falling back to rules-only",
			slog.String("error", err.Error()),
		)
		return rulesResult, nil
	}

	// Weighted ensemble
	ensembleScore := int(
		s.config.RulesWeight*float64(rulesResult.Score) +
			s.config.MLWeight*float64(mlResult.Score),
	)

	// Clamp
	if ensembleScore > 850 {
		ensembleScore = 850
	}
	if ensembleScore < 300 {
		ensembleScore = 300
	}

	// Merge factors from both scorers
	allFactors := append(rulesResult.Factors, mlResult.Factors...)

	return &ScoreResult{
		Score:        ensembleScore,
		Grade:        ScoreGrade(ensembleScore),
		Factors:      allFactors,
		Suggestions:  rulesResult.Suggestions,
		ModelVersion: s.Version(),
		ComputedAt:   time.Now(),
		Features:     fv,
	}, nil
}

// --- A/B Testing Framework ---

// ABTestConfig defines an A/B test between two scorers.
type ABTestConfig struct {
	Name           string  `json:"name"`
	ControlScorer  string  `json:"control_scorer"`  // e.g., "rules-v2.1"
	TreatmentScorer string `json:"treatment_scorer"` // e.g., "hybrid-v3.0"
	TrafficPercent float64 `json:"traffic_percent"`  // 0.0–1.0 percent routed to treatment
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Active         bool    `json:"active"`
}

// ABTestScorer wraps two scorers and routes traffic based on config.
type ABTestScorer struct {
	control   Scorer
	treatment Scorer
	config    ABTestConfig
	logger    *slog.Logger
}

// NewABTestScorer creates an A/B testing scorer.
func NewABTestScorer(control, treatment Scorer, config ABTestConfig, logger *slog.Logger) *ABTestScorer {
	return &ABTestScorer{
		control:   control,
		treatment: treatment,
		config:    config,
		logger:    logger,
	}
}

func (s *ABTestScorer) Version() string {
	return fmt.Sprintf("ab-test(%s)", s.config.Name)
}

func (s *ABTestScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	now := time.Now()
	if !s.config.Active || now.Before(s.config.StartDate) || now.After(s.config.EndDate) {
		return s.control.Score(ctx, fv)
	}

	// Route based on traffic split (deterministic by crew member ID for consistency)
	hash := float64(fv.CrewMemberID[0]) / 256.0
	if hash < s.config.TrafficPercent {
		s.logger.Info("A/B test: routing to treatment",
			slog.String("test", s.config.Name),
			slog.String("crew_member_id", fv.CrewMemberID.String()),
		)
		result, err := s.treatment.Score(ctx, fv)
		if err != nil {
			s.logger.Warn("A/B test: treatment failed, falling back to control")
			return s.control.Score(ctx, fv)
		}
		return result, nil
	}

	return s.control.Score(ctx, fv)
}

// --- Feature Importance Analysis ---

// FeatureImportance represents the importance of a single feature for predictions.
type FeatureImportance struct {
	FeatureName  string  `json:"feature_name"`
	Importance   float64 `json:"importance"`   // 0.0–1.0 normalized
	Direction    string  `json:"direction"`    // "POSITIVE", "NEGATIVE"
	Description  string  `json:"description"`
}

// ComputeFeatureImportance performs a simple permutation-based feature importance analysis.
// In production, this would be replaced by SHAP values from the ML model.
func ComputeFeatureImportance(scorer Scorer, baseline *FeatureVector) ([]FeatureImportance, error) {
	ctx := context.Background()
	baseResult, err := scorer.Score(ctx, baseline)
	if err != nil {
		return nil, err
	}

	// Define feature perturbations
	perturbations := []struct {
		name    string
		perturb func(fv *FeatureVector)
		restore func(fv *FeatureVector, orig interface{})
	}{
		{"completed_shifts_30d", func(fv *FeatureVector) { fv.CompletedShifts30d = 0 }, nil},
		{"total_earnings_30d", func(fv *FeatureVector) { fv.TotalEarnings30dKES = 0 }, nil},
		{"on_time_repayment_rate", func(fv *FeatureVector) { fv.OnTimeRepaymentRate = 0 }, nil},
		{"current_balance", func(fv *FeatureVector) { fv.CurrentBalanceKES = 0 }, nil},
		{"account_age_days", func(fv *FeatureVector) { fv.AccountAgeDays = 0 }, nil},
		{"cancellation_rate", func(fv *FeatureVector) { fv.CancellationRate = 1.0 }, nil},
		{"total_loans_defaulted", func(fv *FeatureVector) { fv.TotalLoansDefaulted = 3 }, nil},
	}

	var importances []FeatureImportance
	for _, p := range perturbations {
		// Deep copy
		copyJSON, _ := json.Marshal(baseline)
		var perturbed FeatureVector
		json.Unmarshal(copyJSON, &perturbed)

		p.perturb(&perturbed)
		perturbedResult, err := scorer.Score(ctx, &perturbed)
		if err != nil {
			continue
		}

		scoreDrop := float64(baseResult.Score - perturbedResult.Score)
		importance := scoreDrop / float64(baseResult.Score)
		if importance < 0 {
			importance = -importance
		}

		direction := "POSITIVE"
		if scoreDrop > 0 {
			direction = "NEGATIVE" // Removing this feature hurts the score
		}

		importances = append(importances, FeatureImportance{
			FeatureName: p.name,
			Importance:  importance,
			Direction:   direction,
			Description: fmt.Sprintf("Score change: %+.0f when zeroed", -scoreDrop),
		})
	}

	return importances, nil
}

// --- Behavioral Signals ---

// BehavioralSignals captures USSD interaction patterns for credit scoring.
// These are logged by the USSD engine and consumed by the FeatureComputer.
type BehavioralSignals struct {
	CrewMemberID       string    `json:"crew_member_id"`
	AvgSessionLength   float64   `json:"avg_session_length_sec"`   // Average USSD session duration
	SessionsPerWeek    float64   `json:"sessions_per_week"`        // Platform engagement frequency
	TimeOfDayPattern   string    `json:"time_of_day_pattern"`      // "MORNING", "AFTERNOON", "EVENING", "NIGHT"
	LanguagePreference string    `json:"language_preference"`      // "en", "sw"
	MenuDepthAvg       float64   `json:"menu_depth_avg"`           // How deep users navigate
	ErrorRate          float64   `json:"error_rate"`               // Invalid input frequency
	BalanceCheckFreq   float64   `json:"balance_check_freq_week"`  // Financial awareness signal
	RecordedAt         time.Time `json:"recorded_at"`
}

// --- Real-Time Scoring Events ---

// ScoringEvent represents a trigger for real-time score recalculation.
type ScoringEvent struct {
	EventType    string    `json:"event_type"`     // "LOAN_COMPLETED", "SHIFT_COMPLETED", "DEPOSIT", etc.
	CrewMemberID string    `json:"crew_member_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ShouldRecalculate returns true if this event type should trigger an immediate score recalculation.
func (e *ScoringEvent) ShouldRecalculate() bool {
	highImpactEvents := map[string]bool{
		"LOAN_COMPLETED":  true,
		"LOAN_DEFAULTED":  true,
		"FRAUD_FLAG":      true,
		"KYC_VERIFIED":    true,
		"LARGE_DEPOSIT":   true,
	}
	return highImpactEvents[e.EventType]
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
