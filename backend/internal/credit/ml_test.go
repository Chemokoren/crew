package credit

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- MLScorerV3 Tests ---

func TestMLScorerV3_ReturnsStub(t *testing.T) {
	ml := NewMLScorerV3(MLConfig{ModelName: "test", ModelVersion: "0.1"}, testLogger())
	fv := makeBaselineFeatures()
	result, err := ml.Score(context.Background(), fv)
	if err != nil {
		t.Fatalf("MLScorer error: %v", err)
	}
	if result.Score < 300 || result.Score > 850 {
		t.Errorf("stub score %d out of range", result.Score)
	}
}

func TestMLScorerV3_Version(t *testing.T) {
	ml := NewMLScorerV3(MLConfig{ModelName: "xgboost", ModelVersion: "1.0"}, testLogger())
	want := "ml-xgboost-1.0"
	if got := ml.Version(); got != want {
		t.Errorf("Version() = %q, want %q", got, want)
	}
}

// --- HybridScorer Tests ---

func TestHybridScorer_Ensemble(t *testing.T) {
	rules := NewRulesScorer()
	ml := NewMLScorerV3(MLConfig{ModelName: "t", ModelVersion: "1"}, testLogger())
	hybrid := NewHybridScorer(rules, ml, HybridScorerConfig{RulesWeight: 0.7, MLWeight: 0.3}, testLogger())

	fv := makeBaselineFeatures()
	result, err := hybrid.Score(context.Background(), fv)
	if err != nil {
		t.Fatalf("Hybrid error: %v", err)
	}
	if result.Score < 300 || result.Score > 850 {
		t.Errorf("hybrid score %d out of range", result.Score)
	}
	if result.ModelVersion == "" {
		t.Error("model version empty")
	}
}

func TestHybridScorer_DefaultWeights(t *testing.T) {
	rules := NewRulesScorer()
	ml := NewMLScorerV3(MLConfig{}, testLogger())
	hybrid := NewHybridScorer(rules, ml, HybridScorerConfig{}, testLogger())

	result, err := hybrid.Score(context.Background(), makeBaselineFeatures())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Score < 300 || result.Score > 850 {
		t.Errorf("default-weight score %d out of range", result.Score)
	}
}

// --- ABTestScorer Tests ---

func TestABTestScorer_InactiveRoutesToControl(t *testing.T) {
	rules := NewRulesScorer()
	ml := NewMLScorerV3(MLConfig{}, testLogger())

	ab := NewABTestScorer(rules, ml, ABTestConfig{
		Name:           "test",
		Active:         false,
		TrafficPercent: 1.0, // Would route all to treatment if active
	}, testLogger())

	result, err := ab.Score(context.Background(), makeBaselineFeatures())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should use control (rules) scorer
	if result.ModelVersion != "rules-v2.1" {
		t.Errorf("inactive AB should use control, got version %q", result.ModelVersion)
	}
}

func TestABTestScorer_ExpiredRoutesToControl(t *testing.T) {
	rules := NewRulesScorer()
	ml := NewMLScorerV3(MLConfig{}, testLogger())

	ab := NewABTestScorer(rules, ml, ABTestConfig{
		Name:           "expired",
		Active:         true,
		TrafficPercent: 1.0,
		StartDate:      time.Now().Add(-48 * time.Hour),
		EndDate:        time.Now().Add(-24 * time.Hour), // Ended yesterday
	}, testLogger())

	result, _ := ab.Score(context.Background(), makeBaselineFeatures())
	if result.ModelVersion != "rules-v2.1" {
		t.Errorf("expired AB should use control, got %q", result.ModelVersion)
	}
}

func TestABTestScorer_ActiveRoutesDeterministic(t *testing.T) {
	rules := NewRulesScorer()
	ml := NewMLScorerV3(MLConfig{ModelName: "t", ModelVersion: "1"}, testLogger())

	ab := NewABTestScorer(rules, ml, ABTestConfig{
		Name:           "det-test",
		Active:         true,
		TrafficPercent: 0.5,
		StartDate:      time.Now().Add(-1 * time.Hour),
		EndDate:        time.Now().Add(24 * time.Hour),
	}, testLogger())

	fv := makeBaselineFeatures()
	r1, _ := ab.Score(context.Background(), fv)
	r2, _ := ab.Score(context.Background(), fv)

	if r1.ModelVersion != r2.ModelVersion {
		t.Error("AB test routing should be deterministic for same crew member")
	}
}

// --- ScoringEvent Tests ---

func TestScoringEvent_ShouldRecalculate(t *testing.T) {
	highImpact := []string{"LOAN_COMPLETED", "LOAN_DEFAULTED", "FRAUD_FLAG", "KYC_VERIFIED", "LARGE_DEPOSIT"}
	for _, et := range highImpact {
		e := ScoringEvent{EventType: et}
		if !e.ShouldRecalculate() {
			t.Errorf("%q should trigger recalculation", et)
		}
	}

	lowImpact := []string{"BALANCE_CHECK", "LOGIN", "MENU_NAVIGATE"}
	for _, et := range lowImpact {
		e := ScoringEvent{EventType: et}
		if e.ShouldRecalculate() {
			t.Errorf("%q should NOT trigger recalculation", et)
		}
	}
}

// --- Feature Importance Tests ---

func TestComputeFeatureImportance(t *testing.T) {
	scorer := NewRulesScorer()
	fv := makeBaselineFeatures()
	importances, err := ComputeFeatureImportance(scorer, fv)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(importances) == 0 {
		t.Fatal("no feature importances returned")
	}
	for _, fi := range importances {
		if fi.FeatureName == "" {
			t.Error("empty feature name")
		}
		if fi.Importance < 0 {
			t.Errorf("negative importance for %s", fi.FeatureName)
		}
		if fi.Direction != "POSITIVE" && fi.Direction != "NEGATIVE" {
			t.Errorf("invalid direction %q for %s", fi.Direction, fi.FeatureName)
		}
	}
}

// --- BehavioralSignals Tests ---

func TestBehavioralSignals_Fields(t *testing.T) {
	sig := BehavioralSignals{
		CrewMemberID:     uuid.New().String(),
		AvgSessionLength: 45.5,
		SessionsPerWeek:  12.0,
		TimeOfDayPattern: "MORNING",
		ErrorRate:        0.05,
		RecordedAt:       time.Now(),
	}
	if sig.AvgSessionLength <= 0 {
		t.Error("session length should be positive")
	}
}
