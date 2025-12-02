from locust import HttpUser, task, between, events
import random

class DummyAppUser(HttpUser):
    """
    Simulates users hitting the intentionally bad endpoints.
    Task weights determine how often each endpoint is called.
    """
    
    # Wait 0.1 to 0.5 seconds between requests (fast, aggressive load)
    wait_time = between(0.1, 0.5)
    
    # Target host (can override with --host flag)
    host = "http://localhost:3002"
    
    def on_start(self):
        """Called once when a user starts"""
        # Health check to verify app is running
        response = self.client.get("/health")
        if response.status_code != 200:
            print("⚠️  WARNING: App health check failed!")
    
    @task(10)  # Weight: 10 (called most often)
    def get_posts_missing_index(self):
        """
        Triggers missing_index detection
        Queries posts by user_id WITHOUT an index
        """
        user_id = random.randint(1, 1000)
        self.client.get(f"/api/posts?user_id={user_id}", name="/api/posts")
    
    @task(3)  # Weight: 3
    def leak_connection(self):
        """
        Triggers connection_pool_exhaustion detection
        Each call holds a connection for 30 seconds
        """
        with self.client.get("/api/leak-connection", catch_response=True, name="/api/leak-connection") as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(2)  # Weight: 2
    def slow_query(self):
        """
        Triggers high_query_latency detection
        Forces a 2-second pg_sleep
        """
        self.client.get("/api/slow-query?sleep=2", name="/api/slow-query")
    
    @task(5)  # Weight: 5
    def expensive_aggregation(self):
        """
        Expensive aggregation query across entire table
        Also triggers missing_index (no index on user_id for GROUP BY)
        """
        self.client.get("/api/stats", name="/api/stats")
    
    @task(1)  # Weight: 1 (least often)
    def good_endpoint(self):
        """
        Normal indexed query (for comparison)
        Should remain fast even under load
        """
        user_id = random.randint(1, 1000)
        self.client.get(f"/api/users/{user_id}", name="/api/users/:id")


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Called when the load test starts"""
    print("\n" + "="*60)
    print("🚀 Starting Load Test - Intentionally Breaking Dummy App")
    print("="*60)
    print("Target endpoints:")
    print("  - /api/posts (missing index)")
    print("  - /api/leak-connection (connection exhaustion)")
    print("  - /api/slow-query (high latency)")
    print("  - /api/stats (expensive aggregation)")
    print("\nExpected detections within 60 seconds:")
    print("  ✓ missing_index (sequential scans)")
    print("  ✓ connection_pool_exhaustion (leaked connections)")
    print("  ✓ high_query_latency (slow queries)")
    print("="*60 + "\n")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Called when the load test stops"""
    print("\n" + "="*60)
    print("🛑 Load Test Complete")
    print("="*60)
    print("Check StartupMonkey logs for detections and actions!")
    print("="*60 + "\n")