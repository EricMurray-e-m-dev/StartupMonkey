package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/actions"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
)

type Server struct {
	detectionHandler *handler.DetectionHandler
	httpServer       *http.Server // Store server instance for graceful shutdown
}

func NewServer(dh *handler.DetectionHandler) *Server {
	return &Server{
		detectionHandler: dh,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Rollback endpoint
	mux.HandleFunc("/api/actions/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		s.handleRollback(w, r)
	})

	// Deploy Redis endpoint
	mux.HandleFunc("/api/deploy-redis", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received deploy request: %s %s", r.Method, r.URL.Path)
		s.handleDeployRedis(w, r)
	})

	// Store server instance for graceful shutdown
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.enableCORS(mux),
	}

	log.Printf("HTTP Server listening on: %s", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server with a timeout.
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	log.Printf("Stopping HTTP server...")

	// 5 second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	log.Printf("HTTP server stopped successfully")
	return nil
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] != "rollback" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	actionID := parts[3]

	log.Printf("Rollback request on action: %s", actionID)

	result, err := s.detectionHandler.RollbackAction(actionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeployRedisRequest represents the JSON payload for Redis deployment
type DeployRedisRequest struct {
	DatabaseID     string `json:"database_id"`
	Port           string `json:"port"`
	MaxMemory      string `json:"max_memory"`
	EvictionPolicy string `json:"eviction_policy"`
}

func (s *Server) handleDeployRedis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req DeployRedisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to parse deploy request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DatabaseID == "" {
		http.Error(w, "database_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Deploy Redis request for database: %s", req.DatabaseID)

	// Generate action ID
	actionID := fmt.Sprintf("action-%d", time.Now().UnixNano())
	detectionID := fmt.Sprintf("manual-redis-%d", time.Now().UnixNano())

	// Create action parameters
	params := map[string]interface{}{
		"port":            req.Port,
		"max_memory":      req.MaxMemory,
		"eviction_policy": req.EvictionPolicy,
	}

	// Create DeployRedisAction
	action, err := actions.NewDeployRedisAction(
		actionID,
		detectionID,
		req.DatabaseID,
		params,
	)
	if err != nil {
		log.Printf("Failed to create Redis action: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create action: %v", err), http.StatusInternalServerError)
		return
	}

	// Create detection for logging/tracking
	detection := &models.Detection{
		DetectionID:    detectionID,
		DetectorName:   "manual_deployment",
		Category:       "cache",
		Severity:       "info",
		DatabaseID:     req.DatabaseID,
		Timestamp:      time.Now().Unix(),
		Title:          "Deploy Redis Cache",
		Description:    "User-triggered Redis deployment from Dashboard",
		Recommendation: "Deploy Redis for application-level caching",
		ActionType:     "deploy_redis",
		ActionMetaData: params,
		Evidence:       map[string]interface{}{},
	}

	// Execute action asynchronously via detection handler
	go func() {
		log.Printf("Executing Redis deployment action: %s", actionID)
		s.detectionHandler.ExecuteActionDirectly(action, detection)
	}()

	// Return immediately with action ID
	response := map[string]interface{}{
		"success":      true,
		"action_id":    actionID,
		"detection_id": detectionID,
		"message":      "Redis deployment started",
		"database_id":  req.DatabaseID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	log.Printf("Redis deployment queued: action_id=%s, database_id=%s", actionID, req.DatabaseID)
}

func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
