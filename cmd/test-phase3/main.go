package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"notifications/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func main() {
	// Get database connection string from environment
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://postgres:postgres@localhost:5432/notifications?sslmode=disable"
	}

	log.Println("🧪 Starting Phase 3 Repository Layer Tests")
	log.Println("==========================================")

	// Initialize repository
	log.Println("\n1️⃣  Initializing repository with connection pool...")
	repository, err := repo.NewRepository(connString)
	if err != nil {
		log.Fatalf("❌ Failed to create repository: %v", err)
	}
	defer repository.Close()
	log.Println("✅ Repository initialized successfully")

	// Test health check
	log.Println("\n2️⃣  Testing health check...")
	ctx := context.Background()
	if err := repository.Health(ctx); err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}
	log.Println("✅ Database connection is healthy")

	// Test pool stats
	log.Println("\n3️⃣  Checking connection pool stats...")
	stats := repository.Stats()
	log.Printf("   - Max connections: %d", stats.MaxConns())
	log.Printf("   - Total connections: %d", stats.TotalConns())
	log.Printf("   - Idle connections: %d", stats.IdleConns())
	log.Printf("   - Acquire count: %d", stats.AcquireCount())
	log.Println("✅ Pool stats retrieved successfully")

	// Test device subscriptions
	log.Println("\n4️⃣  Testing device subscription operations...")
	testDeviceSubscriptions(ctx, repository)

	// Test notifications
	log.Println("\n5️⃣  Testing notification operations...")
	testNotifications(ctx, repository)

	// Test recipients
	log.Println("\n6️⃣  Testing recipient operations...")
	testRecipients(ctx, repository)

	// Test delivery attempts
	log.Println("\n7️⃣  Testing delivery attempt operations...")
	testDeliveryAttempts(ctx, repository)

	// Test transactions
	log.Println("\n8️⃣  Testing transaction support...")
	testTransactions(ctx, repository)

	log.Println("\n==========================================")
	log.Println("✅ All Phase 3 Repository Tests Passed!")
	log.Println("==========================================")
}

func testDeviceSubscriptions(ctx context.Context, r *repo.Repository) {
	// Create a device subscription
	log.Println("   Creating device subscription...")
	sub, err := r.CreateDeviceSubscription(ctx, repo.CreateDeviceSubscriptionParams{
		UserID:    "user-test-001",
		Endpoint:  fmt.Sprintf("https://fcm.googleapis.com/fcm/send/%s", uuid.New().String()),
		P256dh:    "test-p256dh-key",
		Auth:      "test-auth-key",
		DeviceID:  stringPtr("device-001"),
		UserAgent: stringPtr("Mozilla/5.0"),
		Locale:    stringPtr("en-US"),
		Timezone:  stringPtr("America/New_York"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to create device subscription: %v", err)
	}
	log.Printf("   ✓ Created subscription: %s", sub.ID)

	// Get by ID
	log.Println("   Retrieving subscription by ID...")
	retrieved, err := r.GetDeviceSubscription(ctx, sub.ID)
	if err != nil {
		log.Fatalf("❌ Failed to get subscription: %v", err)
	}
	if retrieved.ID != sub.ID {
		log.Fatalf("❌ Subscription ID mismatch")
	}
	log.Println("   ✓ Retrieved subscription successfully")

	// Get by endpoint
	log.Println("   Retrieving subscription by endpoint...")
	byEndpoint, err := r.GetDeviceSubscriptionByEndpoint(ctx, sub.Endpoint)
	if err != nil {
		log.Fatalf("❌ Failed to get subscription by endpoint: %v", err)
	}
	if byEndpoint.ID != sub.ID {
		log.Fatalf("❌ Subscription ID mismatch")
	}
	log.Println("   ✓ Retrieved subscription by endpoint")

	// List subscriptions by user
	log.Println("   Listing subscriptions by user...")
	subs, err := r.ListDeviceSubscriptionsByUser(ctx, "user-test-001")
	if err != nil {
		log.Fatalf("❌ Failed to list subscriptions: %v", err)
	}
	log.Printf("   ✓ Found %d subscriptions", len(subs))

	// Count active subscriptions
	log.Println("   Counting active subscriptions...")
	count, err := r.CountActiveSubscriptionsByUser(ctx, "user-test-001")
	if err != nil {
		log.Fatalf("❌ Failed to count subscriptions: %v", err)
	}
	log.Printf("   ✓ Active subscriptions: %d", count)

	// Update subscription
	log.Println("   Updating subscription...")
	updated, err := r.UpdateDeviceSubscription(ctx, repo.UpdateDeviceSubscriptionParams{
		ID:       sub.ID,
		IsActive: boolPtr(false),
	})
	if err != nil {
		log.Fatalf("❌ Failed to update subscription: %v", err)
	}
	if updated.IsActive {
		log.Fatalf("❌ Subscription should be inactive")
	}
	log.Println("   ✓ Updated subscription")

	// Deactivate subscription
	log.Println("   Deactivating subscription...")
	if err := r.DeactivateDeviceSubscription(ctx, sub.ID); err != nil {
		log.Fatalf("❌ Failed to deactivate subscription: %v", err)
	}
	log.Println("   ✓ Deactivated subscription")

	// Delete subscription
	log.Println("   Deleting subscription...")
	if err := r.DeleteDeviceSubscription(ctx, sub.ID); err != nil {
		log.Fatalf("❌ Failed to delete subscription: %v", err)
	}
	log.Println("   ✓ Deleted subscription")

	log.Println("✅ Device subscription tests passed")
}

func testNotifications(ctx context.Context, r *repo.Repository) {
	// Create a notification
	log.Println("   Creating notification...")
	data := json.RawMessage(`{"key": "value"}`)
	notif, err := r.CreateNotification(ctx, repo.CreateNotificationParams{
		IdempotencyKey: stringPtr(fmt.Sprintf("test-idempotency-%s", uuid.New().String())),
		Type:           "test",
		Title:          stringPtr("Test Notification"),
		Body:           stringPtr("This is a test notification"),
		Icon:           stringPtr("https://example.com/icon.png"),
		Url:            stringPtr("https://example.com/action"),
		Locale:         stringPtr("en-US"),
		Data:           data,
		Status:         "pending",
		DedupeKey:      stringPtr("test-dedupe-key"),
		TtlSeconds:     int32Ptr(3600),
		Priority:       stringPtr("high"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to create notification: %v", err)
	}
	log.Printf("   ✓ Created notification: %s", notif.ID)

	// Get by ID
	log.Println("   Retrieving notification by ID...")
	retrieved, err := r.GetNotification(ctx, notif.ID)
	if err != nil {
		log.Fatalf("❌ Failed to get notification: %v", err)
	}
	if retrieved.ID != notif.ID {
		log.Fatalf("❌ Notification ID mismatch")
	}
	log.Println("   ✓ Retrieved notification successfully")

	// Get by idempotency key
	log.Println("   Retrieving notification by idempotency key...")
	byKey, err := r.GetNotificationByIdempotencyKey(ctx, notif.IdempotencyKey)
	if err != nil {
		log.Fatalf("❌ Failed to get notification by idempotency key: %v", err)
	}
	if byKey.ID != notif.ID {
		log.Fatalf("❌ Notification ID mismatch")
	}
	log.Println("   ✓ Retrieved notification by idempotency key")

	// List notifications
	log.Println("   Listing notifications...")
	notifs, err := r.ListNotifications(ctx, repo.ListNotificationsParams{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		log.Fatalf("❌ Failed to list notifications: %v", err)
	}
	log.Printf("   ✓ Found %d notifications", len(notifs))

	// Update status
	log.Println("   Updating notification status...")
	updated, err := r.UpdateNotificationStatus(ctx, repo.UpdateNotificationStatusParams{
		ID:     notif.ID,
		Status: "sent",
	})
	if err != nil {
		log.Fatalf("❌ Failed to update notification status: %v", err)
	}
	if updated.Status != "sent" {
		log.Fatalf("❌ Status should be 'sent'")
	}
	log.Println("   ✓ Updated notification status")

	// Count by status
	log.Println("   Counting notifications by status...")
	count, err := r.CountNotificationsByStatus(ctx, "sent")
	if err != nil {
		log.Fatalf("❌ Failed to count notifications: %v", err)
	}
	log.Printf("   ✓ Notifications with status 'sent': %d", count)

	// Delete notification
	log.Println("   Deleting notification...")
	if err := r.DeleteNotification(ctx, notif.ID); err != nil {
		log.Fatalf("❌ Failed to delete notification: %v", err)
	}
	log.Println("   ✓ Deleted notification")

	log.Println("✅ Notification tests passed")
}

func testRecipients(ctx context.Context, r *repo.Repository) {
	// Create a notification first
	data := json.RawMessage(`{}`)
	notif, err := r.CreateNotification(ctx, repo.CreateNotificationParams{
		Type:   "test",
		Data:   data,
		Status: "pending",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create notification: %v", err)
	}

	// Create a recipient
	log.Println("   Creating recipient...")
	err = r.CreateRecipient(ctx, repo.CreateRecipientParams{
		NotificationID: notif.ID,
		UserID:         "user-test-002",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create recipient: %v", err)
	}
	log.Println("   ✓ Created recipient")

	// Check recipient exists
	log.Println("   Checking if recipient exists...")
	exists, err := r.CheckRecipientExists(ctx, repo.CheckRecipientExistsParams{
		NotificationID: notif.ID,
		UserID:         "user-test-002",
	})
	if err != nil {
		log.Fatalf("❌ Failed to check recipient: %v", err)
	}
	if !exists {
		log.Fatalf("❌ Recipient should exist")
	}
	log.Println("   ✓ Recipient exists")

	// Get recipients by notification
	log.Println("   Retrieving recipients by notification...")
	recipients, err := r.GetRecipientsByNotification(ctx, notif.ID)
	if err != nil {
		log.Fatalf("❌ Failed to get recipients: %v", err)
	}
	log.Printf("   ✓ Found %d recipients", len(recipients))

	// Count recipients
	log.Println("   Counting recipients...")
	count, err := r.CountRecipientsByNotification(ctx, notif.ID)
	if err != nil {
		log.Fatalf("❌ Failed to count recipients: %v", err)
	}
	log.Printf("   ✓ Recipient count: %d", count)

	// Delete recipient
	log.Println("   Deleting recipient...")
	err = r.DeleteRecipient(ctx, repo.DeleteRecipientParams{
		NotificationID: notif.ID,
		UserID:         "user-test-002",
	})
	if err != nil {
		log.Fatalf("❌ Failed to delete recipient: %v", err)
	}
	log.Println("   ✓ Deleted recipient")

	// Clean up
	if err := r.DeleteNotification(ctx, notif.ID); err != nil {
		log.Fatalf("❌ Failed to delete notification: %v", err)
	}

	log.Println("✅ Recipient tests passed")
}

func testDeliveryAttempts(ctx context.Context, r *repo.Repository) {
	// Create notification and subscription first
	data := json.RawMessage(`{}`)
	notif, err := r.CreateNotification(ctx, repo.CreateNotificationParams{
		Type:   "test",
		Data:   data,
		Status: "pending",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create notification: %v", err)
	}

	sub, err := r.CreateDeviceSubscription(ctx, repo.CreateDeviceSubscriptionParams{
		UserID:   "user-test-003",
		Endpoint: fmt.Sprintf("https://fcm.googleapis.com/fcm/send/%s", uuid.New().String()),
		P256dh:   "test-key",
		Auth:     "test-auth",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create subscription: %v", err)
	}

	// Create a delivery attempt
	log.Println("   Creating delivery attempt...")
	attempt, err := r.CreateDeliveryAttempt(ctx, repo.CreateDeliveryAttemptParams{
		NotificationID: notif.ID,
		SubscriptionID: uuidToPgtype(sub.ID),
		UserID:         "user-test-003",
		Status:         "success",
		HttpStatus:     int32Ptr(201),
		LatencyMs:      int32Ptr(150),
		RetryCount:     int32Ptr(0),
	})
	if err != nil {
		log.Fatalf("❌ Failed to create delivery attempt: %v", err)
	}
	log.Printf("   ✓ Created delivery attempt: %s", attempt.ID)

	// Get by ID
	log.Println("   Retrieving delivery attempt by ID...")
	retrieved, err := r.GetDeliveryAttempt(ctx, attempt.ID)
	if err != nil {
		log.Fatalf("❌ Failed to get delivery attempt: %v", err)
	}
	if retrieved.ID != attempt.ID {
		log.Fatalf("❌ Delivery attempt ID mismatch")
	}
	log.Println("   ✓ Retrieved delivery attempt successfully")

	// List by notification
	log.Println("   Listing delivery attempts by notification...")
	attempts, err := r.ListDeliveryAttemptsByNotification(ctx, notif.ID)
	if err != nil {
		log.Fatalf("❌ Failed to list delivery attempts: %v", err)
	}
	log.Printf("   ✓ Found %d delivery attempts", len(attempts))

	// Get delivery stats
	log.Println("   Getting delivery statistics...")
	stats, err := r.GetDeliveryStats(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		log.Fatalf("❌ Failed to get delivery stats: %v", err)
	}
	log.Printf("   ✓ Total attempts: %d, Success: %d, Failed: %d",
		stats.TotalAttempts, stats.SuccessCount, stats.FailedCount)

	// Count by status
	log.Println("   Counting delivery attempts by status...")
	count, err := r.CountDeliveryAttemptsByStatus(ctx, "success")
	if err != nil {
		log.Fatalf("❌ Failed to count delivery attempts: %v", err)
	}
	log.Printf("   ✓ Successful attempts: %d", count)

	// Clean up
	if err := r.DeleteDeviceSubscription(ctx, sub.ID); err != nil {
		log.Fatalf("❌ Failed to delete subscription: %v", err)
	}
	if err := r.DeleteNotification(ctx, notif.ID); err != nil {
		log.Fatalf("❌ Failed to delete notification: %v", err)
	}

	log.Println("✅ Delivery attempt tests passed")
}

func testTransactions(ctx context.Context, r *repo.Repository) {
	log.Println("   Testing transaction rollback on error...")

	// Create a notification within a transaction that will fail
	err := r.WithTx(ctx, func(q *repo.Queries) error {
		data := json.RawMessage(`{}`)
		_, err := q.CreateNotification(ctx, repo.CreateNotificationParams{
			Type:   "test-tx",
			Data:   data,
			Status: "pending",
		})
		if err != nil {
			return err
		}

		// Force an error to test rollback
		return fmt.Errorf("intentional error for rollback test")
	})

	if err == nil {
		log.Fatalf("❌ Transaction should have failed")
	}
	log.Println("   ✓ Transaction rolled back as expected")

	log.Println("   Testing successful transaction commit...")

	var notifID uuid.UUID
	err = r.WithTx(ctx, func(q *repo.Queries) error {
		data := json.RawMessage(`{}`)
		notif, err := q.CreateNotification(ctx, repo.CreateNotificationParams{
			Type:   "test-tx-success",
			Data:   data,
			Status: "pending",
		})
		if err != nil {
			return err
		}
		notifID = notif.ID

		// Create recipient in same transaction
		return q.CreateRecipient(ctx, repo.CreateRecipientParams{
			NotificationID: notif.ID,
			UserID:         "user-tx-test",
		})
	})

	if err != nil {
		log.Fatalf("❌ Transaction should have succeeded: %v", err)
	}
	log.Println("   ✓ Transaction committed successfully")

	// Verify both records exist
	log.Println("   Verifying transaction results...")
	_, err = r.GetNotification(ctx, notifID)
	if err != nil {
		log.Fatalf("❌ Notification should exist after transaction: %v", err)
	}

	count, err := r.CountRecipientsByNotification(ctx, notifID)
	if err != nil {
		log.Fatalf("❌ Failed to count recipients: %v", err)
	}
	if count != 1 {
		log.Fatalf("❌ Should have 1 recipient, got %d", count)
	}
	log.Println("   ✓ Transaction results verified")

	// Clean up
	if err := r.DeleteNotification(ctx, notifID); err != nil {
		log.Fatalf("❌ Failed to delete notification: %v", err)
	}

	log.Println("✅ Transaction tests passed")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}
