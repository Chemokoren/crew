package gateway

import (
	"testing"
)

func TestExtractLastInput(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		want  string
	}{
		{"empty", "", ""},
		{"single input", "1", "1"},
		{"two inputs", "1*2", "2"},
		{"three inputs", "1*2*3", "3"},
		{"with spaces", "1*2* 3 ", "3"},
		{"text input", "1*John Doe", "John Doe"},
		{"deep navigation", "1*2*3*4*5", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLastInput(tt.text)
			if got != tt.want {
				t.Errorf("extractLastInput(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"already international", "+254712345678", "+254712345678"},
		{"with zero prefix", "0712345678", "+254712345678"},
		{"without plus", "254712345678", "+254712345678"},
		{"with spaces", " +254 712 345 678 ", "+254712345678"},
		{"short number", "+254", "+254"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePhone(tt.phone)
			if got != tt.want {
				t.Errorf("normalizePhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}
