// Package knowledge provides a client for communicating with the Knowledge service.
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

// ListDatabases retrieves all registered databases from Knowledge.
// If enabledOnly is true, only returns databases with enabled=true.
func (c *Client) ListDatabases(ctx context.Context, enabledOnly bool) ([]*pb.RegisteredDatabase, error) {
	req := &pb.ListDatabasesRequest{
		EnabledOnly: enabledOnly,
	}

	resp, err := c.client.ListDatabases(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ListDatabases RPC failed: %w", err)
	}

	return resp.Databases, nil
}

// RegisterDatabase registers a database with the Knowledge service.
func (c *Client) RegisterDatabase(ctx context.Context, info *DatabaseInfo) error {
	host, port := parseConnectionString(info.ConnectionString, info.DatabaseType)

	req := &pb.RegisterDatabaseRequest{
		DatabaseId:       info.DatabaseID,
		ConnectionString: info.ConnectionString,
		DatabaseType:     info.DatabaseType,
		DatabaseName:     info.DatabaseName,
		Host:             host,
		Port:             port,
		Version:          "unknown",
		RegisteredAt:     time.Now().Unix(),
		Metadata:         map[string]string{},
		Enabled:          true, // New databases are enabled by default
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
func (c *Client) UpdateDatabaseHealth(ctx context.Context, databaseID, status string, healthScore float64) error {
	req := &pb.UpdateDatabaseHealthRequest{
		DatabaseId:  databaseID,
		HealthScore: healthScore,
		Status:      status,
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

// GetSystemConfig fetches the system configuration from Knowledge service.
func (c *Client) GetSystemConfig(ctx context.Context) (*pb.SystemConfig, error) {
	resp, err := c.client.GetSystemConfig(ctx, &pb.GetSystemConfigRequest{})
	if err != nil {
		return nil, fmt.Errorf("GetSystemConfig RPC failed: %w", err)
	}

	return resp, nil
}

// Close gracefully closes the gRPC connection to Knowledge service.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// parseConnectionString extracts host and port from a database connection string.
func parseConnectionString(connStr, dbType string) (string, int32) {
	host := "localhost"
	port := int32(5432)

	switch dbType {
	case "postgres", "postgresql":
		port = 5432
	case "mysql":
		port = 3306
	case "mongodb":
		port = 27017
	}

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
