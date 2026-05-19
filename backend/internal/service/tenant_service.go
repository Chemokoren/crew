package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// TenantService manages tenant-level configuration: industry type, job types, and pay schedules.
type TenantService struct {
	saccoRepo    repository.OrganizationRepository
	jobTypeRepo  repository.TenantJobTypeRepository
	scheduleRepo repository.PayScheduleRepository
	logger       *slog.Logger
}

// NewTenantService creates a new TenantService.
func NewTenantService(
	saccoRepo repository.OrganizationRepository,
	jobTypeRepo repository.TenantJobTypeRepository,
	scheduleRepo repository.PayScheduleRepository,
	logger *slog.Logger,
) *TenantService {
	return &TenantService{
		saccoRepo:    saccoRepo,
		jobTypeRepo:  jobTypeRepo,
		scheduleRepo: scheduleRepo,
		logger:       logger,
	}
}

// --- Tenant Config ---

// UpdateTenantConfigInput holds data for updating a tenant's industry config.
type UpdateTenantConfigInput struct {
	IndustryType *models.IndustryType `json:"industry_type"`
	DisplayName  *string              `json:"display_name"`
	TenantConfig *models.TenantConfig `json:"tenant_config"`
}

// GetTenantConfig returns the full tenant configuration for a SACCO.
func (s *TenantService) GetTenantConfig(ctx context.Context, orgID uuid.UUID) (*models.SACCO, error) {
	return s.saccoRepo.GetByID(ctx, orgID)
}

// UpdateTenantConfig updates industry type, display name, and/or tenant config.
func (s *TenantService) UpdateTenantConfig(ctx context.Context, orgID uuid.UUID, input UpdateTenantConfigInput) (*models.SACCO, error) {
	sacco, err := s.saccoRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if input.IndustryType != nil {
		if err := validateIndustryType(*input.IndustryType); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		sacco.IndustryType = *input.IndustryType

		// AD-13: Auto-set organization type from industry template
		template := models.GetIndustryTemplate(*input.IndustryType)
		sacco.OrganizationType = template.OrgType

		// Set UI labels in tenant config
		if template.UILabels != nil {
			cfg, _ := sacco.GetTenantConfig()
			if cfg == nil {
				cfg = &models.TenantConfig{}
			}
			cfg.UILabels = template.UILabels
			_ = sacco.SetTenantConfig(cfg)
		}
	}

	if input.DisplayName != nil {
		sacco.DisplayName = *input.DisplayName
	}

	if input.TenantConfig != nil {
		// Merge with existing config so we don't wipe other fields
		existing, _ := sacco.GetTenantConfig()
		if existing == nil {
			existing = &models.TenantConfig{}
		}
		// Apply non-zero fields from input
		incoming := input.TenantConfig
		if incoming.UILabels != nil {
			existing.UILabels = incoming.UILabels
		}
		if incoming.CreditScoringWeights != nil {
			existing.CreditScoringWeights = incoming.CreditScoringWeights
		}
		if incoming.StatutoryExemptions != nil {
			existing.StatutoryExemptions = incoming.StatutoryExemptions
		}
		if len(incoming.KYCVerificationModes) > 0 {
			existing.KYCVerificationModes = incoming.KYCVerificationModes
		}
		if incoming.KYCRestrictedActions != nil {
			existing.KYCRestrictedActions = incoming.KYCRestrictedActions
		}
		if incoming.KYCDocumentTypes != nil {
			existing.KYCDocumentTypes = incoming.KYCDocumentTypes
		}
		// KYCRequired is a bool — always apply from input
		existing.KYCRequired = incoming.KYCRequired

		// Float top-up configuration
		if incoming.TopUpVerificationMode != "" {
			existing.TopUpVerificationMode = incoming.TopUpVerificationMode
		}
		// AllowedTopUpMethods is a slice — always apply from input (nil means "no change", empty means "all allowed")
		if incoming.AllowedTopUpMethods != nil {
			existing.AllowedTopUpMethods = incoming.AllowedTopUpMethods
		}
		// AllowedTopUpChannels — fine-grained channel control (nil means "no change", empty means "all channels")
		if incoming.AllowedTopUpChannels != nil {
			existing.AllowedTopUpChannels = incoming.AllowedTopUpChannels
		}

		// Payroll: statutory deductions — bool, always apply from input
		existing.HandleStatutoryDeductions = incoming.HandleStatutoryDeductions

		// Payroll: non-statutory deduction types
		// nil means "no change"; empty slice explicitly disables all non-statutory deductions
		if incoming.EnabledDeductions != nil {
			existing.EnabledDeductions = incoming.EnabledDeductions
		}
		// nil map means "no change"; non-nil map (even empty) replaces the custom labels
		if incoming.CustomDeductionLabels != nil {
			existing.CustomDeductionLabels = incoming.CustomDeductionLabels
		}

		if err := sacco.SetTenantConfig(existing); err != nil {
			return nil, fmt.Errorf("%w: invalid tenant config", ErrValidation)
		}
	}

	if err := s.saccoRepo.Update(ctx, sacco); err != nil {
		return nil, fmt.Errorf("update tenant config: %w", err)
	}

	s.logger.Info("tenant config updated",
		slog.String("org_id", orgID.String()),
		slog.String("industry_type", string(sacco.IndustryType)),
		slog.String("org_type", string(sacco.OrganizationType)),
	)

	return sacco, nil
}

// BootstrapIndustry seeds default job types, pay schedules, and config from an industry template.
// Implements AD-13: When an org selects an industry type, auto-populate defaults.
// Only seeds if no job types exist yet (avoids overwriting manual customizations).
func (s *TenantService) BootstrapIndustry(ctx context.Context, orgID uuid.UUID, industry models.IndustryType) (*BootstrapResult, error) {
	// Validate org exists
	org, err := s.saccoRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	template := models.GetIndustryTemplate(industry)
	result := &BootstrapResult{}

	// Set industry on org
	org.IndustryType = industry
	org.OrganizationType = template.OrgType
	if template.UILabels != nil {
		cfg, _ := org.GetTenantConfig()
		if cfg == nil {
			cfg = &models.TenantConfig{}
		}
		cfg.UILabels = template.UILabels
		_ = org.SetTenantConfig(cfg)
	}
	if err := s.saccoRepo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("update org: %w", err)
	}
	result.IndustrySet = true

	// Seed job types: deactivate existing ones and seed the new template's defaults.
	// This ensures switching industries replaces the old roles with the correct ones.
	existingJobs, err := s.jobTypeRepo.ListByOrganization(ctx, orgID)
	if err == nil && len(existingJobs) > 0 {
		for _, jt := range existingJobs {
			jt := jt // copy for pointer
			jt.IsActive = false
			if err := s.jobTypeRepo.Update(ctx, &jt); err != nil {
				s.logger.Warn("bootstrap: failed to deactivate old job type",
					slog.String("code", jt.Code), slog.Any("err", err))
			}
		}
		s.logger.Info("bootstrap: deactivated old job types",
			slog.String("org_id", orgID.String()),
			slog.Int("count", len(existingJobs)),
		)
	}
	for i, dj := range template.DefaultJobTypes {
		jt := &models.TenantJobType{
			OrganizationID: orgID,
			Code:           dj.Code,
			DisplayName:    dj.DisplayName,
			Category:       dj.Category,
			IsActive:       true,
			SortOrder:      i,
		}
		// Check if a deactivated one with the same code exists — reactivate instead of creating duplicate
		existing, existErr := s.jobTypeRepo.GetByCode(ctx, orgID, dj.Code)
		if existErr == nil && existing != nil {
			existing.DisplayName = dj.DisplayName
			existing.Category = dj.Category
			existing.IsActive = true
			existing.SortOrder = i
			if err := s.jobTypeRepo.Update(ctx, existing); err != nil {
				s.logger.Warn("bootstrap: failed to reactivate job type",
					slog.String("code", dj.Code), slog.Any("err", err))
				continue
			}
		} else {
			if err := s.jobTypeRepo.Create(ctx, jt); err != nil {
				s.logger.Warn("bootstrap: failed to seed job type",
					slog.String("code", dj.Code), slog.Any("err", err))
				continue
			}
		}
		result.JobTypesSeeded = append(result.JobTypesSeeded, dj.Code)
	}

	// Seed pay schedules (only if none exist)
	existingScheds, err := s.scheduleRepo.ListByOrganization(ctx, orgID)
	if err == nil && len(existingScheds) == 0 {
		for i, freq := range template.Frequencies {
			name := freqDisplayName(freq)
			payDay := freqDefaultPayDay(freq)
			ps := &models.PaySchedule{
				OrganizationID: orgID,
				Name:           name,
				Frequency:      models.PayFrequency(freq),
				PayDay:         payDay,
				CutoffHour:     17,
				IsDefault:      i == 0, // First is default
				IsActive:       true,
			}
			if err := s.scheduleRepo.Create(ctx, ps); err != nil {
				s.logger.Warn("bootstrap: failed to seed pay schedule",
					slog.String("freq", freq), slog.Any("err", err))
				continue
			}
			result.SchedulesSeeded = append(result.SchedulesSeeded, name)
		}
	} else if len(existingScheds) > 0 {
		result.SchedulesSkipped = true
	}

	s.logger.Info("industry bootstrapped",
		slog.String("org_id", orgID.String()),
		slog.String("industry", string(industry)),
		slog.Int("job_types_seeded", len(result.JobTypesSeeded)),
		slog.Int("schedules_seeded", len(result.SchedulesSeeded)),
	)

	return result, nil
}

// BootstrapResult reports what was seeded by BootstrapIndustry.
type BootstrapResult struct {
	IndustrySet       bool     `json:"industry_set"`
	JobTypesSeeded    []string `json:"job_types_seeded"`
	JobTypesSkipped   bool     `json:"job_types_skipped"`
	SchedulesSeeded   []string `json:"schedules_seeded"`
	SchedulesSkipped  bool     `json:"schedules_skipped"`
	ConfigSeeded      bool     `json:"config_seeded"`
}

func freqDisplayName(freq string) string {
	switch freq {
	case "DAILY":
		return "Daily Cash"
	case "WEEKLY":
		return "Weekly Payout"
	case "BI_WEEKLY":
		return "Bi-Weekly Payout"
	case "MONTHLY":
		return "Monthly Salary"
	default:
		return freq + " Payout"
	}
}

func freqDefaultPayDay(freq string) *int {
	switch freq {
	case "WEEKLY", "BI_WEEKLY":
		d := 5 // Friday
		return &d
	case "MONTHLY":
		d := 28
		return &d
	default:
		return nil
	}
}

// --- Job Types ---

// CreateJobTypeInput holds data for creating a new job type.
type CreateJobTypeInput struct {
	OrganizationID     uuid.UUID              `json:"-"`
	Code        string                 `json:"code" validate:"required"`
	DisplayName string                 `json:"display_name" validate:"required"`
	Category    models.JobTypeCategory `json:"category" validate:"required,oneof=PRIMARY FACILITATOR SUPPORT SUPERVISOR"`
	SortOrder   int                    `json:"sort_order"`
}

// CreateJobType creates a new configurable job type for a tenant.
func (s *TenantService) CreateJobType(ctx context.Context, input CreateJobTypeInput) (*models.TenantJobType, error) {
	// Validate SACCO exists
	if _, err := s.saccoRepo.GetByID(ctx, input.OrganizationID); err != nil {
		return nil, err
	}

	// Normalize code
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" {
		return nil, fmt.Errorf("%w: job type code is required", ErrValidation)
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, fmt.Errorf("%w: job type display name is required", ErrValidation)
	}

	// Check for duplicate code within this sacco
	existing, err := s.jobTypeRepo.GetByCode(ctx, input.OrganizationID, code)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: job type code %q already exists for this organization", ErrConflict, code)
	}

	jt := &models.TenantJobType{
		OrganizationID:     input.OrganizationID,
		Code:        code,
		DisplayName: displayName,
		Category:    input.Category,
		IsActive:    true,
		SortOrder:   input.SortOrder,
	}

	if err := s.jobTypeRepo.Create(ctx, jt); err != nil {
		return nil, fmt.Errorf("create job type: %w", err)
	}

	s.logger.Info("job type created",
		slog.String("sacco_id", input.OrganizationID.String()),
		slog.String("code", code),
		slog.String("category", string(input.Category)),
	)

	return jt, nil
}

// UpdateJobTypeInput holds data for updating a job type.
type UpdateJobTypeInput struct {
	DisplayName *string                 `json:"display_name"`
	Category    *models.JobTypeCategory `json:"category"`
	SortOrder   *int                    `json:"sort_order"`
	IsActive    *bool                   `json:"is_active"`
}

// UpdateJobType updates an existing job type.
func (s *TenantService) UpdateJobType(ctx context.Context, id uuid.UUID, input UpdateJobTypeInput) (*models.TenantJobType, error) {
	jt, err := s.jobTypeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil {
		jt.DisplayName = *input.DisplayName
	}
	if input.Category != nil {
		jt.Category = *input.Category
	}
	if input.SortOrder != nil {
		jt.SortOrder = *input.SortOrder
	}
	if input.IsActive != nil {
		jt.IsActive = *input.IsActive
	}

	if err := s.jobTypeRepo.Update(ctx, jt); err != nil {
		return nil, fmt.Errorf("update job type: %w", err)
	}

	return jt, nil
}

// DeleteJobType deletes a job type.
func (s *TenantService) DeleteJobType(ctx context.Context, id uuid.UUID) error {
	return s.jobTypeRepo.Delete(ctx, id)
}

// ListJobTypes lists all active job types for a tenant.
func (s *TenantService) ListJobTypes(ctx context.Context, orgID uuid.UUID) ([]models.TenantJobType, error) {
	return s.jobTypeRepo.ListByOrganization(ctx, orgID)
}

// --- Pay Schedules ---

// CreatePayScheduleInput holds data for creating a new pay schedule.
type CreatePayScheduleInput struct {
	OrganizationID    uuid.UUID          `json:"-"`
	Name       string             `json:"name" validate:"required"`
	Frequency  models.PayFrequency `json:"frequency" validate:"required,oneof=DAILY WEEKLY BI_WEEKLY MONTHLY"`
	PayDay     *int               `json:"pay_day"`
	CutoffHour int                `json:"cutoff_hour"`
	IsDefault  bool               `json:"is_default"`
}

// CreatePaySchedule creates a new pay schedule for a tenant.
func (s *TenantService) CreatePaySchedule(ctx context.Context, input CreatePayScheduleInput) (*models.PaySchedule, error) {
	// Validate SACCO exists
	if _, err := s.saccoRepo.GetByID(ctx, input.OrganizationID); err != nil {
		return nil, err
	}

	if err := validatePaySchedule(input.Frequency, input.PayDay, input.CutoffHour); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	ps := &models.PaySchedule{
		OrganizationID:    input.OrganizationID,
		Name:       strings.TrimSpace(input.Name),
		Frequency:  input.Frequency,
		PayDay:     input.PayDay,
		CutoffHour: input.CutoffHour,
		IsDefault:  input.IsDefault,
		IsActive:   true,
	}

	if err := s.scheduleRepo.Create(ctx, ps); err != nil {
		return nil, fmt.Errorf("create pay schedule: %w", err)
	}

	s.logger.Info("pay schedule created",
		slog.String("sacco_id", input.OrganizationID.String()),
		slog.String("frequency", string(input.Frequency)),
		slog.Bool("is_default", input.IsDefault),
	)

	return ps, nil
}

// UpdatePayScheduleInput holds data for updating a pay schedule.
type UpdatePayScheduleInput struct {
	Name       *string              `json:"name"`
	Frequency  *models.PayFrequency `json:"frequency"`
	PayDay     *int                 `json:"pay_day"`
	CutoffHour *int                 `json:"cutoff_hour"`
	IsDefault  *bool                `json:"is_default"`
	IsActive   *bool                `json:"is_active"`
}

// UpdatePaySchedule updates an existing pay schedule.
func (s *TenantService) UpdatePaySchedule(ctx context.Context, id uuid.UUID, input UpdatePayScheduleInput) (*models.PaySchedule, error) {
	ps, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		ps.Name = *input.Name
	}
	if input.Frequency != nil {
		ps.Frequency = *input.Frequency
	}
	if input.PayDay != nil {
		ps.PayDay = input.PayDay
	}
	if input.CutoffHour != nil {
		ps.CutoffHour = *input.CutoffHour
	}
	if input.IsDefault != nil {
		ps.IsDefault = *input.IsDefault
	}
	if input.IsActive != nil {
		ps.IsActive = *input.IsActive
	}

	// Re-validate after updates
	freq := ps.Frequency
	if err := validatePaySchedule(freq, ps.PayDay, ps.CutoffHour); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	if err := s.scheduleRepo.Update(ctx, ps); err != nil {
		return nil, fmt.Errorf("update pay schedule: %w", err)
	}

	return ps, nil
}

// DeletePaySchedule deletes a pay schedule.
func (s *TenantService) DeletePaySchedule(ctx context.Context, id uuid.UUID) error {
	return s.scheduleRepo.Delete(ctx, id)
}

// ListPaySchedules lists all active pay schedules for a tenant.
func (s *TenantService) ListPaySchedules(ctx context.Context, orgID uuid.UUID) ([]models.PaySchedule, error) {
	return s.scheduleRepo.ListByOrganization(ctx, orgID)
}

// GetDefaultPaySchedule returns the default pay schedule for a tenant.
func (s *TenantService) GetDefaultPaySchedule(ctx context.Context, orgID uuid.UUID) (*models.PaySchedule, error) {
	return s.scheduleRepo.GetDefault(ctx, orgID)
}

// --- Validators ---

func validateIndustryType(it models.IndustryType) error {
	switch it {
	case models.IndustryTransport, models.IndustryConstruction, models.IndustryHealth,
		models.IndustryLogistics, models.IndustryAgriculture, models.IndustryHospitality,
		models.IndustryGeneral, models.IndustryCustom:
		return nil
	default:
		return fmt.Errorf("invalid industry type %q", it)
	}
}

func validatePaySchedule(freq models.PayFrequency, payDay *int, cutoffHour int) error {
	if cutoffHour < 0 || cutoffHour > 23 {
		return fmt.Errorf("cutoff_hour must be between 0 and 23")
	}

	switch freq {
	case models.PayDaily:
		// No pay_day needed
	case models.PayWeekly, models.PayBiWeekly:
		if payDay == nil {
			return fmt.Errorf("pay_day is required for %s frequency (1=Mon..7=Sun)", freq)
		}
		if *payDay < 1 || *payDay > 7 {
			return fmt.Errorf("pay_day must be 1-7 for weekly/bi-weekly frequency")
		}
	case models.PayMonthly:
		if payDay == nil {
			return fmt.Errorf("pay_day is required for monthly frequency (1-28)")
		}
		if *payDay < 1 || *payDay > 28 {
			return fmt.Errorf("pay_day must be 1-28 for monthly frequency")
		}
	default:
		return fmt.Errorf("invalid pay frequency %q", freq)
	}

	return nil
}
