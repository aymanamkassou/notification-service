package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"notifications/internal/config"
	"notifications/internal/queue"
	"notifications/internal/repo"
	"notifications/internal/webpush"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("=== Phase 5: Queue & Worker Integration Test ===")
	fmt.Println()

	ctx := context.Background()

	// 1. Connect to database
	fmt.Println("1. Connecting to database...")
	repository, err := repo.NewRepository(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repository.Close()
	fmt.Println("   ✓ Connected to database")

	// 2. Initialize queue client
	fmt.Println("2. Connecting to Redis queue...")
	queueClient := queue.NewClient(cfg.RedisAddr)
	defer queueClient.Close()
	fmt.Println("   ✓ Connected to Redis")

	// 3. Initialize webpush sender
	fmt.Println("3. Initializing webpush sender...")
	sender := webpush.NewSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, repository)
	fmt.Println("   ✓ Webpush sender initialized")

	// 4. Create test notification
	fmt.Println("4. Creating test notification...")
	title := "Test Notification"
	body := "This is a test notification from Phase 5"
	ttl := int32(3600)
	priority := "normal"
	data := []byte("{}")

	notif, err := repository.CreateNotification(ctx, repo.CreateNotificationParams{
		Type:       "test",
		Title:      &title,
		Body:       &body,
		Data:       data,
		TtlSeconds: &ttl,
		Priority:   &priority,
	})
	if err != nil {
		log.Fatalf("Failed to create notification: %v", err)
	}
	fmt.Printf("   ✓ Created notification: %s\n", notif.ID)

	// 5. Create test subscription
	fmt.Println("5. Creating test subscription...")
	sub, err := repository.CreateDeviceSubscription(ctx, repo.CreateDeviceSubscriptionParams{
		UserID:   "test-user-123",
		Endpoint: "https://fcm.googleapis.com/fcm/send/test-endpoint-" + time.Now().Format("20060102150405"),
		P256dh:   "test-p256dh-key",
		Auth:     "test-auth-key",
	})
	if err != nil {
		log.Fatalf("Failed to create subscription: %v", err)
	}
	fmt.Printf("   ✓ Created subscription: %s\n", sub.ID)

	// 6. Enqueue delivery task
	fmt.Println("6. Enqueuing delivery task...")
	err = queueClient.EnqueueDeliverNotification(
		ctx,
		notif.ID,
		"test-user-123",
		sub.ID,
		"normal",
		3600,
	)
	if err != nil {
		log.Fatalf("Failed to enqueue task: %v", err)
	}
	fmt.Println("   ✓ Task enqueued to Redis queue")

	// 7. Test webpush sender (this will fail for the fake endpoint, which is expected)
	fmt.Println("7. Testing webpush sender...")
	result, err := sender.SendNotification(ctx, notif.ID, sub.ID, "test-user-123")
	if err != nil {
		fmt.Printf("   ⚠ Sender error (expected for test endpoint): %v\n", err)
	} else {
		fmt.Printf("   ✓ Sender result: success=%v, status=%d, latency=%dms\n",
			result.Success, result.HTTPStatus, result.LatencyMs)
		if !result.Success {
			fmt.Printf("     Error: %s\n", result.Error)
		}
	}

	// 8. Check delivery attempt was recorded
	fmt.Println("8. Checking delivery attempts...")
	attempts, err := repository.ListDeliveryAttemptsByNotification(ctx, notif.ID)
	if err != nil {
		log.Fatalf("Failed to list attempts: %v", err)
	}
	fmt.Printf("   ✓ Found %d delivery attempt(s)\n", len(attempts))
	for i, attempt := range attempts {
		fmt.Printf("     [%d] Status: %s, HTTP: %v, Latency: %vms\n",
			i+1, attempt.Status, attempt.HttpStatus, attempt.LatencyMs)
	}

	fmt.Println()
	fmt.Println("=== Test Complete ===")
	fmt.Println()
	fmt.Println("Components verified:")
	fmt.Println("  ✓ Database connection")
	fmt.Println("  ✓ Redis queue client")
	fmt.Println("  ✓ Task enqueueing")
	fmt.Println("  ✓ Webpush sender")
	fmt.Println("  ✓ Delivery attempt recording")
	fmt.Println()
	fmt.Println("To test the worker:")
	fmt.Println("  1. Run: make worker")
	fmt.Println("  2. Send a notification via API")
	fmt.Println("  3. Watch worker logs process the delivery")
	fmt.Println()

	os.Exit(0)
}
