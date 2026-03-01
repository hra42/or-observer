package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hra42/or-observer/internal/db"
	"github.com/hra42/or-observer/internal/handlers"
	"github.com/hra42/or-observer/internal/worker"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger.
	var logger *zap.Logger
	var err error
	if os.Getenv("LOG_LEVEL") == "debug" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// Read config from env.
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/traces.duckdb"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Initialize database.
	client, err := db.NewClient(dbPath)
	if err != nil {
		logger.Fatal("failed to init database", zap.Error(err))
	}
	defer func() { _ = client.Close() }()

	// Set package-level logger for API handlers.
	handlers.SetLogger(logger)

	// Register routes.
	startTime := time.Now()
	mux := http.NewServeMux()

	mux.Handle("/webhook", handlers.WithCORS(handlers.NewWebhookHandler(client, logger)))
	mux.Handle("/health", handlers.WithCORS(handlers.HealthHandler(client, startTime)))
	mux.Handle("/api/traces", handlers.WithCORS(handlers.TracesHandler(client)))
	mux.Handle("/api/metrics/hourly", handlers.WithCORS(handlers.MetricsHourlyHandler(client)))
	mux.Handle("/api/costs/breakdown", handlers.WithCORS(handlers.CostsBreakdownHandler(client)))

	// Start background worker for hourly aggregation + data retention.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	w := worker.New(client.DB(), logger.Named("worker"), 1*time.Hour, 30*24*time.Hour)
	go w.Run(workerCtx)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen error", zap.Error(err))
		}
	}()

	<-done
	logger.Info("shutting down...")

	// Stop background worker.
	workerCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("server stopped")
}
