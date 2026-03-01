package worker_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hra42/or-observer/internal/db"
	"github.com/hra42/or-observer/internal/worker"
	"go.uber.org/zap"
)

func newTestClient(t *testing.T) *db.Client {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	client, err := db.NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func insertTestTrace(t *testing.T, client *db.Client, traceID, spanID, model, userID string, cost float64) {
	t.Helper()
	ctx := context.Background()
	err := client.InsertTrace(ctx, traceID, spanID, "test-span", model, 100, 50, 150, cost, 500, userID, "", []byte(`{}`))
	if err != nil {
		t.Fatalf("InsertTrace: %v", err)
	}
}

func TestAggregateHourly(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "t1", "s1", "gpt-4", "alice", 0.05)
	insertTestTrace(t, client, "t2", "s2", "gpt-4", "bob", 0.10)
	insertTestTrace(t, client, "t3", "s3", "claude-3", "alice", 0.03)

	log := zap.NewNop()
	w := worker.New(client.DB(), log, time.Hour, 30*24*time.Hour)

	ctx := context.Background()
	w.AggregateHourly(ctx)

	// Verify metrics_hourly was populated.
	var count int
	err := client.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM lake.metrics_hourly`).Scan(&count)
	if err != nil {
		t.Fatalf("count metrics: %v", err)
	}
	if count == 0 {
		t.Error("expected metrics_hourly to have rows after aggregation")
	}

	// Verify it's idempotent: run again, should not double-count.
	w.AggregateHourly(ctx)
	var count2 int
	err = client.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM lake.metrics_hourly`).Scan(&count2)
	if err != nil {
		t.Fatalf("count metrics: %v", err)
	}
	if count2 != count {
		t.Errorf("expected %d rows after re-aggregation, got %d", count, count2)
	}
}

func TestPurgeOldTraces(t *testing.T) {
	client := newTestClient(t)
	ctx := context.Background()

	// Insert a trace, then artificially age it.
	insertTestTrace(t, client, "old-trace", "s1", "gpt-4", "alice", 0.01)

	// Set the created_at to 60 days ago.
	_, err := client.DB().ExecContext(ctx,
		`UPDATE lake.traces SET created_at = NOW() - INTERVAL '60 days' WHERE id = 'old-trace-s1'`)
	if err != nil {
		t.Fatalf("update created_at: %v", err)
	}

	// Insert a recent trace.
	insertTestTrace(t, client, "new-trace", "s1", "gpt-4", "alice", 0.02)

	log := zap.NewNop()
	w := worker.New(client.DB(), log, time.Hour, 30*24*time.Hour)
	w.PurgeOldTraces(ctx)

	var count int64
	err = client.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM lake.traces`).Scan(&count)
	if err != nil {
		t.Fatalf("count traces: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 trace after purge (old one deleted), got %d", count)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
