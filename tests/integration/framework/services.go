package framework

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreateAnalyserClient creates a gRPC client connected to Analyser
func CreateAnalyserClient(t *testing.T, address string) (pb.MetricsServiceClient, *grpc.ClientConn) {
	// grpc.NewClient creates connection (lazy, no WithBlock support)
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		t.Fatalf("Failed to create Analyser client: %v", err)
	}

	// Verify connection works with a quick context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Trigger actual connection by checking state
	conn.Connect()
	conn.WaitForStateChange(ctx, conn.GetState())

	client := pb.NewMetricsServiceClient(conn)
	return client, conn
}

// WaitForMetricsInLogs waits for specific log message to appear
func (e *TestEnvironment) WaitForMetricsInLogs(serviceName string, searchString string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		logs, err := e.GetLogs(serviceName)
		if err != nil {
			return err
		}

		if strings.Contains(logs, searchString) {
			e.t.Logf("Found expected log message in %s", serviceName)
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("did not find '%s' in %s logs within timeout", searchString, serviceName)
}
