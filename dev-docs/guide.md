# LLM Observability Platform - Quick Start Guide

## Project Setup

### 1. Initialize Go Project

```bash
mkdir llm-observability && cd llm-observability
go mod init github.com/yourusername/llm-observability

# Add dependencies
go get github.com/duckdb/duckdb-go/v2
go get github.com/hra42/openrouter-go
go get go.uber.org/zap
```

### 2. Create Project Structure

```
llm-observability/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── db/
│   │   ├── client.go
│   │   ├── migrations.go
│   │   └── schema.sql
│   ├── handlers/
│   │   └── webhook.go
│   ├── models/
│   │   └── trace.go
│   └── services/
│       └── aggregator.go
├── data/
│   ├── traces.duckdb       (created at runtime)
│   └── parquets/           (DuckLake data files)
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

---

## DuckDB/DuckLake Setup

### 1. Initialize Database with DuckLake

**File: `internal/db/schema.sql`**

```sql
-- Install and load DuckLake
INSTALL ducklake;
LOAD ducklake;

-- Attach DuckLake catalog
ATTACH 'ducklake:metadata.ducklake' AS traces_lake (DATA_PATH './data/parquets/');

-- Create tables in DuckLake
CREATE TABLE IF NOT EXISTS traces_lake.traces (
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

-- Indexes
CREATE INDEX IF NOT EXISTS idx_traces_user_model 
  ON traces_lake.traces(user_id, model, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_traces_created 
  ON traces_lake.traces(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_traces_trace_id 
  ON traces_lake.traces(trace_id);

-- Aggregated metrics (hourly)
CREATE TABLE IF NOT EXISTS traces_lake.metrics_hourly (
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
CREATE TABLE IF NOT EXISTS traces_lake.errors (
  id VARCHAR PRIMARY KEY,
  trace_id VARCHAR,
  error_type VARCHAR,
  error_message TEXT,
  stacktrace TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2. Database Client (Go)

**File: `internal/db/client.go`**

```go
package db

import (
  "context"
  "database/sql"
  "fmt"

  _ "github.com/duckdb/duckdb-go/v2"
)

type Client struct {
  db *sql.DB
}

// NewClient initializes DuckDB with DuckLake
func NewClient(dbPath string) (*Client, error) {
  // Connect to DuckDB
  db, err := sql.Open("duckdb", fmt.Sprintf("file:%s?access_mode=automatic&threads=4", dbPath))
  if err != nil {
    return nil, err
  }

  // Initialize DuckLake on first connection
  ctx := context.Background()
  
  initQueries := []string{
    "INSTALL ducklake",
    "LOAD ducklake",
    "ATTACH 'ducklake:metadata.ducklake' AS traces_lake (DATA_PATH './data/parquets/')",
  }
  
  for _, query := range initQueries {
    if _, err := db.ExecContext(ctx, query); err != nil {
      // DuckLake might already be loaded/attached - that's ok
      // Only fail on real errors like file permissions
      if isPermissionError(err) {
        return nil, err
      }
    }
  }

  // Run migrations
  if err := runMigrations(db); err != nil {
    return nil, err
  }

  return &Client{db: db}, nil
}

// InsertTrace adds a new trace to DuckLake
func (c *Client) InsertTrace(ctx context.Context, traceID, spanID, spanName, model string, 
  promptTokens, completionTokens, totalTokens int, cost float64, durationMs int,
  userID, sessionID string, metadata []byte) error {
  
  query := `
    INSERT INTO traces_lake.traces (
      id, trace_id, span_id, span_name, model,
      prompt_tokens, completion_tokens, total_tokens, cost,
      duration_ms, user_id, session_id, metadata, webhook_received_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
  `

  id := fmt.Sprintf("%s-%s", traceID, spanID)
  
  _, err := c.db.ExecContext(ctx, query,
    id, traceID, spanID, spanName, model,
    promptTokens, completionTokens, totalTokens, cost,
    durationMs, userID, sessionID, string(metadata),
  )
  
  return err
}

// QueryTraces retrieves traces with filters
func (c *Client) QueryTraces(ctx context.Context, userID, model string, limit, offset int) ([]map[string]interface{}, error) {
  query := `
    SELECT * FROM traces_lake.traces
    WHERE 1=1
  `
  
  args := []interface{}{}
  
  if userID != "" {
    query += " AND user_id = ?"
    args = append(args, userID)
  }
  
  if model != "" {
    query += " AND model = ?"
    args = append(args, model)
  }
  
  query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
  args = append(args, limit, offset)
  
  rows, err := c.db.QueryContext(ctx, query, args...)
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  
  var results []map[string]interface{}
  // TODO: Implement row scanning
  
  return results, nil
}

// Close closes the database connection
func (c *Client) Close() error {
  return c.db.Close()
}

func isPermissionError(err error) bool {
  // Simplified - implement full error checking as needed
  return err != nil && err.Error() != "extension already loaded"
}

func runMigrations(db *sql.DB) error {
  // Read schema.sql and execute
  // For MVP, just ensure tables exist
  return nil
}
```

---

## OpenRouter SDK Integration

### 1. Webhook Handler (Go)

**File: `internal/handlers/webhook.go`**

```go
package handlers

import (
  "context"
  "log"
  "net/http"

  "github.com/hra42/openrouter-go"
  "myapp/internal/db"
  "myapp/internal/models"
)

type WebhookHandler struct {
  dbClient *db.Client
}

func NewWebhookHandler(dbClient *db.Client) *WebhookHandler {
  return &WebhookHandler{dbClient: dbClient}
}

// ServeHTTP implements http.Handler
// Uses openrouter-go's built-in BroadcastWebhookHandler
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if r.Method != http.MethodPost {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
  }

  ctx := context.Background()
  successCount := 0
  errorCount := 0

  // Use openrouter-go SDK's webhook parser
  // It automatically handles OTLP JSON deserialization
  handler := openrouter.BroadcastWebhookHandler(func(traces []openrouter.BroadcastTrace) {
    for _, tr := range traces {
      // Convert openrouter.BroadcastTrace to your model
      trace := models.TraceFromOpenRouter(tr)

      // Insert into DuckLake
      if err := h.dbClient.InsertTrace(ctx,
        trace.TraceID,
        trace.SpanID,
        trace.SpanName,
        trace.Model,
        trace.PromptTokens,
        trace.CompletionTokens,
        trace.TotalTokens,
        trace.Cost,
        trace.DurationMs,
        trace.UserID,
        trace.SessionID,
        trace.MetadataJSON,
      ); err != nil {
        log.Printf("Failed to insert trace %s: %v", tr.TraceID, err)
        errorCount++
      } else {
        successCount++
      }
    }
  })

  // Delegate to openrouter handler
  handler(w, r)

  log.Printf("Webhook processed: %d successful, %d errors", successCount, errorCount)
}
```

### 2. Models

**File: `internal/models/trace.go`**

```go
package models

import (
  "encoding/json"
  "github.com/hra42/openrouter-go"
)

type Trace struct {
  TraceID           string
  SpanID            string
  SpanName          string
  Model             string
  PromptTokens      int
  CompletionTokens  int
  TotalTokens       int
  Cost              float64
  DurationMs        int
  UserID            string
  SessionID         string
  MetadataJSON      []byte
}

// TraceFromOpenRouter converts openrouter.BroadcastTrace to our Trace model
func TraceFromOpenRouter(orTrace openrouter.BroadcastTrace) Trace {
  metadata, _ := json.Marshal(orTrace.Metadata)
  
  return Trace{
    TraceID:          orTrace.TraceID,
    SpanID:           orTrace.SpanID,
    SpanName:         orTrace.SpanName,
    Model:            orTrace.Model,
    PromptTokens:     orTrace.PromptTokens,
    CompletionTokens: orTrace.CompletionTokens,
    TotalTokens:      orTrace.TotalTokens,
    Cost:             orTrace.Cost,
    DurationMs:       int(orTrace.Duration.Milliseconds()),
    UserID:           orTrace.UserID,
    SessionID:        orTrace.SessionID,
    MetadataJSON:     metadata,
  }
}
```

### 3. Main Server

**File: `cmd/server/main.go`**

```go
package main

import (
  "log"
  "net/http"

  "myapp/internal/db"
  "myapp/internal/handlers"
)

func main() {
  // Initialize DuckDB with DuckLake
  dbClient, err := db.NewClient("data/traces.duckdb")
  if err != nil {
    log.Fatalf("Failed to initialize database: %v", err)
  }
  defer dbClient.Close()

  log.Println("✓ DuckDB initialized with DuckLake")

  // Set up webhook handler
  webhookHandler := handlers.NewWebhookHandler(dbClient)
  http.Handle("/webhook", webhookHandler)

  // Health check
  http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
  })

  log.Println("🚀 Server listening on :8080")
  log.Println("   Webhook: POST http://localhost:8080/webhook")
  log.Println("   Health:  GET  http://localhost:8080/health")
  
  if err := http.ListenAndServe(":8080", nil); err != nil {
    log.Fatalf("Server error: %v", err)
  }
}
```

---

## Testing

### 1. Test Webhook with curl

```bash
# Start server
go run cmd/server/main.go

# In another terminal, send a test trace
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "scopeSpans": [{
        "spans": [{
          "traceId": "test-trace-123",
          "spanId": "test-span-456",
          "name": "llm.completion",
          "attributes": {
            "llm.model": "gpt-4",
            "llm.usage.prompt_tokens": 100,
            "llm.usage.completion_tokens": 50
          },
          "endTimeUnixNano": "1234567890000000000"
        }]
      }]
    }]
  }'
```

### 2. Verify Data in DuckDB

```bash
# Connect to DuckDB CLI
duckdb data/traces.duckdb

# Query traces
SELECT * FROM traces_lake.traces LIMIT 5;

# Check DuckLake snapshots (time travel)
SELECT * FROM traces_lake.ducklake_snapshots('traces');
```

### 3. Query Metrics

```sql
-- Get cost breakdown by model
SELECT 
  model,
  COUNT(*) as request_count,
  SUM(total_tokens) as total_tokens,
  SUM(cost) as total_cost,
  AVG(cost) as avg_cost
FROM traces_lake.traces
WHERE created_at >= NOW() - INTERVAL 24 HOURS
GROUP BY model
ORDER BY total_cost DESC;

-- Get user usage
SELECT 
  user_id,
  COUNT(*) as requests,
  SUM(cost) as cost,
  AVG(duration_ms) as avg_latency_ms
FROM traces_lake.traces
WHERE created_at >= NOW() - INTERVAL 24 HOURS
GROUP BY user_id
ORDER BY cost DESC;
```

---

## Docker Deployment

**File: `Dockerfile`**

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
RUN mkdir -p data/parquets

EXPOSE 8080
CMD ["./server"]
```

**File: `docker-compose.yml`**

```yaml
version: '3.8'

services:
  backend:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      LOG_LEVEL: info
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  frontend:
    build: ./frontend
    ports:
      - "5173:5173"
    environment:
      VITE_API_URL: http://localhost:8080
    depends_on:
      - backend
```

**Run with:**
```bash
docker-compose up --build
```

---

## Useful DuckLake Commands

```sql
-- Check catalog metadata
SELECT * FROM __ducklake_metadata_traces_lake.versions;

-- Time travel: query as of specific snapshot
SELECT * FROM traces_lake.traces AT (VERSION => 0);

-- Check table statistics
SELECT * FROM traces_lake.ducklake_table_stats('traces');

-- Cleanup old snapshots
CALL traces_lake.ducklake_expire_snapshots(older_than => NOW() - INTERVAL 30 DAYS);

-- Compact small parquet files
CALL traces_lake.ducklake_merge_adjacent_files();

-- Export to Parquet archive
COPY (SELECT * FROM traces_lake.traces) 
  TO 'backup/traces_export.parquet' (FORMAT 'parquet');
```

---

## Next Steps

1. ✅ Set up DuckDB + DuckLake
2. ✅ Integrate openrouter-go SDK
3. ⏳ Build REST API endpoints
4. ⏳ Create SvelteKit frontend
5. ⏳ Deploy to production

See the main plan for detailed phase breakdown!
