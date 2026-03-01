package models

import (
	"encoding/json"
	"fmt"

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

	// Add named fields that the SDK extracts from RawAttributes.
	// These no longer appear in RawAttributes as of v1.5.2.
	setIfNotEmpty := func(key, val string) {
		if val != "" {
			merged[key] = val
		}
	}
	setIfNotZeroF := func(key string, val float64) {
		if val != 0 {
			merged[key] = fmt.Sprintf("%g", val)
		}
	}
	setIfNotZeroI := func(key string, val int) {
		if val != 0 {
			merged[key] = fmt.Sprintf("%d", val)
		}
	}

	// GenAI semantic convention fields.
	setIfNotEmpty("gen_ai.operation.name", t.OperationName)
	setIfNotEmpty("gen_ai.system", t.System)
	setIfNotEmpty("gen_ai.provider.name", t.ProviderName)
	setIfNotEmpty("gen_ai.response.model", t.ResponseModel)
	setIfNotEmpty("gen_ai.request.model", t.RequestModel)
	setIfNotEmpty("gen_ai.response.finish_reason", t.FinishReason)
	setIfNotEmpty("gen_ai.response.finish_reasons", t.FinishReasons)
	setIfNotEmpty("gen_ai.prompt", t.Prompt)
	setIfNotEmpty("gen_ai.completion", t.Completion)

	// OpenRouter-specific fields.
	setIfNotEmpty("openrouter.provider_name", t.OpenRouterProviderName)
	setIfNotEmpty("openrouter.provider_slug", t.ProviderSlug)
	setIfNotEmpty("openrouter.api_key_name", t.APIKeyName)
	setIfNotEmpty("openrouter.entity_id", t.EntityID)
	setIfNotEmpty("openrouter.user_id", t.OpenRouterUserID)
	setIfNotEmpty("openrouter.finish_reason", t.OpenRouterFinishReason)
	setIfNotZeroF("openrouter.input_unit_price", t.InputUnitPrice)
	setIfNotZeroF("openrouter.output_unit_price", t.OutputUnitPrice)
	setIfNotEmpty("openrouter.source", t.Source)

	// Span-level fields.
	setIfNotEmpty("span.type", t.SpanType)
	setIfNotEmpty("span.level", t.SpanLevel)
	setIfNotEmpty("span.input", t.SpanInput)
	setIfNotEmpty("span.output", t.SpanOutput)

	// Trace-level fields.
	setIfNotEmpty("trace.name", t.TraceName)
	setIfNotEmpty("trace.input", t.TraceInput)
	setIfNotEmpty("trace.output", t.TraceOutput)
	setIfNotEmpty("trace.tags", t.TraceTags)

	// SpanMetadata (span.metadata.* attributes with prefix stripped by SDK).
	for k, v := range t.SpanMetadata {
		merged["span.metadata."+k] = v
	}

	// Cost breakdown fields.
	setIfNotZeroF("gen_ai.usage.total_cost", t.TotalCost)
	setIfNotZeroF("gen_ai.usage.input_cost", t.InputCost)
	setIfNotZeroF("gen_ai.usage.output_cost", t.OutputCost)

	// Token detail fields.
	setIfNotZeroI("gen_ai.usage.input_tokens", t.InputTokens)
	setIfNotZeroI("gen_ai.usage.output_tokens", t.OutputTokens)
	setIfNotZeroI("gen_ai.usage.input_tokens.cached", t.CachedTokens)
	setIfNotZeroI("gen_ai.usage.input_tokens.audio", t.AudioInputTokens)
	setIfNotZeroI("gen_ai.usage.input_tokens.video", t.VideoInputTokens)
	setIfNotZeroI("gen_ai.usage.output_tokens.image", t.ImageOutputTokens)
	setIfNotZeroI("gen_ai.usage.output_tokens.reasoning", t.ReasoningTokens)

	metaJSON, _ := json.Marshal(merged)

	// Prefer canonical token/cost fields from SDK when available.
	promptTokens := t.PromptTokens
	if t.InputTokens > 0 {
		promptTokens = t.InputTokens
	}
	completionTokens := t.CompletionTokens
	if t.OutputTokens > 0 {
		completionTokens = t.OutputTokens
	}
	cost := t.Cost
	if t.TotalCost > 0 {
		cost = t.TotalCost
	} else if cost == 0 && (t.InputCost+t.OutputCost) > 0 {
		cost = t.InputCost + t.OutputCost
	}

	return Trace{
		TraceID:          t.TraceID,
		SpanID:           t.SpanID,
		SpanName:         t.SpanName,
		Model:            t.Model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      t.TotalTokens,
		Cost:             cost,
		DurationMs:       int(t.Duration.Milliseconds()),
		UserID:           t.UserID,
		SessionID:        t.SessionID,
		MetadataJSON:     metaJSON,
	}
}
