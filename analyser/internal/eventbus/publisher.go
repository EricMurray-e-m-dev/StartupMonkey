package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/models"
	"github.com/nats-io/nats.go"
)

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)

	if err != nil {
		return nil, err
	}

	log.Printf("Analyser connected to NATS at %s", natsURL)

	return &Publisher{
		conn: conn,
	}, nil
}

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

func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
		log.Printf("Analyser disconnected from NATS")
	}
}

func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
