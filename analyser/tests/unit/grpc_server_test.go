package unit

import (
	"context"
	"testing"

	"github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/engine"
	grpcserver "github.com/EricMurray-e-m-dev/StartupMonkey/analyser/internal/grpc"
	pb "github.com/EricMurray-e-m-dev/StartupMonkey/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsServer(t *testing.T) {
	detectionEngine := engine.NewEngine()
	server := grpcserver.NewMetricsServer(detectionEngine)

	assert.NotNil(t, server)
}

func TestRegisterDatabase(t *testing.T) {
	detectionEngine := engine.NewEngine()
	server := grpcserver.NewMetricsServer(detectionEngine)
	ctx := context.Background()

	info := &pb.DatabaseInfo{
		DatabaseId:   "test-db-1",
		DatabaseName: "test_database",
		DatabaseType: "postgresql",
	}

	ack, err := server.RegisterDatabase(ctx, info)

	assert.NoError(t, err)
	assert.NotNil(t, ack)
	assert.True(t, ack.Success)
	assert.Equal(t, "test-db-1", ack.AssignedId)
}

func TestPlaceholder_StreamMetrics(t *testing.T) {
	// Placeholder - full stream testing in integration tests
	t.Log("StreamMetrics tested via integration tests (Issue #10)")
}
