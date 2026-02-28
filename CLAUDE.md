# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**or-observer** is a self-hosted LLM observability platform for tracking cost, performance, errors, and usage analytics from OpenRouter API calls via webhooks.

**Target scale:** < 1K requests/day
**Status:** Phase 2 complete — backend APIs and frontend UI implemented.

## Tech Stack

- **Backend:** Go (`cmd/server/main.go`, `internal/`)
- **Database:** DuckDB embedded with DuckLake extension (`github.com/duckdb/duckdb-go/v2`) — use the official duckdb-go/v2, NOT the legacy `marcboeker/go-duckdb`
- **Frontend:** SvelteKit + Svelte 5 runes, TanStack Query v6, Tailwind CSS, Recharts
- **OpenRouter SDK:** `github.com/hra42/openrouter-go` (local path via `replace` directive) — provides `BroadcastWebhookHandlerWithError` for OTLP JSON parsing

## Project Structure

```
cmd/server/main.go              # Entry point, all routes registered here
internal/
  db/client.go                  # DuckDB connection + InsertTrace + CountTraces
  db/migrations.go              # Schema setup (CREATE TABLE IF NOT EXISTS)
  handlers/webhook.go           # POST /webhook via BroadcastWebhookHandlerWithError
  handlers/health.go            # GET /health
  handlers/api.go               # GET /api/traces, /api/metrics/hourly, /api/costs/breakdown
  handlers/api_test.go          # 10 handler tests
  models/trace.go               # Trace struct + TraceFromOpenRouter converter
data/
  traces.duckdb                 # DuckDB metadata DB (created at runtime)
  metadata.ducklake             # DuckLake catalog
  parquets/                     # Parquet data files (auto-created by DuckLake)
frontend/
  src/routes/dashboard/         # Cost cards, trend chart, top models table
  src/routes/traces/            # Filterable paginated table + detail modal
  src/routes/analytics/         # Cost breakdown + latency tabs with Recharts charts
  src/lib/api.ts                # Typed API client
  src/lib/recharts.ts           # Recharts re-export with `any` cast (Svelte 5 compat fix)
  src/lib/components/Nav.svelte # Navigation bar
```

## Key Architecture Decisions

### Webhook Flow
OpenRouter sends OTLP JSON → `POST /webhook` → `openrouter.BroadcastWebhookHandlerWithError` parses it → traces inserted into DuckDB. The SDK handles all OTLP deserialization; the handler only does DB insertion.

Use `BroadcastWebhookHandlerWithError` (not `BroadcastWebhookHandler`) — it accepts a callback that returns an error.

### DuckDB / DuckLake
- `duckdb.NewConnector(path, nil)` — nil init func is fine for plain embedded mode
- `db.SetMaxOpenConns(1)` required for embedded single-writer mode
- Boot sequence: `INSTALL ducklake` → `LOAD ducklake` → `ATTACH 'ducklake:<path>' AS lake (DATA_PATH '<path>/')`
- Use `isAlreadyExistsError()` helper to swallow "already exists/loaded/installed" errors on boot
- All table names prefixed with `lake.`: `lake.traces`, `lake.metrics_hourly`, `lake.errors`

**DuckLake constraints (discovered at runtime):**
- No `DEFAULT` expressions — only literals (`DEFAULT 'val'`), not `DEFAULT CURRENT_TIMESTAMP`
- No PRIMARY KEY or UNIQUE constraints
- No `ON CONFLICT` clause
- No secondary indexes
- Deduplication must be done manually: `SELECT COUNT(*) > 0 FROM lake.traces WHERE id = ?` before INSERT

### API Dynamic SQL
`api.go` handlers build WHERE clauses by string concatenation — safe because only fixed operator strings are concatenated; all user-supplied values go through `?` placeholders.

### Frontend Pattern (Svelte 5 + TanStack Query v6)

**`createQuery` options must be a function (accessor):**
```svelte
const query = createQuery(() => ({
  queryKey: ['traces', userID, model],   // state refs inside the fn = reactive
  queryFn: () => fetchTraces({ user_id: userID })
}))
```

**Result is a reactive proxy — access fields directly, NOT via `$store`:**
```svelte
{#if query.isLoading} … {/if}
<p>{query.data?.total}</p>
```

**Recharts `Tooltip` type incompatibility with Svelte 5:** Recharts hasn't updated its typedefs for Svelte 5's `Component` type. Fix: re-export with `as any` cast from `src/lib/recharts.ts` and import `Chart_Tooltip` instead of `Tooltip`.

### CORS
All routes are wrapped with `handlers.WithCORS()` middleware in `main.go`.

### Env Vars
- `DB_PATH` (default: `data/traces.duckdb`)
- `ADDR` (default: `:8080`)
- Frontend: `VITE_API_URL` (default: `http://localhost:8080`)

## API Endpoints

| Method | Path | Params | Description |
|--------|------|--------|-------------|
| POST | `/webhook` | — | OpenRouter broadcast receiver |
| GET | `/health` | — | DB status + trace count |
| GET | `/api/traces` | `user_id`, `model`, `start_date`, `end_date`, `limit` (max 500), `offset` | Paginated traces |
| GET | `/api/metrics/hourly` | `start`, `end`, `groupBy` (`model`\|`user`) | On-demand hourly aggregation |
| GET | `/api/costs/breakdown` | `groupBy` (`model`\|`user`), `period` (`hourly`\|`daily`\|`overall`) | Cost summary |

## Database Schema

- `lake.traces` — raw span data (id, trace_id, span_id, span_name, model, status, prompt_tokens, completion_tokens, total_tokens, cost, duration_ms, user_id, session_id, metadata JSON, created_at, webhook_received_at)
- `lake.metrics_hourly` — pre-aggregated hourly rollups
- `lake.errors` — error logs linked by trace_id

Deduplication: `id = trace_id + "-" + span_id`, checked before every INSERT (no UNIQUE constraint in DuckLake).

## Common Commands

```bash
# Backend
go run cmd/server/main.go
go test ./...
go test -v ./internal/handlers/...
go build -o server cmd/server/main.go

# Frontend (run from frontend/)
npm run dev
npm run build
npm run check        # svelte-check type validation

# Docker
docker-compose up --build

# Test webhook manually
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d @sample_trace.json

# Inspect DuckDB directly
duckdb data/traces.duckdb
```

## Testing Notes

- Handler tests use `t.TempDir()` for isolated DuckDB instances per test
- Metrics tests must pass explicit `start`/`end` params (e.g. `2000-01-01T00:00:00Z` to `2099-12-31T23:59:59Z`) — relying on the default "last 24h" window can fail in tests due to `NOW()` vs Go `time.Now()` timestamp precision differences

## Reference Docs

Full architecture details, schema SQL, and code templates are in `dev-docs/plan.md` and `dev-docs/guide.md`.
