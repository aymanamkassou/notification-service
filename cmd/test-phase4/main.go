package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"notifications/internal/config"
)

const (
	baseURL = "http://localhost:8080"
	green   = "\033[32m"
	red     = "\033[31m"
	yellow  = "\033[33m"
	reset   = "\033[0m"
)

var (
	hmacSecret  string
	testsPassed int
	testsFailed int
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	hmacSecret = cfg.HMACSecret

	fmt.Println("\n=== Phase 4 API Testing ===\n")

	// Test 1: Health Check (no auth)
	test("Health Check", testHealthCheck)

	// Test 2: VAPID Public Key (no auth)
	test("VAPID Public Key", testVAPIDPublicKey)

	// Test 3: Metrics Endpoint (no auth)
	test("Metrics Endpoint", testMetrics)

	// Test 4: Register Subscription - Invalid JSON
	test("Register Subscription - Invalid JSON", testRegisterSubscriptionInvalidJSON)

	// Test 5: Register Subscription - Missing Fields
	test("Register Subscription - Missing Fields", testRegisterSubscriptionMissingFields)

	// Test 6: Register Subscription - Valid
	subID := test("Register Subscription - Valid", testRegisterSubscriptionValid)

	// Test 7: Register Subscription - Idempotency
	test("Register Subscription - Idempotency", func() (string, error) {
		return testRegisterSubscriptionIdempotency(subID)
	})

	// Test 8: Unregister Subscription
	test("Unregister Subscription", func() (string, error) {
		return testUnregisterSubscription(subID)
	})

	// Test 9: Unregister Non-Existent Subscription
	test("Unregister Non-Existent Subscription", testUnregisterNonExistentSubscription)

	// Test 10: Send Notification - Invalid JSON
	test("Send Notification - Invalid JSON", testSendNotificationInvalidJSON)

	// Test 11: Send Notification - Missing Fields
	test("Send Notification - Missing Fields", testSendNotificationMissingFields)

	// Test 12: Send Notification - Valid
	notifID := test("Send Notification - Valid", testSendNotificationValid)

	// Test 13: Send Notification - Idempotency
	test("Send Notification - Idempotency", func() (string, error) {
		return testSendNotificationIdempotency(notifID)
	})

	// Test 14: Get Notification
	test("Get Notification", func() (string, error) {
		return testGetNotification(notifID)
	})

	// Test 15: Get Non-Existent Notification
	test("Get Non-Existent Notification", testGetNonExistentNotification)

	// Test 16: List Delivery Attempts
	test("List Delivery Attempts", func() (string, error) {
		return testListDeliveryAttempts(notifID)
	})

	// Test 17: Missing HMAC Headers
	test("Missing HMAC Headers", testMissingHMACHeaders)

	// Test 18: Invalid HMAC Signature
	test("Invalid HMAC Signature", testInvalidHMACSignature)

	// Summary
	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("%sPassed: %d%s\n", green, testsPassed, reset)
	fmt.Printf("%sFailed: %d%s\n", red, testsFailed, reset)
	fmt.Printf("Total: %d\n\n", testsPassed+testsFailed)

	if testsFailed > 0 {
		os.Exit(1)
	}
}

func test(name string, fn func() (string, error)) string {
	result, err := fn()
	if err != nil {
		fmt.Printf("%s✗ %s: %v%s\n", red, name, err, reset)
		testsFailed++
		return ""
	}
	fmt.Printf("%s✓ %s%s\n", green, name, reset)
	testsPassed++
	return result
}

func testHealthCheck() (string, error) {
	resp, err := http.Get(baseURL + "/healthz")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result["status"] != "ok" {
		return "", fmt.Errorf("expected status 'ok', got '%v'", result["status"])
	}

	return "", nil
}

func testVAPIDPublicKey() (string, error) {
	resp, err := http.Get(baseURL + "/v1/push/public-key")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result["public_key"] == "" {
		return "", fmt.Errorf("public_key is empty")
	}

	return result["public_key"], nil
}

func testMetrics() (string, error) {
	resp, err := http.Get(baseURL + "/metrics")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return "", fmt.Errorf("metrics response is empty")
	}

	return "", nil
}

func testRegisterSubscriptionInvalidJSON() (string, error) {
	body := []byte(`{invalid json}`)
	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		return "", fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return "", nil
}

func testRegisterSubscriptionMissingFields() (string, error) {
	payload := map[string]interface{}{
		"user_id": "user123",
		// Missing endpoint and keys
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		return "", fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return "", nil
}

func testRegisterSubscriptionValid() (string, error) {
	payload := map[string]interface{}{
		"user_id":  "user123",
		"endpoint": "https://fcm.googleapis.com/fcm/send/test123",
		"keys": map[string]string{
			"p256dh": "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
			"auth":   "tBHItJI5svbpez7KI4CCXg",
		},
		"device_id":  "device123",
		"user_agent": "Mozilla/5.0",
		"locale":     "en-US",
		"timezone":   "America/New_York",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	subID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("subscription ID not found in response")
	}

	return subID, nil
}

func testRegisterSubscriptionIdempotency(originalID string) (string, error) {
	payload := map[string]interface{}{
		"user_id":  "user123",
		"endpoint": "https://fcm.googleapis.com/fcm/send/test123",
		"keys": map[string]string{
			"p256dh": "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
			"auth":   "tBHItJI5svbpez7KI4CCXg",
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200 (idempotent), got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	returnedID := result["id"].(string)
	if returnedID != originalID {
		return "", fmt.Errorf("expected same subscription ID, got different ID")
	}

	return returnedID, nil
}

func testUnregisterSubscription(subID string) (string, error) {
	req, _ := http.NewRequest("DELETE", baseURL+"/v1/subscriptions/"+subID, nil)
	signRequest(req, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("expected status 204, got %d", resp.StatusCode)
	}

	return "", nil
}

func testUnregisterNonExistentSubscription() (string, error) {
	fakeID := uuid.New().String()
	req, _ := http.NewRequest("DELETE", baseURL+"/v1/subscriptions/"+fakeID, nil)
	signRequest(req, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		return "", fmt.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	return "", nil
}

func testSendNotificationInvalidJSON() (string, error) {
	body := []byte(`{invalid json}`)
	req, _ := http.NewRequest("POST", baseURL+"/v1/notifications", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		return "", fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return "", nil
}

func testSendNotificationMissingFields() (string, error) {
	payload := map[string]interface{}{
		"type": "promotion",
		// Missing user_ids
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/notifications", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		return "", fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return "", nil
}

func testSendNotificationValid() (string, error) {
	idempotencyKey := fmt.Sprintf("test-%d", time.Now().Unix())
	payload := map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"type":            "promotion",
		"user_ids":        []string{"user123", "user456"},
		"title":           "Test Notification",
		"body":            "This is a test notification",
		"icon":            "https://example.com/icon.png",
		"url":             "https://example.com/promo",
		"locale":          "en-US",
		"data": map[string]interface{}{
			"campaign_id": "camp123",
		},
		"ttl_seconds": 86400,
		"priority":    "high",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/notifications", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	notifID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("notification ID not found in response")
	}

	return notifID, nil
}

func testSendNotificationIdempotency(originalID string) (string, error) {
	// Use a known idempotency key to test idempotency
	payload := map[string]interface{}{
		"idempotency_key": "test-idempotency",
		"type":            "promotion",
		"user_ids":        []string{"user789"},
		"title":           "Idempotency Test",
		"body":            "This should be idempotent",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/notifications", bytes.NewReader(body))
	signRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// First call should create
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("expected status 201 on first call, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	firstID := result["id"].(string)

	// Second call with same idempotency key
	req2, _ := http.NewRequest("POST", baseURL+"/v1/notifications", bytes.NewReader(body))
	signRequest(req2, body)

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", fmt.Errorf("second request failed: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200 (idempotent) on second call, got %d", resp2.StatusCode)
	}

	var result2 map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		return "", fmt.Errorf("failed to decode second response: %w", err)
	}

	secondID := result2["id"].(string)
	if firstID != secondID {
		return "", fmt.Errorf("expected same notification ID, got different IDs")
	}

	return firstID, nil
}

func testGetNotification(notifID string) (string, error) {
	req, _ := http.NewRequest("GET", baseURL+"/v1/notifications/"+notifID, nil)
	signRequest(req, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result["id"] != notifID {
		return "", fmt.Errorf("expected notification ID %s, got %v", notifID, result["id"])
	}

	return "", nil
}

func testGetNonExistentNotification() (string, error) {
	fakeID := uuid.New().String()
	req, _ := http.NewRequest("GET", baseURL+"/v1/notifications/"+fakeID, nil)
	signRequest(req, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		return "", fmt.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	return "", nil
}

func testListDeliveryAttempts(notifID string) (string, error) {
	req, _ := http.NewRequest("GET", baseURL+"/v1/notifications/"+notifID+"/attempts", nil)
	signRequest(req, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result["notification_id"] != notifID {
		return "", fmt.Errorf("expected notification ID %s, got %v", notifID, result["notification_id"])
	}

	return "", nil
}

func testMissingHMACHeaders() (string, error) {
	payload := map[string]interface{}{
		"user_id":  "user123",
		"endpoint": "https://fcm.googleapis.com/fcm/send/test",
		"keys": map[string]string{
			"p256dh": "test",
			"auth":   "test",
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	// Don't sign the request

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	return "", nil
}

func testInvalidHMACSignature() (string, error) {
	payload := map[string]interface{}{
		"user_id":  "user123",
		"endpoint": "https://fcm.googleapis.com/fcm/send/test",
		"keys": map[string]string{
			"p256dh": "test",
			"auth":   "test",
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/v1/subscriptions", bytes.NewReader(body))
	timestamp := time.Now().Format(time.RFC3339)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", "invalid-signature")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	return "", nil
}

// signRequest signs the HTTP request with HMAC-SHA256
func signRequest(req *http.Request, body []byte) {
	timestamp := time.Now().Format(time.RFC3339)

	h := hmac.New(sha256.New, []byte(hmacSecret))
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.Path))
	if body != nil {
		h.Write(body)
	}
	h.Write([]byte(timestamp))

	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("Content-Type", "application/json")
}
