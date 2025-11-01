package apihttp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrorResponse represents API error responses.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// RegisterSubscriptionRequest represents device subscription registration.
type RegisterSubscriptionRequest struct {
	UserID    string                 `json:"user_id"`
	Endpoint  string                 `json:"endpoint"`
	Keys      SubscriptionKeys       `json:"keys"`
	DeviceID  *string                `json:"device_id,omitempty"`
	UserAgent *string                `json:"user_agent,omitempty"`
	Locale    *string                `json:"locale,omitempty"`
	Timezone  *string                `json:"timezone,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionKeys contains the P-256 encryption keys for push.
type SubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// Validate checks RegisterSubscriptionRequest fields.
func (r *RegisterSubscriptionRequest) Validate() error {
	if strings.TrimSpace(r.UserID) == "" {
		return fmt.Errorf("user_id is required")
	}
	if len(r.UserID) > 255 {
		return fmt.Errorf("user_id exceeds 255 characters")
	}
	if strings.TrimSpace(r.Endpoint) == "" {
		return fmt.Errorf("endpoint is required")
	}
	if len(r.Endpoint) > 500 {
		return fmt.Errorf("endpoint exceeds 500 characters")
	}
	if !strings.HasPrefix(r.Endpoint, "https://") {
		return fmt.Errorf("endpoint must be HTTPS URL")
	}
	if strings.TrimSpace(r.Keys.P256dh) == "" {
		return fmt.Errorf("keys.p256dh is required")
	}
	if strings.TrimSpace(r.Keys.Auth) == "" {
		return fmt.Errorf("keys.auth is required")
	}
	if r.Locale != nil && len(*r.Locale) > 10 {
		return fmt.Errorf("locale exceeds 10 characters")
	}
	if r.Timezone != nil && len(*r.Timezone) > 50 {
		return fmt.Errorf("timezone exceeds 50 characters")
	}
	return nil
}

// RegisterSubscriptionResponse represents successful subscription registration.
type RegisterSubscriptionResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// SendNotificationRequest represents a notification send request.
type SendNotificationRequest struct {
	IdempotencyKey *string                `json:"idempotency_key,omitempty"`
	Type           string                 `json:"type"`
	UserIDs        []string               `json:"user_ids"`
	Title          *string                `json:"title,omitempty"`
	Body           *string                `json:"body,omitempty"`
	Icon           *string                `json:"icon,omitempty"`
	URL            *string                `json:"url,omitempty"`
	Locale         *string                `json:"locale,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	DedupeKey      *string                `json:"dedupe_key,omitempty"`
	TTLSeconds     *int                   `json:"ttl_seconds,omitempty"`
	Priority       *string                `json:"priority,omitempty"`
}

// Validate checks SendNotificationRequest fields.
func (r *SendNotificationRequest) Validate() error {
	if strings.TrimSpace(r.Type) == "" {
		return fmt.Errorf("type is required")
	}
	if len(r.Type) > 50 {
		return fmt.Errorf("type exceeds 50 characters")
	}
	if len(r.UserIDs) == 0 {
		return fmt.Errorf("user_ids is required and must contain at least one user")
	}
	if len(r.UserIDs) > 1000 {
		return fmt.Errorf("user_ids exceeds maximum of 1000 recipients")
	}
	for i, uid := range r.UserIDs {
		if strings.TrimSpace(uid) == "" {
			return fmt.Errorf("user_ids[%d] is empty", i)
		}
		if len(uid) > 255 {
			return fmt.Errorf("user_ids[%d] exceeds 255 characters", i)
		}
	}
	if r.Title != nil && len(*r.Title) > 255 {
		return fmt.Errorf("title exceeds 255 characters")
	}
	if r.Body != nil && len(*r.Body) > 1000 {
		return fmt.Errorf("body exceeds 1000 characters")
	}
	if r.Icon != nil && len(*r.Icon) > 500 {
		return fmt.Errorf("icon URL exceeds 500 characters")
	}
	if r.URL != nil && len(*r.URL) > 500 {
		return fmt.Errorf("url exceeds 500 characters")
	}
	if r.Locale != nil && len(*r.Locale) > 10 {
		return fmt.Errorf("locale exceeds 10 characters")
	}
	if r.IdempotencyKey != nil && len(*r.IdempotencyKey) > 255 {
		return fmt.Errorf("idempotency_key exceeds 255 characters")
	}
	if r.DedupeKey != nil && len(*r.DedupeKey) > 255 {
		return fmt.Errorf("dedupe_key exceeds 255 characters")
	}
	if r.TTLSeconds != nil && (*r.TTLSeconds < 0 || *r.TTLSeconds > 2419200) {
		return fmt.Errorf("ttl_seconds must be between 0 and 2419200 (28 days)")
	}
	if r.Priority != nil {
		validPriorities := map[string]bool{"low": true, "normal": true, "high": true, "critical": true}
		if !validPriorities[*r.Priority] {
			return fmt.Errorf("priority must be one of: low, normal, high, critical")
		}
	}
	return nil
}

// DataAsJSON returns the data field as JSON bytes.
func (r *SendNotificationRequest) DataAsJSON() (json.RawMessage, error) {
	if r.Data == nil {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(r.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	return b, nil
}

// SendNotificationResponse represents successful notification creation.
type SendNotificationResponse struct {
	ID             uuid.UUID `json:"id"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	RecipientCount int       `json:"recipient_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetNotificationResponse represents notification status details.
type GetNotificationResponse struct {
	ID             uuid.UUID              `json:"id"`
	Type           string                 `json:"type"`
	Title          *string                `json:"title,omitempty"`
	Body           *string                `json:"body,omitempty"`
	Icon           *string                `json:"icon,omitempty"`
	URL            *string                `json:"url,omitempty"`
	Locale         *string                `json:"locale,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Status         string                 `json:"status"`
	RecipientCount int                    `json:"recipient_count"`
	CreatedAt      time.Time              `json:"created_at"`
}

// DeliveryAttemptResponse represents a single delivery attempt.
type DeliveryAttemptResponse struct {
	ID             uuid.UUID  `json:"id"`
	NotificationID uuid.UUID  `json:"notification_id"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	UserID         string     `json:"user_id"`
	Status         string     `json:"status"`
	HTTPStatus     *int       `json:"http_status,omitempty"`
	LatencyMs      *int       `json:"latency_ms,omitempty"`
	Error          *string    `json:"error,omitempty"`
	RetryCount     *int       `json:"retry_count,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ListDeliveryAttemptsResponse represents list of delivery attempts.
type ListDeliveryAttemptsResponse struct {
	NotificationID uuid.UUID                 `json:"notification_id"`
	Attempts       []DeliveryAttemptResponse `json:"attempts"`
	Total          int                       `json:"total"`
}

// HealthResponse represents health check response.
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// VAPIDPublicKeyResponse represents VAPID public key response.
type VAPIDPublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}
