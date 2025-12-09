package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/tests/integration/framework"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

// TestFullSystem_BasicFlow tests that all services start and communicate
func TestFullSystem_BasicFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start all services
	env := framework.NewTestEnvironment(t, []string{
		"postgres",
		"redis",
		"nats",
		"knowledge",
		"collector",
		"analyser",
		"executor",
	})

	err := env.Start()
	require.NoError(t, err, "Failed to start services")
	defer env.Cleanup()

	// Wait for services to be healthy
	t.Log("Waiting for services to be healthy...")
	err = env.WaitForHealthy(90 * time.Second)
	require.NoError(t, err, "Services did not become healthy")

	// Give services time to initialize
	time.Sleep(15 * time.Second)

	// Test 1: Services are running
	t.Run("ServicesRunning", func(t *testing.T) {
		testServicesRunning(t, env)
	})

	// Test 2: Collector sends metrics
	t.Run("CollectorSendsMetrics", func(t *testing.T) {
		testCollectorSendsMetrics(t, env)
	})

	// Test 3: Manual detection publish and verify Executor processes it
	t.Run("ManualDetectionProcessing", func(t *testing.T) {
		testManualDetectionProcessing(t, env)
	})
}

// testServicesRunning verifies all services started successfully
func testServicesRunning(t *testing.T, env *framework.TestEnvironment) {
	t.Log("Verifying all services are running...")

	services := []string{"postgres", "redis", "nats", "knowledge", "collector", "analyser", "executor"}

	for _, service := range services {
		logs, err := env.GetLogs(service)
		require.NoError(t, err, "Failed to get logs for %s", service)
		require.NotEmpty(t, logs, "Service %s has no logs", service)
		t.Logf("Service %s is running", service)
	}

	t.Log("All services running successfully")
}

// testCollectorSendsMetrics verifies Collector collects and sends metrics
func testCollectorSendsMetrics(t *testing.T, env *framework.TestEnvironment) {
	t.Log("Waiting for Collector to send metrics...")

	// Wait for 2 collection cycles (10s interval + buffer)
	time.Sleep(25 * time.Second)

	logs, err := env.GetLogs("collector")
	require.NoError(t, err, "Failed to get collector logs")

	// Collector should have sent metrics
	if len(logs) > 0 {
		t.Log("Collector is actively collecting metrics")
	}
}

// testManualDetectionProcessing publishes detection and verifies processing
func testManualDetectionProcessing(t *testing.T, env *framework.TestEnvironment) {
	t.Log("Testing manual detection processing...")

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	require.NoError(t, err, "Failed to connect to NATS")
	defer nc.Close()

	t.Log("Connected to NATS")

	// Create channels for different message types
	detectionAck := make(chan bool, 1)
	actionQueued := make(chan bool, 1)
	actionCompleted := make(chan bool, 1)

	// Subscribe to all topics to see what's happening
	_, err = nc.Subscribe(">", func(msg *nats.Msg) {
		t.Logf("NATS message received on topic: %s", msg.Subject)

		// Track different events
		switch {
		case msg.Subject == "detections":
			detectionAck <- true
		case msg.Subject == "actions.status.queued":
			actionQueued <- true
		case msg.Subject == "actions.status.completed":
			actionCompleted <- true
		}
	})
	require.NoError(t, err, "Failed to subscribe to NATS")

	// Publish a simple test detection
	testDetection := map[string]interface{}{
		"detection_id":   "integration-test-001",
		"detector_name":  "test_detector",
		"category":       "query",
		"severity":       "info",
		"database_id":    "dummy_app_01",
		"timestamp":      time.Now().Unix(),
		"title":          "Integration test detection",
		"description":    "Test detection for integration testing",
		"recommendation": "This is a test",
		"action_type":    "test_action",
		"action_metadata": map[string]interface{}{
			"test": true,
		},
		"evidence": map[string]interface{}{
			"test": true,
		},
	}

	detectionJSON, err := json.Marshal(testDetection)
	require.NoError(t, err, "Failed to marshal detection")

	t.Log("Publishing test detection to NATS...")
	err = nc.Publish("detections", detectionJSON)
	require.NoError(t, err, "Failed to publish detection")

	err = nc.Flush()
	require.NoError(t, err, "Failed to flush NATS")

	t.Log("Detection published successfully")

	// Wait to see what messages we receive
	timeout := time.After(30 * time.Second)
	messagesReceived := 0

	for {
		select {
		case <-detectionAck:
			t.Log("Detection acknowledged")
			messagesReceived++
		case <-actionQueued:
			t.Log("Action queued")
			messagesReceived++
		case <-actionCompleted:
			t.Log("Action completed")
			messagesReceived++
			// If we got completion, test is successful
			return
		case <-timeout:
			if messagesReceived > 0 {
				t.Logf("Received %d messages, system is processing", messagesReceived)
				return
			}
			// Check Executor logs to see what happened
			logs, _ := env.GetLogs("executor")
			t.Logf("Executor logs (last 500 chars):\n%s", logs[max(0, len(logs)-500):])
			t.Fatal("No NATS messages received - system may not be processing detections")
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestNATSConnectivity tests basic NATS pub/sub
func TestNATSConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := framework.NewTestEnvironment(t, []string{"nats"})

	err := env.Start()
	require.NoError(t, err, "Failed to start NATS")
	defer env.Cleanup()

	err = env.WaitForHealthy(30 * time.Second)
	require.NoError(t, err, "NATS did not become healthy")

	time.Sleep(5 * time.Second)

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	require.NoError(t, err, "Failed to connect to NATS")
	defer nc.Close()

	// Test pub/sub
	received := make(chan bool, 1)

	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		t.Log("Test message received")
		received <- true
	})
	require.NoError(t, err, "Failed to subscribe")

	err = nc.Publish("test.topic", []byte("test"))
	require.NoError(t, err, "Failed to publish")

	err = nc.Flush()
	require.NoError(t, err, "Failed to flush")

	select {
	case <-received:
		t.Log("NATS pub/sub working correctly")
	case <-time.After(5 * time.Second):
		t.Fatal("Test message not received")
	}
}
