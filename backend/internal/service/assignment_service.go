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

// AssignmentService manages crew shift assignments and earnings calculation.
type AssignmentService struct {
	assignmentRepo repository.AssignmentRepository
	earningRepo    repository.EarningRepository
	walletSvc      *WalletService
	logger         *slog.Logger
}

// NewAssignmentService creates a new AssignmentService.
func NewAssignmentService(
	assignmentRepo repository.AssignmentRepository,
	earningRepo repository.EarningRepository,
	walletSvc *WalletService,
	logger *slog.Logger,
) *AssignmentService {
	return &AssignmentService{
		assignmentRepo: assignmentRepo,
		earningRepo:    earningRepo,
		walletSvc:      walletSvc,
		logger:         logger,
	}
}

// CreateAssignmentInput holds the data for creating a shift assignment.
type CreateAssignmentInput struct {
	CrewMemberID    uuid.UUID              `json:"crew_member_id" validate:"required"`
	VehicleID       uuid.UUID              `json:"vehicle_id" validate:"required"`
	SaccoID         uuid.UUID              `json:"sacco_id" validate:"required"`
	RouteID         *uuid.UUID             `json:"route_id"`
	ShiftDate       time.Time              `json:"shift_date" validate:"required"`
	ShiftStart      time.Time              `json:"shift_start" validate:"required"`
	EarningModel    models.EarningModel    `json:"earning_model" validate:"required,oneof=FIXED COMMISSION HYBRID"`
	FixedAmountCents int64                 `json:"fixed_amount_cents"`
	CommissionRate  float64                `json:"commission_rate"`
	HybridBaseCents int64                  `json:"hybrid_base_cents"`
	CommissionBasis models.CommissionBasis `json:"commission_basis"`
	Notes           string                 `json:"notes"`
	CreatedByID     uuid.UUID              `json:"-"` // Set from JWT claims
}

// CreateAssignment creates a new shift assignment after double-booking check.
func (s *AssignmentService) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*models.Assignment, error) {
	// Guard: prevent double-booking
	hasActive, err := s.assignmentRepo.HasActiveAssignment(ctx, input.CrewMemberID, input.ShiftDate)
	if err != nil {
		return nil, fmt.Errorf("check active assignment: %w", err)
	}
	if hasActive {
		return nil, fmt.Errorf("%w: crew member already has an active assignment on this date", ErrConflict)
	}

	assignment := &models.Assignment{
		CrewMemberID:     input.CrewMemberID,
		VehicleID:        input.VehicleID,
		SaccoID:          input.SaccoID,
		RouteID:          input.RouteID,
		ShiftDate:        input.ShiftDate,
		ShiftStart:       input.ShiftStart,
		Status:           models.AssignmentScheduled,
		EarningModel:     input.EarningModel,
		FixedAmountCents: input.FixedAmountCents,
		CommissionRate:   input.CommissionRate,
		HybridBaseCents:  input.HybridBaseCents,
		CommissionBasis:  input.CommissionBasis,
		Notes:            input.Notes,
		CreatedByID:      input.CreatedByID,
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}

	s.logger.Info("assignment created",
		slog.String("assignment_id", assignment.ID.String()),
		slog.String("crew_member_id", input.CrewMemberID.String()),
		slog.String("shift_date", input.ShiftDate.Format("2006-01-02")),
	)

	return assignment, nil
}

// CompleteAssignment marks an assignment as COMPLETED and calculates earnings.
func (s *AssignmentService) CompleteAssignment(ctx context.Context, assignmentID uuid.UUID, totalRevenueCents int64) (*models.Earning, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	if assignment.Status != models.AssignmentActive && assignment.Status != models.AssignmentScheduled {
		return nil, fmt.Errorf("%w: assignment is not active/scheduled (status: %s)", ErrValidation, assignment.Status)
	}

	// Calculate earnings based on model
	var amountCents int64
	var earningType models.EarningType

	switch assignment.EarningModel {
	case models.EarningFixed:
		amountCents = assignment.FixedAmountCents
		earningType = models.EarningTypeShiftPay
	case models.EarningCommission:
		amountCents = int64(float64(totalRevenueCents) * assignment.CommissionRate)
		earningType = models.EarningTypeCommission
	case models.EarningHybrid:
		commission := int64(float64(totalRevenueCents) * assignment.CommissionRate)
		amountCents = assignment.HybridBaseCents + commission
		earningType = models.EarningTypeShiftPay
	default:
		return nil, fmt.Errorf("%w: unknown earning model %q", ErrValidation, assignment.EarningModel)
	}

	if amountCents <= 0 {
		return nil, fmt.Errorf("%w: calculated earnings must be positive (got %d cents)", ErrValidation, amountCents)
	}

	// Mark assignment completed
	now := time.Now()
	assignment.Status = models.AssignmentCompleted
	assignment.ShiftEnd = &now
	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("update assignment: %w", err)
	}

	// Create earning record
	earning := &models.Earning{
		AssignmentID: assignment.ID,
		CrewMemberID: assignment.CrewMemberID,
		AmountCents:  amountCents,
		Currency:     "KES",
		EarningType:  earningType,
		Description:  fmt.Sprintf("Shift on %s", assignment.ShiftDate.Format("2006-01-02")),
		EarnedAt:     now,
	}

	if err := s.earningRepo.Create(ctx, earning); err != nil {
		return nil, fmt.Errorf("create earning: %w", err)
	}

	// Credit wallet automatically
	idempotencyKey := fmt.Sprintf("earn-%s", earning.ID.String())
	_, err = s.walletSvc.Credit(ctx, CreditInput{
		CrewMemberID:   assignment.CrewMemberID,
		AmountCents:    amountCents,
		Category:       models.TxCatEarning,
		IdempotencyKey: idempotencyKey,
		Reference:      earning.ID.String(),
		Description:    earning.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("credit wallet for earning: %w", err)
	}

	s.logger.Info("assignment completed + earning credited",
		slog.String("assignment_id", assignmentID.String()),
		slog.Int64("earned_cents", amountCents),
		slog.String("earning_model", string(assignment.EarningModel)),
	)

	return earning, nil
}

// ListAssignments returns filtered, paginated assignments.
func (s *AssignmentService) ListAssignments(ctx context.Context, filter repository.AssignmentFilter, page, perPage int) ([]models.Assignment, int64, error) {
	return s.assignmentRepo.List(ctx, filter, page, perPage)
}

// GetAssignment retrieves a single assignment by ID.
func (s *AssignmentService) GetAssignment(ctx context.Context, id uuid.UUID) (*models.Assignment, error) {
	return s.assignmentRepo.GetByID(ctx, id)
}
