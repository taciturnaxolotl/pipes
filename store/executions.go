package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PipeExecution struct {
	ID             string  `json:"id"`
	PipeID         string  `json:"pipe_id"`
	Status         string  `json:"status"`
	TriggerType    string  `json:"trigger_type"`
	StartedAt      int64   `json:"started_at"`
	CompletedAt    *int64  `json:"completed_at,omitempty"`
	DurationMs     *int64  `json:"duration_ms,omitempty"`
	ItemsProcessed *int    `json:"items_processed,omitempty"`
	ErrorMessage   *string `json:"error_message,omitempty"`
	Metadata       *string `json:"metadata,omitempty"`
}

type ExecutionLog struct {
	ID          string  `json:"id"`
	ExecutionID string  `json:"execution_id"`
	NodeID      string  `json:"node_id"`
	Level       string  `json:"level"`
	Message     string  `json:"message"`
	Timestamp   int64   `json:"timestamp"`
	Metadata    *string `json:"metadata,omitempty"`
}

func (db *DB) CreateExecution(id, pipeID, triggerType string, startedAt int64) error {
	_, err := db.Exec(`
		INSERT INTO pipe_executions (id, pipe_id, status, trigger_type, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, pipeID, "running", triggerType, startedAt)

	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}

	return nil
}

func (db *DB) UpdateExecutionSuccess(id string, completedAt, durationMs int64, itemsProcessed int) error {
	_, err := db.Exec(`
		UPDATE pipe_executions
		SET status = ?, completed_at = ?, duration_ms = ?, items_processed = ?
		WHERE id = ?
	`, "success", completedAt, durationMs, itemsProcessed, id)

	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}

	return nil
}

func (db *DB) UpdateExecutionFailed(id string, completedAt, durationMs int64, errorMessage string) error {
	_, err := db.Exec(`
		UPDATE pipe_executions
		SET status = ?, completed_at = ?, duration_ms = ?, error_message = ?
		WHERE id = ?
	`, "failed", completedAt, durationMs, errorMessage, id)

	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}

	return nil
}

func (db *DB) GetExecution(id string) (*PipeExecution, error) {
	exec := &PipeExecution{}
	var completedAt, durationMs sql.NullInt64
	var itemsProcessed sql.NullInt64
	var errorMessage, metadata sql.NullString

	err := db.QueryRow(`
		SELECT id, pipe_id, status, trigger_type, started_at, completed_at, duration_ms, items_processed, error_message, metadata
		FROM pipe_executions
		WHERE id = ?
	`, id).Scan(&exec.ID, &exec.PipeID, &exec.Status, &exec.TriggerType, &exec.StartedAt, &completedAt, &durationMs, &itemsProcessed, &errorMessage, &metadata)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query execution: %w", err)
	}

	if completedAt.Valid {
		val := completedAt.Int64
		exec.CompletedAt = &val
	}

	if durationMs.Valid {
		val := durationMs.Int64
		exec.DurationMs = &val
	}

	if itemsProcessed.Valid {
		val := int(itemsProcessed.Int64)
		exec.ItemsProcessed = &val
	}

	if errorMessage.Valid {
		exec.ErrorMessage = &errorMessage.String
	}

	if metadata.Valid {
		exec.Metadata = &metadata.String
	}

	return exec, nil
}

func (db *DB) GetPipeExecutions(pipeID string, limit int) ([]*PipeExecution, error) {
	rows, err := db.Query(`
		SELECT id, pipe_id, status, trigger_type, started_at, completed_at, duration_ms, items_processed, error_message, metadata
		FROM pipe_executions
		WHERE pipe_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, pipeID, limit)

	if err != nil {
		return nil, fmt.Errorf("query executions: %w", err)
	}
	defer rows.Close()

	var executions []*PipeExecution
	for rows.Next() {
		exec := &PipeExecution{}
		var completedAt, durationMs sql.NullInt64
		var itemsProcessed sql.NullInt64
		var errorMessage, metadata sql.NullString

		if err := rows.Scan(&exec.ID, &exec.PipeID, &exec.Status, &exec.TriggerType, &exec.StartedAt, &completedAt, &durationMs, &itemsProcessed, &errorMessage, &metadata); err != nil {
			return nil, fmt.Errorf("scan execution: %w", err)
		}

		if completedAt.Valid {
			val := completedAt.Int64
			exec.CompletedAt = &val
		}

		if durationMs.Valid {
			val := durationMs.Int64
			exec.DurationMs = &val
		}

		if itemsProcessed.Valid {
			val := int(itemsProcessed.Int64)
			exec.ItemsProcessed = &val
		}

		if errorMessage.Valid {
			exec.ErrorMessage = &errorMessage.String
		}

		if metadata.Valid {
			exec.Metadata = &metadata.String
		}

		executions = append(executions, exec)
	}

	return executions, nil
}

func (db *DB) LogExecution(executionID, nodeID, level, message string) error {
	logID := uuid.New().String()
	timestamp := time.Now().Unix()

	_, err := db.Exec(`
		INSERT INTO execution_logs (id, execution_id, node_id, level, message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, logID, executionID, nodeID, level, message, timestamp)

	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}

	return nil
}

func (db *DB) LogExecutionWithData(executionID, nodeID, level, message, data string) error {
	logID := uuid.New().String()
	timestamp := time.Now().Unix()

	_, err := db.Exec(`
		INSERT INTO execution_logs (id, execution_id, node_id, level, message, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, logID, executionID, nodeID, level, message, timestamp, data)

	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}

	return nil
}

func (db *DB) GetExecutionLogs(executionID string) ([]*ExecutionLog, error) {
	rows, err := db.Query(`
		SELECT id, execution_id, node_id, level, message, timestamp, metadata
		FROM execution_logs
		WHERE execution_id = ?
		ORDER BY timestamp ASC
	`, executionID)

	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var logs []*ExecutionLog
	for rows.Next() {
		log := &ExecutionLog{}
		var metadata sql.NullString

		if err := rows.Scan(&log.ID, &log.ExecutionID, &log.NodeID, &log.Level, &log.Message, &log.Timestamp, &metadata); err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}

		if metadata.Valid {
			log.Metadata = &metadata.String
		}

		logs = append(logs, log)
	}

	return logs, nil
}
