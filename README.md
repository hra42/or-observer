# or-observer

Self-hosted LLM observability platform for tracking cost, performance, errors, and usage analytics from OpenRouter API calls via webhooks.

## What It Does

or-observer receives traces from OpenRouter via webhooks and provides:

- Cost tracking and breakdowns by model or user
- Request latency metrics (avg, p95, p99) with hourly aggregation
- Trace inspection with filterable, paginated table and detail view
- Dashboard and analytics UI

Target scale: < 1K requests/day. Runs self-hosted with no external dependencies.

## Features

- Webhook receiver for OpenRouter broadcast OTLP JSON events
- Cost dashboard: real-time cost cards, hourly trend chart, top models table
- Trace explorer: filterable and paginated with per-trace metadata modal
- Analytics: cost breakdown by model/user, latency percentile charts
- Embedded DuckDB with DuckLake extension — fast columnar storage, single file
- Full REST API with pagination and time-range filtering

## Quick Start

```bash
docker-compose up --build
```

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173`

Data persists in `./data/`. Send test webhooks to `http://localhost:8080/webhook`.

## Manual Setup

### Prerequisites

- Go 1.26+
- Node.js 18+

### Backend

```bash
go mod download
go run cmd/server/main.go
```

Starts on `:8080` by default.

### Frontend

```bash
cd frontend
npm install
npm run dev       # dev server with hot reload
npm run build     # production build
npm run check     # type check
```

Connects to `http://localhost:8080` by default. Override with `VITE_API_URL`.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DB_PATH` | `data/traces.duckdb` | DuckDB file path |
| `ADDR` | `:8080` | Backend listen address |
| `VITE_API_URL` | `http://localhost:8080` | API base URL (frontend only) |

## API Reference

### POST /webhook

Receives OpenRouter broadcast OTLP JSON. Returns `200 OK` on success.

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @sample_trace.json
```

### GET /health

```json
{
  "status": "ok",
  "database": "connected",
  "traces_ingested": 42,
  "uptime_seconds": 120
}
```

### GET /api/traces

Paginated trace list.

| Param | Type | Description |
|-------|------|-------------|
| `limit` | int | Max results (default 50, max 500) |
| `offset` | int | Pagination offset |
| `user_id` | string | Filter by user (exact match) |
| `model` | string | Filter by model (exact match) |
| `start_date` | ISO 8601 | Filter by created_at >= |
| `end_date` | ISO 8601 | Filter by created_at <= |

```json
{
  "total": 1234,
  "limit": 50,
  "offset": 0,
  "traces": [
    {
      "id": "abc-123-span-456",
      "trace_id": "abc-123",
      "span_id": "span-456",
      "model": "openai/gpt-4",
      "prompt_tokens": 100,
      "completion_tokens": 50,
      "total_tokens": 150,
      "cost": 0.00315,
      "duration_ms": 1234,
      "user_id": "user-789",
      "created_at": "2026-02-28T14:30:00Z"
    }
  ]
}
```

### GET /api/metrics/hourly

On-demand hourly aggregation from raw traces.

| Param | Type | Description |
|-------|------|-------------|
| `start` | ISO 8601 | Range start (default: 24h ago) |
| `end` | ISO 8601 | Range end (default: now) |
| `groupBy` | `model` \| `user` | Dimension to group by (default: overall) |

```json
{
  "metrics": [
    {
      "hour": "2026-02-28T14:00:00Z",
      "dimension": "openai/gpt-4",
      "request_count": 42,
      "avg_latency_ms": 1500.0,
      "p95_latency_ms": 2800.0,
      "p99_latency_ms": 3200.0,
      "total_tokens": 6300,
      "total_cost": 0.132,
      "error_count": 1
    }
  ]
}
```

### GET /api/costs/breakdown

Cost summary by dimension and period.

| Param | Type | Description |
|-------|------|-------------|
| `groupBy` | `model` \| `user` | Required. Dimension to group by |
| `period` | `hourly` \| `daily` \| `overall` | Required. Lookback window |

```json
{
  "period": "daily",
  "group_by": "model",
  "breakdown": [
    {
      "dimension": "openai/gpt-4",
      "request_count": 340,
      "total_cost": 12.45,
      "avg_cost": 0.036,
      "total_tokens": 45600
    }
  ]
}
```

## How It Works

1. **Ingest** — OpenRouter sends OTLP JSON to `POST /webhook`. The `openrouter-go` SDK deserializes the payload; the handler inserts each span into DuckDB.
2. **Deduplicate** — Each trace is keyed by `trace_id + span_id`. Duplicate webhook deliveries are silently ignored.
3. **Query** — API handlers compute metrics on-demand via SQL (no background aggregation worker needed at this scale).
4. **Visualize** — SvelteKit frontend fetches from `/api/*` with TanStack Query, renders charts with Recharts.

## Development

### Run tests

```bash
go test ./...
go test -v ./internal/handlers/...
```

### Inspect the database

```bash
duckdb data/traces.duckdb
# then:
SELECT COUNT(*) FROM lake.traces;
SELECT model, SUM(cost) FROM lake.traces GROUP BY model ORDER BY 2 DESC;
```

## Database Schema

**lake.traces**

| Column | Type | Notes |
|--------|------|-------|
| id | VARCHAR | trace_id + "-" + span_id |
| trace_id | VARCHAR | |
| span_id | VARCHAR | |
| span_name | VARCHAR | |
| model | VARCHAR | e.g. `openai/gpt-4` |
| status | VARCHAR | `ok` or error |
| prompt_tokens | INTEGER | |
| completion_tokens | INTEGER | |
| total_tokens | INTEGER | |
| cost | DECIMAL(10,6) | USD |
| duration_ms | INTEGER | |
| user_id | VARCHAR | |
| session_id | VARCHAR | |
| metadata | JSON | merged span + resource attributes |
| created_at | TIMESTAMP | |
| webhook_received_at | TIMESTAMP | |

**lake.metrics_hourly** — pre-aggregated rollups (hour, model, user_id, request_count, latency percentiles, total_cost, error_count)

**lake.errors** — error logs (id, trace_id, error_type, error_message, stacktrace, created_at)

## Troubleshooting

**Frontend can't reach backend** — check `VITE_API_URL`. In Docker, services use `http://backend:8080`.

**Webhook returns 400** — validate JSON matches OpenRouter OTLP format; check server logs for parse errors.

**Traces not appearing** — same `trace_id + span_id` pair is deduplicated silently; verify with `SELECT * FROM lake.traces` in the duckdb CLI.

**DuckDB locked** — embedded DuckDB allows one writer at a time. Ensure only one backend process is running and no duckdb CLI session has the file open.
