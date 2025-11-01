package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"notifications/internal/repo"
	"notifications/internal/webpush"
)

// Worker processes tasks from Redis/Asynq
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	repo   *repo.Repository
	sender *webpush.Sender
	logger *slog.Logger
}

// WorkerConfig contains configuration for the worker
type WorkerConfig struct {
	RedisAddr   string
	Concurrency int
	Queues      map[string]int // Queue name to priority weight
}

// NewWorker creates a new worker
func NewWorker(
	cfg WorkerConfig,
	repository *repo.Repository,
	sender *webpush.Sender,
	logger *slog.Logger,
) *Worker {
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues:      cfg.Queues,
			Logger:      &asynqLogger{logger: logger},
		},
	)

	w := &Worker{
		server: server,
		mux:    asynq.NewServeMux(),
		repo:   repository,
		sender: sender,
		logger: logger,
	}

	// Register task handlers
	w.mux.HandleFunc(TypeDeliverNotification, w.handleDeliverNotification)

	return w
}

// Start starts the worker
func (w *Worker) Start() error {
	w.logger.Info("Starting worker")
	return w.server.Start(w.mux)
}

// Stop stops the worker gracefully
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker")
	w.server.Shutdown()
}

// handleDeliverNotification processes a notification delivery task
func (w *Worker) handleDeliverNotification(ctx context.Context, task *asynq.Task) error {
	// Parse the payload
	var payload DeliverNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error("Failed to unmarshal task payload",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	w.logger.Info("Processing notification delivery",
		slog.String("notification_id", payload.NotificationID.String()),
		slog.String("user_id", payload.UserID),
		slog.String("subscription_id", payload.SubscriptionID.String()),
	)

	// Retry count will be tracked manually through task retries
	retryCount := 0

	// Send the notification
	result, err := w.sender.SendNotification(
		ctx,
		payload.NotificationID,
		payload.SubscriptionID,
		payload.UserID,
	)
	if err != nil {
		w.logger.Error("Failed to send notification",
			slog.String("notification_id", payload.NotificationID.String()),
			slog.String("error", err.Error()),
		)
		// Record failed attempt
		_ = w.recordAttempt(ctx, payload, "failed", nil, nil, retryCount, err.Error())
		return fmt.Errorf("failed to send notification: %w", err)
	}

	// Record the delivery attempt
	status := "delivered"
	if !result.Success {
		status = "failed"
	}

	httpStatus := result.HTTPStatus
	latencyMs := int32(result.LatencyMs)
	errorMsg := result.Error

	if err := w.recordAttempt(
		ctx,
		payload,
		status,
		&httpStatus,
		&latencyMs,
		retryCount,
		errorMsg,
	); err != nil {
		w.logger.Error("Failed to record delivery attempt",
			slog.String("notification_id", payload.NotificationID.String()),
			slog.String("error", err.Error()),
		)
	}

	// Prune subscription if needed (mark as inactive)
	if result.ShouldPrune {
		w.logger.Info("Deactivating subscription",
			slog.String("subscription_id", payload.SubscriptionID.String()),
			slog.String("reason", result.Error),
		)
		if err := w.repo.DeactivateDeviceSubscription(ctx, payload.SubscriptionID); err != nil {
			w.logger.Error("Failed to deactivate subscription",
				slog.String("subscription_id", payload.SubscriptionID.String()),
				slog.String("error", err.Error()),
			)
		}
	}

	// If the delivery failed but shouldn't be pruned, return an error to trigger retry
	if !result.Success && !result.ShouldPrune {
		return fmt.Errorf("delivery failed: %s", result.Error)
	}

	return nil
}

// recordAttempt records a delivery attempt in the database
func (w *Worker) recordAttempt(
	ctx context.Context,
	payload DeliverNotificationPayload,
	status string,
	httpStatus *int,
	latencyMs *int32,
	retryCount int,
	errorMsg string,
) error {
	var httpStatusInt *int32
	if httpStatus != nil {
		val := int32(*httpStatus)
		httpStatusInt = &val
	}

	var errorStr *string
	if errorMsg != "" {
		errorStr = &errorMsg
	}

	retryCountInt := int32(retryCount)

	params := repo.CreateDeliveryAttemptParams{
		NotificationID: payload.NotificationID,
		SubscriptionID: pgtype.UUID{Bytes: payload.SubscriptionID, Valid: true},
		UserID:         payload.UserID,
		Status:         status,
		HttpStatus:     httpStatusInt,
		LatencyMs:      latencyMs,
		Error:          errorStr,
		RetryCount:     &retryCountInt,
	}

	if _, err := w.repo.CreateDeliveryAttempt(ctx, params); err != nil {
		return fmt.Errorf("failed to create delivery attempt: %w", err)
	}

	return nil
}

// asynqLogger adapts slog.Logger to asynq's Logger interface
type asynqLogger struct {
	logger *slog.Logger
}

func (l *asynqLogger) Debug(args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
	panic(fmt.Sprint(args...))
}
