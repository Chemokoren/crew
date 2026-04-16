// Package money provides utilities for handling monetary amounts as integer cents.
// All monetary values in AMY MIS are stored as int64 cents to avoid floating-point errors.
// KES 1,500.00 = 150000 cents.
package money

import (
	"fmt"
	"math"
)

// ToCents converts a float64 amount to integer cents.
// Uses math.Round to handle floating-point imprecision.
func ToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// FromCents converts integer cents to a float64 amount.
// Use only for display purposes — never for calculations.
func FromCents(cents int64) float64 {
	return float64(cents) / 100.0
}

// FormatKES formats cents as a KES currency string.
// Example: 150000 → "KES 1,500.00"
func FormatKES(cents int64) string {
	return Format(cents, "KES")
}

// Format formats cents as a currency string with the given currency code.
// Example: Format(150000, "KES") → "KES 1,500.00"
func Format(cents int64, currency string) string {
	negative := ""
	if cents < 0 {
		negative = "-"
		cents = -cents
	}

	whole := cents / 100
	frac := cents % 100

	// Add thousand separators
	wholeStr := formatWithCommas(whole)

	return fmt.Sprintf("%s%s %s.%02d", negative, currency, wholeStr, frac)
}

// formatWithCommas adds thousand separators to an integer.
func formatWithCommas(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// ValidatePositive returns an error if the amount is not positive.
func ValidatePositive(cents int64) error {
	if cents <= 0 {
		return fmt.Errorf("amount must be positive, got %d cents", cents)
	}
	return nil
}
