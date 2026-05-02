package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// PayrollScheduleService manages schedule-aware payroll: period generation,
// earnings aggregation, statutory proration, and per-worker schedule overrides.
type PayrollScheduleService struct {
	payrollRepo    repository.PayrollRepository
	earningRepo    repository.EarningRepository
	rateRepo       repository.StatutoryRateRepository
	membershipRepo repository.MembershipRepository
	scheduleRepo   repository.PayScheduleRepository
	saccoRepo      repository.OrganizationRepository
	crewRepo       repository.CrewRepository
	logger         *slog.Logger
}

// NewPayrollScheduleService creates a new PayrollScheduleService.
func NewPayrollScheduleService(
	payrollRepo repository.PayrollRepository,
	earningRepo repository.EarningRepository,
	rateRepo repository.StatutoryRateRepository,
	membershipRepo repository.MembershipRepository,
	scheduleRepo repository.PayScheduleRepository,
	saccoRepo repository.OrganizationRepository,
	crewRepo repository.CrewRepository,
	logger *slog.Logger,
) *PayrollScheduleService {
	return &PayrollScheduleService{
		payrollRepo:    payrollRepo,
		earningRepo:    earningRepo,
		rateRepo:       rateRepo,
		membershipRepo: membershipRepo,
		scheduleRepo:   scheduleRepo,
		saccoRepo:      saccoRepo,
		crewRepo:       crewRepo,
		logger:         logger,
	}
}

// GeneratePayPeriod calculates the next period window for a schedule
// and creates it if it doesn't already exist.
func (s *PayrollScheduleService) GeneratePayPeriod(ctx context.Context, scheduleID uuid.UUID) (*models.PayPeriod, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	start, end := s.calculatePeriodWindow(schedule, time.Now())

	period := &models.PayPeriod{
		PayScheduleID: schedule.ID,
		OrganizationID:       schedule.OrganizationID,
		PeriodStart:   start,
		PeriodEnd:     end,
		Status:        models.PeriodOpen,
	}

	if err := s.payrollRepo.CreatePayPeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("create pay period: %w", err)
	}

	s.logger.Info("pay period generated",
		slog.String("schedule_id", scheduleID.String()),
		slog.String("frequency", string(schedule.Frequency)),
		slog.String("start", start.Format("2006-01-02")),
		slog.String("end", end.Format("2006-01-02")),
	)

	return period, nil
}

// calculatePeriodWindow determines the start/end dates for the current pay period.
func (s *PayrollScheduleService) calculatePeriodWindow(schedule *models.PaySchedule, now time.Time) (time.Time, time.Time) {
	today := now.Truncate(24 * time.Hour)

	switch schedule.Frequency {
	case models.PayDaily:
		return today, today

	case models.PayWeekly:
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1))
		end := start.AddDate(0, 0, 6)
		return start, end

	case models.PayBiWeekly:
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := today.AddDate(0, 0, -(weekday - 1))
		dayOfYear := today.YearDay()
		if (dayOfYear/7)%2 == 1 {
			start = start.AddDate(0, 0, -7)
		}
		end := start.AddDate(0, 0, 13)
		return start, end

	case models.PayMonthly:
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		end := start.AddDate(0, 1, -1)
		return start, end

	default:
		return today, today
	}
}

// ProcessScheduledPayroll runs payroll for a specific pay period, aggregating
// earnings for all crew members on the matching schedule.
// Implements D4 (schedule-aware), D5 (per-worker overrides), D8 (proration), D10 (exemptions).
func (s *PayrollScheduleService) ProcessScheduledPayroll(ctx context.Context, periodID uuid.UUID, processedByID *uuid.UUID) (*models.PayrollRun, error) {
	period, err := s.payrollRepo.GetPayPeriodByID(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("get period: %w", err)
	}

	if period.Status != models.PeriodOpen && period.Status != models.PeriodClosed {
		return nil, fmt.Errorf("%w: period must be OPEN or CLOSED (status: %s)", ErrValidation, period.Status)
	}

	schedule, err := s.scheduleRepo.GetByID(ctx, period.PayScheduleID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	// Get active statutory rates
	rates, err := s.rateRepo.GetActiveRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rates: %w", err)
	}
	rateLookup := make(map[string]models.StatutoryRate)
	for _, r := range rates {
		rateLookup[r.Name] = r
	}

	// Get SACCO config for statutory exemptions (D10)
	sacco, err := s.saccoRepo.GetByID(ctx, period.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("get sacco: %w", err)
	}
	exemptCodes := make(map[string]bool)
	if cfg, cfgErr := sacco.GetTenantConfig(); cfgErr == nil && cfg != nil {
		for _, code := range cfg.StatutoryExemptions {
			exemptCodes[code] = true
		}
	}

	// Get all active members in this SACCO
	members, _, err := s.membershipRepo.ListByOrganization(ctx, period.OrganizationID, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	// Filter members to those on this schedule (D5: per-worker override)
	var eligibleMembers []models.CrewSACCOMembership
	for _, m := range members {
		if !m.IsActive {
			continue
		}
		memberScheduleID := s.resolveScheduleID(m, schedule)
		if memberScheduleID == period.PayScheduleID {
			eligibleMembers = append(eligibleMembers, m)
		}
	}

	// Aggregate earnings for each eligible member within the period window
	var entries []models.PayrollEntry
	var totalGross, totalDed, totalNet int64

	for _, m := range eligibleMembers {
		crew, err := s.crewRepo.GetByID(ctx, m.CrewMemberID)
		if err != nil {
			s.logger.Error("skip crew member: failed to load", slog.String("crew_id", m.CrewMemberID.String()), slog.Any("err", err))
			continue
		}

		earningFilter := repository.EarningFilter{
			CrewMemberID: &m.CrewMemberID,
			DateFrom:     &period.PeriodStart,
			DateTo:       &period.PeriodEnd,
		}
		earnings, _, err := s.earningRepo.List(ctx, earningFilter, 1, 10000)
		if err != nil {
			s.logger.Error("skip crew: failed to load earnings", slog.String("crew_id", m.CrewMemberID.String()), slog.Any("err", err))
			continue
		}

		var gross int64
		for _, e := range earnings {
			gross += e.AmountCents
		}
		if gross == 0 {
			continue
		}

		// D10: Check statutory exemptions by job type code
		isExempt := s.isStatutoryExempt(crew, exemptCodes)

		// D8: Calculate statutory deductions with proration
		var sha, nssf, housing int64
		if !isExempt {
			prorationFactor := s.prorationFactor(schedule.Frequency)
			sha = prorateDeduction(gross, rateLookup, "SHA", prorationFactor)
			nssf = prorateDeduction(gross, rateLookup, "NSSF", prorationFactor)
			housing = prorateDeduction(gross, rateLookup, "HOUSING_LEVY", prorationFactor)
		}

		ded := sha + nssf + housing
		net := gross - ded

		entries = append(entries, models.PayrollEntry{
			CrewMemberID:              m.CrewMemberID,
			GrossEarningsCents:        gross,
			SHADeductionCents:         sha,
			NSSFDeductionCents:        nssf,
			HousingLevyDeductionCents: housing,
			NetPayCents:               net,
		})

		totalGross += gross
		totalDed += ded
		totalNet += net
	}

	// Create the payroll run linked to the period
	run := &models.PayrollRun{
		OrganizationID:              period.OrganizationID,
		PeriodStart:          period.PeriodStart,
		PeriodEnd:            period.PeriodEnd,
		Status:               models.PayrollProcessing,
		TotalGrossCents:      totalGross,
		TotalDeductionsCents: totalDed,
		TotalNetCents:        totalNet,
		Currency:             "KES",
		ProcessedByID:        processedByID,
		PayScheduleID:        &period.PayScheduleID,
		PayPeriodID:          &period.ID,
	}

	if err := s.payrollRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create payroll run: %w", err)
	}

	for i := range entries {
		entries[i].PayrollRunID = run.ID
	}
	if len(entries) > 0 {
		if err := s.payrollRepo.CreateEntries(ctx, entries); err != nil {
			return nil, fmt.Errorf("create entries: %w", err)
		}
	}

	// Mark period as processing
	period.Status = models.PeriodProcessing
	if err := s.payrollRepo.UpdatePayPeriod(ctx, period); err != nil {
		s.logger.Error("failed to update period status", slog.Any("err", err))
	}

	s.logger.Info("scheduled payroll processed",
		slog.String("run_id", run.ID.String()),
		slog.String("schedule", schedule.Name),
		slog.String("frequency", string(schedule.Frequency)),
		slog.Int("entries", len(entries)),
		slog.Int64("total_gross", totalGross),
		slog.Int64("total_net", totalNet),
	)

	return run, nil
}

// resolveScheduleID determines which pay schedule applies to a member.
func (s *PayrollScheduleService) resolveScheduleID(m models.CrewSACCOMembership, defaultSchedule *models.PaySchedule) uuid.UUID {
	if m.PayScheduleID != nil {
		return *m.PayScheduleID
	}
	return defaultSchedule.ID
}

// isStatutoryExempt checks if a crew member's job type is exempt from statutory deductions.
func (s *PayrollScheduleService) isStatutoryExempt(crew *models.CrewMember, exemptCodes map[string]bool) bool {
	if crew.JobType != nil && exemptCodes[crew.JobType.Code] {
		return true
	}
	return false
}

// prorationFactor returns the monthly-equivalent fraction for a pay frequency.
func (s *PayrollScheduleService) prorationFactor(freq models.PayFrequency) float64 {
	switch freq {
	case models.PayDaily:
		return 1.0 / 22.0
	case models.PayWeekly:
		return 7.0 / 30.0
	case models.PayBiWeekly:
		return 14.0 / 30.0
	case models.PayMonthly:
		return 1.0
	default:
		return 1.0
	}
}

// prorateDeduction calculates a statutory deduction prorated for the pay frequency.
func prorateDeduction(grossCents int64, rates map[string]models.StatutoryRate, name string, prorationFactor float64) int64 {
	rate, ok := rates[name]
	if !ok {
		return 0
	}
	switch rate.RateType {
	case models.RatePercentage:
		return int64(float64(grossCents) * rate.Rate * prorationFactor)
	case models.RateFixed:
		return int64(rate.Rate * 100 * prorationFactor)
	default:
		return 0
	}
}

// ListPayPeriods returns pay periods for a schedule.
func (s *PayrollScheduleService) ListPayPeriods(ctx context.Context, scheduleID uuid.UUID, page, perPage int) ([]models.PayPeriod, int64, error) {
	return s.payrollRepo.ListPayPeriods(ctx, scheduleID, page, perPage)
}

// GetPayPeriod retrieves a single pay period by ID.
func (s *PayrollScheduleService) GetPayPeriod(ctx context.Context, id uuid.UUID) (*models.PayPeriod, error) {
	return s.payrollRepo.GetPayPeriodByID(ctx, id)
}

// ClosePayPeriod marks a period as CLOSED, preventing new earnings from being added.
func (s *PayrollScheduleService) ClosePayPeriod(ctx context.Context, periodID uuid.UUID) (*models.PayPeriod, error) {
	period, err := s.payrollRepo.GetPayPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if period.Status != models.PeriodOpen {
		return nil, fmt.Errorf("%w: can only close OPEN periods", ErrValidation)
	}
	now := time.Now()
	period.Status = models.PeriodClosed
	period.ClosedAt = &now
	if err := s.payrollRepo.UpdatePayPeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("close period: %w", err)
	}
	return period, nil
}
