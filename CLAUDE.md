# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**or-observer** is a self-hosted LLM observability platform for tracking cost, performance, errors, and usage analytics from OpenRouter API calls via webhooks.

**Target scale:** < 1K requests/day
**Status:** Phase 3 complete — production-ready with structured logging, graceful shutdown, background workers, dark mode, SSR, and alerts.

## Tech Stack

- **Backend:** Go (`cmd/server/main.go`, `internal/`)
- **Database:** DuckDB embedded with DuckLake extension (`github.com/duckdb/duckdb-go/v2`) — use the official duckdb-go/v2, NOT the legacy `marcboeker/go-duckdb`
- **Frontend:** SvelteKit + Svelte 5 runes, TanStack Query v6, Tailwind CSS, Recharts
- **OpenRouter SDK:** `github.com/hra42/openrouter-go` (local path via `replace` directive) — provides `BroadcastWebhookHandlerWithError` for OTLP JSON parsing

## Project Structure

```
cmd/server/main.go              # Entry point: routes, graceful shutdown, worker
internal/
  db/client.go                  # DuckDB connection + InsertTrace + CountTraces
  db/migrations.go              # Schema setup (CREATE TABLE IF NOT EXISTS)
  handlers/webhook.go           # POST /webhook via BroadcastWebhookHandlerWithError
  handlers/health.go            # GET /health
  handlers/api.go               # GET /api/traces, /api/metrics/hourly, /api/costs/breakdown + SetLogger
  handlers/api_test.go          # 14 handler tests (incl. date validation, date-range costs)
  models/trace.go               # Trace struct + TraceFromOpenRouter converter
  worker/worker.go              # Background worker: hourly aggregation + 30-day retention
  worker/worker_test.go         # Worker tests
data/
  traces.duckdb                 # DuckDB metadata DB (created at runtime)
  metadata.ducklake             # DuckLake catalog
  parquets/                     # Parquet data files (auto-created by DuckLake)
frontend/
  src/routes/+layout.svelte     # QueryClientProvider + HydrationBoundary + dark mode
  src/routes/+error.svelte      # SvelteKit error boundary page
  src/routes/dashboard/         # Cost cards, trend chart, top models table, alert banner
  src/routes/traces/            # Filterable paginated table + detail modal
  src/routes/analytics/         # Cost breakdown + latency tabs with advanced filters
  src/routes/alerts/            # Alert threshold configuration (localStorage)
  src/lib/api.ts                # Typed API client (costs breakdown supports start/end)
  src/lib/recharts.ts           # Recharts re-export with `any` cast (Svelte 5 compat fix)
  src/lib/stores/theme.svelte.ts # Dark/light mode store with localStorage persistence
  src/lib/components/Nav.svelte # Navigation bar + theme toggle + mobile hamburger
  src/lib/components/Spinner.svelte   # Animated loading spinner
  src/lib/components/ErrorAlert.svelte # Error display with retry button
  src/lib/components/AlertBanner.svelte # Dismissible threshold alert banners
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

### Structured Logging
`go.uber.org/zap` is used throughout. The webhook handler receives a logger via constructor; API handlers use a package-level logger set via `handlers.SetLogger()`. Default is `zap.NewNop()` so tests pass without logger init.

### Graceful Shutdown
`cmd/server/main.go` listens for SIGINT/SIGTERM, cancels the background worker context, and calls `srv.Shutdown()` with a 15s timeout.

### Background Worker
`internal/worker/worker.go` runs on a 1-hour ticker:
- **Hourly aggregation:** DELETE last 2h from `lake.metrics_hourly`, then INSERT...SELECT from `lake.traces` (delete-then-insert to work around no ON CONFLICT)
- **Data retention:** DELETE traces older than 30 days

### Dark Mode
Class-based dark mode using Tailwind v4 `@custom-variant dark (&:where(.dark *))`. Theme state in `$lib/stores/theme.svelte.ts` with localStorage persistence. Toggle button in Nav. All pages use `bg-gray-100 dark:bg-gray-800` dual-class pattern.

### SSR Prefetching
Each route has a `+page.ts` universal load that creates a temporary `QueryClient`, calls `prefetchQuery` (via `Promise.allSettled` for graceful failure), and returns `dehydrate(queryClient)`. The layout wraps children in `HydrationBoundary`.

### Alerts
Frontend-only threshold alerts stored in localStorage (`or-observer-alert-thresholds`). `AlertBanner` component on dashboard shows dismissible warnings when 24h cost or error count exceeds configured thresholds. `/alerts` page for threshold configuration.

### CORS
All routes are wrapped with `handlers.WithCORS()` middleware in `main.go`.

### Env Vars
- `DB_PATH` (default: `data/traces.duckdb`)
- `ADDR` (default: `:8080`)
- `LOG_LEVEL` (set to `debug` for development logger, otherwise production)
- Frontend: `VITE_API_URL` (default: `http://localhost:8080`)

## API Endpoints

| Method | Path | Params | Description |
|--------|------|--------|-------------|
| POST | `/webhook` | — | OpenRouter broadcast receiver |
| GET | `/health` | — | DB status + trace count |
| GET | `/api/traces` | `user_id`, `model`, `start_date`, `end_date`, `limit` (max 500), `offset` | Paginated traces |
| GET | `/api/metrics/hourly` | `start`, `end`, `groupBy` (`model`\|`user`) | On-demand hourly aggregation |
| GET | `/api/costs/breakdown` | `groupBy` (`model`\|`user`), `period` (`hourly`\|`daily`\|`overall`), `start`, `end` (optional RFC3339 date range) | Cost summary |

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
