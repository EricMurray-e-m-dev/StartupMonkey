// Package eventbus provides NATS event publishing for the Collector service.
package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
	"github.com/nats-io/nats.go"
)

// Publisher handles publishing metrics to the NATS event bus.
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher creates a new NATS publisher with retry logic.
func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, err
	}

	log.Printf("Collector connected to NATS at %s", natsURL)

	return &Publisher{conn: conn}, nil
}

// PublishMetrics publishes normalised metrics to the "metrics" subject.
func (p *Publisher) PublishMetrics(metrics *normaliser.NormalisedMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err := p.conn.Publish("metrics", data); err != nil {
		return err
	}

	return nil
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
		log.Printf("Collector disconnected from NATS")
	}
}

// IsConnected returns true if the NATS connection is active.
func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
