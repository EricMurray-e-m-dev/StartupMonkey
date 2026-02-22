// Package health provides HTTP health check endpoints for the Collector service.
package health

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	startTime           time.Time
	unavailableFeatures []string
	mu                  sync.RWMutex
	server              *http.Server
)

func init() {
	startTime = time.Now()
}

// HealthResponse represents the JSON response from the health endpoint.
type HealthResponse struct {
	Status              string   `json:"status"`
	Service             string   `json:"service"`
	UptimeSeconds       int64    `json:"uptime_seconds"`
	Timestamp           int64    `json:"timestamp"`
	UnavailableFeatures []string `json:"unavailable_features,omitempty"`
}

// SetUnavailableFeatures updates the list of unavailable database features.
func SetUnavailableFeatures(features []string) {
	mu.Lock()
	defer mu.Unlock()
	unavailableFeatures = features
}

// StartHealthCheckServer starts the HTTP health check server on the given port.
func StartHealthCheckServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Health check listening on :%s", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Health server failed: %v", err)
		}
	}()
}

// StopHealthCheckServer gracefully stops the health check server.
func StopHealthCheckServer() {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error stopping health server: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	features := unavailableFeatures
	mu.RUnlock()

	response := &HealthResponse{
		Status:              "healthy",
		Service:             "collector",
		UptimeSeconds:       int64(time.Since(startTime).Seconds()),
		Timestamp:           time.Now().Unix(),
		UnavailableFeatures: features,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
