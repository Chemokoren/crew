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
	"github.com/redis/go-redis/v9"
)

func getTestRedis(t *testing.T) *redis.Client {
	opts := &redis.Options{Addr: "localhost:6379"}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("skipping scheduler test: redis not running on localhost:6379")
	}
	client.FlushDB(ctx)
	return client
}

func TestScheduler(t *testing.T) {
	client := getTestRedis(t)
	defer client.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler := worker.NewScheduler(logger, client)

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
	client := getTestRedis(t)
	defer client.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	scheduler := worker.NewScheduler(logger, client)

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
