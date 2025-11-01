package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"notifications/internal/config"
	"notifications/internal/logger"
	"notifications/internal/queue"
	"notifications/internal/repo"
	"notifications/internal/webpush"
)

func main() {
	// Load .env file if present (development convenience)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize structured logger
	appLogger, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Convert zap logger to slog for compatibility
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slogger.Info("Starting notifications worker",
		slog.String("log_level", cfg.LogLevel),
		slog.String("redis_addr", cfg.RedisAddr),
	)

	// Initialize database repository
	repository, err := repo.NewRepository(cfg.DatabaseURL)
	if err != nil {
		slogger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer repository.Close()

	slogger.Info("Connected to database")

	// Initialize webpush sender
	sender := webpush.NewSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, repository)
	slogger.Info("Initialized webpush sender")

	// Initialize worker
	worker := queue.NewWorker(
		queue.WorkerConfig{
			RedisAddr:   cfg.RedisAddr,
			Concurrency: 10, // Default concurrency
			Queues: map[string]int{
				"high":    6, // Priority weight 6
				"default": 3, // Priority weight 3
				"low":     1, // Priority weight 1
			},
		},
		repository,
		sender,
		slogger,
	)

	// Start worker
	if err := worker.Start(); err != nil {
		slogger.Error("Failed to start worker", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slogger.Info("Worker started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slogger.Info("Shutting down worker...")

	// Graceful shutdown
	worker.Stop()

	// Give time for workers to finish
	time.Sleep(2 * time.Second)

	slogger.Info("Worker stopped")

	// Final sync
	_ = appLogger.Sync()
}
