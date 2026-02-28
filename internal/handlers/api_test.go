package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hra42/or-observer/internal/db"
	"github.com/hra42/or-observer/internal/handlers"
)

func newTestClient(t *testing.T) *db.Client {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.duckdb")
	client, err := db.NewClient(dbPath)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func insertTestTrace(t *testing.T, client *db.Client, traceID, spanID, model, userID string, cost float64) {
	t.Helper()
	ctx := t.Context()
	err := client.InsertTrace(ctx, traceID, spanID, "test-span", model, 100, 50, 150, cost, 500, userID, "", []byte(`{}`))
	if err != nil {
		t.Fatalf("InsertTrace: %v", err)
	}
}

// ─── TracesHandler ──────────────────────────────────────────────────────────

func TestTracesHandler_EmptyDB(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.TracesHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/traces", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
}

func TestTracesHandler_FilterByModel(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "trace-1", "span-1", "gpt-4", "user-a", 0.01)
	insertTestTrace(t, client, "trace-2", "span-2", "gpt-3.5", "user-b", 0.001)

	handler := handlers.TracesHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/traces?model=gpt-4", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total=1 filtering by model=gpt-4, got %v", resp["total"])
	}
}

func TestTracesHandler_FilterByUserID(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "trace-1", "span-1", "gpt-4", "alice", 0.01)
	insertTestTrace(t, client, "trace-2", "span-2", "gpt-4", "bob", 0.01)
	insertTestTrace(t, client, "trace-3", "span-3", "gpt-4", "alice", 0.01)

	handler := handlers.TracesHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/api/traces?user_id=alice", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["total"].(float64) != 2 {
		t.Errorf("expected total=2 filtering by user_id=alice, got %v", resp["total"])
	}
}

func TestTracesHandler_Pagination(t *testing.T) {
	client := newTestClient(t)
	for i := 0; i < 5; i++ {
		insertTestTrace(t, client, "trace-"+string(rune('a'+i)), "span-"+string(rune('a'+i)), "gpt-4", "user", 0.01)
	}

	handler := handlers.TracesHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/api/traces?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	traces := resp["traces"].([]any)
	if len(traces) != 2 {
		t.Errorf("expected 2 traces in page, got %d", len(traces))
	}
	if resp["total"].(float64) != 5 {
		t.Errorf("expected total=5, got %v", resp["total"])
	}
}

func TestTracesHandler_InvalidLimit(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.TracesHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/traces?limit=-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── MetricsHourlyHandler ───────────────────────────────────────────────────

func TestMetricsHourlyHandler_EmptyDB(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.MetricsHourlyHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/hourly", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	metrics := resp["metrics"].([]any)
	if len(metrics) != 0 {
		t.Errorf("expected empty metrics, got %d rows", len(metrics))
	}
}

func TestMetricsHourlyHandler_GroupByModel(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "t1", "s1", "gpt-4", "user1", 0.05)
	insertTestTrace(t, client, "t2", "s2", "gpt-3.5", "user1", 0.005)
	insertTestTrace(t, client, "t3", "s3", "gpt-4", "user2", 0.05)

	handler := handlers.MetricsHourlyHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/api/metrics/hourly?groupBy=model&start=2000-01-01T00:00:00Z&end=2099-12-31T23:59:59Z", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	metrics := resp["metrics"].([]any)
	// Should have 2 groups: gpt-4 and gpt-3.5
	if len(metrics) != 2 {
		t.Errorf("expected 2 metric rows (by model), got %d", len(metrics))
	}
}

func TestMetricsHourlyHandler_InvalidGroupBy(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.MetricsHourlyHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/hourly?groupBy=invalid", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── CostsBreakdownHandler ──────────────────────────────────────────────────

func TestCostsBreakdownHandler_ByModel(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "t1", "s1", "gpt-4", "user1", 0.10)
	insertTestTrace(t, client, "t2", "s2", "gpt-4", "user2", 0.10)
	insertTestTrace(t, client, "t3", "s3", "claude-3", "user1", 0.05)

	handler := handlers.CostsBreakdownHandler(client)
	req := httptest.NewRequest(http.MethodGet, "/api/costs/breakdown?groupBy=model&period=overall", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["group_by"] != "model" {
		t.Errorf("expected group_by=model, got %v", resp["group_by"])
	}
	breakdown := resp["breakdown"].([]any)
	if len(breakdown) != 2 {
		t.Errorf("expected 2 breakdown rows, got %d", len(breakdown))
	}
	// gpt-4 should be first (highest cost)
	first := breakdown[0].(map[string]any)
	if first["dimension"] != "gpt-4" {
		t.Errorf("expected gpt-4 first (highest cost), got %v", first["dimension"])
	}
}

func TestCostsBreakdownHandler_InvalidPeriod(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.CostsBreakdownHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/costs/breakdown?period=weekly", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── Date Validation Tests ──────────────────────────────────────────────────

func TestTracesHandler_InvalidStartDate(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.TracesHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/traces?start_date=not-a-date", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMetricsHourlyHandler_InvalidStartDate(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.MetricsHourlyHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/hourly?start=bad-date", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCostsBreakdownHandler_WithDateRange(t *testing.T) {
	client := newTestClient(t)
	insertTestTrace(t, client, "t1", "s1", "gpt-4", "user1", 0.10)
	insertTestTrace(t, client, "t2", "s2", "claude-3", "user1", 0.05)

	handler := handlers.CostsBreakdownHandler(client)
	req := httptest.NewRequest(http.MethodGet,
		"/api/costs/breakdown?groupBy=model&start=2000-01-01T00:00:00Z&end=2099-12-31T23:59:59Z", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	breakdown := resp["breakdown"].([]any)
	if len(breakdown) != 2 {
		t.Errorf("expected 2 breakdown rows with date range, got %d", len(breakdown))
	}
}

func TestCostsBreakdownHandler_InvalidStartDate(t *testing.T) {
	client := newTestClient(t)
	handler := handlers.CostsBreakdownHandler(client)

	req := httptest.NewRequest(http.MethodGet, "/api/costs/breakdown?start=not-valid", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
