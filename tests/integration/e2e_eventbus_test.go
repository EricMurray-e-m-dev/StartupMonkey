package integration

import (
	"testing"
	"time"

	"github.com/EricMurray-e-m-dev/StartupMonkey/tests/integration/framework"
)

func TestFullPipeline_WithEventbus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	env := framework.NewTestEnvironment(t, []string{
		"postgres",
		"nats",
		"analyser",
		"executor",
		"collector",
	})

	err := env.Start()
	if err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	defer env.Cleanup()

	t.Log("Waiting for all services to be healthy...")
	err = env.WaitForHealthy(60 * time.Second)
	if err != nil {
		t.Fatalf("Services health check timed out: %v", err)
	}

	framework.AssertLogsContain(t, env, "nats", "Server is ready")

	framework.AssertLogsContain(t, env, "analyser", "Analyser connected to NATS")

	framework.AssertLogsContain(t, env, "executor", "Connected to NATS")
	framework.AssertLogsContain(t, env, "executor", "Subscribed to 'detections'")

	t.Log("Waiting for collection cycle and detection flow (35s)...")
	time.Sleep(35 * time.Second)

	t.Log("Verifying Collector -> Analyser flow...")
	framework.AssertLogsContain(t, env, "collector", "Metrics sent successfully")
	framework.AssertLogsContain(t, env, "collector", "Collection Cycle Complete")

	t.Log("Verifying Analyser received metrics...")
	framework.AssertLogsContain(t, env, "analyser", "Metric #1 received")
	framework.AssertLogsContain(t, env, "analyser", "Active Connections")

	t.Log("Verifying Analyser published to NATS...")
	framework.AssertLogsContain(t, env, "analyser", "Published detection to event bus")
	framework.AssertLogsContain(t, env, "analyser", "Published to event bus")

	t.Log("Verifying Executor received detection from NATS...")
	framework.AssertLogsContain(t, env, "executor", "Action queued")
	framework.AssertLogsContain(t, env, "executor", "Detection processed successfully")

	t.Log("Full pipeline test passed: Collector -> Analyser -> NATS -> Executor")
}
