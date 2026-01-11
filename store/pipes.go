package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Pipe struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"`
	IsPublic    bool   `json:"is_public"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type ScheduledJob struct {
	ID             string
	PipeID         string
	CronExpression string
	NextRunAt      int64
	LastRunAt      *int64
	Enabled        bool
	CreatedAt      int64
	UpdatedAt      int64
}

func (db *DB) CreatePipe(userID, name, description, config string, isPublic bool) (*Pipe, error) {
	now := time.Now().Unix()
	pipe := &Pipe{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		Config:      config,
		IsPublic:    isPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := db.Exec(`
		INSERT INTO pipes (id, user_id, name, description, config, is_public, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, pipe.ID, pipe.UserID, pipe.Name, pipe.Description, pipe.Config, btoi(pipe.IsPublic), pipe.CreatedAt, pipe.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert pipe: %w", err)
	}

	return pipe, nil
}

func (db *DB) GetPipe(id string) (*Pipe, error) {
	pipe := &Pipe{}
	var isPublic int

	err := db.QueryRow(`
		SELECT id, user_id, name, description, config, is_public, created_at, updated_at
		FROM pipes
		WHERE id = ?
	`, id).Scan(&pipe.ID, &pipe.UserID, &pipe.Name, &pipe.Description, &pipe.Config, &isPublic, &pipe.CreatedAt, &pipe.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query pipe: %w", err)
	}

	pipe.IsPublic = isPublic == 1
	return pipe, nil
}

func (db *DB) GetUserPipes(userID string) ([]*Pipe, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, description, config, is_public, created_at, updated_at
		FROM pipes
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)

	if err != nil {
		return nil, fmt.Errorf("query pipes: %w", err)
	}
	defer rows.Close()

	var pipes []*Pipe
	for rows.Next() {
		pipe := &Pipe{}
		var isPublic int

		if err := rows.Scan(&pipe.ID, &pipe.UserID, &pipe.Name, &pipe.Description, &pipe.Config, &isPublic, &pipe.CreatedAt, &pipe.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan pipe: %w", err)
		}

		pipe.IsPublic = isPublic == 1
		pipes = append(pipes, pipe)
	}

	return pipes, nil
}

func (db *DB) UpdatePipe(pipe *Pipe) error {
	pipe.UpdatedAt = time.Now().Unix()

	_, err := db.Exec(`
		UPDATE pipes
		SET name = ?, description = ?, config = ?, is_public = ?, updated_at = ?
		WHERE id = ?
	`, pipe.Name, pipe.Description, pipe.Config, btoi(pipe.IsPublic), pipe.UpdatedAt, pipe.ID)

	if err != nil {
		return fmt.Errorf("update pipe: %w", err)
	}

	return nil
}

func (db *DB) DeletePipe(id string) error {
	_, err := db.Exec("DELETE FROM pipes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete pipe: %w", err)
	}
	return nil
}

func (db *DB) CreateScheduledJob(pipeID, cronExpression string, nextRunAt int64) (*ScheduledJob, error) {
	now := time.Now().Unix()
	job := &ScheduledJob{
		ID:             uuid.New().String(),
		PipeID:         pipeID,
		CronExpression: cronExpression,
		NextRunAt:      nextRunAt,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := db.Exec(`
		INSERT INTO scheduled_jobs (id, pipe_id, cron_expression, next_run_at, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.PipeID, job.CronExpression, job.NextRunAt, btoi(job.Enabled), job.CreatedAt, job.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert scheduled job: %w", err)
	}

	return job, nil
}

func (db *DB) GetDueJobs(now int64) ([]*ScheduledJob, error) {
	rows, err := db.Query(`
		SELECT id, pipe_id, cron_expression, next_run_at, last_run_at, enabled, created_at, updated_at
		FROM scheduled_jobs
		WHERE enabled = 1 AND next_run_at <= ?
	`, now)

	if err != nil {
		return nil, fmt.Errorf("query due jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*ScheduledJob
	for rows.Next() {
		job := &ScheduledJob{}
		var enabled int
		var lastRunAt sql.NullInt64

		if err := rows.Scan(&job.ID, &job.PipeID, &job.CronExpression, &job.NextRunAt, &lastRunAt, &enabled, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		job.Enabled = enabled == 1
		if lastRunAt.Valid {
			val := lastRunAt.Int64
			job.LastRunAt = &val
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (db *DB) UpdateJobAfterRun(id string, lastRunAt, nextRunAt int64) error {
	now := time.Now().Unix()

	_, err := db.Exec(`
		UPDATE scheduled_jobs
		SET last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ?
	`, lastRunAt, nextRunAt, now, id)

	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	return nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
