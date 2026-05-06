package routing

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestHardcodedRoles_MatchBackendTemplates compares the hardcoded job types in
// routing.go with the backend's industry_templates.go to detect drift.
//
// This test reads the backend source file and extracts Code values from each
// industry's DefaultJobTypes slice, then verifies every PRIMARY/FACILITATOR
// code in the backend exists in the USSD hardcoded fallback.
//
// Run this as part of CI to catch drift between the two independently-owned
// role definitions.
func TestHardcodedRoles_MatchBackendTemplates(t *testing.T) {
	backendPath := "../../../backend/internal/models/industry_templates.go"

	// If the backend file isn't available (e.g., running in isolation), skip
	if _, err := os.Stat(backendPath); os.IsNotExist(err) {
		t.Skipf("backend file not found at %s — skipping drift check (run from repo root)", backendPath)
	}

	backendRoles, err := parseBackendRoles(backendPath)
	if err != nil {
		t.Fatalf("failed to parse backend templates: %v", err)
	}

	// Map USSD hardcoded roles for quick lookup
	ussdRoles := make(map[string]map[string]bool)
	for industry, jts := range industryJobTypes {
		ussdRoles[industry] = make(map[string]bool)
		for _, jt := range jts {
			ussdRoles[industry][jt.Code] = true
		}
	}

	// Supervisory/support roles that should NOT be in USSD (self-registration exclusions)
	excludedCategories := map[string]bool{
		"JobCategorySupervisor": true,
		"JobCategorySupport":   true,
	}

	// Skip pseudo-industries that are only fallback templates in the backend
	skipIndustries := map[string]bool{
		"GENERAL": true, // General fallback in backend, not a real industry
	}

	// Compare: every non-excluded backend role should exist in USSD
	for industry, roles := range backendRoles {
		ussdIndustryKey := normalizeIndustryName(industry)
		if skipIndustries[ussdIndustryKey] {
			continue
		}
		ussdCodes, ok := ussdRoles[ussdIndustryKey]
		if !ok {
			t.Errorf("DRIFT: backend has industry %q but USSD routing.go has no hardcoded roles for it", ussdIndustryKey)
			continue
		}

		for _, role := range roles {
			if excludedCategories[role.category] {
				continue // SUPERVISOR/SUPPORT roles are intentionally excluded from USSD
			}
			if !ussdCodes[role.code] {
				t.Errorf("DRIFT: backend %s has role %q (%s) but USSD routing.go is missing it",
					ussdIndustryKey, role.code, role.displayName)
			}
		}
	}

	// Reverse check: every USSD role should exist in backend (to catch orphaned hardcoded roles)
	for industry, codes := range ussdRoles {
		backendIndustry, ok := backendRoles[industryConstName(industry)]
		if !ok {
			// Try direct match
			found := false
			for bName := range backendRoles {
				if normalizeIndustryName(bName) == industry {
					backendIndustry = backendRoles[bName]
					found = true
					break
				}
			}
			if !found {
				continue // Industry is USSD-only (e.g., future industry not yet in backend)
			}
		}

		backendCodes := make(map[string]bool)
		for _, role := range backendIndustry {
			backendCodes[role.code] = true
		}

		for code := range codes {
			if !backendCodes[code] {
				t.Logf("NOTE: USSD %s has role %q not in backend template (may be intentional)", industry, code)
			}
		}
	}
}

// --- Source file parser (reads Go source, no AST needed) ---

type backendRole struct {
	code        string
	displayName string
	category    string
}

// parseBackendRoles extracts role codes from the backend's industry_templates.go
// by scanning for {Code: "...", DisplayName: "...", Category: ...} patterns
// within the templates map. Stops parsing at the map's closing brace.
func parseBackendRoles(path string) (map[string][]backendRole, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	result := make(map[string][]backendRole)
	scanner := bufio.NewScanner(f)

	// Regex patterns
	industryRe := regexp.MustCompile(`Industry(\w+):\s*\{`)
	codeRe := regexp.MustCompile(`\{Code:\s*"([^"]+)",\s*DisplayName:\s*"([^"]+)",\s*Category:\s*(\w+)\}`)
	mapStartRe := regexp.MustCompile(`templates\s*:=\s*map\[`)

	currentIndustry := ""
	inJobTypes := false
	inMap := false
	braceDepth := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect the start of the templates map
		if mapStartRe.MatchString(line) {
			inMap = true
			braceDepth = 0
			continue
		}

		// Only parse inside the templates map
		if !inMap {
			continue
		}

		// Track brace depth to detect map end
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
		if braceDepth <= 0 {
			break // Exited the templates map — stop parsing
		}

		// Track which industry block we're in
		if matches := industryRe.FindStringSubmatch(line); len(matches) > 1 {
			currentIndustry = matches[1]
			inJobTypes = false
		}

		// Track when we're inside DefaultJobTypes
		if strings.Contains(line, "DefaultJobTypes:") {
			inJobTypes = true
		}

		// Parse role entries
		if inJobTypes && currentIndustry != "" {
			if matches := codeRe.FindStringSubmatch(line); len(matches) > 3 {
				result[currentIndustry] = append(result[currentIndustry], backendRole{
					code:        matches[1],
					displayName: matches[2],
					category:    matches[3],
				})
			}

			// End of DefaultJobTypes slice
			if strings.HasPrefix(line, "},") || line == "}," {
				if len(result[currentIndustry]) > 0 {
					inJobTypes = false
				}
			}
		}
	}

	return result, scanner.Err()
}

// normalizeIndustryName converts backend const names like "Transport" to USSD keys like "TRANSPORT"
func normalizeIndustryName(name string) string {
	return strings.ToUpper(name)
}

// industryConstName converts USSD keys like "TRANSPORT" to backend const names like "Transport"
func industryConstName(industry string) string {
	if len(industry) == 0 {
		return ""
	}
	return strings.ToUpper(industry[:1]) + strings.ToLower(industry[1:])
}
