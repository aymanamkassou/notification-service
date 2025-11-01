package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Task types
const (
	TypeDeliverNotification = "notification:deliver"
)

// Task priorities
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// DeliverNotificationPayload contains the data needed to deliver a notification
type DeliverNotificationPayload struct {
	NotificationID uuid.UUID `json:"notification_id"`
	UserID         string    `json:"user_id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

// Client handles enqueuing tasks to Redis/Asynq
type Client struct {
	client *asynq.Client
}

// NewClient creates a new queue client
func NewClient(redisAddr string) *Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddr,
	})

	return &Client{
		client: client,
	}
}

// Close closes the queue client
func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueDeliverNotification enqueues a notification delivery task
func (c *Client) EnqueueDeliverNotification(
	ctx context.Context,
	notificationID uuid.UUID,
	userID string,
	subscriptionID uuid.UUID,
	priority string,
	ttlSeconds int,
) error {
	payload := DeliverNotificationPayload{
		NotificationID: notificationID,
		UserID:         userID,
		SubscriptionID: subscriptionID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeDeliverNotification, data)

	// Configure task options
	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Timeout(30 * time.Second),
	}

	// Set priority
	switch priority {
	case PriorityHigh:
		opts = append(opts, asynq.Queue("high"))
	case PriorityLow:
		opts = append(opts, asynq.Queue("low"))
	default:
		opts = append(opts, asynq.Queue("default"))
	}

	// Set TTL retention time (how long the task info is kept after processing)
	if ttlSeconds > 0 {
		opts = append(opts, asynq.Retention(time.Duration(ttlSeconds)*time.Second))
	}

	// Enqueue the task
	info, err := c.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	_ = info // Task enqueued successfully

	return nil
}
