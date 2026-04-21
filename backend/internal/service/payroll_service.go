package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payroll"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// PayrollService handles payroll business logic.
type PayrollService struct {
	payrollRepo repository.PayrollRepository
	earningRepo repository.EarningRepository
	rateRepo    repository.StatutoryRateRepository
	crewRepo    repository.CrewRepository
	payrollMgr  *payroll.Manager
	logger      *slog.Logger
}

func NewPayrollService(
	payrollRepo repository.PayrollRepository,
	earningRepo repository.EarningRepository,
	rateRepo repository.StatutoryRateRepository,
	crewRepo repository.CrewRepository,
	payrollMgr *payroll.Manager,
	logger *slog.Logger,
) *PayrollService {
	return &PayrollService{
		payrollRepo: payrollRepo,
		earningRepo: earningRepo,
		rateRepo:    rateRepo,
		crewRepo:    crewRepo,
		payrollMgr:  payrollMgr,
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

// SubmitPayrollRun submits all entries in an approved run to PerPay.
func (s *PayrollService) SubmitPayrollRun(ctx context.Context, runID uuid.UUID) (*models.PayrollRun, error) {
	run, err := s.payrollRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != models.PayrollApproved {
		return nil, fmt.Errorf("run must be APPROVED to submit (current: %s)", run.Status)
	}

	if s.payrollMgr == nil {
		return nil, fmt.Errorf("payroll manager not configured")
	}

	entries, err := s.payrollRepo.GetEntries(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get entries: %w", err)
	}

	// Submit each entry
	for _, entry := range entries {
		crew, err := s.crewRepo.GetByID(ctx, entry.CrewMemberID)
		if err != nil {
			s.logger.Error("failed to get crew member for payroll submission", slog.String("crew_id", entry.CrewMemberID.String()), slog.Any("err", err))
			continue
		}

		req := payroll.SubmitRequest{
			EmployeeID:     crew.ID.String(),
			FullName:       crew.FirstName + " " + crew.LastName,
			EmployeePIN:    crew.NationalID,
			Currency:       run.Currency,
			PayPeriodStart: run.PeriodStart.Format("2006-01-02"),
			PayPeriodEnd:   run.PeriodEnd.Format("2006-01-02"),
			IdempotencyKey: fmt.Sprintf("payroll-%s-%s", runID.String(), crew.ID.String()),
			PayComponents: []payroll.PayComponent{
				{ID: "GROSS", Amount: float64(entry.GrossEarningsCents) / 100.0, Description: "Gross Earnings"},
			},
			Deductions: []payroll.Deduction{
				{ID: "SHA", Amount: float64(entry.SHADeductionCents) / 100.0, Type: "STATUTORY", PreTax: true},
				{ID: "NSSF", Amount: float64(entry.NSSFDeductionCents) / 100.0, Type: "STATUTORY", PreTax: true},
				{ID: "HOUSING_LEVY", Amount: float64(entry.HousingLevyDeductionCents) / 100.0, Type: "STATUTORY", PreTax: true},
			},
		}
		
		_, err = s.payrollMgr.SubmitPayroll(ctx, req)
		if err != nil {
			s.logger.Error("failed to submit payroll to PerPay", slog.String("crew_id", crew.ID.String()), slog.Any("err", err))
			// Continue with other entries despite failure
		}
	}

	now := time.Now()
	run.Status = models.PayrollSubmitted
	run.SubmittedAt = &now
	run.PerpayReference = fmt.Sprintf("run-%s", runID.String()) // Or another bulk reference if supported
	
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update run status: %w", err)
	}

	s.logger.Info("payroll run submitted", slog.String("run_id", runID.String()))
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
