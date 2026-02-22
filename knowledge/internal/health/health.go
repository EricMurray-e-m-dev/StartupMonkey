// Package health provides HTTP health check endpoints for the Knowledge service.
package health

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
)

// HealthServer provides HTTP health check endpoints.
type HealthServer struct {
	redisClient *redis.Client
	server      *http.Server
}

// NewHealthServer creates a new HealthServer instance.
func NewHealthServer(redisClient *redis.Client) *HealthServer {
	return &HealthServer{
		redisClient: redisClient,
	}
}

// Start begins listening for health check requests on the given address.
func (h *HealthServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthCheckHandler)

	h.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return h.server.ListenAndServe()
}

// Shutdown gracefully stops the health server.
func (h *HealthServer) Shutdown(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}
	return nil
}

func (h *HealthServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.redisClient.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"redis":  "disconnected",
			"error":  err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"redis":  "connected",
	})
}
