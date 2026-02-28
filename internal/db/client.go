package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// Client wraps a DuckDB connection.
type Client struct {
	db        *sql.DB
	connector *duckdb.Connector
}

// NewClient opens (or creates) the DuckDB file at dbPath and runs migrations.
func NewClient(dbPath string) (*Client, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "parquets"), 0o755); err != nil {
		return nil, fmt.Errorf("create parquets dir: %w", err)
	}

	connector, err := duckdb.NewConnector(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1) // DuckDB embedded: single writer

	// Boot DuckLake
	ctx := context.Background()
	catalogPath := filepath.Join(dataDir, "metadata.ducklake")
	parquetsPath := filepath.Join(dataDir, "parquets") + "/"
	bootQueries := []string{
		"INSTALL ducklake",
		"LOAD ducklake",
		fmt.Sprintf("ATTACH 'ducklake:%s' AS lake (DATA_PATH '%s')", catalogPath, parquetsPath),
	}
	for _, q := range bootQueries {
		if _, err := db.ExecContext(ctx, q); err != nil {
			if !isAlreadyExistsError(err) {
				_ = db.Close()
				return nil, fmt.Errorf("ducklake init: %w", err)
			}
		}
	}

	if err := runMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return &Client{db: db, connector: connector}, nil
}

func isAlreadyExistsError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "already exists") ||
		strings.Contains(s, "already loaded") ||
		strings.Contains(s, "already installed")
}

// InsertTrace inserts a single trace row, ignoring duplicate (trace_id, span_id) pairs.
func (c *Client) InsertTrace(ctx context.Context,
	traceID, spanID, spanName, model string,
	promptTokens, completionTokens, totalTokens int,
	cost float64,
	durationMs int,
	userID, sessionID string,
	metadata []byte,
) error {
	id := traceID + "-" + spanID

	// DuckLake does not support PRIMARY KEY/UNIQUE constraints, so deduplicate manually.
	var exists bool
	err := c.db.QueryRowContext(ctx,
		`SELECT COUNT(*) > 0 FROM lake.traces WHERE id = ?`, id,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if exists {
		return nil
	}

	const query = `
		INSERT INTO lake.traces (
			id, trace_id, span_id, span_name, model,
			prompt_tokens, completion_tokens, total_tokens, cost,
			duration_ms, user_id, session_id, metadata, created_at, webhook_received_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`

	_, err = c.db.ExecContext(ctx, query,
		id, traceID, spanID, spanName, model,
		promptTokens, completionTokens, totalTokens, cost,
		durationMs, userID, sessionID, string(metadata),
	)
	if err != nil {
		return fmt.Errorf("insert trace: %w", err)
	}
	return nil
}

// CountTraces returns the total number of traces in the database.
func (c *Client) CountTraces(ctx context.Context) (int64, error) {
	var count int64
	err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lake.traces`).Scan(&count)
	return count, err
}

// DB returns the underlying *sql.DB for direct queries.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close closes the database connection.
func (c *Client) Close() error {
	return c.db.Close()
}
