package unit

import (
	"testing"

	grpcclient "github.com/EricMurray-e-m-dev/StartupMonkey/collector/internal/grpc"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsClient(t *testing.T) {
	address := "localhost:50051"

	client := grpcclient.NewMetricsClient(address)

	assert.NotNil(t, client)
}

func TestMetricsClient_Connect(t *testing.T) {
	client := grpcclient.NewMetricsClient("localhost:50051")

	err := client.Connect()

	assert.NoError(t, err)

	client.Close()
}

func TestMetricsClient_Connect_InvalidAddress(t *testing.T) {
	client := grpcclient.NewMetricsClient("")

	err := client.Connect()

	assert.Error(t, err)
}

func TestMetricsClient_Close_SafeWhenNotConnected(t *testing.T) {
	client := grpcclient.NewMetricsClient("localhost:50051")

	// Close should not error if not connected
	err := client.Close()
	assert.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}
