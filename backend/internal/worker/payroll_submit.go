package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// PayrollAutoSubmitJob automatically submits approved payroll runs that haven't been submitted.
type PayrollAutoSubmitJob struct {
	payrollSvc  *service.PayrollService
	payrollRepo repository.PayrollRepository
	logger      *slog.Logger
}

func NewPayrollAutoSubmitJob(svc *service.PayrollService, repo repository.PayrollRepository, logger *slog.Logger) *PayrollAutoSubmitJob {
	return &PayrollAutoSubmitJob{payrollSvc: svc, payrollRepo: repo, logger: logger}
}

func (j *PayrollAutoSubmitJob) AsJob() Job {
	return Job{
		Name:     "payroll_auto_submit",
		Interval: 1 * time.Hour,
		RunFunc:  j.Run,
	}
}

func (j *PayrollAutoSubmitJob) Run(ctx context.Context) error {
	// List all payroll runs (no SACCO filter = all)
	runs, _, err := j.payrollRepo.List(ctx, nil, 1, 1000)
	if err != nil {
		return err
	}

	var submitted int
	for _, run := range runs {
		if run.Status == models.PayrollApproved {
			if _, err := j.payrollSvc.SubmitPayrollRun(ctx, run.ID); err != nil {
				j.logger.Error("auto-submit failed",
					slog.String("run_id", run.ID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}
			submitted++
		}
	}

	j.logger.Info("payroll auto-submit complete", slog.Int("submitted", submitted))
	return nil
}
