package engine

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
	"github.com/kierank/pipes/store"
)

type Scheduler struct {
	db       *store.DB
	executor *Executor
	ticker   *time.Ticker
	done     chan struct{}
	logger   *log.Logger
}

func NewScheduler(db *store.DB, logger *log.Logger) *Scheduler {
	return &Scheduler{
		db:       db,
		executor: NewExecutor(db),
		done:     make(chan struct{}),
		logger:   logger,
	}
}

func (s *Scheduler) Start() {
	s.logger.Info("scheduler starting")

	s.ticker = time.NewTicker(1 * time.Minute)

	// Run immediately on start
	go s.tick()

	// Then run every minute
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.tick()
			case <-s.done:
				return
			}
		}
	}()
}

func (s *Scheduler) tick() {
	ctx := context.Background()
	now := time.Now().Unix()

	jobs, err := s.db.GetDueJobs(now)
	if err != nil {
		s.logger.Error("error fetching jobs", "error", err)
		return
	}

	if len(jobs) > 0 {
		s.logger.Info("found jobs to execute", "count", len(jobs))
	}

	for _, job := range jobs {
		if err := s.executeJob(ctx, job); err != nil {
			s.logger.Error("job execution failed", "job_id", job.ID, "error", err)
		}
	}
}

func (s *Scheduler) executeJob(ctx context.Context, job *store.ScheduledJob) error {
	// Execute pipeline
	_, err := s.executor.Execute(ctx, job.PipeID, "scheduled")
	if err != nil {
		s.logger.Error("pipeline execution failed", "pipe_id", job.PipeID, "error", err)
	}

	// Calculate next run time (simplified: add 1 hour for now)
	// In production, use a proper cron parser
	nextRun := time.Now().Add(1 * time.Hour).Unix()

	// Update job
	now := time.Now().Unix()
	return s.db.UpdateJobAfterRun(job.ID, now, nextRun)
}

func (s *Scheduler) Stop() {
	s.logger.Info("scheduler stopping")
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.done)
	s.logger.Info("scheduler stopped")
}
