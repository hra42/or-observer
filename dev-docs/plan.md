# LLM Observability Platform - Architecture & Implementation Plan

**Project Overview:** Build a lightweight, self-hosted LLM observability platform for tracking cost, performance, errors, and usage analytics from OpenRouter LLM API calls.

**Target Scale:** < 1K requests/day  
**Tech Stack:** Go (Backend) + SvelteKit + Svelte 5 (Frontend) + DuckDB/DuckLake (Data Layer)

---

## 1. Executive Summary: Why This Stack

### DuckDB/DuckLake Choice ✅

DuckDB's columnar storage and vectorized query execution offer remarkable performance for analytical workloads, even when dealing with large volumes of log data, ensuring that log analytics queries are completed quickly and efficiently.

For your LLM observability use case:

- **Perfect JSON Support**: DuckDB has excellent JSON support, allowing you to query complex JSON files using SQL without writing complex parsing code or heavyweight database setup
- **Cost Tracking**: You can track and optimize LLM costs with detailed token analytics, process millions of tokens on modest hardware, and do it all without complex infrastructure setup or cloud costs
- **Small Scale Efficiency**: For < 1K requests/day, an embedded database eliminates the need for separate database infrastructure

### DuckLake: The Next Evolution

DuckLake is an open Lakehouse format that stores metadata in a catalog database and data in Parquet files, allowing DuckDB to directly read and write data.

**Key advantages for your platform:**
- DuckLake delivers advanced data lake features without traditional lakehouse complexity by using Parquet files and a SQL database
- **Time Travel Queries**: DuckLake provides true ACID transactions across multiple tables, dramatically improved performance for small changes, and simplified operations
- **Future-Proof**: If you scale to multi-user or need archival, DuckLake enables "multiplayer DuckDB" without code rewrites

**Recommendation**: Start with vanilla DuckDB (embedded), migrate to DuckLake when you need:
- Multi-user concurrent access
- Time-travel/audit queries
- Data partitioning across files
- S3 integration for cold storage

---

## 2. System Architecture

### 2.1 High-Level Flow

```
OpenRouter Broadcast
        ↓
  Webhook (JSON)
        ↓
   Go Backend
   (Port 8080)
        ├─ Validate & Parse
        ├─ Enrich (user context, tags)
        └─ Insert into DuckDB
        ↓
   DuckDB Embedded
   (SQLite-compatible)
        ├─ Raw Traces Table
        ├─ Aggregated Metrics
        └─ Parquet Export (optional)
        ↓
   Go REST API
   (/api/traces, /api/metrics, /api/costs)
        ↓
   SvelteKit Frontend
   (Port 5173)
        ├─ TanStack Query
        ├─ Real-time Dashboard
        ├─ Trace Explorer
        └─ Cost Breakdown
```

### 2.2 Component Breakdown

**Backend (Go)**

**Responsibilities:**
- HTTP webhook receiver (`POST /webhook`)
- OTLP JSON parsing via openrouter-go SDK → DuckDB ingestion
- Query API endpoints
- Authentication/API key validation
- Graceful error handling & deduplication

**Key Dependencies:**
- `github.com/hra42/openrouter-go` - OpenRouter Broadcast webhook parsing
- `github.com/duckdb/duckdb-go/v2` - Official DuckDB Go driver (migrated from marcboeker/go-duckdb)
- `github.com/gin-gonic/gin` or `net/http` - HTTP server
- `go.uber.org/zap` - Structured logging

**OpenRouter SDK Integration:**

The `openrouter-go` library provides a convenient `BroadcastWebhookHandler` that automatically parses incoming OTLP JSON payloads and extracts structured trace data:

```go
// The SDK does the parsing for you
handler := openrouter.BroadcastWebhookHandler(func(traces []openrouter.BroadcastTrace) {
  // traces is already a []openrouter.BroadcastTrace with:
  // - TraceID, SpanID, SpanName
  // - Model, PromptTokens, CompletionTokens, TotalTokens
  // - Cost, Duration
  // - UserID, SessionID, Metadata
  for _, tr := range traces {
    // Insert into DuckDB/DuckLake
  }
})
```

This eliminates manual OTLP JSON parsing and makes your webhook handler clean and focused on data insertion.

#### Database (DuckDB)

**Schema (Initial MVP):**

```sql
-- Raw traces (partitioned by day)
CREATE TABLE traces (
  id VARCHAR PRIMARY KEY,
  trace_id VARCHAR,
  span_id VARCHAR,
  span_name VARCHAR,
  model VARCHAR,
  status VARCHAR,
  
  prompt_tokens INT,
  completion_tokens INT,
  total_tokens INT,
  cost DECIMAL(10, 6),
  
  duration_ms INT,
  user_id VARCHAR,
  session_id VARCHAR,
  metadata JSON,
  
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  webhook_received_at TIMESTAMP
);

-- Pre-aggregated metrics (hourly)
CREATE TABLE metrics_hourly (
  hour TIMESTAMP,
  model VARCHAR,
  user_id VARCHAR,
  
  request_count INT,
  avg_latency_ms FLOAT,
  p95_latency_ms FLOAT,
  p99_latency_ms FLOAT,
  
  total_tokens INT,
  total_cost DECIMAL(10, 6),
  
  error_count INT,
  PRIMARY KEY (hour, model, user_id)
);

-- Error logs
CREATE TABLE errors (
  id VARCHAR PRIMARY KEY,
  trace_id VARCHAR,
  error_type VARCHAR,
  error_message TEXT,
  stacktrace TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Indexing Strategy:**
```sql
CREATE INDEX idx_traces_user_model ON traces(user_id, model, created_at DESC);
CREATE INDEX idx_traces_created ON traces(created_at DESC);
CREATE INDEX idx_metrics_time ON metrics_hourly(hour DESC, model);
```

#### Frontend (SvelteKit + Svelte 5)

**Technology Stack:**
- **Framework**: SvelteKit (latest) with Svelte 5 runes
- **Data Fetching**: @tanstack/svelte-query returns Svelte stores, making values reactive by prefixing with $
- **Styling**: Tailwind CSS + shadcn/ui Svelte components
- **Charts**: Recharts (built-in, no external deps)
- **State**: TanStack Query for server state, Svelte stores for UI state

**Key Pages:**
1. **Dashboard** (`/dashboard`)
   - Real-time cost summary
   - Request count & latency trend (last 24h)
   - Top models by cost/requests
   - Error rate widget

2. **Trace Explorer** (`/traces`)
   - Filterable table (model, user, date range)
   - Click-to-expand trace details
   - Raw JSON viewer

3. **Analytics** (`/analytics`)
   - Cost breakdown by model/user
   - Latency distributions (P50, P95, P99)
   - Hourly/daily/monthly views
   - Custom date range picker

4. **Settings** (`/settings`)
   - API key management
   - Webhook URL (copy to OpenRouter)
   - Retention policies
   - Export data

---

## 3. Detailed Implementation Roadmap

### Phase 1: Foundation (Week 1-2)

**Important Note on duckdb-go:**
The DuckDB Go driver moved from `github.com/marcboeker/go-duckdb` to the official `github.com/duckdb/duckdb-go/v2` (starting with v2.5.0). Use the official repository for new projects. If upgrading from marcboeker's version, see the [migration guide](https://github.com/duckdb/duckdb-go/blob/main/MIGRATE.md).

**Backend:**
- [ ] Set up Go project structure
  ```
  llm-observability/
  ├── cmd/server/main.go
  ├── internal/
  │   ├── db/
  │   │   ├── client.go
  │   │   └── migrations.go
  │   ├── handlers/
  │   │   ├── webhook.go
  │   │   └── api.go
  │   ├── models/
  │   │   └── trace.go
  │   └── services/
  │       ├── parser.go
  │       └── aggregator.go
  ├── Dockerfile
  ├── go.mod
  └── go.sum
  ```
- [ ] Implement webhook handler (from your example)
- [ ] Initialize DuckDB with schema
- [ ] Write trace insertion logic
- [ ] Add deduplication (check for webhook retries via trace_id)

**Frontend:**
- [ ] Create SvelteKit project
  ```bash
  npm create svelte@latest llm-obs-ui
  npm install @tanstack/svelte-query tailwindcss recharts
  ```
- [ ] Set up root `+layout.svelte` with TanStack QueryClientProvider
- [ ] Create `/api` route handlers for proxying backend calls

**Database:**
- [ ] Design & test schema
- [ ] Create migrations script
- [ ] Test ingestion with sample OpenRouter data

### Phase 2: Core Features (Week 3-4)

**Backend:**
- [ ] `/api/traces` endpoint with filtering
  - Query params: `?user_id=&model=&start_date=&end_date=`
  - Return paginated results
- [ ] `/api/metrics/hourly` for dashboard trends
- [ ] `/api/costs/breakdown?groupBy=model|user`
- [ ] Error handling middleware
- [ ] Rate limiting (optional for now)

**Frontend:**
- [ ] Dashboard with:
  - TanStack Query reactive query using Svelte 5's $effect for reactive dependencies
  - Cost card (total $, today, this week)
  - Trend chart (Recharts Line chart)
  - Top models table
- [ ] Trace Explorer page
  - Client-side table filtering (TanStack Table integration optional)
  - Trace detail modal with JSON viewer
- [ ] Basic Auth (API key in header)

**Operations:**
- [ ] Docker compose for local development
- [ ] Environment variables (.env)
- [ ] Health check endpoint

### Phase 3: Polish & Observability (Week 5-6)

**Backend:**
- [ ] Structured logging (zap)
- [ ] Metrics export (Prometheus format optional)
- [ ] Graceful shutdown
- [ ] Request validation & sanitization
- [ ] Webhook signature verification (if OpenRouter supports)

**Frontend:**
- [ ] Analytics page with advanced filters
- [ ] Real-time alerts (mock for MVP)
- [ ] Dark mode toggle
- [ ] Mobile responsive design
- [ ] Loading states & error boundaries
- [ ] SvelteKit SSR support with prefetchQuery for server-side data loading

**Data Management:**
- [ ] Implement hourly aggregation job (Cron or background worker)
- [ ] Data retention policy (delete traces > 30 days)
- [ ] Parquet export for archival

---

## 4. Database Design Details

### 4.1 DuckDB Setup (Embedded)

**File Structure:**
```
/data
├── traces.duckdb          # Main database
├── traces.duckdb.wal      # Write-ahead log
└── backups/
    └── traces_20250228.duckdb  # Daily backup
```

**Connection String (Go - duckdb-go/v2):**

```go
import (
    "database/sql"
    _ "github.com/duckdb/duckdb-go/v2"
)

// Simple in-memory connection
db, err := sql.Open("duckdb", "")

// Or with a file and configuration
connector, err := duckdb.NewConnector("/data/traces.duckdb?access_mode=automatic&threads=4", 
    func(execer driver.ExecerContext) error {
        // Initialize DuckLake extension
        bootQueries := []string{
            `INSTALL ducklake`,
            `LOAD ducklake`,
        }
        for _, query := range bootQueries {
            _, err := execer.ExecContext(context.Background(), query, nil)
            if err != nil {
                return err
            }
        }
        return nil
    })
defer connector.Close()

db := sql.OpenDB(connector)
defer db.Close()
```

For bulk inserts, use the Appender API:
```go
conn, _ := connector.Connect(context.Background())
defer conn.Close()

appender, _ := duckdb.NewAppenderFromConn(conn, "", "traces")
defer appender.Close()

appender.AppendRow(traceID, spanID, model, cost, ...)
appender.Flush()
```

### 4.2 Migration to DuckLake (Future)

When you need multi-user concurrent writes:

```sql
-- Install extension
INSTALL ducklake;
LOAD ducklake;

-- Attach DuckLake with local Parquet storage
ATTACH 'ducklake:metadata.ducklake' AS my_lake (DATA_PATH './data/parquets/');

-- Create table in DuckLake
CREATE TABLE my_lake.traces (
  id VARCHAR,
  trace_id VARCHAR,
  -- ... schema
);

-- Time travel query
SELECT * FROM my_lake.traces AT (VERSION => snapshot_version);
```

---

## 5. API Reference

### Webhook Endpoint

**Request:**
```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "scopeSpans": [{
        "spans": [{
          "traceId": "abc123",
          "spanId": "def456",
          "name": "llm.completion",
          "attributes": {
            "llm.model": "gpt-4",
            "llm.usage.prompt_tokens": 100,
            "llm.usage.completion_tokens": 50,
            "user_id": "user_123"
          },
          "events": [],
          "endTimeUnixNano": "1234567890"
        }]
      }]
    }]
  }'
```

**Response:**
```json
{
  "success": true,
  "traceId": "abc123",
  "message": "Trace ingested"
}
```

### Query Endpoints

**GET /api/traces**
```bash
curl http://localhost:8080/api/traces?user_id=user_123&model=gpt-4&limit=50&offset=0
```

Response:
```json
{
  "total": 1234,
  "traces": [
    {
      "traceId": "abc123",
      "spanName": "llm.completion",
      "model": "gpt-4",
      "totalTokens": 150,
      "cost": 0.004500,
      "durationMs": 1234,
      "userId": "user_123",
      "createdAt": "2025-02-28T10:30:00Z"
    }
  ]
}
```

**GET /api/metrics/hourly**
```bash
curl http://localhost:8080/api/metrics/hourly?start=2025-02-27&end=2025-02-28&groupBy=model
```

**GET /api/costs/breakdown**
```bash
curl http://localhost:8080/api/costs/breakdown?groupBy=model&period=daily
```

---

## 6. Frontend Architecture

### 6.1 TanStack Query Setup

**Layout (+layout.svelte):**
```svelte
<script>
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query'
  import { browser } from '$app/environment'

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        enabled: browser,
        staleTime: 1000 * 60 * 5, // 5 minutes
        gcTime: 1000 * 60 * 10,    // 10 minutes
      },
    },
  })
</script>

<QueryClientProvider client={queryClient}>
  <slot />
</QueryClientProvider>
```

**Page Component (Dashboard):**
```svelte
<script>
  import { createQuery } from '@tanstack/svelte-query'

  let dateRange = { start: new Date(Date.now() - 86400000), end: new Date() }

  $: metricsQuery = createQuery({
    queryKey: ['metrics', dateRange],
    queryFn: async () => {
      const res = await fetch(`/api/metrics/hourly?start=${dateRange.start}&end=${dateRange.end}`)
      return res.json()
    },
  })

  $: ({ data, isLoading, error } = $metricsQuery)
</script>

{#if $isLoading}
  <p>Loading...</p>
{:else if error}
  <p>Error: {error.message}</p>
{:else}
  <div class="grid gap-4">
    <CostCard cost={data.totalCost} />
    <TrendChart data={data.metrics} />
  </div>
{/if}
```

### 6.2 Reactive Queries (Svelte 5 Runes)

With Svelte 5 runes, you can use $effect.pre() to create reactive query arguments:

```svelte
<script>
  import { createQuery } from '@tanstack/svelte-query'
  import { writable } from 'svelte/store'

  let searchTerm = $state('')
  let page = $state(1)

  const reactiveQueryArgs = () => ({
    queryKey: ['traces', `search:${searchTerm}`, `page:${page}`],
    queryFn: () => fetch(`/api/traces?q=${searchTerm}&page=${page}`).then(r => r.json())
  })

  let tracesQuery = createQuery(reactiveQueryArgs())

  $effect.pre(() => {
    tracesQuery = createQuery(reactiveQueryArgs())
  })
</script>

<input bind:value={searchTerm} placeholder="Search..." />
{#if $tracesQuery.isLoading}
  Loading...
{:else}
  {#each $tracesQuery.data.traces as trace}
    <TraceRow {trace} />
  {/each}
{/if}
```

---

## 7. Operational Considerations

### 7.1 Deployment Options

**Option A: Docker Compose (Recommended for MVP)**
```yaml
version: '3.8'
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      LOG_LEVEL: info
      DB_PATH: /app/data/traces.duckdb
  
  frontend:
    build: ./frontend
    ports:
      - "5173:5173"
    environment:
      VITE_API_URL: http://localhost:8080
    depends_on:
      - backend
```

**Option B: Single Binary + Embedded UI**
- Use `embed` in Go to bundle SvelteKit build
- Deploy as single binary
- Perfect for self-hosted

### 7.2 Data Retention & Cleanup

**Strategy:**
```sql
-- Daily maintenance job (run via cron or background worker)
DELETE FROM traces 
WHERE created_at < NOW() - INTERVAL '30 days';

DELETE FROM metrics_hourly
WHERE hour < NOW() - INTERVAL '365 days';

-- Defragment database
PRAGMA database_size;
```

### 7.3 Monitoring the Platform

**Health Check Endpoint:**
```go
GET /health
{
  "status": "ok",
  "database": "connected",
  "traces_ingested": 45678,
  "uptime_seconds": 123456
}
```

**Key Metrics:**
- Webhook ingestion latency (p95, p99)
- Database query latency
- Disk usage
- Memory usage

---

## 8. Testing Strategy

### 8.1 Backend Tests

```go
// Test webhook parsing
func TestParseWebhookTrace(t *testing.T) {
  payload := loadFixture("openrouter_trace.json")
  trace, err := parser.Parse(payload)
  assert.NoError(t, err)
  assert.Equal(t, "gpt-4", trace.Model)
}

// Test deduplication
func TestWebhookDeduplication(t *testing.T) {
  trace := &models.Trace{ID: "abc123", ...}
  db.Insert(trace)
  db.Insert(trace) // Same trace again
  
  count, _ := db.Count(&models.Trace{ID: "abc123"})
  assert.Equal(t, 1, count) // Only one inserted
}
```

### 8.2 Frontend Tests

```svelte
<script>
  import { render, screen } from '@testing-library/svelte'
  import Dashboard from './+page.svelte'

  test('displays cost summary', async () => {
    render(Dashboard)
    await screen.findByText(/\$[\d.]+/)
  })
</script>
```

### 8.3 E2E Tests

```bash
# Send sample trace, verify dashboard update
curl -X POST http://localhost:8080/webhook -d @sample.json
sleep 1
curl http://localhost:8080/api/metrics/hourly | jq .
```

---

## 9. Cost Considerations

### DuckDB vs. Alternatives

| Aspect | DuckDB | PostgreSQL | ClickHouse |
|--------|--------|------------|-----------|
| Setup | 1 file | Separate service | Complex cluster |
| Query Speed (analytics) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Embedded | Yes | No | No |
| Scaling | Single-node | Horizontal | Horizontal |
| Cost | Free | Free | Free |
| For < 1K req/day | Perfect | Overkill | Overkill |

**Recommendation:** DuckDB is ideal for your scale. Migrate to DuckLake when:
- Need multi-user concurrent writes
- Want S3 integration
- Require audit/time-travel
- Adding distributed compute

---

## 10. Sample Implementation Files

### Backend: Webhook Handler (Go) - Using openrouter-go SDK

The `openrouter-go` SDK provides a built-in `BroadcastWebhookHandler` that parses OTLP JSON traces:

```go
package handlers

import (
  "context"
  "database/sql"
  "fmt"
  "log"
  "net/http"

  "github.com/hra42/openrouter-go"
  "github.com/duckdb/duckdb-go/v2"
)

// Handler wraps DuckDB connection for webhook handling
type WebhookHandler struct {
  db *sql.DB
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(db *sql.DB) *WebhookHandler {
  return &WebhookHandler{db: db}
}

// ServeHTTP implements http.Handler
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if r.Method != http.MethodPost {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
  }

  // Use openrouter-go's built-in webhook handler
  handler := openrouter.BroadcastWebhookHandler(func(traces []openrouter.BroadcastTrace) {
    ctx := context.Background()
    
    for _, tr := range traces {
      // Insert each trace into DuckLake
      query := `
        INSERT INTO traces (
          id, trace_id, span_id, span_name, model, 
          prompt_tokens, completion_tokens, total_tokens, cost,
          duration_ms, user_id, session_id, metadata, webhook_received_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
      `
      
      _, err := h.db.ExecContext(ctx, query,
        generateID(),           // id (unique per trace)
        tr.TraceID,            // trace_id
        tr.SpanID,             // span_id
        tr.SpanName,           // span_name
        tr.Model,              // model
        tr.PromptTokens,       // prompt_tokens
        tr.CompletionTokens,   // completion_tokens
        tr.TotalTokens,        // total_tokens
        tr.Cost,               // cost
        tr.Duration.Milliseconds(), // duration_ms
        tr.UserID,             // user_id
        tr.SessionID,          // session_id
        toJSON(tr.Metadata),   // metadata
      )
      
      if err != nil {
        // Log but don't fail - webhook should be idempotent
        log.Printf("Failed to insert trace %s: %v", tr.TraceID, err)
      }
    }

    fmt.Printf("✓ Ingested %d traces\n", len(traces))
  })

  // Delegate to openrouter handler
  handler(w, r)
}

// Helper to generate unique IDs
func generateID() string {
  return fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String())
}

// Helper to marshal metadata to JSON
func toJSON(metadata map[string]interface{}) string {
  if len(metadata) == 0 {
    return "{}"
  }
  b, _ := json.Marshal(metadata)
  return string(b)
}
```

**Usage in main.go:**
```go
package main

import (
  "database/sql"
  "log"
  "net/http"
  _ "github.com/duckdb/duckdb-go/v2"
  "myapp/internal/handlers"
)

func main() {
  // Initialize DuckDB with DuckLake
  db, err := sql.Open("duckdb", "file:data/traces.duckdb")
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()

  // Register webhook handler
  webhookHandler := handlers.NewWebhookHandler(db)
  http.Handle("/webhook", webhookHandler)

  log.Println("Listening on :8080/webhook")
  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Frontend: Dashboard Component

```svelte
<script lang="ts">
  import { createQuery } from '@tanstack/svelte-query'
  import { LineChart, Line, XAxis, YAxis } from 'recharts'

  const metricsQuery = createQuery({
    queryKey: ['metrics-dashboard'],
    queryFn: async () => {
      const res = await fetch('/api/metrics/hourly?period=24h')
      return res.json()
    },
    staleTime: 60 * 1000, // 1 minute
  })
</script>

<div class="grid grid-cols-4 gap-4">
  <div class="bg-white p-4 rounded">
    <div class="text-sm text-gray-600">Total Cost (24h)</div>
    <div class="text-2xl font-bold">
      ${($metricsQuery.data?.totalCost ?? 0).toFixed(2)}
    </div>
  </div>

  <div class="bg-white p-4 rounded">
    <div class="text-sm text-gray-600">Requests</div>
    <div class="text-2xl font-bold">
      {$metricsQuery.data?.requestCount ?? 0}
    </div>
  </div>

  <div class="bg-white p-4 rounded">
    <div class="text-sm text-gray-600">Avg Latency</div>
    <div class="text-2xl font-bold">
      {($metricsQuery.data?.avgLatency ?? 0).toFixed(0)}ms
    </div>
  </div>

  <div class="bg-white p-4 rounded">
    <div class="text-sm text-gray-600">Error Rate</div>
    <div class="text-2xl font-bold">
      {($metricsQuery.data?.errorRate ?? 0).toFixed(1)}%
    </div>
  </div>
</div>

<div class="mt-8">
  {#if $metricsQuery.isLoading}
    <p>Loading chart...</p>
  {:else if $metricsQuery.data}
    <LineChart width={800} height={300} data={$metricsQuery.data.chartData}>
      <XAxis dataKey="time" />
      <YAxis />
      <Line type="monotone" dataKey="cost" stroke="#3b82f6" />
    </LineChart>
  {/if}
</div>
```

---

## 11. Timeline & Effort Estimate

| Phase | Duration | Effort | Deliverable |
|-------|----------|--------|-------------|
| Phase 1: Foundation | 2 weeks | 40 hours | Working webhook + basic dashboard |
| Phase 2: Core Features | 2 weeks | 40 hours | Full CRUD, filtering, analytics |
| Phase 3: Polish | 2 weeks | 30 hours | Production-ready, docs, deployment |
| **Total** | **6 weeks** | **110 hours** | **MVP Platform** |

**Realistic breakdown for solo dev:**
- Week 1-2: Backend (webhook, DuckDB schema)
- Week 2-3: Frontend (dashboard, TanStack integration)
- Week 3-4: Connect them (API endpoints, error handling)
- Week 4-5: Features (trace explorer, analytics)
- Week 5-6: Polish (UI, deployment, docs)

---

## 12. Next Steps

1. **Validate DuckDB Performance**
   ```bash
   go get github.com/marcboeker/go-duckdb
   # Write benchmark: 1000 trace insertions
   go test -bench=BenchmarkInsert -benchtime=10s
   ```

2. **Set Up Project Structure**
   ```bash
   mkdir llm-observability
   cd llm-observability
   go mod init github.com/yourname/llm-observability
   npm create svelte@latest frontend
   ```

3. **Design Database Schema (Finalize)**
   - Export as SQL file
   - Add sample queries
   - Test with mock data

4. **Create MVP Checklist**
   - [ ] Webhook receives traces
   - [ ] Dashboard shows cost card
   - [ ] Trace explorer table works
   - [ ] All CRUD operations work

---

## References & Resources

- **DuckDB Docs**: https://duckdb.org/docs
- **DuckLake Docs**: https://ducklake.select
- **TanStack Query Svelte**: https://tanstack.com/query/v4/docs/framework/svelte
- **SvelteKit Docs**: https://kit.svelte.dev
- **OpenRouter Broadcast**: [Your reference]

---

## FAQ

**Q: Will DuckDB handle 1000 requests/day easily?**  
A: Yes. DuckDB can analyze entire JSON logs with simple queries and has been tested with millions of records.

**Q: Should I start with DuckLake or plain DuckDB?**  
A: Start with DuckDB (embedded file). DuckLake adds complexity you don't need for single-user. Migrate in Phase 2 if needed.

**Q: Can I export traces to S3?**  
A: Yes, DuckDB supports Parquet export, and you can partition by date for archival.

**Q: How do I handle webhook retries?**  
A: Use `trace_id` as PRIMARY KEY or UNIQUE constraint. Second insert with same ID will fail gracefully.

**Q: Should I add real-time updates?**  
A: For MVP, polling via TanStack Query (staleTime: 10s) is sufficient. Add WebSockets in Phase 3 if needed.

---

**Document Version:** 1.0  
**Last Updated:** 2025-02-28  
**Author:** Claude
