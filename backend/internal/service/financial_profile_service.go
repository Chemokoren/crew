package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/credit"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// FinancialProfile is a unified, cross-org financial identity for a crew member.
// This is the core output exposed to lenders, insurers, and financial service providers.
type FinancialProfile struct {
	CrewMemberID       uuid.UUID                  `json:"crew_member_id"`
	FullName           string                     `json:"full_name"`
	NationalID         string                     `json:"national_id"`
	KYCStatus          string                     `json:"kyc_status"`
	PrimaryWorkType    string                     `json:"primary_work_type,omitempty"`
	ComputedAt         time.Time                  `json:"computed_at"`

	// Cross-org composite score
	CompositeScore     int                        `json:"composite_score"`
	ScoreGrade         string                     `json:"score_grade"`
	OrgCount           int                        `json:"org_count"`
	CrossOrgTenure     int                        `json:"cross_org_tenure_months"`

	// Per-org breakdown
	OrgProfiles        []OrgProfile               `json:"org_profiles"`

	// Earnings summary
	TotalEarnings30d   int64                      `json:"total_earnings_30d_cents"`
	TotalEarnings90d   int64                      `json:"total_earnings_90d_cents"`
	AvgDailyEarnings   int64                      `json:"avg_daily_earnings_cents"`
	EarningTrend       string                     `json:"earning_trend"`

	// Wallet health
	WalletBalance      int64                      `json:"wallet_balance_cents"`
	SavingsRate        float64                    `json:"savings_rate"`

	// Loan history
	TotalLoansCompleted int                       `json:"total_loans_completed"`
	TotalLoansDefaulted int                       `json:"total_loans_defaulted"`
	OnTimeRepayment     float64                   `json:"on_time_repayment_rate"`

	// Insurance
	ActivePolicies     int                        `json:"active_insurance_policies"`

	// Available products
	AvailableLoanProducts  []LoanProduct           `json:"available_loan_products,omitempty"`
	AvailableInsurance     []InsuranceProduct       `json:"available_insurance,omitempty"`

	// Score factors and suggestions
	Factors            []credit.ScoreFactor        `json:"factors,omitempty"`
	Suggestions        []string                    `json:"suggestions,omitempty"`
}

// OrgProfile represents a worker's profile within a single organization.
type OrgProfile struct {
	OrgID              uuid.UUID    `json:"org_id"`
	OrgName            string       `json:"org_name"`
	Industry           string       `json:"industry"`
	Role               string       `json:"role"`
	JoinedAt           time.Time    `json:"joined_at"`
	TenureMonths       int          `json:"tenure_months"`
	IsActive           bool         `json:"is_active"`
	EarningsCents30d   int64        `json:"earnings_30d_cents"`
	AssignmentCount30d int          `json:"assignment_count_30d"`
}

// LoanProduct represents an industry-specific loan category available to the worker.
type LoanProduct struct {
	Category    string `json:"category"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	MaxAmountKES int64 `json:"max_amount_kes,omitempty"`
}

// InsuranceProduct represents an industry-specific insurance product recommendation.
type InsuranceProduct struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Provider    string `json:"provider,omitempty"`
}

// FinancialProfileService builds cross-org financial profiles for crew members.
// Implements E3 (composite scoring) and E4 (unified API) from the workplan.
type FinancialProfileService struct {
	creditSvc      CreditService
	crewRepo       repository.CrewRepository
	membershipRepo repository.MembershipRepository
	saccoRepo      repository.OrganizationRepository
	earningRepo    repository.EarningRepository
	walletRepo     repository.WalletRepository
	loanRepo       repository.LoanApplicationRepository
	insuranceRepo  repository.InsurancePolicyRepository
	logger         *slog.Logger
}

// NewFinancialProfileService creates a new FinancialProfileService.
func NewFinancialProfileService(
	creditSvc CreditService,
	crewRepo repository.CrewRepository,
	membershipRepo repository.MembershipRepository,
	saccoRepo repository.OrganizationRepository,
	earningRepo repository.EarningRepository,
	walletRepo repository.WalletRepository,
	loanRepo repository.LoanApplicationRepository,
	insuranceRepo repository.InsurancePolicyRepository,
	logger *slog.Logger,
) *FinancialProfileService {
	return &FinancialProfileService{
		creditSvc:      creditSvc,
		crewRepo:       crewRepo,
		membershipRepo: membershipRepo,
		saccoRepo:      saccoRepo,
		earningRepo:    earningRepo,
		walletRepo:     walletRepo,
		loanRepo:       loanRepo,
		insuranceRepo:  insuranceRepo,
		logger:         logger,
	}
}

// GetProfile builds a comprehensive, cross-org financial profile for a crew member.
func (s *FinancialProfileService) GetProfile(ctx context.Context, crewMemberID uuid.UUID) (*FinancialProfile, error) {
	crew, err := s.crewRepo.GetByID(ctx, crewMemberID)
	if err != nil {
		return nil, fmt.Errorf("get crew member: %w", err)
	}

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	profile := &FinancialProfile{
		CrewMemberID: crewMemberID,
		FullName:     crew.FirstName + " " + crew.LastName,
		NationalID:   crew.NationalID,
		KYCStatus:    string(crew.KYCStatus),
		ComputedAt:   now,
	}

	// --- Credit Score ---
	scoreResult, err := s.creditSvc.GetDetailedScore(ctx, crewMemberID)
	if err != nil {
		s.logger.Warn("financial profile: credit score unavailable", slog.Any("err", err))
	} else {
		profile.CompositeScore = scoreResult.Score
		profile.ScoreGrade = scoreResult.Grade
		profile.Factors = scoreResult.Factors
		profile.Suggestions = scoreResult.Suggestions
		if scoreResult.Features != nil {
			profile.PrimaryWorkType = scoreResult.Features.PrimaryWorkType
			profile.OrgCount = scoreResult.Features.OrgCount
			profile.CrossOrgTenure = scoreResult.Features.CrossOrgTenureMonths
		}
	}

	// --- Cross-org profiles ---
	memberships, err := s.membershipRepo.ListByCrewMember(ctx, crewMemberID)
	if err != nil {
		s.logger.Warn("financial profile: membership query failed", slog.Any("err", err))
	} else {
		for _, m := range memberships {
			orgProfile := OrgProfile{
				OrgID:    m.OrganizationID,
				JoinedAt: m.JoinedAt,
				IsActive: m.IsActive,
				Role:     string(m.RoleInOrg),
			}

			// Tenure
			end := now
			if m.LeftAt != nil {
				end = *m.LeftAt
			}
			orgProfile.TenureMonths = int(end.Sub(m.JoinedAt).Hours() / (24 * 30))

			// Org details
			sacco, err := s.saccoRepo.GetByID(ctx, m.OrganizationID)
			if err == nil {
				orgProfile.OrgName = sacco.Name
				orgProfile.Industry = string(sacco.IndustryType)
			}

			// Per-org earnings (30d)
			earnings, _, err := s.earningRepo.List(ctx, repository.EarningFilter{
				CrewMemberID: &crewMemberID,
				DateFrom:     &thirtyDaysAgo,
			}, 1, 10000)
			if err == nil {
				for _, e := range earnings {
					orgProfile.EarningsCents30d += e.AmountCents
					orgProfile.AssignmentCount30d++
				}
			}

			profile.OrgProfiles = append(profile.OrgProfiles, orgProfile)
		}
	}

	// --- Earnings aggregation ---
	isVerified := true
	earnings30, _, _ := s.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &thirtyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	for _, e := range earnings30 {
		profile.TotalEarnings30d += e.AmountCents
	}

	earnings90, _, _ := s.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &ninetyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	for _, e := range earnings90 {
		profile.TotalEarnings90d += e.AmountCents
	}

	activeDays := 22.0
	if profile.TotalEarnings30d > 0 && len(earnings30) > 0 {
		profile.AvgDailyEarnings = profile.TotalEarnings30d / int64(activeDays)
	}

	// Earnings trend
	sixtyDaysAgo := now.AddDate(0, 0, -60)
	earningsPrev, _, _ := s.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &sixtyDaysAgo,
		DateTo:       &thirtyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 10000)
	var prevTotal int64
	for _, e := range earningsPrev {
		prevTotal += e.AmountCents
	}
	if prevTotal > 0 {
		ratio := float64(profile.TotalEarnings30d) / float64(prevTotal)
		if ratio > 1.1 {
			profile.EarningTrend = "GROWING"
		} else if ratio > 0.9 {
			profile.EarningTrend = "STABLE"
		} else {
			profile.EarningTrend = "DECLINING"
		}
	} else if profile.TotalEarnings30d > 0 {
		profile.EarningTrend = "GROWING"
	} else {
		profile.EarningTrend = "STABLE"
	}

	// --- Wallet ---
	wallet, err := s.walletRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err == nil {
		profile.WalletBalance = wallet.BalanceCents
		if wallet.TotalCreditedCents > 0 {
			profile.SavingsRate = float64(wallet.BalanceCents) / float64(wallet.TotalCreditedCents)
		}
	}

	// --- Loans ---
	loans, _, _ := s.loanRepo.List(ctx, repository.LoanApplicationFilter{
		CrewMemberID: &crewMemberID,
	}, 1, 1000)
	for _, l := range loans {
		switch l.Status {
		case models.LoanCompleted:
			profile.TotalLoansCompleted++
		case models.LoanDefaulted:
			profile.TotalLoansDefaulted++
		}
	}
	if profile.TotalLoansCompleted+profile.TotalLoansDefaulted > 0 {
		profile.OnTimeRepayment = float64(profile.TotalLoansCompleted) / float64(profile.TotalLoansCompleted+profile.TotalLoansDefaulted)
	}

	// --- Insurance ---
	policies, _, _ := s.insuranceRepo.List(ctx, repository.InsurancePolicyFilter{
		CrewMemberID: &crewMemberID,
		Status:       "ACTIVE",
	}, 1, 100)
	profile.ActivePolicies = len(policies)

	// --- E5: Industry-specific loan products ---
	profile.AvailableLoanProducts = s.getAvailableLoanProducts(profile)

	// --- E6: Industry-specific insurance recommendations ---
	profile.AvailableInsurance = s.getInsuranceRecommendations(profile)

	return profile, nil
}

// getAvailableLoanProducts returns loan categories relevant to the worker's industry context.
func (s *FinancialProfileService) getAvailableLoanProducts(profile *FinancialProfile) []LoanProduct {
	// Base products available to all workers
	products := []LoanProduct{
		{Category: "PERSONAL", Label: "Personal Loan", Description: "General-purpose personal loan"},
		{Category: "EMERGENCY", Label: "Emergency Loan", Description: "Urgent short-term financial needs"},
	}

	// Industry-specific products based on org profiles
	industries := make(map[string]bool)
	for _, op := range profile.OrgProfiles {
		industries[op.Industry] = true
	}

	if industries["TRANSPORT"] {
		products = append(products, LoanProduct{
			Category:    "ASSET",
			Label:       "Vehicle Maintenance Loan",
			Description: "Cover vehicle repair and maintenance costs",
		})
	}
	if industries["CONSTRUCTION"] {
		products = append(products, LoanProduct{
			Category:    "ASSET",
			Label:       "Tool & Equipment Loan",
			Description: "Purchase or upgrade construction tools and PPE",
		})
		products = append(products, LoanProduct{
			Category:    "BUSINESS",
			Label:       "Site Materials Advance",
			Description: "Working capital for construction materials",
		})
	}
	if industries["HEALTH"] {
		products = append(products, LoanProduct{
			Category:    "EDUCATION",
			Label:       "Professional Development Loan",
			Description: "Certification, training, and upskilling",
		})
	}
	if industries["AGRICULTURE"] {
		products = append(products, LoanProduct{
			Category:    "BUSINESS",
			Label:       "Seasonal Advance",
			Description: "Bridge financing between harvest cycles",
		})
		products = append(products, LoanProduct{
			Category:    "ASSET",
			Label:       "Farm Equipment Loan",
			Description: "Equipment and input financing",
		})
	}

	return products
}

// getInsuranceRecommendations returns industry-specific insurance product recommendations.
func (s *FinancialProfileService) getInsuranceRecommendations(profile *FinancialProfile) []InsuranceProduct {
	var products []InsuranceProduct

	// Universal products
	products = append(products, InsuranceProduct{
		Type:        "PERSONAL_ACCIDENT",
		Label:       "Personal Accident Cover",
		Description: "Protection against accidental injury or death",
	})

	industries := make(map[string]bool)
	for _, op := range profile.OrgProfiles {
		industries[op.Industry] = true
	}

	if industries["TRANSPORT"] {
		products = append(products, InsuranceProduct{
			Type:        "MOTOR_VEHICLE",
			Label:       "Motor Vehicle Insurance",
			Description: "Third-party and comprehensive vehicle cover",
		})
		products = append(products, InsuranceProduct{
			Type:        "PSV_OCCUPATIONAL",
			Label:       "PSV Occupational Cover",
			Description: "Mandatory public service vehicle operator insurance",
		})
	}
	if industries["CONSTRUCTION"] {
		products = append(products, InsuranceProduct{
			Type:        "OCCUPATIONAL_HAZARD",
			Label:       "Construction Occupational Cover",
			Description: "Work injury compensation and occupational disease coverage",
		})
		products = append(products, InsuranceProduct{
			Type:        "TOOL_EQUIPMENT",
			Label:       "Tool & Equipment Insurance",
			Description: "Protection for personal construction tools and equipment",
		})
	}
	if industries["HEALTH"] {
		products = append(products, InsuranceProduct{
			Type:        "PROFESSIONAL_INDEMNITY",
			Label:       "Professional Indemnity",
			Description: "Protection against clinical or professional negligence claims",
		})
		products = append(products, InsuranceProduct{
			Type:        "MEDICAL",
			Label:       "Medical Cover",
			Description: "Comprehensive outpatient and inpatient health insurance",
		})
	}
	if industries["AGRICULTURE"] {
		products = append(products, InsuranceProduct{
			Type:        "CROP_LIVESTOCK",
			Label:       "Crop & Livestock Insurance",
			Description: "Protection against crop failure, disease, and weather events",
		})
	}

	return products
}
