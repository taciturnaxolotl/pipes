package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable foreign keys and WAL mode
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	store := &DB{DB: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

func (db *DB) initSchema() error {
	schema := `
	-- Users (OAuth profiles)
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		indiko_sub TEXT UNIQUE NOT NULL,
		username TEXT,
		name TEXT,
		email TEXT,
		photo TEXT,
		url TEXT,
		role TEXT NOT NULL DEFAULT 'user',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	-- Sessions (OAuth sessions)
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		access_token TEXT NOT NULL,
		refresh_token TEXT,
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

	-- Pipes (pipeline configurations)
	CREATE TABLE IF NOT EXISTS pipes (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT,
		config TEXT NOT NULL,
		is_public INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_pipes_user_id ON pipes(user_id);

	-- Scheduled jobs
	CREATE TABLE IF NOT EXISTS scheduled_jobs (
		id TEXT PRIMARY KEY,
		pipe_id TEXT NOT NULL UNIQUE REFERENCES pipes(id) ON DELETE CASCADE,
		cron_expression TEXT NOT NULL,
		next_run_at INTEGER NOT NULL,
		last_run_at INTEGER,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_next_run ON scheduled_jobs(next_run_at, enabled);

	-- Execution history
	CREATE TABLE IF NOT EXISTS pipe_executions (
		id TEXT PRIMARY KEY,
		pipe_id TEXT NOT NULL REFERENCES pipes(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		trigger_type TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		completed_at INTEGER,
		duration_ms INTEGER,
		items_processed INTEGER,
		error_message TEXT,
		metadata TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_executions_pipe_id ON pipe_executions(pipe_id);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON pipe_executions(status);

	-- Execution logs (detailed step logs)
	CREATE TABLE IF NOT EXISTS execution_logs (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL REFERENCES pipe_executions(id) ON DELETE CASCADE,
		node_id TEXT NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		metadata TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_logs_execution_id ON execution_logs(execution_id);

	-- Source cache (avoid redundant fetches)
	CREATE TABLE IF NOT EXISTS source_cache (
		id TEXT PRIMARY KEY,
		pipe_id TEXT NOT NULL REFERENCES pipes(id) ON DELETE CASCADE,
		node_id TEXT NOT NULL,
		cache_key TEXT NOT NULL,
		data TEXT NOT NULL,
		etag TEXT,
		last_modified TEXT,
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_cache_pipe_node ON source_cache(pipe_id, node_id);
	CREATE INDEX IF NOT EXISTS idx_cache_expires ON source_cache(expires_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	return nil
}
