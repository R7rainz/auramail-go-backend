package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduler_AddJob(t *testing.T) {
	s := New()

	s.AddJob("test", time.Hour, func(ctx context.Context) error {
		return nil
	})

	if len(s.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(s.jobs))
	}

	if s.jobs[0].Name != "test" {
		t.Errorf("expected job name 'test', got '%s'", s.jobs[0].Name)
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	s := New()

	var count int32
	s.AddJob("counter", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx := context.Background()
	s.Start(ctx)

	// Wait for a few executions
	time.Sleep(50 * time.Millisecond)

	s.Stop()

	finalCount := atomic.LoadInt32(&count)
	if finalCount < 2 {
		t.Errorf("expected at least 2 executions, got %d", finalCount)
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	s := New()

	var count int32
	s.AddJob("counter", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Wait for a few executions
	time.Sleep(30 * time.Millisecond)

	// Cancel context
	cancel()

	// Give time for goroutine to stop
	time.Sleep(20 * time.Millisecond)

	countAfterCancel := atomic.LoadInt32(&count)
	time.Sleep(30 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	// Count shouldn't have increased much after cancellation
	if finalCount > countAfterCancel+1 {
		t.Error("jobs continued running after context cancellation")
	}
}

func TestScheduler_JobError(t *testing.T) {
	s := New()

	var successCount int32
	var errorCount int32

	s.AddJob("error_job", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&errorCount, 1)
		return errors.New("intentional error")
	})

	s.AddJob("success_job", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&successCount, 1)
		return nil
	})

	ctx := context.Background()
	s.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	s.Stop()

	// Both jobs should have been called
	if atomic.LoadInt32(&errorCount) < 2 {
		t.Error("error job should have been called multiple times")
	}
	if atomic.LoadInt32(&successCount) < 2 {
		t.Error("success job should have been called multiple times")
	}
}

func TestScheduler_RunOnce(t *testing.T) {
	s := New()

	var called bool
	s.AddJob("once", time.Hour, func(ctx context.Context) error {
		called = true
		return nil
	})

	ctx := context.Background()
	err := s.RunOnce(ctx, "once")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("job should have been called")
	}
}

func TestScheduler_MultipleStarts(t *testing.T) {
	s := New()

	var count int32
	s.AddJob("counter", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx := context.Background()
	s.Start(ctx)
	s.Start(ctx) // Second start should be ignored
	s.Start(ctx) // Third start should be ignored

	time.Sleep(30 * time.Millisecond)
	s.Stop()

	// Should have only one set of jobs running
	finalCount := atomic.LoadInt32(&count)
	if finalCount > 10 {
		t.Errorf("multiple goroutines may have been started, count: %d", finalCount)
	}
}

func TestEmailSyncJob(t *testing.T) {
	called := false
	job := EmailSyncJob(time.Hour, func(ctx context.Context) error {
		called = true
		return nil
	})

	if job.Name != "email_sync" {
		t.Errorf("expected name 'email_sync', got '%s'", job.Name)
	}

	err := job.Fn(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("function should have been called")
	}
}
