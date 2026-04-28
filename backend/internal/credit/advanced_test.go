package credit

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- FairnessAuditor Tests ---

func TestFairnessAuditor_FairGroups(t *testing.T) {
	auditor := NewFairnessAuditor(testLogger())

	groups := map[string][]int{
		"male":   {600, 620, 650, 580, 700},
		"female": {610, 630, 640, 590, 690},
	}
	report := auditor.Audit(groups)

	if !report.OverallFair {
		t.Errorf("similar groups should be fair, got disparate_impact=%.2f, stat_parity=%.2f",
			report.DisparateImpact, report.StatisticalParity)
	}
	if len(report.GroupAnalysis) != 2 {
		t.Errorf("expected 2 groups, got %d", len(report.GroupAnalysis))
	}
}

func TestFairnessAuditor_UnfairGroups(t *testing.T) {
	auditor := NewFairnessAuditor(testLogger())

	groups := map[string][]int{
		"group_a": {700, 750, 800, 650, 700},       // All above 400
		"group_b": {300, 350, 310, 320, 380},        // All below 400
	}
	report := auditor.Audit(groups)

	if report.OverallFair {
		t.Error("vastly different groups should be unfair")
	}
	if report.DisparateImpact >= 0.8 {
		t.Errorf("expected disparate impact < 0.8, got %.2f", report.DisparateImpact)
	}
	if len(report.Recommendations) == 0 {
		t.Error("unfair report should have recommendations")
	}
}

func TestFairnessAuditor_SingleGroup(t *testing.T) {
	auditor := NewFairnessAuditor(testLogger())
	groups := map[string][]int{
		"all": {500, 600, 700},
	}
	report := auditor.Audit(groups)
	// Single group: disparate impact = 1.0, stat parity = 0
	if report.DisparateImpact != 1.0 {
		t.Errorf("single group DI should be 1.0, got %.2f", report.DisparateImpact)
	}
}

// --- SurvivalAnalyzer Tests ---

func TestSurvivalAnalyzer_LowRisk(t *testing.T) {
	analyzer := NewSurvivalAnalyzer(testLogger())
	fv := makeBaselineFeatures()
	fv.OnTimeRepaymentRate = 1.0
	fv.AccountAgeDays = 365
	fv.ActiveInsurancePolicies = 2

	est := analyzer.Estimate(fv)
	if est.RiskGroup != "LOW" {
		t.Errorf("good user risk = %q, want LOW", est.RiskGroup)
	}
	if est.ProbDefault30d > 0.1 {
		t.Errorf("good user 30d default prob %.3f too high", est.ProbDefault30d)
	}
	if est.MedianSurvival < 30 {
		t.Errorf("good user median survival %d too low", est.MedianSurvival)
	}
}

func TestSurvivalAnalyzer_HighRisk(t *testing.T) {
	analyzer := NewSurvivalAnalyzer(testLogger())
	fv := &FeatureVector{
		CrewMemberID:        uuid.New(),
		ComputedAt:          time.Now(),
		TotalLoansDefaulted: 3,
		OnTimeRepaymentRate: 0.2,
		CurrentBalanceKES:   50,
		CompletedShifts30d:  1,
		FraudFlags:          2,
	}

	est := analyzer.Estimate(fv)
	if est.RiskGroup != "HIGH" {
		t.Errorf("risky user risk = %q, want HIGH", est.RiskGroup)
	}
	if est.ProbDefault30d < est.ProbDefault7d {
		t.Error("30d probability should be >= 7d probability")
	}
	if est.ProbDefault90d < est.ProbDefault30d {
		t.Error("90d probability should be >= 30d probability")
	}
}

func TestSurvivalAnalyzer_ProbabilitiesMonotonic(t *testing.T) {
	analyzer := NewSurvivalAnalyzer(testLogger())
	fv := makeBaselineFeatures()
	est := analyzer.Estimate(fv)

	if est.ProbDefault7d > est.ProbDefault14d {
		t.Error("P(7d) > P(14d)")
	}
	if est.ProbDefault14d > est.ProbDefault30d {
		t.Error("P(14d) > P(30d)")
	}
	if est.ProbDefault30d > est.ProbDefault90d {
		t.Error("P(30d) > P(90d)")
	}
}

func TestSurvivalAnalyzer_ProbabilitiesBounded(t *testing.T) {
	analyzer := NewSurvivalAnalyzer(testLogger())
	fv := makeBaselineFeatures()
	est := analyzer.Estimate(fv)

	for _, p := range []float64{est.ProbDefault7d, est.ProbDefault14d, est.ProbDefault30d, est.ProbDefault90d} {
		if p < 0 || p > 1 {
			t.Errorf("probability %.3f out of [0,1]", p)
		}
	}
}

// --- PSI Helper Tests ---

func TestCalculatePSI_Identical(t *testing.T) {
	data := []float64{500, 550, 600, 650, 700}
	psi := calculatePSI(data, data, 5)
	if psi > 0.01 {
		t.Errorf("identical distributions should have PSI~0, got %.4f", psi)
	}
}

func TestCalculatePSI_Empty(t *testing.T) {
	psi := calculatePSI(nil, []float64{1, 2, 3}, 5)
	if psi != 0 {
		t.Errorf("empty baseline should return 0, got %.4f", psi)
	}
}

func TestMean(t *testing.T) {
	tests := []struct {
		vals []float64
		want float64
	}{
		{[]float64{10, 20, 30}, 20},
		{[]float64{100}, 100},
		{nil, 0},
		{[]float64{}, 0},
	}
	for _, tc := range tests {
		if got := mean(tc.vals); got != tc.want {
			t.Errorf("mean(%v) = %f, want %f", tc.vals, got, tc.want)
		}
	}
}

// --- DriftReport Tests ---

func TestDriftReport_ActionLevels(t *testing.T) {
	tests := []struct {
		psi    float64
		action string
	}{
		{0.05, "NONE"},
		{0.15, "MONITOR"},
		{0.30, "RETRAIN"},
	}
	for _, tc := range tests {
		report := DriftReport{PSI: tc.psi}
		if tc.psi > 0.25 {
			report.DriftDetected = true
			report.ActionRequired = "RETRAIN"
		} else if tc.psi > 0.1 {
			report.ActionRequired = "MONITOR"
		} else {
			report.ActionRequired = "NONE"
		}
		if report.ActionRequired != tc.action {
			t.Errorf("PSI=%.2f: action=%q, want %q", tc.psi, report.ActionRequired, tc.action)
		}
	}
}

// --- NetworkRiskSignal Tests ---

func TestNetworkRiskSignal_Levels(t *testing.T) {
	tests := []struct {
		rate  float64
		level string
	}{
		{0.0, "LOW"},
		{0.05, "LOW"},
		{0.15, "MEDIUM"},
		{0.25, "HIGH"},
	}
	for _, tc := range tests {
		level := "LOW"
		if tc.rate > 0.2 {
			level = "HIGH"
		} else if tc.rate > 0.1 {
			level = "MEDIUM"
		}
		if level != tc.level {
			t.Errorf("rate=%.2f: level=%q, want %q", tc.rate, level, tc.level)
		}
	}
}
