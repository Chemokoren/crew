// Package worker provides background job scheduling for AMY MIS.
// Uses Go's standard library (goroutines + tickers) for lightweight scheduling.
// Designed to be upgraded to Asynq when persistent job queues are needed.
package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Job represents a named background task that runs on a schedule.
type Job struct {
	Name     string
	Interval time.Duration
	RunFunc  func(ctx context.Context) error
}

// Scheduler manages periodic background jobs with graceful shutdown.
type Scheduler struct {
	jobs   []Job
	logger *slog.Logger
	redis  *redis.Client
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewScheduler creates a new background job scheduler.
func NewScheduler(logger *slog.Logger, redisClient *redis.Client) *Scheduler {
	return &Scheduler{
		logger: logger,
		redis:  redisClient,
	}
}

// Register adds a job to the scheduler. Must be called before Start.
func (s *Scheduler) Register(job Job) {
	s.jobs = append(s.jobs, job)
}

// Start begins running all registered jobs on their configured intervals.
// Each job runs in its own goroutine. The first execution is delayed by
// the configured interval (no immediate run on startup).
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	for _, job := range s.jobs {
		s.wg.Add(1)
		go s.runJob(ctx, job)
	}

	s.logger.Info("worker scheduler started",
		slog.Int("job_count", len(s.jobs)),
	)
}

// Stop gracefully shuts down all running jobs and waits for them to finish.
// Should be called during application shutdown.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.logger.Info("worker scheduler stopped")
}

// runJob executes a single job on a recurring timer.
func (s *Scheduler) runJob(ctx context.Context, job Job) {
	defer s.wg.Done()

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	s.logger.Info("job registered",
		slog.String("job", job.Name),
		slog.Duration("interval", job.Interval),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("job stopping", slog.String("job", job.Name))
			return
		case <-ticker.C:
			// Attempt to acquire distributed lock for this job run
			lockKey := "worker_lock:" + job.Name
			// Lock expires slightly before the next interval to allow the next run
			lockDuration := job.Interval - (1 * time.Second)
			if lockDuration <= 0 {
				lockDuration = job.Interval / 2
			}

			acquired, err := s.redis.SetNX(ctx, lockKey, time.Now().String(), lockDuration).Result()
			if err != nil || !acquired {
				s.logger.Debug("job skipped (lock not acquired)", slog.String("job", job.Name))
				continue // Another instance is running it
			}

			start := time.Now()
			s.logger.Info("job started", slog.String("job", job.Name))

			if err := job.RunFunc(ctx); err != nil {
				s.logger.Error("job failed",
					slog.String("job", job.Name),
					slog.Duration("duration", time.Since(start)),
					slog.String("error", err.Error()),
				)
			} else {
				s.logger.Info("job completed",
					slog.String("job", job.Name),
					slog.Duration("duration", time.Since(start)),
				)
			}
		}
	}
}
