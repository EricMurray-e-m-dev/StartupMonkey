package eventbus

import (
	"encoding/json"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/handler"
	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/nats-io/nats.go"
)

type Subscriber struct {
	conn         *nats.Conn
	subscription *nats.Subscription
	handler      *handler.DetectionHandler
}

func NewSubscriber(natsURL string, h *handler.DetectionHandler) (*Subscriber, error) {
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
		conn:    conn,
		handler: h,
	}, nil

}

func (s *Subscriber) Start() error {
	var err error

	log.Printf("Subscribing to 'detections'")

	s.subscription, err = s.conn.Subscribe("detections", func(msg *nats.Msg) {
		s.handleMessage(msg)
	})

	if err != nil {
		return err
	}

	log.Printf("Subscribed to 'detections'")

	return nil
}

func (s *Subscriber) handleMessage(msg *nats.Msg) {
	log.Printf("Received message from event bus (%d bytes)", len(msg.Data))

	var detection models.Detection
	if err := json.Unmarshal(msg.Data, &detection); err != nil {
		log.Printf("Failed to unmarshal detection: %v", err)
		return
	}

	result, err := s.handler.HandleDetection(&detection)
	if err != nil {
		log.Printf("Failed to handle detection: %v", err)
		return
	}

	log.Printf("Detection processed successfully (Action ID: %s)", result.ActionID)
}

func (s *Subscriber) Close() {
	if s.subscription != nil {
		s.subscription.Unsubscribe()
	}

	if s.conn != nil {
		s.conn.Close()
		log.Printf("Disconnected from NATS")
	}
}

func (s *Subscriber) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}
