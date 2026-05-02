package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
)

func TestHourlyEarningCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:    crew.ID,
		OrganizationID:         uuid.New(),
		ShiftDate:       time.Now(),
		ShiftStart:      time.Now(),
		EarningModel:    models.EarningHourly,
		HourlyRateCents: 50000, // 500 KES/hr
		WorkType:        models.WorkTypeHourly,
		WorkSite:        "Kiambu Road Site",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	hours := 8.0
	earning, err := svc.CompleteAssignment(ctx, a.ID, CompleteAssignmentInput{HoursWorked: &hours})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	// 8 hours * 500 KES = 4000 KES = 400000 cents
	if earning.AmountCents != 400000 {
		t.Errorf("hourly = %d, want 400000", earning.AmountCents)
	}
	if earning.EarningType != models.EarningTypeHourly {
		t.Errorf("type = %s, want HOURLY_PAY", earning.EarningType)
	}
}

func TestHourlyWithOvertimeCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:      crew.ID,
		OrganizationID:           uuid.New(),
		ShiftDate:         time.Now(),
		ShiftStart:        time.Now(),
		EarningModel:      models.EarningHourly,
		HourlyRateCents:   40000, // 400 KES/hr
		OvertimeRateCents: 60000, // 600 KES/hr (1.5x)
		WorkType:          models.WorkTypeHourly,
		WorkSite:          "Hospital Wing B",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	hours := 8.0
	overtime := 2.0
	earning, err := svc.CompleteAssignment(ctx, a.ID, CompleteAssignmentInput{
		HoursWorked:   &hours,
		OvertimeHours: &overtime,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	// 8h * 400 = 3200 KES + 2h * 600 = 1200 KES = 4400 KES = 440000 cents
	if earning.AmountCents != 440000 {
		t.Errorf("hourly+overtime = %d, want 440000", earning.AmountCents)
	}
}

func TestDailyRateCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:   crew.ID,
		OrganizationID:        uuid.New(),
		ShiftDate:      time.Now(),
		ShiftStart:     time.Now(),
		EarningModel:   models.EarningDailyRate,
		DailyRateCents: 150000, // 1500 KES/day
		WorkType:       models.WorkTypeDaily,
		WorkSite:       "Thika Road Construction",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	earning, err := svc.CompleteAssignment(ctx, a.ID, CompleteAssignmentInput{})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if earning.AmountCents != 150000 {
		t.Errorf("daily = %d, want 150000", earning.AmountCents)
	}
	if earning.EarningType != models.EarningTypeDailyRate {
		t.Errorf("type = %s, want DAILY_RATE", earning.EarningType)
	}
}

func TestPerTaskCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:     crew.ID,
		OrganizationID:          uuid.New(),
		ShiftDate:        time.Now(),
		ShiftStart:       time.Now(),
		EarningModel:     models.EarningPerTask,
		PerUnitRateCents: 20000, // 200 KES/delivery
		WorkType:         models.WorkTypeTask,
		WorkSite:         "Nairobi CBD Deliveries",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	units := 12
	earning, err := svc.CompleteAssignment(ctx, a.ID, CompleteAssignmentInput{UnitsCompleted: &units})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	// 12 * 200 = 2400 KES = 240000 cents
	if earning.AmountCents != 240000 {
		t.Errorf("per-task = %d, want 240000", earning.AmountCents)
	}
	if earning.EarningType != models.EarningTypeTaskPay {
		t.Errorf("type = %s, want TASK_PAY", earning.EarningType)
	}
}

func TestCheckInCheckOut(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:    crew.ID,
		OrganizationID:         uuid.New(),
		ShiftDate:       time.Now(),
		ShiftStart:      time.Now(),
		EarningModel:    models.EarningHourly,
		HourlyRateCents: 30000,
		WorkType:        models.WorkTypeHourly,
		WorkSite:        "Community Health Center",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Check in
	result, err := svc.CheckIn(ctx, a.ID)
	if err != nil {
		t.Fatalf("check-in: %v", err)
	}
	if result.Status != models.AssignmentActive {
		t.Errorf("status after check-in = %s, want ACTIVE", result.Status)
	}
	if result.CheckInAt == nil {
		t.Error("check_in_at should be set")
	}

	// Check out
	result, err = svc.CheckOut(ctx, a.ID)
	if err != nil {
		t.Fatalf("check-out: %v", err)
	}
	if result.CheckOutAt == nil {
		t.Error("check_out_at should be set")
	}
	if result.HoursWorked == nil {
		t.Error("hours_worked should be auto-calculated")
	}
}

func TestValidation_ShiftRequiresVehicle(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	_, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:     crew.ID,
		OrganizationID:          uuid.New(),
		ShiftDate:        time.Now(),
		ShiftStart:       time.Now(),
		EarningModel:     models.EarningFixed,
		FixedAmountCents: 100000,
		WorkType:         models.WorkTypeShift,
		// VehicleID intentionally nil
	})
	if err == nil {
		t.Fatal("expected validation error for SHIFT without vehicle_id")
	}
}

func TestValidation_DailyRequiresWorkSite(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	_, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID:   crew.ID,
		OrganizationID:        uuid.New(),
		ShiftDate:      time.Now(),
		ShiftStart:     time.Now(),
		EarningModel:   models.EarningDailyRate,
		DailyRateCents: 150000,
		WorkType:       models.WorkTypeDaily,
		// WorkSite intentionally empty
	})
	if err == nil {
		t.Fatal("expected validation error for DAILY without work_site")
	}
}

func TestValidation_HourlyRequiresRate(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	_, err := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID: crew.ID,
		OrganizationID:      uuid.New(),
		ShiftDate:    time.Now(),
		ShiftStart:   time.Now(),
		EarningModel: models.EarningHourly,
		WorkType:     models.WorkTypeHourly,
		WorkSite:     "Test Site",
		// HourlyRateCents = 0
	})
	if err == nil {
		t.Fatal("expected validation error for HOURLY without hourly_rate_cents")
	}
}
