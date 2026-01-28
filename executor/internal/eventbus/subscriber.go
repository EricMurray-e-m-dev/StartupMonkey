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

type ApprovalRequest struct {
	ActionID string `json:"action_id"`
}

type ApprovalProcessor interface {
	ApproveAction(actionID string) (*models.ActionResult, error)
	RejectAction(actionID string) (*models.ActionResult, error)
}

type Subscriber struct {
	conn              *nats.Conn
	detectionSub      *nats.Subscription
	rollbackSub       *nats.Subscription
	approveSub        *nats.Subscription
	rejectSub         *nats.Subscription
	processor         DetectionProcessor
	rollbackProcessor RollbackProcessor
	approvalProcessor ApprovalProcessor
}

func NewSubscriber(natsURL string, processor DetectionProcessor, rollbackProcessor RollbackProcessor, approvalProcessor ApprovalProcessor) (*Subscriber, error) {
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
		approvalProcessor: approvalProcessor,
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

	// Approval subscriptions
	if s.approvalProcessor != nil {
		log.Printf("Subscribing to 'actions.approve'")
		s.approveSub, err = s.conn.Subscribe("actions.approve", func(msg *nats.Msg) {
			s.handleApproveMessage(msg)
		})
		if err != nil {
			return err
		}
		log.Printf("Subscribed to 'actions.approve'")

		log.Printf("Subscribing to 'actions.reject'")
		s.rejectSub, err = s.conn.Subscribe("actions.reject", func(msg *nats.Msg) {
			s.handleRejectMessage(msg)
		})
		if err != nil {
			return err
		}
		log.Printf("Subscribed to 'actions.reject'")
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

func (s *Subscriber) handleApproveMessage(msg *nats.Msg) {
	log.Printf("Received approval request from event bus (%d bytes)", len(msg.Data))

	var request ApprovalRequest
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		log.Printf("Failed to unmarshal approval request: %v", err)
		return
	}

	log.Printf("Processing action approval: %s", request.ActionID)

	result, err := s.approvalProcessor.ApproveAction(request.ActionID)
	if err != nil {
		log.Printf("Action approval failed: %v", err)
		return
	}

	log.Printf("Action approved and executing: %s -> %s", request.ActionID, result.Status)
}

func (s *Subscriber) handleRejectMessage(msg *nats.Msg) {
	log.Printf("Received rejection request from event bus (%d bytes)", len(msg.Data))

	var request ApprovalRequest
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		log.Printf("Failed to unmarshal rejection request: %v", err)
		return
	}

	log.Printf("Processing action rejection: %s", request.ActionID)

	result, err := s.approvalProcessor.RejectAction(request.ActionID)
	if err != nil {
		log.Printf("Action rejection failed: %v", err)
		return
	}

	log.Printf("Action rejected: %s -> %s", request.ActionID, result.Status)
}

func (s *Subscriber) Close() {
	if s.detectionSub != nil {
		s.detectionSub.Unsubscribe()
	}
	if s.rollbackSub != nil {
		s.rollbackSub.Unsubscribe()
	}
	if s.approveSub != nil {
		s.approveSub.Unsubscribe()
	}
	if s.rejectSub != nil {
		s.rejectSub.Unsubscribe()
	}

	if s.conn != nil {
		s.conn.Close()
		log.Printf("Disconnected from NATS")
	}
}

func (s *Subscriber) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}
