package apihttp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"notifications/internal/auth"
	"notifications/internal/config"
	"notifications/internal/middleware"
	"notifications/internal/repo"
)

// NewRouter wires routes and middleware.
func NewRouter(cfg config.Config, r *repo.Repository, logger *zap.Logger) http.Handler {
	mux := chi.NewRouter()

	// Global middleware
	mux.Use(middleware.Recovery(logger))
	mux.Use(middleware.RequestLogger(logger))
	mux.Use(corsMiddleware(cfg.CORSAllowedOrigins))

	// Public routes (no auth)
	mux.Get("/healthz", healthCheckHandler(r))
	mux.Get("/metrics", promhttp.Handler().ServeHTTP)
	mux.Get("/v1/push/public-key", vapidPublicKeyHandler(cfg))

	// Protected routes (require HMAC auth)
	h := NewHandler(r, logger)
	mux.Group(func(protected chi.Router) {
		protected.Use(auth.VerifyHMACMiddleware([]byte(cfg.HMACSecret), 5*time.Minute))

		// Subscriptions
		protected.Post("/v1/subscriptions", h.RegisterSubscription)
		protected.Delete("/v1/subscriptions/{id}", h.UnregisterSubscription)

		// Notifications
		protected.Post("/v1/notifications", h.SendNotification)
		protected.Get("/v1/notifications/{id}", h.GetNotification)
		protected.Get("/v1/notifications/{id}/attempts", h.ListDeliveryAttempts)
	})

	return mux
}

// healthCheckHandler returns health status with database check
func healthCheckHandler(r *repo.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := "ok"
		checks := make(map[string]string)

		// Check database
		if err := r.Health(req.Context()); err != nil {
			status = "degraded"
			checks["database"] = "unhealthy: " + err.Error()
		} else {
			checks["database"] = "healthy"
		}

		resp := HealthResponse{
			Status:    status,
			Timestamp: time.Now(),
			Checks:    checks,
		}

		if status == "degraded" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		_ = json.NewEncoder(w).Encode(resp)
	}
}

// vapidPublicKeyHandler returns the VAPID public key for browser subscriptions
func vapidPublicKeyHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cfg.VAPIDPublicKey == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Error: "VAPID public key not configured",
				Code:  "NOT_CONFIGURED",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(VAPIDPublicKeyResponse{
			PublicKey: cfg.VAPIDPublicKey,
		})
	}
}

func corsMiddleware(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if originAllowed(origin, allowed) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Signature,X-Timestamp")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" || strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}
