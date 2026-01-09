package eventbus

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/knowledge"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/verification"
	"github.com/nats-io/nats.go"
)

// ActionCompletedEvent represents an action completion event
type ActionCompletedEvent struct {
	ActionID     string `json:"action_id"`
	DetectionID  string `json:"detection_id"`
	DetectionKey string `json:"detection_key"`
	ActionType   string `json:"action_type"`
	DatabaseID   string `json:"database_id"`
	Status       string `json:"status"`
	Solution     string `json:"solution"`
	Message      string `json:"message"`
	Timestamp    int64  `json:"timestamp"`
}

type Subscriber struct {
	conn                *nats.Conn
	subscription        *nats.Subscription
	knowledgeClient     *knowledge.KnowledgeClient
	verificationTracker *verification.Tracker
}

func NewSubscriber(natsURL string, knowledgeClient *knowledge.KnowledgeClient, tracker *verification.Tracker) (*Subscriber, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)

	if err != nil {
		return nil, err
	}

	log.Printf("Analyser (Sub) connected to NATS at %s", natsURL)

	return &Subscriber{
		conn:                conn,
		knowledgeClient:     knowledgeClient,
		verificationTracker: tracker,
	}, nil
}

// Start begins listening for action completion events
func (s *Subscriber) Start() error {
	var err error

	log.Printf("Subscribing to 'actions.completed' for feedback loop...")

	s.subscription, err = s.conn.Subscribe("actions.completed", func(msg *nats.Msg) {
		s.handleActionCompleted(msg)
	})

	if err != nil {
		return err
	}

	log.Printf("Subscribed to 'actions.completed'")

	return nil
}

func (s *Subscriber) handleActionCompleted(msg *nats.Msg) {
	log.Printf("Received action completion event (%d bytes)", len(msg.Data))

	var event ActionCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal action completion: %v", err)
		return
	}

	// Only process completed actions (not failed)
	if event.Status != "completed" {
		log.Printf("Action %s not completed (status: %s), skipping verification", event.ActionID, event.Status)
		return
	}

	log.Printf("Action completed: %s (type: %s)", event.ActionID, event.ActionType)
	log.Printf("   Detection: %s", event.DetectionID)
	log.Printf("   Detection Key: %s", event.DetectionKey)
	log.Printf("   Solution: %s", event.Solution)

	// Check if this action type supports autonomous verification
	if s.supportsAutonomousVerification(event.ActionType) {
		// Add to verification tracker instead of marking resolved immediately
		if event.DetectionKey != "" {
			s.verificationTracker.AddPendingVerification(
				event.DetectionKey,
				event.DetectionID,
				event.ActionID,
				event.ActionType,
				event.DatabaseID,
			)
			log.Printf("Action %s added to verification queue (will verify after %d cycles)",
				event.ActionID, verification.DefaultVerificationCycles)
		} else {
			log.Printf("Warning: Action %s has no detection key, marking resolved immediately", event.ActionID)
			s.markResolved(event.DetectionID, event.Solution)
		}
	} else {
		// Actions that can't be autonomously verified (e.g., PgBouncer, Redis)
		// Mark as resolved immediately
		log.Printf("Action type %s does not support autonomous verification, marking resolved", event.ActionType)
		s.markResolved(event.DetectionID, event.Solution)
	}
}

// supportsAutonomousVerification returns true if the action type can be verified by monitoring metrics
func (s *Subscriber) supportsAutonomousVerification(actionType string) bool {
	switch actionType {
	case "create_index":
		// Can verify: seq_scans should decrease
		return true
	case "tune_config_high_latency":
		// Can verify: latency should decrease
		return true
	case "cache_optimization_recommendation":
		// Can verify: cache hit rate should increase
		return true
	case "deploy_connection_pooler", "deploy_redis":
		// Cannot verify: requires app code changes to use
		return false
	default:
		return false
	}
}

func (s *Subscriber) markResolved(detectionID, solution string) {
	ctx := context.Background()
	if err := s.knowledgeClient.MarkDetectionResolved(ctx, detectionID, solution); err != nil {
		log.Printf("Warning: Failed to mark detection resolved in Knowledge: %v", err)
		return
	}
	log.Printf("Detection marked as resolved in Knowledge: %s", detectionID)
}

func (s *Subscriber) Close() {
	if s.subscription != nil {
		s.subscription.Unsubscribe()
	}

	if s.conn != nil {
		s.conn.Close()
		log.Printf("Analyser disconnected from NATS")
	}
}

func (s *Subscriber) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}
