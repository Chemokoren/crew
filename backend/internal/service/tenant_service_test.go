package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func newTestTenantService() (*TenantService, *mock.SACCORepo, *mock.TenantJobTypeRepo, *mock.PayScheduleRepo) {
	saccoRepo := mock.NewSACCORepo()
	jobTypeRepo := mock.NewTenantJobTypeRepo()
	scheduleRepo := mock.NewPayScheduleRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewTenantService(saccoRepo, jobTypeRepo, scheduleRepo, logger)
	return svc, saccoRepo, jobTypeRepo, scheduleRepo
}

func createTestSACCO(t *testing.T, repo *mock.SACCORepo) *models.SACCO {
	t.Helper()
	sacco := &models.SACCO{
		Name:               "Test SACCO",
		RegistrationNumber: "TST-001",
		County:             "Nairobi",
		ContactPhone:       "254700000000",
		Currency:           "KES",
		IsActive:           true,
		IndustryType:       models.IndustryTransport,
	}
	if err := repo.Create(context.Background(), sacco); err != nil {
		t.Fatalf("failed to create test sacco: %v", err)
	}
	return sacco
}

func TestTenantService_UpdateTenantConfig(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	construction := models.IndustryConstruction
	displayName := "Njoro Construction Ltd"

	updated, err := svc.UpdateTenantConfig(ctx, sacco.ID, UpdateTenantConfigInput{
		IndustryType: &construction,
		DisplayName:  &displayName,
	})
	if err != nil {
		t.Fatalf("UpdateTenantConfig failed: %v", err)
	}
	if updated.IndustryType != models.IndustryConstruction {
		t.Errorf("expected industry CONSTRUCTION, got %s", updated.IndustryType)
	}
	if updated.DisplayName != displayName {
		t.Errorf("expected display name %q, got %q", displayName, updated.DisplayName)
	}
}

func TestTenantService_UpdateTenantConfig_InvalidIndustry(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	invalid := models.IndustryType("INVALID")
	_, err := svc.UpdateTenantConfig(ctx, sacco.ID, UpdateTenantConfigInput{
		IndustryType: &invalid,
	})
	if err == nil {
		t.Fatal("expected validation error for invalid industry type")
	}
}

func TestTenantService_CreateJobType(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	jt, err := svc.CreateJobType(ctx, CreateJobTypeInput{
		OrganizationID:     sacco.ID,
		Code:        "mason",
		DisplayName: "Mason",
		Category:    models.JobCategoryPrimary,
		SortOrder:   1,
	})
	if err != nil {
		t.Fatalf("CreateJobType failed: %v", err)
	}
	if jt.Code != "MASON" {
		t.Errorf("expected normalized code MASON, got %s", jt.Code)
	}
	if jt.Category != models.JobCategoryPrimary {
		t.Errorf("expected PRIMARY category, got %s", jt.Category)
	}
}

func TestTenantService_CreateJobType_DuplicateCode(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	input := CreateJobTypeInput{
		OrganizationID:     sacco.ID,
		Code:        "DRIVER",
		DisplayName: "Driver",
		Category:    models.JobCategoryPrimary,
	}
	if _, err := svc.CreateJobType(ctx, input); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.CreateJobType(ctx, input)
	if err == nil {
		t.Fatal("expected conflict error for duplicate code")
	}
}

func TestTenantService_CreateJobType_InvalidSacco(t *testing.T) {
	svc, _, _, _ := newTestTenantService()
	ctx := context.Background()

	_, err := svc.CreateJobType(ctx, CreateJobTypeInput{
		OrganizationID:     uuid.New(),
		Code:        "MASON",
		DisplayName: "Mason",
		Category:    models.JobCategoryPrimary,
	})
	if err == nil {
		t.Fatal("expected not-found error for non-existent sacco")
	}
}

func TestTenantService_ListJobTypes(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	for _, code := range []string{"MASON", "FOREMAN", "LABORER"} {
		if _, err := svc.CreateJobType(ctx, CreateJobTypeInput{
			OrganizationID:     sacco.ID,
			Code:        code,
			DisplayName: code,
			Category:    models.JobCategoryPrimary,
		}); err != nil {
			t.Fatalf("create %s failed: %v", code, err)
		}
	}

	jobTypes, err := svc.ListJobTypes(ctx, sacco.ID)
	if err != nil {
		t.Fatalf("ListJobTypes failed: %v", err)
	}
	if len(jobTypes) != 3 {
		t.Errorf("expected 3 job types, got %d", len(jobTypes))
	}
}

func TestTenantService_UpdateJobType(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	jt, _ := svc.CreateJobType(ctx, CreateJobTypeInput{
		OrganizationID:     sacco.ID,
		Code:        "MASON",
		DisplayName: "Mason",
		Category:    models.JobCategoryPrimary,
	})

	newName := "Senior Mason"
	newCategory := models.JobCategorySupervisor
	updated, err := svc.UpdateJobType(ctx, jt.ID, UpdateJobTypeInput{
		DisplayName: &newName,
		Category:    &newCategory,
	})
	if err != nil {
		t.Fatalf("UpdateJobType failed: %v", err)
	}
	if updated.DisplayName != newName {
		t.Errorf("expected %q, got %q", newName, updated.DisplayName)
	}
	if updated.Category != models.JobCategorySupervisor {
		t.Errorf("expected SUPERVISOR, got %s", updated.Category)
	}
}

func TestTenantService_CreatePaySchedule(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	ps, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID:    sacco.ID,
		Name:       "Weekly Friday",
		Frequency:  models.PayWeekly,
		PayDay:     intPtr(5),
		CutoffHour: 17,
		IsDefault:  true,
	})
	if err != nil {
		t.Fatalf("CreatePaySchedule failed: %v", err)
	}
	if ps.Frequency != models.PayWeekly {
		t.Errorf("expected WEEKLY, got %s", ps.Frequency)
	}
	if *ps.PayDay != 5 {
		t.Errorf("expected pay_day 5, got %d", *ps.PayDay)
	}
}

func TestTenantService_CreatePaySchedule_DailyNoPayDay(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	ps, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID:    sacco.ID,
		Name:       "Daily Payout",
		Frequency:  models.PayDaily,
		CutoffHour: 17,
		IsDefault:  true,
	})
	if err != nil {
		t.Fatalf("CreatePaySchedule daily failed: %v", err)
	}
	if ps.Frequency != models.PayDaily {
		t.Errorf("expected DAILY, got %s", ps.Frequency)
	}
}

func TestTenantService_CreatePaySchedule_WeeklyMissingPayDay(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	_, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID:    sacco.ID,
		Name:       "Weekly No Day",
		Frequency:  models.PayWeekly,
		CutoffHour: 17,
	})
	if err == nil {
		t.Fatal("expected validation error for weekly schedule without pay_day")
	}
}

func TestTenantService_CreatePaySchedule_MonthlyInvalidPayDay(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	_, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID:    sacco.ID,
		Name:       "Monthly 31st",
		Frequency:  models.PayMonthly,
		PayDay:     intPtr(31),
		CutoffHour: 17,
	})
	if err == nil {
		t.Fatal("expected validation error for monthly pay_day > 28")
	}
}

func TestTenantService_ListPaySchedules(t *testing.T) {
	svc, saccoRepo, _, _ := newTestTenantService()
	ctx := context.Background()
	sacco := createTestSACCO(t, saccoRepo)

	if _, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID: sacco.ID, Name: "Daily", Frequency: models.PayDaily, CutoffHour: 17, IsDefault: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePaySchedule(ctx, CreatePayScheduleInput{
		OrganizationID: sacco.ID, Name: "Monthly", Frequency: models.PayMonthly, PayDay: intPtr(25), CutoffHour: 17,
	}); err != nil {
		t.Fatal(err)
	}

	schedules, err := svc.ListPaySchedules(ctx, sacco.ID)
	if err != nil {
		t.Fatalf("ListPaySchedules failed: %v", err)
	}
	if len(schedules) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(schedules))
	}
}

func intPtr(i int) *int { return &i }
