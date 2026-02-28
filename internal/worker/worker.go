package worker

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
)

// Worker runs periodic maintenance tasks: hourly aggregation and data retention.
type Worker struct {
	db       *sql.DB
	log      *zap.Logger
	interval time.Duration
	retain   time.Duration // traces older than this are deleted
}

// New creates a Worker that ticks every interval.
// retention sets the maximum age for traces (e.g. 30*24*time.Hour).
func New(db *sql.DB, log *zap.Logger, interval, retention time.Duration) *Worker {
	return &Worker{
		db:       db,
		log:      log,
		interval: interval,
		retain:   retention,
	}
}

// Run starts the ticker loop. It blocks until ctx is cancelled.
// It runs one cycle immediately on start, then once per interval.
func (w *Worker) Run(ctx context.Context) {
	w.log.Info("worker started",
		zap.Duration("interval", w.interval),
		zap.Duration("retention", w.retain),
	)

	w.runOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	w.AggregateHourly(ctx)
	w.PurgeOldTraces(ctx)
}

// AggregateHourly refreshes the last 2 hours of lake.metrics_hourly from lake.traces.
// Uses delete-then-insert because DuckLake does not support ON CONFLICT.
func (w *Worker) AggregateHourly(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)

	// Delete stale rollup rows for the overlap window.
	_, err := w.db.ExecContext(ctx,
		`DELETE FROM lake.metrics_hourly WHERE hour >= ?`, cutoff)
	if err != nil {
		w.log.Error("aggregation: delete failed", zap.Error(err))
		return
	}

	// Re-aggregate from raw traces.
	_, err = w.db.ExecContext(ctx, `
		INSERT INTO lake.metrics_hourly (
			hour, model, user_id,
			request_count, avg_latency_ms, p95_latency_ms, p99_latency_ms,
			total_tokens, total_cost, error_count
		)
		SELECT
			DATE_TRUNC('hour', created_at) AS hour,
			COALESCE(model, 'unknown'),
			COALESCE(user_id, 'unknown'),
			COUNT(*),
			COALESCE(AVG(duration_ms), 0)::DOUBLE,
			COALESCE(QUANTILE_DISC(duration_ms, 0.95), 0)::DOUBLE,
			COALESCE(QUANTILE_DISC(duration_ms, 0.99), 0)::DOUBLE,
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(CAST(cost AS DOUBLE)), 0),
			COUNT(CASE WHEN status = 'error' THEN 1 END)
		FROM lake.traces
		WHERE created_at >= ?
		GROUP BY DATE_TRUNC('hour', created_at), COALESCE(model, 'unknown'), COALESCE(user_id, 'unknown')`,
		cutoff)
	if err != nil {
		w.log.Error("aggregation: insert failed", zap.Error(err))
		return
	}

	w.log.Debug("aggregation completed")
}

// PurgeOldTraces deletes traces older than the retention window.
func (w *Worker) PurgeOldTraces(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-w.retain).Format(time.RFC3339)

	res, err := w.db.ExecContext(ctx,
		`DELETE FROM lake.traces WHERE created_at < ?`, cutoff)
	if err != nil {
		w.log.Error("purge: delete failed", zap.Error(err))
		return
	}

	n, _ := res.RowsAffected()
	if n > 0 {
		w.log.Info("purge completed", zap.Int64("deleted", n))
	}
}
