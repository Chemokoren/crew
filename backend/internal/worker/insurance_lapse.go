package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// InsuranceLapseJob checks for expired insurance policies and marks them as lapsed.
type InsuranceLapseJob struct {
	insuranceRepo repository.InsurancePolicyRepository
	logger        *slog.Logger
}

func NewInsuranceLapseJob(repo repository.InsurancePolicyRepository, logger *slog.Logger) *InsuranceLapseJob {
	return &InsuranceLapseJob{insuranceRepo: repo, logger: logger}
}

func (j *InsuranceLapseJob) AsJob() Job {
	return Job{
		Name:     "insurance_lapse_checker",
		Interval: 24 * time.Hour,
		RunFunc:  j.Run,
	}
}

func (j *InsuranceLapseJob) Run(ctx context.Context) error {
	filter := repository.InsurancePolicyFilter{Status: string(models.PolicyActive)}
	policies, _, err := j.insuranceRepo.List(ctx, filter, 1, 10000)
	if err != nil {
		return err
	}

	var lapsed int
	now := time.Now()
	for _, p := range policies {
		if p.EndDate.Before(now) {
			p.Status = models.PolicyLapsed
			if err := j.insuranceRepo.Update(ctx, &p); err != nil {
				j.logger.Error("failed to lapse policy", slog.String("id", p.ID.String()), slog.String("error", err.Error()))
				continue
			}
			lapsed++
		}
	}

	j.logger.Info("insurance lapse check complete", slog.Int("lapsed", lapsed), slog.Int("checked", len(policies)))
	return nil
}
