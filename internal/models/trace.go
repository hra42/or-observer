package models

import (
	"encoding/json"

	openrouter "github.com/hra42/openrouter-go"
)

// Trace is the internal representation of a single LLM span.
type Trace struct {
	TraceID          string
	SpanID           string
	SpanName         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Cost             float64
	DurationMs       int
	UserID           string
	SessionID        string
	MetadataJSON     []byte
}

// TraceFromOpenRouter converts an openrouter.BroadcastTrace to the internal Trace model.
func TraceFromOpenRouter(t openrouter.BroadcastTrace) Trace {
	// Merge Metadata and RawAttributes into the stored JSON blob.
	merged := make(map[string]string, len(t.Metadata)+len(t.RawAttributes))
	for k, v := range t.RawAttributes {
		merged[k] = v
	}
	for k, v := range t.Metadata {
		merged[k] = v
	}

	metaJSON, _ := json.Marshal(merged)

	return Trace{
		TraceID:          t.TraceID,
		SpanID:           t.SpanID,
		SpanName:         t.SpanName,
		Model:            t.Model,
		PromptTokens:     t.PromptTokens,
		CompletionTokens: t.CompletionTokens,
		TotalTokens:      t.TotalTokens,
		Cost:             t.Cost,
		DurationMs:       int(t.Duration.Milliseconds()),
		UserID:           t.UserID,
		SessionID:        t.SessionID,
		MetadataJSON:     metaJSON,
	}
}
