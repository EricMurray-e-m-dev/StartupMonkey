package integration

import (
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/tests/integration/framework"
)

// TestCollectorToAnalyser_HappyPath tests the complete flow:
// Collector collects metrics from PostgreSQL → sends to Analyser via gRPC
func TestCollectorToAnalyser_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Start all required services
	env := framework.NewTestEnvironment(t, []string{"postgres", "analyser", "collector"})

	err := env.Start()
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer env.Cleanup()

	// Wait for services to be healthy
	err = env.WaitForHealthy(60 * time.Second)
	if err != nil {
		t.Fatalf("Services did not become healthy: %v", err)
	}

	// Wait for at least one collection cycle to complete
	// Collection interval is 30s, so wait 35s to be safe
	t.Log("Waiting for collection cycle to complete (35s)...")
	time.Sleep(35 * time.Second)

	// Assert: Verify Collector sent metrics
	framework.AssertLogsContain(t, env, "collector", "Metrics sent successfully")
	framework.AssertLogsContain(t, env, "collector", "Collection Cycle Complete")

	// Assert: Verify Analyser received metrics
	framework.AssertLogsContain(t, env, "analyser", "Metric #1 received")
	framework.AssertLogsContain(t, env, "analyser", "Active Connections")

	t.Log("Integration test passed: Collector → Analyser communication verified")
}
