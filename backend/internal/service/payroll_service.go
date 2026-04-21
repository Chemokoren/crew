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

// PayrollService handles payroll business logic.
type PayrollService struct {
	payrollRepo repository.PayrollRepository
	earningRepo repository.EarningRepository
	rateRepo    repository.StatutoryRateRepository
	logger      *slog.Logger
}

func NewPayrollService(
	payrollRepo repository.PayrollRepository,
	earningRepo repository.EarningRepository,
	rateRepo repository.StatutoryRateRepository,
	logger *slog.Logger,
) *PayrollService {
	return &PayrollService{
		payrollRepo: payrollRepo,
		earningRepo: earningRepo,
		rateRepo:    rateRepo,
		logger:      logger,
	}
}

type CreatePayrollRunInput struct {
	SaccoID       uuid.UUID `json:"sacco_id" binding:"required"`
	PeriodStart   string    `json:"period_start" binding:"required"`
	PeriodEnd     string    `json:"period_end" binding:"required"`
	ProcessedByID uuid.UUID `json:"processed_by_id"`
}

func (s *PayrollService) CreatePayrollRun(ctx context.Context, input CreatePayrollRunInput) (*models.PayrollRun, error) {
	start, err := time.Parse("2006-01-02", input.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start: %w", err)
	}
	end, err := time.Parse("2006-01-02", input.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end: %w", err)
	}

	run := &models.PayrollRun{
		SaccoID:       input.SaccoID,
		PeriodStart:   start,
		PeriodEnd:     end,
		Status:        models.PayrollDraft,
		Currency:      "KES",
		ProcessedByID: input.ProcessedByID,
	}
	if err := s.payrollRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create payroll run: %w", err)
	}
	s.logger.Info("payroll run created", slog.String("id", run.ID.String()))
	return run, nil
}

func (s *PayrollService) GetPayrollRun(ctx context.Context, id uuid.UUID) (*models.PayrollRun, error) {
	return s.payrollRepo.GetByID(ctx, id)
}

func (s *PayrollService) ListPayrollRuns(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.PayrollRun, int64, error) {
	return s.payrollRepo.List(ctx, saccoID, page, perPage)
}

func (s *PayrollService) GetPayrollEntries(ctx context.Context, runID uuid.UUID) ([]models.PayrollEntry, error) {
	return s.payrollRepo.GetEntries(ctx, runID)
}

// ProcessPayrollRun calculates statutory deductions and creates entries.
func (s *PayrollService) ProcessPayrollRun(ctx context.Context, runID uuid.UUID) (*models.PayrollRun, error) {
	run, err := s.payrollRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != models.PayrollDraft {
		return nil, fmt.Errorf("run not in DRAFT status (current: %s)", run.Status)
	}

	rates, err := s.rateRepo.GetActiveRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rates: %w", err)
	}
	rateLookup := make(map[string]models.StatutoryRate)
	for _, r := range rates {
		rateLookup[r.Name] = r
	}

	earnings, _, err := s.earningRepo.List(ctx, repository.EarningFilter{
		DateFrom: &run.PeriodStart, DateTo: &run.PeriodEnd,
	}, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("get earnings: %w", err)
	}

	crewEarnings := make(map[uuid.UUID]int64)
	for _, e := range earnings {
		crewEarnings[e.CrewMemberID] += e.AmountCents
	}

	var entries []models.PayrollEntry
	var totalGross, totalDed, totalNet int64

	for crewID, gross := range crewEarnings {
		sha := calcDeduction(gross, rateLookup, "SHA")
		nssf := calcDeduction(gross, rateLookup, "NSSF")
		housing := calcDeduction(gross, rateLookup, "HOUSING_LEVY")
		ded := sha + nssf + housing
		net := gross - ded

		entries = append(entries, models.PayrollEntry{
			PayrollRunID: runID, CrewMemberID: crewID,
			GrossEarningsCents: gross, SHADeductionCents: sha,
			NSSFDeductionCents: nssf, HousingLevyDeductionCents: housing,
			NetPayCents: net,
		})
		totalGross += gross
		totalDed += ded
		totalNet += net
	}

	if len(entries) > 0 {
		if err := s.payrollRepo.CreateEntries(ctx, entries); err != nil {
			return nil, fmt.Errorf("create entries: %w", err)
		}
	}

	run.Status = models.PayrollProcessing
	run.TotalGrossCents = totalGross
	run.TotalDeductionsCents = totalDed
	run.TotalNetCents = totalNet
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update run: %w", err)
	}

	s.logger.Info("payroll processed", slog.String("run_id", runID.String()), slog.Int("entries", len(entries)))
	return run, nil
}

func (s *PayrollService) ApprovePayrollRun(ctx context.Context, runID, approverID uuid.UUID) (*models.PayrollRun, error) {
	run, err := s.payrollRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != models.PayrollProcessing {
		return nil, fmt.Errorf("run must be PROCESSING to approve (current: %s)", run.Status)
	}
	run.Status = models.PayrollApproved
	run.ApprovedByID = &approverID
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("approve run: %w", err)
	}
	return run, nil
}

func calcDeduction(grossCents int64, rates map[string]models.StatutoryRate, name string) int64 {
	rate, ok := rates[name]
	if !ok {
		return 0
	}
	switch rate.RateType {
	case models.RatePercentage:
		return int64(float64(grossCents) * rate.Rate)
	case models.RateFixed:
		return int64(rate.Rate * 100)
	default:
		return 0
	}
}
