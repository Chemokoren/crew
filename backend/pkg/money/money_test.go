package money

import (
	"testing"
)

func TestToCents(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		expected int64
	}{
		{"zero", 0, 0},
		{"whole number", 100.00, 10000},
		{"with cents", 15.50, 1550},
		{"typical KES amount", 1500.00, 150000},
		{"small amount", 0.01, 1},
		{"rounding up", 19.999, 2000},
		{"rounding down", 19.994, 1999},
		{"large amount", 999999.99, 99999999},
		{"negative", -50.00, -5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCents(tt.amount)
			if got != tt.expected {
				t.Errorf("ToCents(%f) = %d, want %d", tt.amount, got, tt.expected)
			}
		})
	}
}

func TestFromCents(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		expected float64
	}{
		{"zero", 0, 0},
		{"whole", 10000, 100.00},
		{"with fraction", 1550, 15.50},
		{"single cent", 1, 0.01},
		{"negative", -5000, -50.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromCents(tt.cents)
			if got != tt.expected {
				t.Errorf("FromCents(%d) = %f, want %f", tt.cents, got, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Converting to cents and back should preserve value
	amounts := []float64{0, 1.00, 15.50, 1500.00, 99999.99}
	for _, amount := range amounts {
		cents := ToCents(amount)
		back := FromCents(cents)
		if back != amount {
			t.Errorf("round trip failed: %f -> %d -> %f", amount, cents, back)
		}
	}
}

func TestFormatKES(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		expected string
	}{
		{"zero", 0, "KES 0.00"},
		{"small", 100, "KES 1.00"},
		{"typical", 150000, "KES 1,500.00"},
		{"with cents", 150050, "KES 1,500.50"},
		{"large", 9999999, "KES 99,999.99"},
		{"very large", 100000000, "KES 1,000,000.00"},
		{"negative", -150000, "-KES 1,500.00"},
		{"single digit cents", 5, "KES 0.05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatKES(tt.cents)
			if got != tt.expected {
				t.Errorf("FormatKES(%d) = %q, want %q", tt.cents, got, tt.expected)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	got := Format(150000, "USD")
	expected := "USD 1,500.00"
	if got != expected {
		t.Errorf("Format(150000, USD) = %q, want %q", got, expected)
	}
}

func TestValidatePositive(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		wantErr bool
	}{
		{"positive", 100, false},
		{"large positive", 99999999, false},
		{"zero", 0, true},
		{"negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositive(tt.cents)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePositive(%d) error = %v, wantErr %v", tt.cents, err, tt.wantErr)
			}
		})
	}
}
