package db

import (
	"context"
	"database/sql"
	"fmt"
)

func runMigrations(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS lake.traces (
			id VARCHAR,
			trace_id VARCHAR NOT NULL,
			span_id VARCHAR NOT NULL,
			span_name VARCHAR,
			model VARCHAR,
			status VARCHAR,
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			cost DECIMAL(10, 6),
			duration_ms INTEGER,
			user_id VARCHAR,
			session_id VARCHAR,
			metadata JSON,
			created_at TIMESTAMP,
			webhook_received_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS lake.metrics_hourly (
			hour TIMESTAMP NOT NULL,
			model VARCHAR NOT NULL,
			user_id VARCHAR NOT NULL,
			request_count INTEGER,
			avg_latency_ms FLOAT,
			p95_latency_ms FLOAT,
			p99_latency_ms FLOAT,
			total_tokens INTEGER,
			total_cost DECIMAL(10, 6),
			error_count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS lake.errors (
			id VARCHAR,
			trace_id VARCHAR,
			error_type VARCHAR,
			error_message TEXT,
			stacktrace TEXT,
			created_at TIMESTAMP
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w\nstatement: %s", err, stmt)
		}
	}

	return nil
}
