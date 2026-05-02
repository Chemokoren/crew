package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/credit"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func TestFinancialProfileService_GetProfile(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	crewRepo := mock.NewCrewRepo()
	membershipRepo := mock.NewMembershipRepo()
	saccoRepo := mock.NewSACCORepo()
	earningRepo := mock.NewEarningRepo()
	walletRepo := mock.NewWalletRepo()
	loanRepo := mock.NewLoanApplicationRepo()
	insuranceRepo := mock.NewInsurancePolicyRepo()

	// Credit scoring infra (simplified — use mock)
	creditScoreRepo := mock.NewCreditScoreRepo()
	historyRepo := mock.NewCreditScoreHistoryRepo()

	// Build the minimal credit service
	creditSvc := &mockCreditService{}

	profileSvc := NewFinancialProfileService(
		creditSvc, crewRepo, membershipRepo, saccoRepo,
		earningRepo, walletRepo, loanRepo, insuranceRepo, logger,
	)

	// Seed data
	crew := &models.CrewMember{
		CrewID: "CRW-PROF-01", FirstName: "Jane", LastName: "Wanjiku",
		NationalID: "33333333", Role: models.RoleOther, IsActive: true,
		KYCStatus: models.KYCVerified,
	}
	crewRepo.Create(ctx, crew)

	saccoTransport := &models.SACCO{
		ID: uuid.New(), Name: "City Express", Currency: "KES",
		IndustryType: models.IndustryTransport, IsActive: true,
	}
	saccoRepo.Create(ctx, saccoTransport)

	saccoConstruction := &models.SACCO{
		ID: uuid.New(), Name: "BuildRight Ltd", Currency: "KES",
		IndustryType: models.IndustryConstruction, IsActive: true,
	}
	saccoRepo.Create(ctx, saccoConstruction)

	// Memberships (cross-org)
	membershipRepo.Create(ctx, &models.CrewSACCOMembership{
		CrewMemberID: crew.ID, OrganizationID: saccoTransport.ID,
		IsActive: true, JoinedAt: time.Now().AddDate(0, -6, 0),
	})
	membershipRepo.Create(ctx, &models.CrewSACCOMembership{
		CrewMemberID: crew.ID, OrganizationID: saccoConstruction.ID,
		IsActive: true, JoinedAt: time.Now().AddDate(0, -3, 0),
	})

	// Earnings
	earningRepo.Create(ctx, &models.Earning{
		CrewMemberID: crew.ID, AmountCents: 500000, Currency: "KES",
		EarningType: models.EarningTypeDailyRate, EarnedAt: time.Now().AddDate(0, 0, -5),
		IsVerified: true,
	})

	profile, err := profileSvc.GetProfile(ctx, crew.ID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	// Verify core fields
	if profile.FullName != "Jane Wanjiku" {
		t.Errorf("name = %s, want Jane Wanjiku", profile.FullName)
	}
	if profile.KYCStatus != "VERIFIED" {
		t.Errorf("kyc = %s, want VERIFIED", profile.KYCStatus)
	}
	if len(profile.OrgProfiles) != 2 {
		t.Errorf("org_profiles = %d, want 2", len(profile.OrgProfiles))
	}

	// Verify industry-specific products
	if len(profile.AvailableLoanProducts) < 3 {
		t.Errorf("expected >= 3 loan products (base + transport + construction), got %d", len(profile.AvailableLoanProducts))
	}
	if len(profile.AvailableInsurance) < 3 {
		t.Errorf("expected >= 3 insurance products (PA + transport + construction), got %d", len(profile.AvailableInsurance))
	}

	// Verify earnings
	if profile.TotalEarnings30d != 500000 {
		t.Errorf("earnings_30d = %d, want 500000", profile.TotalEarnings30d)
	}

	// Ignore unused repos
	_ = creditScoreRepo
	_ = historyRepo
}

// mockCreditService is a minimal mock for tests that don't need real scoring.
type mockCreditService struct{}

func (m *mockCreditService) CalculateScore(_ context.Context, _ uuid.UUID) (*models.CreditScore, error) {
	return &models.CreditScore{Score: 650}, nil
}
func (m *mockCreditService) GetScore(_ context.Context, _ uuid.UUID) (*models.CreditScore, error) {
	return &models.CreditScore{Score: 650}, nil
}
func (m *mockCreditService) GetDetailedScore(_ context.Context, _ uuid.UUID) (*credit.ScoreResult, error) {
	return &credit.ScoreResult{
		Score: 650, Grade: "GOOD", ModelVersion: "mock-v1",
		Features: &credit.FeatureVector{OrgCount: 2, CrossOrgTenureMonths: 9},
	}, nil
}
func (m *mockCreditService) GetScoreHistory(_ context.Context, _ uuid.UUID, _ int) ([]models.CreditScoreHistory, error) {
	return nil, nil
}
