package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job represents a scheduled task
type Job struct {
	Name     string
	Interval time.Duration
	Fn       func(ctx context.Context) error
}

// Scheduler manages periodic background tasks
type Scheduler struct {
	jobs     []*Job
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// New creates a new scheduler
func New() *Scheduler {
	return &Scheduler{
		jobs:     make([]*Job, 0),
		stopChan: make(chan struct{}),
	}
}

// AddJob registers a new job to be executed periodically
func (s *Scheduler) AddJob(name string, interval time.Duration, fn func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = append(s.jobs, &Job{
		Name:     name,
		Interval: interval,
		Fn:       fn,
	})
}

// Start begins executing all registered jobs
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	for _, job := range s.jobs {
		s.wg.Add(1)
		go s.runJob(ctx, job)
	}

	slog.Info("scheduler started", "jobs", len(s.jobs))
}

// runJob executes a single job on its schedule
func (s *Scheduler) runJob(ctx context.Context, job *Job) {
	defer s.wg.Done()

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	// Run immediately on start
	s.executeJob(ctx, job)

	for {
		select {
		case <-ctx.Done():
			slog.Info("job stopped due to context cancellation", "job", job.Name)
			return
		case <-s.stopChan:
			slog.Info("job stopped", "job", job.Name)
			return
		case <-ticker.C:
			s.executeJob(ctx, job)
		}
	}
}

// executeJob runs the job function with error handling
func (s *Scheduler) executeJob(ctx context.Context, job *Job) {
	start := time.Now()
	slog.Debug("job starting", "job", job.Name)

	if err := job.Fn(ctx); err != nil {
		slog.Error("job failed", "job", job.Name, "error", err, "duration", time.Since(start))
		return
	}

	slog.Debug("job completed", "job", job.Name, "duration", time.Since(start))
}

// Stop gracefully stops all running jobs
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

// RunOnce executes a job immediately without scheduling
func (s *Scheduler) RunOnce(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.Name == name {
			return job.Fn(ctx)
		}
	}
	return nil
}

// Example jobs that could be added:

// EmailSyncJob creates a job that syncs emails for all users periodically
func EmailSyncJob(interval time.Duration, syncFn func(ctx context.Context) error) *Job {
	return &Job{
		Name:     "email_sync",
		Interval: interval,
		Fn:       syncFn,
	}
}

// CacheCleanupJob creates a job that cleans up expired cache entries
func CacheCleanupJob(interval time.Duration, cleanupFn func(ctx context.Context) error) *Job {
	return &Job{
		Name:     "cache_cleanup",
		Interval: interval,
		Fn:       cleanupFn,
	}
}

// TokenRefreshJob creates a job that refreshes OAuth tokens before expiry
func TokenRefreshJob(interval time.Duration, refreshFn func(ctx context.Context) error) *Job {
	return &Job{
		Name:     "token_refresh",
		Interval: interval,
		Fn:       refreshFn,
	}
}
