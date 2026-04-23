package handler

import (
	"testing"
)

func TestSanitizeCSVCell_Empty(t *testing.T) {
	if got := sanitizeCSVCell(""); got != "" {
		t.Errorf("sanitizeCSVCell(\"\") = %q, want \"\"", got)
	}
}

func TestSanitizeCSVCell_NormalString(t *testing.T) {
	if got := sanitizeCSVCell("Regular description"); got != "Regular description" {
		t.Errorf("sanitizeCSVCell(\"Regular description\") = %q, want \"Regular description\"", got)
	}
}

func TestSanitizeCSVCell_FormulaEquals(t *testing.T) {
	input := "=CMD('calc')"
	got := sanitizeCSVCell(input)
	if got[0] != '\'' {
		t.Errorf("sanitizeCSVCell(%q) = %q, want leading single-quote", input, got)
	}
}

func TestSanitizeCSVCell_FormulaPlus(t *testing.T) {
	input := "+1+2"
	got := sanitizeCSVCell(input)
	if got[0] != '\'' {
		t.Errorf("sanitizeCSVCell(%q) = %q, want leading single-quote", input, got)
	}
}

func TestSanitizeCSVCell_FormulaMinus(t *testing.T) {
	input := "-SUM(A1:A5)"
	got := sanitizeCSVCell(input)
	if got[0] != '\'' {
		t.Errorf("sanitizeCSVCell(%q) = %q, want leading single-quote", input, got)
	}
}

func TestSanitizeCSVCell_FormulaAt(t *testing.T) {
	input := "@malicious"
	got := sanitizeCSVCell(input)
	if got[0] != '\'' {
		t.Errorf("sanitizeCSVCell(%q) = %q, want leading single-quote", input, got)
	}
}

func TestSanitizeCSVCell_CommasReplaced(t *testing.T) {
	input := "hello,world,test"
	got := sanitizeCSVCell(input)
	for i := 0; i < len(got); i++ {
		if got[i] == ',' {
			t.Errorf("sanitizeCSVCell(%q) = %q, expected commas replaced", input, got)
			break
		}
	}
}

func TestSanitizeCSVCell_NewlinesReplaced(t *testing.T) {
	input := "line1\nline2\rline3"
	got := sanitizeCSVCell(input)
	for i := 0; i < len(got); i++ {
		if got[i] == '\n' || got[i] == '\r' {
			t.Errorf("sanitizeCSVCell(%q) = %q, expected newlines replaced", input, got)
			break
		}
	}
}

func TestSanitizeCSVCell_CombinedThreats(t *testing.T) {
	// Formula injection + comma + newline
	input := "=SUM(A1,A2)\nEvilRow"
	got := sanitizeCSVCell(input)
	if got[0] != '\'' {
		t.Errorf("should start with single-quote, got %q", got)
	}
	for i := 0; i < len(got); i++ {
		if got[i] == ',' || got[i] == '\n' {
			t.Errorf("should not contain comma or newline, got %q", got)
			break
		}
	}
}
