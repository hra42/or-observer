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

	// Backfill prompt_tokens, completion_tokens, total_tokens, and cost from
	// metadata JSON for rows inserted before the SDK v1.5.2 field extraction fix.
	backfill := []string{
		`UPDATE lake.traces
		 SET prompt_tokens = CAST(json_extract_string(metadata, '$.gen_ai.usage.input_tokens') AS INTEGER)
		 WHERE (prompt_tokens IS NULL OR prompt_tokens = 0)
		   AND json_extract_string(metadata, '$.gen_ai.usage.input_tokens') IS NOT NULL`,

		`UPDATE lake.traces
		 SET completion_tokens = CAST(json_extract_string(metadata, '$.gen_ai.usage.output_tokens') AS INTEGER)
		 WHERE (completion_tokens IS NULL OR completion_tokens = 0)
		   AND json_extract_string(metadata, '$.gen_ai.usage.output_tokens') IS NOT NULL`,

		`UPDATE lake.traces
		 SET total_tokens = CAST(json_extract_string(metadata, '$.gen_ai.usage.input_tokens') AS INTEGER)
		              + CAST(json_extract_string(metadata, '$.gen_ai.usage.output_tokens') AS INTEGER)
		 WHERE (total_tokens IS NULL OR total_tokens = 0)
		   AND json_extract_string(metadata, '$.gen_ai.usage.input_tokens') IS NOT NULL
		   AND json_extract_string(metadata, '$.gen_ai.usage.output_tokens') IS NOT NULL`,

		`UPDATE lake.traces
		 SET cost = CAST(json_extract_string(metadata, '$.gen_ai.usage.total_cost') AS DOUBLE)
		 WHERE (cost IS NULL OR cost = 0)
		   AND json_extract_string(metadata, '$.gen_ai.usage.total_cost') IS NOT NULL`,
	}

	for _, stmt := range backfill {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			// Non-fatal: log but don't block startup if backfill fails
			// (e.g., metadata column doesn't contain expected keys)
			_ = err
		}
	}

	return nil
}
