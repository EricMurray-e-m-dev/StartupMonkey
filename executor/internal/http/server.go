package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
)

type Server struct {
	detectionHandler *handler.DetectionHandler
}

func NewServer(dh *handler.DetectionHandler) *Server {
	return &Server{detectionHandler: dh}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/actions/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		s.handleRollback(w, r)
	})

	log.Printf("HTTP Server listening on: %s", addr)
	return http.ListenAndServe(addr, s.enableCORS(mux))
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
