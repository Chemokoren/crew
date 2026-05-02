package models

// ResolveLanguage implements Decision D5: User preference > Tenant config > System default.
// The user note explicitly says: "user configuration should override the rest —
// even if the transport sector chose Swahili, if I choose English then I will always access it in English."
func ResolveLanguage(userPref string, orgDefault string) string {
	// Priority 1: User preference always wins
	if userPref != "" && userPref != "default" {
		return userPref
	}
	// Priority 2: Organization (tenant) default
	if orgDefault != "" {
		return orgDefault
	}
	// Priority 3: System default
	return SystemDefaultLanguage
}

// SystemDefaultLanguage is the global system default (Swahili).
const SystemDefaultLanguage = "sw"

// SupportedLanguages lists all supported language codes.
var SupportedLanguages = []string{"sw", "en", "luo", "kik"}

// IsValidLanguage checks if a language code is supported.
func IsValidLanguage(lang string) bool {
	for _, l := range SupportedLanguages {
		if l == lang {
			return true
		}
	}
	return false
}
