package engine

import (
	"testing"
)

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		currency string
		want     string
	}{
		{"zero", 0, "KES", "KES 0.00"},
		{"whole amount", 10000, "KES", "KES 100.00"},
		{"with cents", 15050, "KES", "KES 150.50"},
		{"small amount", 50, "KES", "KES 0.50"},
		{"large amount", 100000000, "KES", "KES 1000000.00"},
		{"one cent", 1, "KES", "KES 0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMoney(tt.cents, tt.currency)
			if got != tt.want {
				t.Errorf("formatMoney(%d, %q) = %q, want %q", tt.cents, tt.currency, got, tt.want)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"whole number", "100", 10000, false},
		{"with decimals", "150.50", 15050, false},
		{"single decimal", "50.5", 5050, false},
		{"zero", "0", 0, false},
		{"empty", "", 0, true},
		{"non-numeric", "abc", 0, true},
		{"with spaces", " 100 ", 10000, false},
		{"large number", "1000000", 100000000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmount(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
