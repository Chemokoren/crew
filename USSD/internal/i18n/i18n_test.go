package i18n

import (
	"testing"
)

func TestTranslator_English(t *testing.T) {
	tr := NewTranslator("en")

	tests := []struct {
		key  string
		want string
	}{
		{"menu.welcome", "Welcome to CrewPay"},
		{"menu.check_balance", "Check Balance"},
		{"menu.withdraw", "Withdraw"},
		{"menu.earnings", "My Earnings"},
		{"menu.exit", "Exit"},
		{"goodbye", "Thank you for using CrewPay.\nGoodbye!"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := tr.T("en", tt.key)
			if got != tt.want {
				t.Errorf("T('en', %q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestTranslator_Swahili(t *testing.T) {
	tr := NewTranslator("en")

	got := tr.T("sw", "menu.welcome")
	if got != "Karibu CrewPay" {
		t.Errorf("T('sw', 'menu.welcome') = %q, want %q", got, "Karibu CrewPay")
	}

	got = tr.T("sw", "menu.check_balance")
	if got != "Angalia Salio" {
		t.Errorf("T('sw', 'menu.check_balance') = %q, want %q", got, "Angalia Salio")
	}
}

func TestTranslator_Fallback(t *testing.T) {
	tr := NewTranslator("en")

	// Unknown language should fall back to English
	got := tr.T("fr", "menu.welcome")
	if got != "Welcome to CrewPay" {
		t.Errorf("T('fr', 'menu.welcome') should fallback to English, got %q", got)
	}
}

func TestTranslator_MissingKey(t *testing.T) {
	tr := NewTranslator("en")

	got := tr.T("en", "nonexistent.key")
	if got != "[nonexistent.key]" {
		t.Errorf("T for missing key should return [key], got %q", got)
	}
}

func TestTranslator_EmptyLanguage(t *testing.T) {
	tr := NewTranslator("en")

	got := tr.T("", "menu.welcome")
	if got != "Welcome to CrewPay" {
		t.Errorf("T('', key) should use default language, got %q", got)
	}
}

func TestTranslator_SupportedLanguages(t *testing.T) {
	tr := NewTranslator("en")

	langs := tr.SupportedLanguages()
	if len(langs) < 2 {
		t.Errorf("expected at least 2 languages, got %d", len(langs))
	}

	if !tr.HasLanguage("en") {
		t.Error("should support English")
	}
	if !tr.HasLanguage("sw") {
		t.Error("should support Swahili")
	}
	if tr.HasLanguage("fr") {
		t.Error("should not support French")
	}
}

func TestTranslator_SetMessage(t *testing.T) {
	tr := NewTranslator("en")

	// Add custom message
	tr.SetMessage("en", "custom.test", "Hello World")
	got := tr.T("en", "custom.test")
	if got != "Hello World" {
		t.Errorf("custom message = %q, want %q", got, "Hello World")
	}

	// Add new language
	tr.SetMessage("fr", "menu.welcome", "Bienvenue à CrewPay")
	got = tr.T("fr", "menu.welcome")
	if got != "Bienvenue à CrewPay" {
		t.Errorf("French message = %q, want %q", got, "Bienvenue à CrewPay")
	}
}

func TestTruncateToLimit(t *testing.T) {
	tests := []struct {
		name  string
		msg   string
		limit int
		short bool // true if result should be shorter than original
	}{
		{"short message", "Hello", 160, false},
		{"exactly at limit", "x", 1, false},
		{"over limit", "This is a very long message that exceeds the limit set for USSD screens which is typically around 160 characters and we need to truncate it properly to avoid issues", 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateToLimit(tt.msg, tt.limit)
			if tt.short && len(got) > tt.limit {
				t.Errorf("truncated message len=%d exceeds limit=%d", len(got), tt.limit)
			}
			if !tt.short && got != tt.msg {
				t.Errorf("should not truncate, got %q", got)
			}
		})
	}
}
