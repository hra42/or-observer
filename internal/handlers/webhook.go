package handlers

import (
	"context"
	"net/http"

	openrouter "github.com/hra42/openrouter-go"
	"github.com/hra42/or-observer/internal/db"
	"github.com/hra42/or-observer/internal/models"
	"go.uber.org/zap"
)

// WebhookHandler receives OpenRouter Broadcast traces via HTTP POST.
type WebhookHandler struct {
	client *db.Client
	log    *zap.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(client *db.Client, log *zap.Logger) *WebhookHandler {
	return &WebhookHandler{client: client, log: log}
}

// ServeHTTP implements http.Handler using the openrouter-go SDK's
// BroadcastWebhookHandler for OTLP parsing, then inserts into DuckDB.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	successCount := 0
	errorCount := 0

	inner := openrouter.BroadcastWebhookHandlerWithError(func(traces []openrouter.BroadcastTrace) error {
		ctx := context.Background()
		for _, t := range traces {
			trace := models.TraceFromOpenRouter(t)
			err := h.client.InsertTrace(ctx,
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
			)
			if err != nil {
				h.log.Warn("failed to insert trace",
					zap.String("trace_id", t.TraceID),
					zap.String("span_id", t.SpanID),
					zap.Error(err),
				)
				errorCount++
			} else {
				successCount++
			}
		}
		return nil
	})

	inner(w, r)

	if successCount > 0 || errorCount > 0 {
		h.log.Info("webhook processed",
			zap.Int("ingested", successCount),
			zap.Int("errors", errorCount),
		)
	}
}
