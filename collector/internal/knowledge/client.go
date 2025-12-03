package knowledge

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client handles communication with the Knowledge service.
// It manages database registration and health status updates.
type Client struct {
	conn   *grpc.ClientConn
	client pb.KnowledgeServiceClient
}

// DatabaseInfo contains information needed to register a database with Knowledge.
type DatabaseInfo struct {
	DatabaseID       string
	ConnectionString string
	DatabaseType     string
	DatabaseName     string
}

// NewClient creates a new Knowledge service client and establishes a gRPC connection.
func NewClient(address string) (*Client, error) {
	log.Printf("Connecting to Knowledge service at: %s", address)

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Knowledge service: %w", err)
	}

	log.Printf("Connected to Knowledge service")

	return &Client{
		conn:   conn,
		client: pb.NewKnowledgeServiceClient(conn),
	}, nil
}

// RegisterDatabase registers a database with the Knowledge service.
// This allows other services (like Executor) to fetch connection strings for autonomous actions.
func (c *Client) RegisterDatabase(ctx context.Context, info *DatabaseInfo) error {
	host, port := parseConnectionString(info.ConnectionString, info.DatabaseType)

	req := &pb.RegisterDatabaseRequest{
		DatabaseId:       info.DatabaseID,
		ConnectionString: info.ConnectionString,
		DatabaseType:     info.DatabaseType,
		DatabaseName:     info.DatabaseName,
		Host:             host,
		Port:             port,
		Version:          "unknown", // TODO: Query database for actual version
		RegisteredAt:     time.Now().Unix(),
		Metadata:         map[string]string{},
	}

	resp, err := c.client.RegisterDatabase(ctx, req)
	if err != nil {
		return fmt.Errorf("registration RPC failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Knowledge rejected registration: %s", resp.Message)
	}

	log.Printf("Database registered with Knowledge: %s (%s)", info.DatabaseID, info.DatabaseType)
	return nil
}

// UpdateDatabaseHealth updates the health status of a registered database.
// TODO: Call this periodically from Orchestrator to keep Knowledge in sync.
func (c *Client) UpdateDatabaseHealth(ctx context.Context, databaseID string, healthScore float64) error {
	req := &pb.UpdateDatabaseHealthRequest{
		DatabaseId:  databaseID,
		HealthScore: healthScore,
		Status:      determineStatus(healthScore),
		LastSeen:    time.Now().Unix(),
	}

	resp, err := c.client.UpdateDatabaseHealth(ctx, req)
	if err != nil {
		return fmt.Errorf("health update RPC failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Knowledge rejected health update: %s", resp.Message)
	}

	return nil
}

// Close gracefully closes the gRPC connection to Knowledge service.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// parseConnectionString extracts host and port from a database connection string.
// Supports standard URL formats (postgresql://, mysql://, mongodb://).
// Returns sensible defaults based on database type if parsing fails.
func parseConnectionString(connStr, dbType string) (string, int32) {
	// Default values based on database type
	host := "localhost"
	port := int32(5432) // PostgreSQL default

	switch dbType {
	case "postgres", "postgresql":
		port = 5432
	case "mysql":
		port = 3306
	case "mongodb":
		port = 27017
	}

	// Parse URL-formatted connection strings
	if strings.Contains(connStr, "://") {
		u, err := url.Parse(connStr)
		if err == nil {
			if u.Hostname() != "" {
				host = u.Hostname()
			}
			if u.Port() != "" {
				if p, err := strconv.Atoi(u.Port()); err == nil {
					port = int32(p)
				}
			}
		}
	}

	return host, port
}

// determineStatus converts a health score to a status string.
func determineStatus(healthScore float64) string {
	if healthScore >= 0.8 {
		return "healthy"
	} else if healthScore >= 0.5 {
		return "degraded"
	}
	return "offline"
}
