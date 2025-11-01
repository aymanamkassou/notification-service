package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"notifications/internal/config"
	apihttp "notifications/internal/http"
	"notifications/internal/logger"
	"notifications/internal/repo"
)

func main() {
	// Load .env file if present (development convenience)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	appLogger, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Sync()

	appLogger.Info("starting notifications api server",
		zap.String("log_level", cfg.LogLevel),
		zap.String("port", cfg.Port),
	)

	// Connect to database
	appLogger.Info("connecting to database")
	repository, err := repo.NewRepository(cfg.DatabaseURL)
	if err != nil {
		appLogger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer repository.Close()

	appLogger.Info("database connection established")

	// Create HTTP router
	router := apihttp.NewRouter(*cfg, repository, appLogger)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Info("api server listening", zap.String("addr", server.Addr))
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		appLogger.Fatal("server error", zap.Error(err))

	case sig := <-shutdown:
		appLogger.Info("shutdown signal received",
			zap.String("signal", sig.String()),
		)

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			appLogger.Error("graceful shutdown failed, forcing close",
				zap.Error(err),
			)
			if err := server.Close(); err != nil {
				appLogger.Error("server close error", zap.Error(err))
			}
		}

		appLogger.Info("server stopped gracefully")
	}
}
