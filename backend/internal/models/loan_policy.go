package models

// =============================================================================
// Loan Category System
// =============================================================================
//
// Market Research — Best Practices:
//
// 1. M-Shwari / KCB M-PESA:
//    - One active loan per PRODUCT (M-Shwari vs KCB M-PESA are separate products)
//    - Can hold M-Shwari + KCB M-PESA simultaneously
//
// 2. Tala / Branch:
//    - Strictly one active loan at a time across all categories
//    - Must fully repay before re-applying
//
// 3. Traditional Banks (KCB, Equity, Co-op):
//    - Multiple concurrent loans allowed across different categories
//    - Mortgage + Personal + Education can coexist
//    - Controlled by aggregate exposure limits (% of income)
//
// 4. SACCO Model:
//    - Multiple concurrent loans with share-capital backing
//    - Categories: Emergency, Development, School Fees, Normal
//    - Aggregate limit = 3× share capital
//
// CrewPay Implementation:
//    Configurable via LOAN_CONCURRENCY_POLICY env var:
//    - "SINGLE"       → Only one active loan at a time (Tala-style, default)
//    - "PER_CATEGORY" → One active loan per category (M-Shwari-style)
//    - "AGGREGATE"    → Multiple loans up to aggregate exposure limit (Bank-style)
// =============================================================================

// LoanCategory defines the purpose/type of a loan.
// Different categories may have different limits and can coexist
// depending on the concurrency policy.
type LoanCategory string

const (
	// LoanCatPersonal covers general-purpose personal loans.
	// This is the default category for uncategorized applications.
	LoanCatPersonal LoanCategory = "PERSONAL"

	// LoanCatEmergency covers urgent short-term needs (medical, etc).
	// Typically has faster approval and shorter tenure.
	LoanCatEmergency LoanCategory = "EMERGENCY"

	// LoanCatEducation covers school fees and education expenses.
	// Often has longer tenure and lower interest rates.
	LoanCatEducation LoanCategory = "EDUCATION"

	// LoanCatBusiness covers working capital and business investment.
	// May have higher limits for higher-tier borrowers.
	LoanCatBusiness LoanCategory = "BUSINESS"

	// LoanCatAsset covers asset acquisition (vehicle, equipment).
	// Longest tenure, may require higher credit score.
	LoanCatAsset LoanCategory = "ASSET"
)

// AllLoanCategories returns all defined loan categories for validation and UI.
func AllLoanCategories() []LoanCategory {
	return []LoanCategory{
		LoanCatPersonal,
		LoanCatEmergency,
		LoanCatEducation,
		LoanCatBusiness,
		LoanCatAsset,
	}
}

// IsValid checks if a loan category is recognized.
func (c LoanCategory) IsValid() bool {
	for _, valid := range AllLoanCategories() {
		if c == valid {
			return true
		}
	}
	return false
}

// Label returns a human-friendly display name for USSD menus.
func (c LoanCategory) Label() string {
	switch c {
	case LoanCatPersonal:
		return "Personal"
	case LoanCatEmergency:
		return "Emergency"
	case LoanCatEducation:
		return "Education"
	case LoanCatBusiness:
		return "Business"
	case LoanCatAsset:
		return "Asset Finance"
	default:
		return string(c)
	}
}

// LabelSW returns a Swahili display name for USSD menus.
func (c LoanCategory) LabelSW() string {
	switch c {
	case LoanCatPersonal:
		return "Binafsi"
	case LoanCatEmergency:
		return "Dharura"
	case LoanCatEducation:
		return "Elimu"
	case LoanCatBusiness:
		return "Biashara"
	case LoanCatAsset:
		return "Mali"
	default:
		return string(c)
	}
}

// =============================================================================
// Loan Concurrency Policy
// =============================================================================

// LoanConcurrencyPolicy defines the system-wide rule for concurrent loans.
type LoanConcurrencyPolicy string

const (
	// PolicySingle allows only one active loan at a time across all categories.
	// Simplest and most conservative — mirrors Tala/Branch model.
	// Best for: Early-stage lenders, high-risk borrower pools.
	PolicySingle LoanConcurrencyPolicy = "SINGLE"

	// PolicyPerCategory allows one active loan PER category simultaneously.
	// A borrower can hold Personal + Emergency + Education at the same time,
	// but NOT two Personal loans concurrently.
	// Mirrors M-Shwari / KCB M-PESA multi-product model.
	// Best for: Growing lenders with diversified product lines.
	PolicyPerCategory LoanConcurrencyPolicy = "PER_CATEGORY"

	// PolicyAggregate allows multiple loans up to an aggregate exposure limit.
	// Total outstanding across all loans cannot exceed MaxAggregateCents or
	// a configurable multiplier of the borrower's tier limit.
	// Mirrors traditional bank model (KCB, Equity, Co-op).
	// Best for: Mature lenders with strong underwriting.
	PolicyAggregate LoanConcurrencyPolicy = "AGGREGATE"
)

// IsValid checks if a concurrency policy is recognized.
func (p LoanConcurrencyPolicy) IsValid() bool {
	return p == PolicySingle || p == PolicyPerCategory || p == PolicyAggregate
}

// =============================================================================
// Loan Policy Configuration
// =============================================================================

// LoanPolicyConfig holds the complete configurable lending policy.
// Loaded from environment variables at startup.
type LoanPolicyConfig struct {
	// ConcurrencyPolicy controls how multiple loans are handled.
	// Default: PolicySingle
	ConcurrencyPolicy LoanConcurrencyPolicy

	// MaxConcurrentLoans limits total active loans under AGGREGATE policy.
	// Ignored for SINGLE (always 1) and PER_CATEGORY (1 per category).
	// Default: 3
	MaxConcurrentLoans int

	// MaxAggregateExposureCents limits total outstanding principal across
	// all active loans under AGGREGATE policy (in cents).
	// 0 means no absolute limit (tier limits still apply per-loan).
	// Default: 0 (disabled — use tier multiplier instead)
	MaxAggregateExposureCents int64

	// AggregateExposureMultiplier limits total exposure as a multiple of
	// the borrower's tier max loan amount under AGGREGATE policy.
	// e.g., 2.0 means total outstanding ≤ 2× tier MaxLoanCents.
	// Default: 2.0
	AggregateExposureMultiplier float64

	// CategoryEnabled controls which categories are available for new loans.
	// If empty, all categories are enabled.
	// Default: all categories enabled
	CategoryEnabled map[LoanCategory]bool
}

// DefaultLoanPolicy returns the production-safe default policy (conservative single-loan).
func DefaultLoanPolicy() *LoanPolicyConfig {
	return &LoanPolicyConfig{
		ConcurrencyPolicy:          PolicySingle,
		MaxConcurrentLoans:         3,
		MaxAggregateExposureCents:  0, // No absolute cap — per-tier limits apply
		AggregateExposureMultiplier: 2.0,
		CategoryEnabled: map[LoanCategory]bool{
			LoanCatPersonal:  true,
			LoanCatEmergency: true,
			LoanCatEducation: true,
			LoanCatBusiness:  true,
			LoanCatAsset:     true,
		},
	}
}

// IsCategoryEnabled checks if a specific category is available for lending.
func (p *LoanPolicyConfig) IsCategoryEnabled(cat LoanCategory) bool {
	if len(p.CategoryEnabled) == 0 {
		return true // All enabled by default
	}
	return p.CategoryEnabled[cat]
}

// EnabledCategories returns only the categories currently available.
func (p *LoanPolicyConfig) EnabledCategories() []LoanCategory {
	var cats []LoanCategory
	for _, c := range AllLoanCategories() {
		if p.IsCategoryEnabled(c) {
			cats = append(cats, c)
		}
	}
	return cats
}
