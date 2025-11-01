package webpush

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"

	"notifications/internal/repo"
)

// Sender handles sending Web Push notifications
type Sender struct {
	vapidPublicKey  string
	vapidPrivateKey string
	repo            *repo.Repository
}

// NewSender creates a new Web Push sender
func NewSender(vapidPublicKey, vapidPrivateKey string, repository *repo.Repository) *Sender {
	return &Sender{
		vapidPublicKey:  vapidPublicKey,
		vapidPrivateKey: vapidPrivateKey,
		repo:            repository,
	}
}

// DeliveryResult contains the outcome of a push delivery attempt
type DeliveryResult struct {
	Success     bool
	HTTPStatus  int
	LatencyMs   int
	Error       string
	ShouldPrune bool // True if subscription should be removed (410 Gone)
}

// SendNotification sends a push notification to a specific subscription
func (s *Sender) SendNotification(
	ctx context.Context,
	notificationID uuid.UUID,
	subscriptionID uuid.UUID,
	userID string,
) (*DeliveryResult, error) {
	startTime := time.Now()

	// Get the subscription details
	sub, err := s.repo.GetDeviceSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Check if subscription is active
	if !sub.IsActive {
		return &DeliveryResult{
			Success:     false,
			Error:       "subscription is not active",
			ShouldPrune: false,
		}, nil
	}

	// Get the notification details
	notif, err := s.repo.GetNotification(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Build the push payload
	payload, err := s.buildPayload(notif)
	if err != nil {
		return &DeliveryResult{
			Success: false,
			Error:   fmt.Sprintf("failed to build payload: %v", err),
		}, nil
	}

	// Create the subscription object for webpush-go
	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	// Set up options
	ttl := 3600 // Default TTL in seconds (1 hour)
	if notif.TtlSeconds != nil {
		ttl = int(*notif.TtlSeconds)
	}
	options := &webpush.Options{
		Subscriber:      "mailto:admin@example.com", // TODO: Make configurable
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             ttl,
	}

	// Send the push notification
	resp, err := webpush.SendNotificationWithContext(ctx, payload, subscription, options)

	// Calculate latency
	latencyMs := int(time.Since(startTime).Milliseconds())

	// Handle response
	result := &DeliveryResult{
		LatencyMs: latencyMs,
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode

	// Check response status
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// Success
		result.Success = true

	case resp.StatusCode == http.StatusGone: // 410
		// Subscription expired or invalid - should be pruned
		result.Success = false
		result.Error = "subscription gone (410)"
		result.ShouldPrune = true

	case resp.StatusCode == http.StatusNotFound: // 404
		// Subscription not found - should be pruned
		result.Success = false
		result.Error = "subscription not found (404)"
		result.ShouldPrune = true

	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Client error - likely permanent failure
		result.Success = false
		result.Error = fmt.Sprintf("client error: %d", resp.StatusCode)
		result.ShouldPrune = false // Don't prune on client errors (except 404/410)

	case resp.StatusCode >= 500:
		// Server error - temporary failure, should retry
		result.Success = false
		result.Error = fmt.Sprintf("server error: %d", resp.StatusCode)
		result.ShouldPrune = false

	default:
		result.Success = false
		result.Error = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}

	return result, nil
}

// buildPayload creates the JSON payload for the push notification
func (s *Sender) buildPayload(notif repo.Notification) ([]byte, error) {
	payload := map[string]interface{}{
		"notification_id": notif.ID.String(),
		"type":            notif.Type,
	}

	// Add optional fields
	if notif.Title != nil {
		payload["title"] = *notif.Title
	}
	if notif.Body != nil {
		payload["body"] = *notif.Body
	}
	if notif.Icon != nil {
		payload["icon"] = *notif.Icon
	}
	if notif.Url != nil {
		payload["url"] = *notif.Url
	}
	if notif.Locale != nil {
		payload["locale"] = *notif.Locale
	}

	// Add custom data
	if len(notif.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(notif.Data, &data); err == nil {
			payload["data"] = data
		}
	}

	return json.Marshal(payload)
}
