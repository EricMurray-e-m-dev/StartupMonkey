package eventbus

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/executor/internal/models"
	"github.com/nats-io/nats.go"
)

type ActionCompletedEvent struct {
	ActionID    string `json:"action_id"`
	DetectionID string `json:"detection_id"`
	ActionType  string `json:"action_type"`
	Status      string `json:"status"`
	Solution    string `json:"solution"`
	Message     string `json:"message"`
	Timestamp   int64  `json:"timestamp"`
}

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

func (p *Publisher) PublishActionCompleted(result *models.ActionResult, detection *models.Detection) error {
	solution := generateSolution(result, detection)

	event := ActionCompletedEvent{
		ActionID:    result.ActionID,
		DetectionID: detection.DetectionID,
		ActionType:  result.ActionType,
		Status:      result.Status,
		Solution:    solution,
		Message:     result.Message,
		Timestamp:   time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal action completion: %w", err)
	}

	if err := p.conn.Publish("actions.completed", data); err != nil {
		return fmt.Errorf("failed to published data to actions.completed: %w", err)
	}

	log.Printf("Published completed action: %s -> %s", result.ActionID, solution)

	return nil
}

func generateSolution(result *models.ActionResult, detection *models.Detection) string {
	switch result.ActionType {
	case "create_index":
		tableName := ""
		columnName := ""

		if detection.ActionMetaData != nil {
			if t, ok := detection.ActionMetaData["table_name"].(string); ok {
				tableName = t
			}
			if c, ok := detection.ActionMetaData["column_name"].(string); ok {
				columnName = c
			}
		}

		if tableName != "" && columnName != "" {
			indexName := fmt.Sprintf("%s_%s_idx", tableName, columnName)
			return fmt.Sprintf("index_created:%s", indexName)
		}
		return "index_created:unknown"

	case "deploy_pgbouncer":
		return "pgbouncer_deployed"

	case "deploy_redis":
		return "redis_deployed"

	case "increase_cache_size":
		return "cache_size_increased"

	default:
		return fmt.Sprintf("%s_applied", result.ActionType)
	}
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
