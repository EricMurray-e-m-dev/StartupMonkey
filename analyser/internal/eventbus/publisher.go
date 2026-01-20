package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/verification"
	"github.com/nats-io/nats.go"
)

// Publisher publishes events to NATS
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher creates a new event bus publisher
func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Analyser (Pub) connected to NATS at %s", natsURL)

	return &Publisher{
		conn: conn,
	}, nil
}

// PublishDetection publishes a detection to the "detections" topic
func (p *Publisher) PublishDetection(detection *models.Detection) error {
	data, err := json.Marshal(detection)
	if err != nil {
		return err
	}

	if err := p.conn.Publish("detections", data); err != nil {
		return err
	}

	log.Printf("Published detection to event bus: [%s] %s", detection.Severity, detection.Title)

	return nil
}

// PublishRollbackRequest publishes a rollback request to the "rollback.requested" topic
func (p *Publisher) PublishRollbackRequest(request *verification.RollbackRequest) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	if err := p.conn.Publish("rollback.requested", data); err != nil {
		return err
	}

	log.Printf("Published rollback request to event bus: action=%s reason=%s",
		request.ActionID, request.Reason)

	return nil
}

// Close closes the NATS connection
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
		log.Println("Analyser (Pub) disconnected from NATS")
	}
}

// IsConnected returns true if connected to NATS
func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
