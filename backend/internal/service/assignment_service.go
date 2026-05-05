package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// AssignmentService manages crew shift assignments and earnings calculation.
type AssignmentService struct {
	assignmentRepo repository.AssignmentRepository
	earningRepo    repository.EarningRepository
	walletSvc      *WalletService
	notifSvc       *NotificationService
	txMgr          *database.TxManager
	logger         *slog.Logger
}

// NewAssignmentService creates a new AssignmentService.
func NewAssignmentService(
	assignmentRepo repository.AssignmentRepository,
	earningRepo repository.EarningRepository,
	walletSvc *WalletService,
	notifSvc *NotificationService,
	txMgr *database.TxManager,
	logger *slog.Logger,
) *AssignmentService {
	return &AssignmentService{
		assignmentRepo: assignmentRepo,
		earningRepo:    earningRepo,
		walletSvc:      walletSvc,
		notifSvc:       notifSvc,
		txMgr:          txMgr,
		logger:         logger,
	}
}

// CreateAssignmentInput holds the data for creating a work assignment.
type CreateAssignmentInput struct {
	CrewMemberID     uuid.UUID              `json:"crew_member_id" validate:"required"`
	VehicleID        *uuid.UUID             `json:"vehicle_id"`
	OrganizationID          uuid.UUID              `json:"sacco_id" validate:"required"`
	RouteID          *uuid.UUID             `json:"route_id"`
	ShiftDate        time.Time              `json:"shift_date" validate:"required"`
	ShiftStart       time.Time              `json:"shift_start" validate:"required"`
	EarningModel     models.EarningModel    `json:"earning_model" validate:"required"`
	FixedAmountCents int64                  `json:"fixed_amount_cents"`
	CommissionRate   float64                `json:"commission_rate"`
	HybridBaseCents  int64                  `json:"hybrid_base_cents"`
	CommissionBasis  models.CommissionBasis `json:"commission_basis"`
	Notes            string                 `json:"notes"`
	CreatedByID      uuid.UUID              `json:"-"` // Set from JWT claims

	// Generalized fields (Phase C)
	WorkType         models.WorkType `json:"work_type"`
	WorkSite         string          `json:"work_site"`
	ProjectRef       string          `json:"project_ref"`
	HourlyRateCents  int64           `json:"hourly_rate_cents"`
	DailyRateCents   int64           `json:"daily_rate_cents"`
	PerUnitRateCents int64           `json:"per_unit_rate_cents"`
	OvertimeRateCents int64          `json:"overtime_rate_cents"`
	PayScheduleID    *uuid.UUID      `json:"pay_schedule_id"`
}

// CreateAssignment creates a new work assignment after double-booking check.
func (s *AssignmentService) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*models.Assignment, error) {
	// Default work type
	if input.WorkType == "" {
		input.WorkType = models.WorkTypeShift
	}

	// Industry-aware validation (C5)
	if err := s.validateAssignmentInput(input); err != nil {
		return nil, err
	}

	// Guard: prevent double-booking
	hasActive, err := s.assignmentRepo.HasActiveAssignment(ctx, input.CrewMemberID, input.ShiftDate)
	if err != nil {
		return nil, fmt.Errorf("check active assignment: %w", err)
	}
	if hasActive {
		return nil, fmt.Errorf("%w: crew member already has an active assignment on this date", ErrConflict)
	}

	assignment := &models.Assignment{
		CrewMemberID:      input.CrewMemberID,
		VehicleID:         input.VehicleID,
		OrganizationID:           input.OrganizationID,
		RouteID:           input.RouteID,
		ShiftDate:         input.ShiftDate,
		ShiftStart:        input.ShiftStart,
		Status:            models.AssignmentScheduled,
		EarningModel:      input.EarningModel,
		FixedAmountCents:  input.FixedAmountCents,
		CommissionRate:    input.CommissionRate,
		HybridBaseCents:   input.HybridBaseCents,
		CommissionBasis:   input.CommissionBasis,
		Notes:             input.Notes,
		CreatedByID:       input.CreatedByID,
		WorkType:          input.WorkType,
		WorkSite:          input.WorkSite,
		ProjectRef:        input.ProjectRef,
		HourlyRateCents:   input.HourlyRateCents,
		DailyRateCents:    input.DailyRateCents,
		PerUnitRateCents:  input.PerUnitRateCents,
		OvertimeRateCents: input.OvertimeRateCents,
		PayScheduleID:     input.PayScheduleID,
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}

	s.logger.Info("assignment created",
		slog.String("assignment_id", assignment.ID.String()),
		slog.String("crew_member_id", input.CrewMemberID.String()),
		slog.String("work_type", string(input.WorkType)),
		slog.String("shift_date", input.ShiftDate.Format("2006-01-02")),
	)

	return assignment, nil
}

// validateAssignmentInput performs industry-aware validation (C5).
func (s *AssignmentService) validateAssignmentInput(input CreateAssignmentInput) error {
	switch input.WorkType {
	case models.WorkTypeShift:
		// Transport: vehicle_id is required
		if input.VehicleID == nil {
			return fmt.Errorf("%w: vehicle_id is required for SHIFT assignments", ErrValidation)
		}
	case models.WorkTypeDaily, models.WorkTypeHourly, models.WorkTypeTask, models.WorkTypeProject:
		// Non-transport: work_site is required
		if input.WorkSite == "" {
			return fmt.Errorf("%w: work_site is required for %s assignments", ErrValidation, input.WorkType)
		}
	case models.WorkTypeBooking:
		// Facilitator bookings: no special requirements
	default:
		return fmt.Errorf("%w: unknown work_type %q", ErrValidation, input.WorkType)
	}

	// Validate rate fields match earning model
	switch input.EarningModel {
	case models.EarningHourly:
		if input.HourlyRateCents <= 0 {
			return fmt.Errorf("%w: hourly_rate_cents is required for HOURLY earning model", ErrValidation)
		}
	case models.EarningDailyRate:
		if input.DailyRateCents <= 0 {
			return fmt.Errorf("%w: daily_rate_cents is required for DAILY_RATE earning model", ErrValidation)
		}
	case models.EarningPerTask, models.EarningPerPiece:
		if input.PerUnitRateCents <= 0 {
			return fmt.Errorf("%w: per_unit_rate_cents is required for %s earning model", ErrValidation, input.EarningModel)
		}
	case models.EarningFixed:
		if input.FixedAmountCents <= 0 {
			return fmt.Errorf("%w: fixed_amount_cents is required for FIXED earning model", ErrValidation)
		}
	case models.EarningCommission:
		if input.CommissionRate <= 0 {
			return fmt.Errorf("%w: commission_rate is required for COMMISSION earning model", ErrValidation)
		}
	case models.EarningHybrid:
		// Both base and commission needed
	case models.EarningSalary:
		if input.FixedAmountCents <= 0 {
			return fmt.Errorf("%w: fixed_amount_cents is required for SALARY earning model", ErrValidation)
		}
	}

	return nil
}

// CompleteAssignmentInput holds the data for completing a work assignment.
type CompleteAssignmentInput struct {
	TotalRevenueCents int64    `json:"total_revenue_cents"` // For commission/hybrid models
	HoursWorked       *float64 `json:"hours_worked"`        // For hourly models
	UnitsCompleted    *int     `json:"units_completed"`     // For per-task/per-piece models
	OvertimeHours     *float64 `json:"overtime_hours"`      // Optional overtime
}

// CompleteAssignment marks an assignment as COMPLETED and calculates earnings.
// The entire flow (update assignment + create earning + credit wallet) runs
// inside a database transaction to ensure atomicity.
func (s *AssignmentService) CompleteAssignment(ctx context.Context, assignmentID uuid.UUID, input CompleteAssignmentInput) (*models.Earning, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	if assignment.Status != models.AssignmentActive && assignment.Status != models.AssignmentScheduled {
		return nil, fmt.Errorf("%w: assignment is not active/scheduled (status: %s)", ErrValidation, assignment.Status)
	}

	// Update hours/units on assignment
	if input.HoursWorked != nil {
		assignment.HoursWorked = input.HoursWorked
	}
	if input.UnitsCompleted != nil {
		assignment.UnitsCompleted = input.UnitsCompleted
	}
	if input.OvertimeHours != nil {
		assignment.OvertimeHours = input.OvertimeHours
	}

	// Calculate earnings based on model (C6)
	amountCents, earningType, err := s.calculateEarnings(assignment, input)
	if err != nil {
		return nil, err
	}

	if amountCents <= 0 {
		return nil, fmt.Errorf("%w: calculated earnings must be positive (got %d cents)", ErrValidation, amountCents)
	}

	now := time.Now()
	description := s.buildEarningDescription(assignment)
	earning := &models.Earning{
		AssignmentID: assignment.ID,
		CrewMemberID: assignment.CrewMemberID,
		AmountCents:  amountCents,
		Currency:     "KES",
		EarningType:  earningType,
		Description:  description,
		EarnedAt:     now,
	}

	// Wrap the multi-step completion in a transaction
	completeFn := func(txCtx context.Context) error {
		// Mark assignment completed
		assignment.Status = models.AssignmentCompleted
		assignment.ShiftEnd = &now
		if err := s.assignmentRepo.Update(txCtx, assignment); err != nil {
			return fmt.Errorf("update assignment: %w", err)
		}

		// Create earning record
		if err := s.earningRepo.Create(txCtx, earning); err != nil {
			return fmt.Errorf("create earning: %w", err)
		}

		// Credit wallet automatically
		idempotencyKey := fmt.Sprintf("earn-%s", earning.ID.String())
		_, err := s.walletSvc.Credit(txCtx, CreditInput{
			CrewMemberID:   assignment.CrewMemberID,
			AmountCents:    amountCents,
			Category:       models.TxCatEarning,
			IdempotencyKey: idempotencyKey,
			Reference:      earning.ID.String(),
			Description:    earning.Description,
		})
		if err != nil {
			return fmt.Errorf("credit wallet for earning: %w", err)
		}

		return nil
	}

	if s.txMgr != nil {
		if err := s.txMgr.RunInTx(ctx, completeFn); err != nil {
			return nil, err
		}
	} else {
		if err := completeFn(ctx); err != nil {
			return nil, err
		}
	}

	s.logger.Info("assignment completed + earning credited",
		slog.String("assignment_id", assignmentID.String()),
		slog.Int64("earned_cents", amountCents),
		slog.String("earning_model", string(assignment.EarningModel)),
		slog.String("work_type", string(assignment.WorkType)),
	)

	if s.notifSvc != nil {
		body := fmt.Sprintf("Your %s on %s is completed. You earned KES %.2f.",
			s.workTypeLabel(assignment.WorkType),
			assignment.ShiftDate.Format("2006-01-02"),
			float64(amountCents)/100.0)

		// Dispatch notification synchronously — failures are best-effort and logged, not returned.
		if _, err := s.notifSvc.SendToCrewMember(ctx, assignment.CrewMemberID, models.ChannelSMS, "Work Completed", body); err != nil {
			s.logger.Error("failed to dispatch completion SMS", slog.String("error", err.Error()))
		}
	}

	return earning, nil
}

// calculateEarnings computes the earned amount based on the assignment's earning model.
func (s *AssignmentService) calculateEarnings(a *models.Assignment, input CompleteAssignmentInput) (int64, models.EarningType, error) {
	switch a.EarningModel {
	case models.EarningFixed:
		amount := a.FixedAmountCents
		if amount <= 0 && input.TotalRevenueCents > 0 {
			// Admin can override at completion time if not pre-set
			amount = input.TotalRevenueCents
		}
		return amount, models.EarningTypeShiftPay, nil

	case models.EarningCommission:
		amount := int64(float64(input.TotalRevenueCents) * a.CommissionRate)
		return amount, models.EarningTypeCommission, nil

	case models.EarningHybrid:
		commission := int64(float64(input.TotalRevenueCents) * a.CommissionRate)
		return a.HybridBaseCents + commission, models.EarningTypeShiftPay, nil

	case models.EarningHourly:
		hours := float64(0)
		if input.HoursWorked != nil {
			hours = *input.HoursWorked
		} else if a.HoursWorked != nil {
			hours = *a.HoursWorked
		}
		if hours <= 0 {
			return 0, "", fmt.Errorf("%w: hours_worked is required for HOURLY earnings", ErrValidation)
		}
		base := int64(math.Round(hours * float64(a.HourlyRateCents)))
		// Overtime calculation
		var overtime int64
		if input.OvertimeHours != nil && *input.OvertimeHours > 0 && a.OvertimeRateCents > 0 {
			overtime = int64(math.Round(*input.OvertimeHours * float64(a.OvertimeRateCents)))
		}
		return base + overtime, models.EarningTypeHourly, nil

	case models.EarningDailyRate:
		return a.DailyRateCents, models.EarningTypeDailyRate, nil

	case models.EarningPerTask, models.EarningPerPiece:
		units := 0
		if input.UnitsCompleted != nil {
			units = *input.UnitsCompleted
		} else if a.UnitsCompleted != nil {
			units = *a.UnitsCompleted
		}
		if units <= 0 {
			return 0, "", fmt.Errorf("%w: units_completed is required for %s earnings", ErrValidation, a.EarningModel)
		}
		return int64(units) * a.PerUnitRateCents, models.EarningTypeTaskPay, nil

	case models.EarningSalary:
		amount := a.FixedAmountCents
		if amount <= 0 && input.TotalRevenueCents > 0 {
			amount = input.TotalRevenueCents
		}
		return amount, models.EarningTypeSalary, nil

	default:
		return 0, "", fmt.Errorf("%w: unknown earning model %q", ErrValidation, a.EarningModel)
	}
}

// buildEarningDescription creates a human-readable earning description.
func (s *AssignmentService) buildEarningDescription(a *models.Assignment) string {
	label := s.workTypeLabel(a.WorkType)
	base := fmt.Sprintf("%s on %s", label, a.ShiftDate.Format("2006-01-02"))
	if a.WorkSite != "" {
		base += " at " + a.WorkSite
	}
	return base
}

// workTypeLabel returns a human-readable label for work types.
func (s *AssignmentService) workTypeLabel(wt models.WorkType) string {
	switch wt {
	case models.WorkTypeShift:
		return "Shift"
	case models.WorkTypeDaily:
		return "Daily assignment"
	case models.WorkTypeHourly:
		return "Hourly assignment"
	case models.WorkTypeTask:
		return "Task"
	case models.WorkTypeProject:
		return "Project work"
	case models.WorkTypeBooking:
		return "Booking"
	default:
		return "Assignment"
	}
}

// CheckIn records a worker's check-in time for an assignment.
func (s *AssignmentService) CheckIn(ctx context.Context, assignmentID uuid.UUID) (*models.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	if assignment.Status != models.AssignmentScheduled {
		return nil, fmt.Errorf("%w: can only check in to SCHEDULED assignments (status: %s)", ErrValidation, assignment.Status)
	}

	now := time.Now()
	assignment.CheckInAt = &now
	assignment.Status = models.AssignmentActive

	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("check in: %w", err)
	}

	s.logger.Info("check-in recorded",
		slog.String("assignment_id", assignmentID.String()),
		slog.String("check_in_at", now.Format(time.RFC3339)),
	)

	return assignment, nil
}

// CheckOut records a worker's check-out time and auto-calculates hours worked.
func (s *AssignmentService) CheckOut(ctx context.Context, assignmentID uuid.UUID) (*models.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	if assignment.Status != models.AssignmentActive {
		return nil, fmt.Errorf("%w: can only check out of ACTIVE assignments (status: %s)", ErrValidation, assignment.Status)
	}
	if assignment.CheckInAt == nil {
		return nil, fmt.Errorf("%w: cannot check out without a prior check-in", ErrValidation)
	}

	now := time.Now()
	assignment.CheckOutAt = &now

	// Auto-calculate hours worked from check-in/check-out
	duration := now.Sub(*assignment.CheckInAt)
	hours := math.Round(duration.Hours()*100) / 100 // 2 decimal places
	assignment.HoursWorked = &hours

	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("check out: %w", err)
	}

	s.logger.Info("check-out recorded",
		slog.String("assignment_id", assignmentID.String()),
		slog.Float64("hours_worked", hours),
	)

	return assignment, nil
}

// ListAssignments returns filtered, paginated assignments.
func (s *AssignmentService) ListAssignments(ctx context.Context, filter repository.AssignmentFilter, page, perPage int) ([]models.Assignment, int64, error) {
	return s.assignmentRepo.List(ctx, filter, page, perPage)
}

// UpdateAssignmentInput holds partial update data for an assignment.
type UpdateAssignmentInput struct {
	VehicleID        *uuid.UUID
	ShiftDate        *time.Time
	ShiftStart       *time.Time
	EarningModel     *models.EarningModel
	FixedAmountCents *int64
	CommissionRate   *float64
	WorkType         *models.WorkType
	WorkSite         *string
	ProjectRef       *string
	HourlyRateCents  *int64
	Notes            *string
}

// UpdateAssignment applies partial updates to a SCHEDULED or ACTIVE assignment.
func (s *AssignmentService) UpdateAssignment(ctx context.Context, id uuid.UUID, input UpdateAssignmentInput) (*models.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if assignment.Status == models.AssignmentCompleted || assignment.Status == models.AssignmentCancelled {
		return nil, fmt.Errorf("%w: cannot edit %s assignments", ErrValidation, assignment.Status)
	}

	if input.VehicleID != nil {
		assignment.VehicleID = input.VehicleID
	}
	if input.ShiftDate != nil {
		assignment.ShiftDate = *input.ShiftDate
	}
	if input.ShiftStart != nil {
		assignment.ShiftStart = *input.ShiftStart
	}
	if input.EarningModel != nil {
		assignment.EarningModel = *input.EarningModel
	}
	if input.FixedAmountCents != nil {
		assignment.FixedAmountCents = *input.FixedAmountCents
	}
	if input.CommissionRate != nil {
		assignment.CommissionRate = *input.CommissionRate
	}
	if input.WorkType != nil {
		assignment.WorkType = *input.WorkType
	}
	if input.WorkSite != nil {
		assignment.WorkSite = *input.WorkSite
	}
	if input.ProjectRef != nil {
		assignment.ProjectRef = *input.ProjectRef
	}
	if input.HourlyRateCents != nil {
		assignment.HourlyRateCents = *input.HourlyRateCents
	}
	if input.Notes != nil {
		assignment.Notes = *input.Notes
	}

	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("update assignment: %w", err)
	}

	s.logger.Info("assignment updated", slog.String("assignment_id", id.String()))
	return assignment, nil
}

// GetAssignment retrieves a single assignment by ID.
func (s *AssignmentService) GetAssignment(ctx context.Context, id uuid.UUID) (*models.Assignment, error) {
	return s.assignmentRepo.GetByID(ctx, id)
}

// CancelAssignment cancels a pending or assigned shift.
func (s *AssignmentService) CancelAssignment(ctx context.Context, assignmentID uuid.UUID, reason string) (*models.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.Status == models.AssignmentCompleted {
		return nil, ErrInvalidStatus
	}
	assignment.Status = models.AssignmentCancelled
	assignment.Notes = assignment.Notes + " | Cancelled: " + reason
	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("cancel assignment: %w", err)
	}
	s.logger.Info("assignment cancelled",
		slog.String("assignment_id", assignmentID.String()),
		slog.String("reason", reason),
	)
	return assignment, nil
}

// ReassignAssignment reassigns a shift to a different crew member.
func (s *AssignmentService) ReassignAssignment(ctx context.Context, assignmentID, newCrewMemberID uuid.UUID) (*models.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.Status == models.AssignmentCompleted || assignment.Status == models.AssignmentCancelled {
		return nil, ErrInvalidStatus
	}
	oldCrewID := assignment.CrewMemberID
	assignment.CrewMemberID = newCrewMemberID
	assignment.Notes = assignment.Notes + fmt.Sprintf(" | Reassigned from %s", oldCrewID.String())
	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("reassign assignment: %w", err)
	}
	s.logger.Info("assignment reassigned",
		slog.String("assignment_id", assignmentID.String()),
		slog.String("from", oldCrewID.String()),
		slog.String("to", newCrewMemberID.String()),
	)
	return assignment, nil
}

// BulkCreateAssignments creates many assignments in a single operation.
func (s *AssignmentService) BulkCreateAssignments(ctx context.Context, inputs []CreateAssignmentInput) (int, []repository.BulkError, error) {
	assignments := make([]models.Assignment, 0, len(inputs))
	for _, input := range inputs {
		if input.WorkType == "" {
			input.WorkType = models.WorkTypeShift
		}
		assignments = append(assignments, models.Assignment{
			CrewMemberID:      input.CrewMemberID,
			VehicleID:         input.VehicleID,
			OrganizationID:    input.OrganizationID,
			RouteID:           input.RouteID,
			ShiftDate:         input.ShiftDate,
			ShiftStart:        input.ShiftStart,
			Status:            models.AssignmentScheduled,
			EarningModel:      input.EarningModel,
			FixedAmountCents:  input.FixedAmountCents,
			CommissionRate:    input.CommissionRate,
			HybridBaseCents:   input.HybridBaseCents,
			CommissionBasis:   input.CommissionBasis,
			Notes:             input.Notes,
			CreatedByID:       input.CreatedByID,
			WorkType:          input.WorkType,
			WorkSite:          input.WorkSite,
			ProjectRef:        input.ProjectRef,
			HourlyRateCents:   input.HourlyRateCents,
			DailyRateCents:    input.DailyRateCents,
			PerUnitRateCents:  input.PerUnitRateCents,
			OvertimeRateCents: input.OvertimeRateCents,
			PayScheduleID:     input.PayScheduleID,
		})
	}

	created, bulkErrors, err := s.assignmentRepo.BulkCreate(ctx, assignments)
	if err != nil {
		return 0, nil, fmt.Errorf("bulk create assignments: %w", err)
	}

	s.logger.Info("bulk assignments created",
		slog.Int("created", created),
		slog.Int("errors", len(bulkErrors)),
	)

	return created, bulkErrors, nil
}
