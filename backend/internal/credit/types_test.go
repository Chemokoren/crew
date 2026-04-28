package credit

import "testing"

func TestScoreGrade(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{850, "EXCELLENT"}, {750, "EXCELLENT"}, {800, "EXCELLENT"},
		{749, "GOOD"}, {650, "GOOD"}, {700, "GOOD"},
		{649, "FAIR"}, {500, "FAIR"}, {550, "FAIR"},
		{499, "POOR"}, {400, "POOR"}, {450, "POOR"},
		{399, "VERY_POOR"}, {300, "VERY_POOR"}, {0, "VERY_POOR"}, {-10, "VERY_POOR"},
	}
	for _, tc := range tests {
		got := ScoreGrade(tc.score)
		if got != tc.want {
			t.Errorf("ScoreGrade(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestScoreGrade_BoundaryExact(t *testing.T) {
	boundaries := map[int]string{750: "EXCELLENT", 650: "GOOD", 500: "FAIR", 400: "POOR"}
	for score, want := range boundaries {
		if got := ScoreGrade(score); got != want {
			t.Errorf("ScoreGrade(%d) = %q, want %q (boundary)", score, got, want)
		}
		belowGrade := ScoreGrade(score - 1)
		if belowGrade == want {
			t.Errorf("ScoreGrade(%d) should not be %q", score-1, want)
		}
	}
}
