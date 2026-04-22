package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kibsoft/amy-mis/internal/worker"
)

func TestScheduler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler := worker.NewScheduler(logger)

	var runCount int32

	job := worker.Job{
		Name:     "test_job",
		Interval: 50 * time.Millisecond,
		RunFunc: func(ctx context.Context) error {
			atomic.AddInt32(&runCount, 1)
			return nil
		},
	}

	scheduler.Register(job)

	// Start the scheduler
	scheduler.Start()

	// Wait for a couple of ticks
	time.Sleep(120 * time.Millisecond)

	// Stop the scheduler
	scheduler.Stop()

	// Verify job ran
	count := atomic.LoadInt32(&runCount)
	if count < 2 {
		t.Errorf("expected job to run at least twice, got %d", count)
	}
}

func TestScheduler_ErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler := worker.NewScheduler(logger)

	job := worker.Job{
		Name:     "error_job",
		Interval: 10 * time.Millisecond,
		RunFunc: func(ctx context.Context) error {
			return errors.New("simulated error")
		},
	}

	scheduler.Register(job)
	scheduler.Start()

	time.Sleep(30 * time.Millisecond)
	scheduler.Stop()
	// Should not panic or crash
}
