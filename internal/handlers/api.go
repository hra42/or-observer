package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/hra42/or-observer/internal/db"
	"go.uber.org/zap"
)

// logger is the package-level logger. Defaults to no-op so existing tests
// continue to pass without having to initialise a logger.
var logger *zap.Logger = zap.NewNop()

// SetLogger replaces the package-level logger (called from main).
func SetLogger(l *zap.Logger) {
	logger = l
}

// corsMiddleware adds CORS headers for frontend development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithCORS wraps any handler with CORS middleware.
func WithCORS(h http.Handler) http.Handler {
	return corsMiddleware(h)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	logger.Warn("api error", zap.Int("status", status), zap.String("msg", msg))
	writeJSON(w, status, map[string]any{"error": msg, "status": status})
}

// validateDateParam checks that v is a valid RFC3339 timestamp (if non-empty).
func validateDateParam(v string) error {
	if v == "" {
		return nil
	}
	_, err := time.Parse(time.RFC3339, v)
	return err
}

// sanitizeString caps s at maxLen characters.
func sanitizeString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// ─── /api/traces ────────────────────────────────────────────────────────────

type traceRow struct {
	ID               string    `json:"id"`
	TraceID          string    `json:"trace_id"`
	SpanID           string    `json:"span_id"`
	SpanName         string    `json:"span_name"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Cost             float64   `json:"cost"`
	DurationMs       int       `json:"duration_ms"`
	UserID           string    `json:"user_id"`
	SessionID        string    `json:"session_id"`
	Metadata         string    `json:"metadata"`
	CreatedAt        time.Time `json:"created_at"`
}

type tracesResponse struct {
	Total  int64      `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Traces []traceRow `json:"traces"`
}

// TracesHandler handles GET /api/traces.
func TracesHandler(client *db.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		q := r.URL.Query()

		limit := 50
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				writeError(w, http.StatusBadRequest, "invalid limit")
				return
			}
			if n > 500 {
				n = 500
			}
			limit = n
		}

		offset := 0
		if v := q.Get("offset"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				writeError(w, http.StatusBadRequest, "invalid offset")
				return
			}
			offset = n
		}

		userID := sanitizeString(q.Get("user_id"), 255)
		model := sanitizeString(q.Get("model"), 255)
		startDate := q.Get("start_date")
		endDate := q.Get("end_date")

		if err := validateDateParam(startDate); err != nil {
			writeError(w, http.StatusBadRequest, "invalid start_date: must be RFC3339 format")
			return
		}
		if err := validateDateParam(endDate); err != nil {
			writeError(w, http.StatusBadRequest, "invalid end_date: must be RFC3339 format")
			return
		}

		sqlDB := client.DB()
		ctx := r.Context()

		// Build WHERE clause dynamically.
		where := "1=1"
		args := []any{}
		if userID != "" {
			where += " AND user_id = ?"
			args = append(args, userID)
		}
		if model != "" {
			where += " AND model = ?"
			args = append(args, model)
		}
		if startDate != "" {
			where += " AND created_at >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			where += " AND created_at <= ?"
			args = append(args, endDate)
		}

		// Total count.
		var total int64
		countSQL := "SELECT COUNT(*) FROM lake.traces WHERE " + where
		if err := sqlDB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
			writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
			return
		}

		// Paginated rows.
		querySQL := `SELECT id, trace_id, span_id, COALESCE(span_name,''), COALESCE(model,''),
			COALESCE(prompt_tokens,0), COALESCE(completion_tokens,0), COALESCE(total_tokens,0),
			COALESCE(CAST(cost AS DOUBLE),0), COALESCE(duration_ms,0),
			COALESCE(user_id,''), COALESCE(session_id,''), COALESCE(CAST(metadata AS VARCHAR),'{}'), created_at
			FROM lake.traces WHERE ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = append(args, limit, offset)

		rows, err := sqlDB.QueryContext(ctx, querySQL, args...)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
			return
		}
		defer rows.Close()

		traces := []traceRow{}
		for rows.Next() {
			var t traceRow
			if err := rows.Scan(
				&t.ID, &t.TraceID, &t.SpanID, &t.SpanName, &t.Model,
				&t.PromptTokens, &t.CompletionTokens, &t.TotalTokens,
				&t.Cost, &t.DurationMs, &t.UserID, &t.SessionID, &t.Metadata, &t.CreatedAt,
			); err != nil {
				writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
				return
			}
			traces = append(traces, t)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, tracesResponse{
			Total:  total,
			Limit:  limit,
			Offset: offset,
			Traces: traces,
		})
	}
}

// ─── /api/metrics/hourly ────────────────────────────────────────────────────

type metricRow struct {
	Hour         time.Time `json:"hour"`
	Dimension    string    `json:"dimension"`
	RequestCount int       `json:"request_count"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P95LatencyMs float64   `json:"p95_latency_ms"`
	P99LatencyMs float64   `json:"p99_latency_ms"`
	TotalTokens  int64     `json:"total_tokens"`
	TotalCost    float64   `json:"total_cost"`
	ErrorCount   int       `json:"error_count"`
}

type metricsResponse struct {
	Metrics []metricRow `json:"metrics"`
}

// MetricsHourlyHandler handles GET /api/metrics/hourly.
func MetricsHourlyHandler(client *db.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		q := r.URL.Query()
		start := q.Get("start")
		end := q.Get("end")
		groupBy := q.Get("groupBy") // "model", "user", or ""

		// Validate date params before applying defaults.
		if err := validateDateParam(start); err != nil {
			writeError(w, http.StatusBadRequest, "invalid start: must be RFC3339 format")
			return
		}
		if err := validateDateParam(end); err != nil {
			writeError(w, http.StatusBadRequest, "invalid end: must be RFC3339 format")
			return
		}

		// Default time range: last 24 hours.
		if start == "" {
			start = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		}
		if end == "" {
			end = time.Now().UTC().Format(time.RFC3339)
		}

		// Validate groupBy.
		if groupBy != "" && groupBy != "model" && groupBy != "user" {
			writeError(w, http.StatusBadRequest, "groupBy must be 'model', 'user', or empty")
			return
		}

		var dimensionExpr string
		switch groupBy {
		case "model":
			dimensionExpr = "COALESCE(model, 'unknown')"
		case "user":
			dimensionExpr = "COALESCE(user_id, 'unknown')"
		default:
			dimensionExpr = "'all'"
		}

		sqlDB := client.DB()
		ctx := r.Context()

		querySQL := `SELECT
			DATE_TRUNC('hour', created_at) as hour,
			` + dimensionExpr + ` as dimension,
			COUNT(*) as request_count,
			COALESCE(AVG(duration_ms), 0)::DOUBLE as avg_latency_ms,
			COALESCE(QUANTILE_DISC(duration_ms, 0.95), 0)::DOUBLE as p95_latency_ms,
			COALESCE(QUANTILE_DISC(duration_ms, 0.99), 0)::DOUBLE as p99_latency_ms,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(CAST(cost AS DOUBLE)), 0) as total_cost,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count
		FROM lake.traces
		WHERE created_at BETWEEN ? AND ?
		GROUP BY DATE_TRUNC('hour', created_at), ` + dimensionExpr + `
		ORDER BY hour DESC`

		rows, err := sqlDB.QueryContext(ctx, querySQL, start, end)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
			return
		}
		defer rows.Close()

		metrics := []metricRow{}
		for rows.Next() {
			var m metricRow
			if err := rows.Scan(
				&m.Hour, &m.Dimension,
				&m.RequestCount, &m.AvgLatencyMs, &m.P95LatencyMs, &m.P99LatencyMs,
				&m.TotalTokens, &m.TotalCost, &m.ErrorCount,
			); err != nil {
				writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
				return
			}
			metrics = append(metrics, m)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, metricsResponse{Metrics: metrics})
	}
}

// ─── /api/costs/breakdown ───────────────────────────────────────────────────

type breakdownRow struct {
	Dimension    string  `json:"dimension"`
	RequestCount int     `json:"request_count"`
	TotalCost    float64 `json:"total_cost"`
	AvgCost      float64 `json:"avg_cost"`
	TotalTokens  int64   `json:"total_tokens"`
}

type costsResponse struct {
	Period    string         `json:"period"`
	GroupBy   string         `json:"group_by"`
	Breakdown []breakdownRow `json:"breakdown"`
}

// CostsBreakdownHandler handles GET /api/costs/breakdown.
func CostsBreakdownHandler(client *db.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		q := r.URL.Query()
		groupBy := q.Get("groupBy")
		period := q.Get("period")
		startParam := q.Get("start")
		endParam := q.Get("end")

		if groupBy == "" {
			groupBy = "model"
		}
		if groupBy != "model" && groupBy != "user" {
			writeError(w, http.StatusBadRequest, "groupBy must be 'model' or 'user'")
			return
		}
		if period == "" {
			period = "daily"
		}

		// Validate optional date range params.
		if err := validateDateParam(startParam); err != nil {
			writeError(w, http.StatusBadRequest, "invalid start: must be RFC3339 format")
			return
		}
		if err := validateDateParam(endParam); err != nil {
			writeError(w, http.StatusBadRequest, "invalid end: must be RFC3339 format")
			return
		}

		// Build time filter: use explicit date range if provided, otherwise use period-based interval.
		var whereTime string
		queryArgs := []any{}
		if startParam != "" && endParam != "" {
			whereTime = "created_at BETWEEN ? AND ?"
			queryArgs = append(queryArgs, startParam, endParam)
		} else if startParam != "" {
			whereTime = "created_at >= ?"
			queryArgs = append(queryArgs, startParam)
		} else if endParam != "" {
			whereTime = "created_at <= ?"
			queryArgs = append(queryArgs, endParam)
		} else {
			var intervalExpr string
			switch period {
			case "hourly":
				intervalExpr = "INTERVAL '1 hour'"
			case "daily":
				intervalExpr = "INTERVAL '1 day'"
			case "overall":
				intervalExpr = "INTERVAL '100 year'"
			default:
				writeError(w, http.StatusBadRequest, "period must be 'hourly', 'daily', or 'overall'")
				return
			}
			whereTime = "created_at >= NOW() - " + intervalExpr
		}

		var dimensionExpr string
		switch groupBy {
		case "model":
			dimensionExpr = "COALESCE(model, 'unknown')"
		case "user":
			dimensionExpr = "COALESCE(user_id, 'unknown')"
		}

		querySQL := `SELECT
			` + dimensionExpr + ` as dimension,
			COUNT(*) as request_count,
			COALESCE(SUM(CAST(cost AS DOUBLE)), 0) as total_cost,
			COALESCE(AVG(CAST(cost AS DOUBLE)), 0) as avg_cost,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM lake.traces
		WHERE ` + whereTime + `
		GROUP BY ` + dimensionExpr + `
		ORDER BY total_cost DESC`

		sqlDB := client.DB()
		ctx := r.Context()

		rows, err := sqlDB.QueryContext(ctx, querySQL, queryArgs...)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "query failed: "+err.Error())
			return
		}
		defer rows.Close()

		breakdown := []breakdownRow{}
		for rows.Next() {
			var b breakdownRow
			var totalCost, avgCost sql.NullFloat64
			if err := rows.Scan(&b.Dimension, &b.RequestCount, &totalCost, &avgCost, &b.TotalTokens); err != nil {
				writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
				return
			}
			b.TotalCost = totalCost.Float64
			b.AvgCost = avgCost.Float64
			breakdown = append(breakdown, b)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "rows error: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, costsResponse{
			Period:    period,
			GroupBy:   groupBy,
			Breakdown: breakdown,
		})
	}
}
