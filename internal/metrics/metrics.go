package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// NotificationsSent tracks notifications successfully queued
	NotificationsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notifications_sent_total",
			Help: "Total number of notifications queued",
		},
		[]string{"notification_type"},
	)

	// NotificationDeliveries tracks push delivery attempts
	NotificationDeliveries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_deliveries_total",
			Help: "Total number of notification delivery attempts",
		},
		[]string{"status", "notification_type"},
	)

	// NotificationLatency tracks push delivery latency
	NotificationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_delivery_duration_seconds",
			Help:    "Notification delivery latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status", "notification_type"},
	)

	// SubscriptionCount tracks active device subscriptions
	SubscriptionCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_subscriptions_count",
			Help: "Current number of active device subscriptions",
		},
	)

	// QueueSize tracks Asynq queue depth
	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_size",
			Help: "Number of tasks in Asynq queues",
		},
		[]string{"queue", "state"},
	)

	// HTTPRequestDuration tracks API request latency
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestsTotal tracks total API requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
)
