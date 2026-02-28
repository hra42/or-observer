package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hra42/or-observer/internal/db"
)

type healthResponse struct {
	Status         string `json:"status"`
	Database       string `json:"database"`
	TracesIngested int64  `json:"traces_ingested"`
	UptimeSeconds  int64  `json:"uptime_seconds"`
}

// HealthHandler returns a simple health check response.
func HealthHandler(client *db.Client, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		count, err := client.CountTraces(ctx)
		dbStatus := "connected"
		if err != nil {
			dbStatus = "error: " + err.Error()
		}

		resp := healthResponse{
			Status:         "ok",
			Database:       dbStatus,
			TracesIngested: count,
			UptimeSeconds:  int64(time.Since(startTime).Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
