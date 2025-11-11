package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/nats-io/nats.go"
)

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second))

	if err != nil {
		return nil, err
	}

	log.Printf("Executor Pub connected to NATS: %s", natsURL)

	return &Publisher{
		conn: conn,
	}, nil
}

func (p *Publisher) PublishActionStatus(result *models.ActionResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	if err := p.conn.Publish("actions.status", data); err != nil {
		return err
	}

	log.Printf("Published action status to event bus: [%s] %s", result.Status, result.ActionID)

	return nil
}

func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		log.Printf("Executor Pub disconnected from NATS")
	}
}

func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
