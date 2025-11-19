package eventbus

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/knowledge"
	"github.com/nats-io/nats.go"
)

// ActionCompletedEvent represents an action completion event
type ActionCompletedEvent struct {
	ActionID    string `json:"action_id"`
	DetectionID string `json:"detection_id"`
	ActionType  string `json:"action_type"`
	Status      string `json:"status"`
	Solution    string `json:"solution"`
	Message     string `json:"message"`
	Timestamp   int64  `json:"timestamp"`
}

type Subscriber struct {
	conn            *nats.Conn
	subscription    *nats.Subscription
	knowledgeClient *knowledge.KnowledgeClient
}

func NewSubscriber(natsURL string, knowledgeClient *knowledge.KnowledgeClient) (*Subscriber, error) {
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
		conn:            conn,
		knowledgeClient: knowledgeClient,
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
		log.Printf("Action %s not completed (status: %s), skipping resolution", event.ActionID, event.Status)
		return
	}

	log.Printf("Action completed: %s (type: %s)", event.ActionID, event.ActionType)
	log.Printf("   Detection: %s", event.DetectionID)
	log.Printf("   Solution: %s", event.Solution)

	// Mark detection as resolved in Knowledge
	ctx := context.Background()
	if err := s.knowledgeClient.MarkDetectionResolved(ctx, event.DetectionID, event.Solution); err != nil {
		log.Printf("Warning: Failed to mark detection resolved in Knowledge: %v", err)
		return
	}

	log.Printf("Detection marked as resolved in Knowledge: %s", event.DetectionID)
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
