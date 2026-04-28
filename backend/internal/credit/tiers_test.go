package credit

import "testing"

func TestGetTierForScore(t *testing.T) {
	tests := []struct {
		score     int
		wantGrade string
		wantNil   bool
	}{
		{850, "EXCELLENT", false},
		{750, "EXCELLENT", false},
		{749, "GOOD", false},
		{650, "GOOD", false},
		{649, "FAIR", false},
		{500, "FAIR", false},
		{499, "POOR", false},
		{400, "POOR", false},
		{399, "", true},
		{300, "", true},
		{0, "", true},
	}
	for _, tc := range tests {
		tier := GetTierForScore(tc.score)
		if tc.wantNil {
			if tier != nil {
				t.Errorf("GetTierForScore(%d) = %+v, want nil", tc.score, tier)
			}
			continue
		}
		if tier == nil {
			t.Fatalf("GetTierForScore(%d) = nil, want %q", tc.score, tc.wantGrade)
		}
		if tier.Grade != tc.wantGrade {
			t.Errorf("GetTierForScore(%d).Grade = %q, want %q", tc.score, tier.Grade, tc.wantGrade)
		}
	}
}

func TestDefaultTiers_Invariants(t *testing.T) {
	tiers := DefaultTiers()

	if len(tiers) != 4 {
		t.Fatalf("expected 4 tiers, got %d", len(tiers))
	}

	// Tiers must be sorted descending by MinScore
	for i := 1; i < len(tiers); i++ {
		if tiers[i].MinScore >= tiers[i-1].MinScore {
			t.Errorf("tiers not sorted desc: tier[%d].MinScore=%d >= tier[%d].MinScore=%d",
				i, tiers[i].MinScore, i-1, tiers[i-1].MinScore)
		}
	}

	// Interest rates: higher tier = lower rate
	for i := 1; i < len(tiers); i++ {
		if tiers[i].InterestRate <= tiers[i-1].InterestRate {
			t.Errorf("lower tiers should have higher rates: %s=%.2f <= %s=%.2f",
				tiers[i].Grade, tiers[i].InterestRate, tiers[i-1].Grade, tiers[i-1].InterestRate)
		}
	}

	// Max loan: higher tier = higher limit
	for i := 1; i < len(tiers); i++ {
		if tiers[i].MaxLoanCents >= tiers[i-1].MaxLoanCents {
			t.Errorf("lower tiers should have lower limits: %s=%d >= %s=%d",
				tiers[i].Grade, tiers[i].MaxLoanCents, tiers[i-1].Grade, tiers[i-1].MaxLoanCents)
		}
	}

	// All tiers must have positive values
	for _, tier := range tiers {
		if tier.MaxLoanCents <= 0 {
			t.Errorf("tier %s: MaxLoanCents must be positive", tier.Grade)
		}
		if tier.InterestRate <= 0 || tier.InterestRate > 1 {
			t.Errorf("tier %s: InterestRate %.2f out of (0,1] range", tier.Grade, tier.InterestRate)
		}
		if tier.MaxTenureDays <= 0 {
			t.Errorf("tier %s: MaxTenureDays must be positive", tier.Grade)
		}
	}
}

func TestLoanTier_FormatMaxLoanKES(t *testing.T) {
	tier := LoanTier{MaxLoanCents: 5_000_000}
	if got := tier.FormatMaxLoanKES(); got != 50000 {
		t.Errorf("FormatMaxLoanKES() = %f, want 50000", got)
	}
}

func TestLoanTier_FormatInterestPercent(t *testing.T) {
	tier := LoanTier{InterestRate: 0.05}
	if got := tier.FormatInterestPercent(); got != 5.0 {
		t.Errorf("FormatInterestPercent() = %f, want 5.0", got)
	}
}

func TestGetTierForScore_AllGradesReachable(t *testing.T) {
	grades := map[string]bool{"EXCELLENT": false, "GOOD": false, "FAIR": false, "POOR": false}
	for score := 400; score <= 850; score++ {
		tier := GetTierForScore(score)
		if tier != nil {
			grades[tier.Grade] = true
		}
	}
	for g, reached := range grades {
		if !reached {
			t.Errorf("grade %q is never reached for scores 400-850", g)
		}
	}
}
