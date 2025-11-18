package health

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/EricMurray-e-m-dev/StartupMonkey/knowledge/internal/redis"
)

type HealthServer struct {
	redisClient *redis.Client
	server      *http.Server
}

func NewHealthServer(redisClient *redis.Client) *HealthServer {
	return &HealthServer{
		redisClient: redisClient,
	}
}

func (h *HealthServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthCheckHandler)

	h.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return h.server.ListenAndServe()
}

func (h *HealthServer) Shutdown(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}
	return nil
}

func (h *HealthServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Check Redis connection
	if err := h.redisClient.Ping(ctx); err != nil {
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
