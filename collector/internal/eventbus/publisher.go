package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/collector/normaliser"
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

	log.Printf("Collector connected to NATS at %s", natsURL)

	return &Publisher{
		conn: conn,
	}, nil
}

func (p *Publisher) PublishMetrics(metrics *normaliser.NormalisedMetrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err := p.conn.Publish("metrics", data); err != nil {
		return err
	}

	log.Printf("Published metrics to event bus [%s]", metrics.DatabaseID)

	return nil
}

func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
		log.Printf("Collector disconnected from NATS")
	}
}

func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
