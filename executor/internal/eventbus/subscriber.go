package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/nats-io/nats.go"
)

type DetectionProcessor interface {
	HandleDetection(detection *models.Detection) (*models.ActionResult, error)
}

type RollbackRequest struct {
	ActionID    string `json:"action_id"`
	DetectionID string `json:"detection_id"`
	ActionType  string `json:"action_type"`
	DatabaseID  string `json:"database_id"`
	Reason      string `json:"reason"`
	Timestamp   int64  `json:"timestamp"`
}

type RollbackProcessor interface {
	RollbackAction(actionID string) (*models.ActionResult, error)
}

type Subscriber struct {
	conn              *nats.Conn
	detectionSub      *nats.Subscription
	rollbackSub       *nats.Subscription
	processor         DetectionProcessor
	rollbackProcessor RollbackProcessor
}

func NewSubscriber(natsURL string, processor DetectionProcessor, rollbackProcessor RollbackProcessor) (*Subscriber, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)

	if err != nil {
		return nil, err
	}

	log.Printf("Connected to NATS at %s", natsURL)

	return &Subscriber{
		conn:              conn,
		processor:         processor,
		rollbackProcessor: rollbackProcessor,
	}, nil
}

func (s *Subscriber) Start() error {
	var err error

	// Detection subscription
	log.Printf("Subscribing to 'detections'")
	s.detectionSub, err = s.conn.Subscribe("detections", func(msg *nats.Msg) {
		s.handleDetectionMessage(msg)
	})
	if err != nil {
		return err
	}
	log.Printf("Subscribed to 'detections'")

	// Rollback subscription
	if s.rollbackProcessor != nil {
		log.Printf("Subscribing to 'rollback.requested'")
		s.rollbackSub, err = s.conn.Subscribe("rollback.requested", func(msg *nats.Msg) {
			s.handleRollbackMessage(msg)
		})
		if err != nil {
			return err
		}
		log.Printf("Subscribed to 'rollback.requested'")
	}

	return nil
}

func (s *Subscriber) handleDetectionMessage(msg *nats.Msg) {
	log.Printf("Received detection from event bus (%d bytes)", len(msg.Data))

	var detection models.Detection
	if err := json.Unmarshal(msg.Data, &detection); err != nil {
		log.Printf("Failed to unmarshal detection: %v", err)
		return
	}

	result, err := s.processor.HandleDetection(&detection)
	if err != nil {
		log.Printf("Failed to handle detection: %v", err)
		return
	}

	if result != nil {
		log.Printf("Detection processed successfully (Action ID: %s)", result.ActionID)
	}
}

func (s *Subscriber) handleRollbackMessage(msg *nats.Msg) {
	log.Printf("Received rollback request from event bus (%d bytes)", len(msg.Data))

	var request RollbackRequest
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		log.Printf("Failed to unmarshal rollback request: %v", err)
		return
	}

	log.Printf("Processing autonomous rollback: action=%s reason=%s", request.ActionID, request.Reason)

	result, err := s.rollbackProcessor.RollbackAction(request.ActionID)
	if err != nil {
		log.Printf("Autonomous rollback failed: %v", err)
		return
	}

	log.Printf("Autonomous rollback completed: %s -> %s", request.ActionID, result.Status)
}

func (s *Subscriber) Close() {
	if s.detectionSub != nil {
		s.detectionSub.Unsubscribe()
	}
	if s.rollbackSub != nil {
		s.rollbackSub.Unsubscribe()
	}

	if s.conn != nil {
		s.conn.Close()
		log.Printf("Disconnected from NATS")
	}
}

func (s *Subscriber) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}
