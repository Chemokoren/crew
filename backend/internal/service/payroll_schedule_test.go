package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func newScheduleTestEnv() (*PayrollScheduleService, *mock.PayrollRepo, *mock.PayScheduleRepo, *mock.MembershipRepo, *mock.EarningRepo, *mock.CrewRepo) {
	payrollRepo := mock.NewPayrollRepo()
	earningRepo := mock.NewEarningRepo()
	rateRepo := mock.NewStatutoryRateRepo([]models.StatutoryRate{
		{ID: uuid.New(), Name: "SHA", Rate: 0.0275, RateType: models.RatePercentage, IsActive: true},
		{ID: uuid.New(), Name: "NSSF", Rate: 0.06, RateType: models.RatePercentage, IsActive: true},
		{ID: uuid.New(), Name: "HOUSING_LEVY", Rate: 0.015, RateType: models.RatePercentage, IsActive: true},
	})
	membershipRepo := mock.NewMembershipRepo()
	scheduleRepo := mock.NewPayScheduleRepo()
	saccoRepo := mock.NewSACCORepo()
	crewRepo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewPayrollScheduleService(payrollRepo, earningRepo, rateRepo, membershipRepo, scheduleRepo, saccoRepo, crewRepo, logger)
	return svc, payrollRepo, scheduleRepo, membershipRepo, earningRepo, crewRepo
}

func TestGeneratePayPeriod_Daily(t *testing.T) {
	svc, _, scheduleRepo, _, _, _ := newScheduleTestEnv()
	ctx := context.Background()

	orgID := uuid.New()
	schedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Daily Transport", Frequency: models.PayDaily, IsDefault: true, IsActive: true,
	}
	scheduleRepo.Create(ctx, schedule)

	period, err := svc.GeneratePayPeriod(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("generate daily period: %v", err)
	}
	if period.PeriodStart != period.PeriodEnd {
		t.Errorf("daily period should have same start/end, got %s — %s",
			period.PeriodStart.Format("2006-01-02"), period.PeriodEnd.Format("2006-01-02"))
	}
}

func TestGeneratePayPeriod_Weekly(t *testing.T) {
	svc, _, scheduleRepo, _, _, _ := newScheduleTestEnv()
	ctx := context.Background()

	orgID := uuid.New()
	payDay := 5 // Friday
	schedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Weekly Construction", Frequency: models.PayWeekly, PayDay: &payDay, IsDefault: true, IsActive: true,
	}
	scheduleRepo.Create(ctx, schedule)

	period, err := svc.GeneratePayPeriod(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("generate weekly period: %v", err)
	}
	duration := period.PeriodEnd.Sub(period.PeriodStart)
	if duration.Hours() != 144 { // 6 days difference (Mon-Sun)
		t.Errorf("weekly period span = %v hours, want 144", duration.Hours())
	}
}

func TestGeneratePayPeriod_Monthly(t *testing.T) {
	svc, _, scheduleRepo, _, _, _ := newScheduleTestEnv()
	ctx := context.Background()

	orgID := uuid.New()
	payDay := 28
	schedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Monthly Office", Frequency: models.PayMonthly, PayDay: &payDay, IsDefault: true, IsActive: true,
	}
	scheduleRepo.Create(ctx, schedule)

	period, err := svc.GeneratePayPeriod(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("generate monthly period: %v", err)
	}
	// Period should span 1st to last day of current month
	if period.PeriodStart.Day() != 1 {
		t.Errorf("monthly start day = %d, want 1", period.PeriodStart.Day())
	}
}

func TestProcessScheduledPayroll_WithProration(t *testing.T) {
	svc, payrollRepo, scheduleRepo, membershipRepo, earningRepo, crewRepo := newScheduleTestEnv()
	ctx := context.Background()

	orgID := uuid.New()
	schedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Weekly Construction", Frequency: models.PayWeekly, IsDefault: true, IsActive: true,
	}
	scheduleRepo.Create(ctx, schedule)

	// Create a SACCO (needed for config check)
	saccoRepo := mock.NewSACCORepo()
	sacco := &models.SACCO{ID: orgID, Name: "Test SACCO", Currency: "KES", IsActive: true}
	saccoRepo.Create(ctx, sacco)
	// Re-create service with proper saccoRepo
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	rateRepo := mock.NewStatutoryRateRepo([]models.StatutoryRate{
		{ID: uuid.New(), Name: "SHA", Rate: 0.0275, RateType: models.RatePercentage, IsActive: true},
		{ID: uuid.New(), Name: "NSSF", Rate: 0.06, RateType: models.RatePercentage, IsActive: true},
		{ID: uuid.New(), Name: "HOUSING_LEVY", Rate: 0.015, RateType: models.RatePercentage, IsActive: true},
	})
	svc = NewPayrollScheduleService(payrollRepo, earningRepo, rateRepo, membershipRepo, scheduleRepo, saccoRepo, crewRepo, logger)

	// Create crew + membership
	crew := &models.CrewMember{
		CrewID: "CRW-PAY-01", FirstName: "James", LastName: "Mwangi",
		NationalID: "99999999", Role: models.RoleOther, IsActive: true,
	}
	crewRepo.Create(ctx, crew)
	membership := &models.CrewSACCOMembership{
		CrewMemberID: crew.ID, OrganizationID: orgID, IsActive: true, JoinedAt: time.Now(),
		// No PayScheduleID override — uses SACCO default
	}
	membershipRepo.Create(ctx, membership)

	// Create a pay period
	today := time.Now().Truncate(24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()-time.Monday))
	period := &models.PayPeriod{
		PayScheduleID: schedule.ID, OrganizationID: orgID,
		PeriodStart: weekStart, PeriodEnd: weekStart.AddDate(0, 0, 6),
		Status: models.PeriodOpen,
	}
	payrollRepo.CreatePayPeriod(ctx, period)

	// Add earnings within the period
	earningRepo.Create(ctx, &models.Earning{
		CrewMemberID: crew.ID, AmountCents: 500000, Currency: "KES",
		EarningType: models.EarningTypeDailyRate, EarnedAt: weekStart,
	})
	earningRepo.Create(ctx, &models.Earning{
		CrewMemberID: crew.ID, AmountCents: 500000, Currency: "KES",
		EarningType: models.EarningTypeDailyRate, EarnedAt: weekStart.AddDate(0, 0, 1),
	})

	// Process payroll
	run, err := svc.ProcessScheduledPayroll(ctx, period.ID, nil)
	if err != nil {
		t.Fatalf("process scheduled payroll: %v", err)
	}

	if run.TotalGrossCents != 1000000 {
		t.Errorf("gross = %d, want 1000000", run.TotalGrossCents)
	}
	if run.TotalDeductionsCents == 0 {
		t.Error("expected non-zero deductions for non-exempt worker")
	}
	if run.TotalNetCents != run.TotalGrossCents-run.TotalDeductionsCents {
		t.Errorf("net = %d, want gross-deductions = %d", run.TotalNetCents, run.TotalGrossCents-run.TotalDeductionsCents)
	}
	if run.PayScheduleID == nil || *run.PayScheduleID != schedule.ID {
		t.Error("payroll run should be linked to pay schedule")
	}
	if run.PayPeriodID == nil || *run.PayPeriodID != period.ID {
		t.Error("payroll run should be linked to pay period")
	}

	// Verify proration: weekly SHA = gross * 0.0275 * (7/30)
	entries, _ := payrollRepo.GetEntries(ctx, run.ID)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	shaFloat := float64(1000000) * 0.0275 * 7.0 / 30.0
	expectedSHA := int64(shaFloat)
	if entry.SHADeductionCents != expectedSHA {
		t.Errorf("SHA = %d, want %d (prorated for weekly)", entry.SHADeductionCents, expectedSHA)
	}
}

func TestProcessPayroll_PerWorkerScheduleOverride(t *testing.T) {
	svc, payrollRepo, scheduleRepo, membershipRepo, earningRepo, crewRepo := newScheduleTestEnv()
	ctx := context.Background()

	orgID := uuid.New()
	saccoRepo := mock.NewSACCORepo()
	sacco := &models.SACCO{ID: orgID, Name: "Override SACCO", Currency: "KES", IsActive: true}
	saccoRepo.Create(ctx, sacco)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	rateRepo := mock.NewStatutoryRateRepo([]models.StatutoryRate{
		{ID: uuid.New(), Name: "SHA", Rate: 0.0275, RateType: models.RatePercentage, IsActive: true},
	})
	svc = NewPayrollScheduleService(payrollRepo, earningRepo, rateRepo, membershipRepo, scheduleRepo, saccoRepo, crewRepo, logger)

	// Default schedule: WEEKLY
	weeklySchedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Weekly Default", Frequency: models.PayWeekly, IsDefault: true, IsActive: true,
	}
	scheduleRepo.Create(ctx, weeklySchedule)

	// Override schedule: MONTHLY (for office staff)
	monthlySchedule := &models.PaySchedule{
		OrganizationID: orgID, Name: "Monthly Office", Frequency: models.PayMonthly, IsDefault: false, IsActive: true,
	}
	scheduleRepo.Create(ctx, monthlySchedule)

	// Worker 1: on weekly (no override)
	worker1 := &models.CrewMember{CrewID: "CRW-W1", FirstName: "A", LastName: "B", NationalID: "11111111", Role: models.RoleOther, IsActive: true}
	crewRepo.Create(ctx, worker1)
	membershipRepo.Create(ctx, &models.CrewSACCOMembership{
		CrewMemberID: worker1.ID, OrganizationID: orgID, IsActive: true, JoinedAt: time.Now(),
	})

	// Worker 2: overridden to monthly
	worker2 := &models.CrewMember{CrewID: "CRW-W2", FirstName: "C", LastName: "D", NationalID: "22222222", Role: models.RoleOther, IsActive: true}
	crewRepo.Create(ctx, worker2)
	membershipRepo.Create(ctx, &models.CrewSACCOMembership{
		CrewMemberID: worker2.ID, OrganizationID: orgID, IsActive: true, JoinedAt: time.Now(),
		PayScheduleID: &monthlySchedule.ID,
	})

	// Add earnings for both
	today := time.Now().Truncate(24 * time.Hour)
	earningRepo.Create(ctx, &models.Earning{CrewMemberID: worker1.ID, AmountCents: 300000, EarningType: models.EarningTypeDailyRate, EarnedAt: today, Currency: "KES"})
	earningRepo.Create(ctx, &models.Earning{CrewMemberID: worker2.ID, AmountCents: 500000, EarningType: models.EarningTypeSalary, EarnedAt: today, Currency: "KES"})

	// Create weekly period — only worker1 should be included
	weekStart := today.AddDate(0, 0, -int(today.Weekday()-time.Monday))
	weeklyPeriod := &models.PayPeriod{
		PayScheduleID: weeklySchedule.ID, OrganizationID: orgID,
		PeriodStart: weekStart, PeriodEnd: weekStart.AddDate(0, 0, 6), Status: models.PeriodOpen,
	}
	payrollRepo.CreatePayPeriod(ctx, weeklyPeriod)

	run, err := svc.ProcessScheduledPayroll(ctx, weeklyPeriod.ID, nil)
	if err != nil {
		t.Fatalf("process weekly: %v", err)
	}

	entries, _ := payrollRepo.GetEntries(ctx, run.ID)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (worker1 only), got %d", len(entries))
	}
	if entries[0].CrewMemberID != worker1.ID {
		t.Error("expected worker1 in weekly payroll, got different crew member")
	}
}

func TestClosePayPeriod(t *testing.T) {
	svc, payrollRepo, scheduleRepo, _, _, _ := newScheduleTestEnv()
	ctx := context.Background()

	schedule := &models.PaySchedule{OrganizationID: uuid.New(), Name: "Test", Frequency: models.PayDaily, IsActive: true}
	scheduleRepo.Create(ctx, schedule)

	period := &models.PayPeriod{
		PayScheduleID: schedule.ID, OrganizationID: schedule.OrganizationID,
		PeriodStart: time.Now(), PeriodEnd: time.Now(), Status: models.PeriodOpen,
	}
	payrollRepo.CreatePayPeriod(ctx, period)

	closed, err := svc.ClosePayPeriod(ctx, period.ID)
	if err != nil {
		t.Fatalf("close period: %v", err)
	}
	if closed.Status != models.PeriodClosed {
		t.Errorf("status = %s, want CLOSED", closed.Status)
	}
	if closed.ClosedAt == nil {
		t.Error("closed_at should be set")
	}
}
