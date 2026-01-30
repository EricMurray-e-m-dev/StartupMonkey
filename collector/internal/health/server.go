package health

import (
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
)

func init() {
	startTime = time.Now()
}

type HealthResponse struct {
	Status              string   `json:"status"`
	Service             string   `json:"service"`
	UptimeSeconds       int64    `json:"uptime_seconds"`
	Timestamp           int64    `json:"timestamp"`
	UnavailableFeatures []string `json:"unavailable_features,omitempty"`
}

func SetUnavailableFeatures(features []string) {
	mu.Lock()
	defer mu.Unlock()
	unavailableFeatures = features
}

func StartHealthCheckServer(port string) {
	http.HandleFunc("/health", healthHandler)

	log.Printf("Health check listening on : %s", port)

	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Health server failed: %v", err)
		}
	}()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	features := unavailableFeatures
	mu.RUnlock()

	response := &HealthResponse{
		Status:              "healthy",
		Service:             "collector",
		UptimeSeconds:       int64(time.Since(startTime).Seconds()),
		Timestamp:           int64(time.Now().Unix()),
		UnavailableFeatures: features,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
