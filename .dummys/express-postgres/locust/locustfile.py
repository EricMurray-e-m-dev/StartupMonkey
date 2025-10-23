import random
import time
from locust import HttpUser, task, between
from faker import Faker

fake = Faker()

class BlogAPIUser(HttpUser):
    wait_time = between(1, 3)

    def on_start(self):
        """Called when a user starts. Initialize user-specific data."""
        self.categories = [
            'Technology', 'Startups', 'Development', 'DevOps',
            'Architecture', 'Performance', 'Security', 'Data', 'Mobile', 'AI/ML'
        ]
        self.search_terms = [
            'react', 'node', 'python', 'javascript', 'api', 'database',
            'performance', 'startup', 'scaling', 'microservices', 'docker',
            'kubernetes', 'postgresql', 'mongodb', 'redis', 'optimization',
            'security', 'authentication', 'aws', 'gcp', 'azure', 'deployment',
            'testing', 'monitoring', 'architecture', 'design patterns'
        ]

        # Simulate different user behaviors
        self.user_type = random.choice(['browser', 'power_user', 'mobile_app', 'api_client'])

    @task(30)
    def browse_posts(self):
        """Most common action - browsing posts (30% weight)"""
        params = {
            'limit': random.choice([10, 20, 50, 100]),
            'offset': random.randint(0, 1000)
        }

        # 40% chance to filter by category
        if random.random() < 0.4:
            params['category'] = random.choice(self.categories)

        # 20% chance to filter by author
        if random.random() < 0.2:
            params['author_id'] = random.randint(1, 10000)

        with self.client.get("/api/posts", params=params, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")

    @task(15)
    def heavy_query_load(self):
        """Heavy analytical queries (15% weight) - simulates dashboards/analytics"""
        params = {
            'limit': random.choice([5, 10, 20, 50])
        }

        with self.client.get("/api/heavy-query", params=params, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Heavy query failed with status {response.status_code}")

    @task(25)
    def search_content(self):
        """Text search functionality (25% weight)"""
        search_term = random.choice(self.search_terms)

        # Sometimes use partial terms or combine terms
        if random.random() < 0.3:
            search_term = search_term[:random.randint(3, len(search_term))]
        elif random.random() < 0.2:
            search_term = f"{search_term} {random.choice(self.search_terms)}"

        params = {
            'q': search_term,
            'limit': random.choice([10, 25, 50]),
            'offset': random.randint(0, 100)
        }

        with self.client.get("/api/search", params=params, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 400:
                # Expected for very short search terms
                response.success()
            else:
                response.failure(f"Search failed with status {response.status_code}")

    @task(20)
    def random_discovery(self):
        """Random content discovery (20% weight) - cache-busting behavior"""
        params = {
            'count': random.choice([3, 5, 10, 15])
        }

        with self.client.get("/api/random-posts", params=params, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Random posts failed with status {response.status_code}")

    @task(5)
    def health_check(self):
        """Health check endpoint (5% weight)"""
        with self.client.get("/health", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Health check failed with status {response.status_code}")

    @task(3)
    def browse_homepage(self):
        """Visit homepage (3% weight)"""
        with self.client.get("/", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Homepage failed with status {response.status_code}")

class PowerUser(BlogAPIUser):
    """Power users that make more intensive queries"""
    wait_time = between(0.5, 1.5)

    @task(40)
    def intensive_browsing(self):
        """Power users browse more intensively"""
        for _ in range(random.randint(2, 5)):
            params = {
                'limit': random.choice([50, 100, 200]),
                'offset': random.randint(0, 5000),
                'category': random.choice(self.categories) if random.random() < 0.7 else None,
                'author_id': random.randint(1, 10000) if random.random() < 0.4 else None
            }

            # Remove None values
            params = {k: v for k, v in params.items() if v is not None}

            self.client.get("/api/posts", params=params)
            time.sleep(0.1)

    @task(25)
    def complex_searches(self):
        """Power users perform more complex searches"""
        search_terms = random.sample(self.search_terms, random.randint(1, 3))
        search_term = " ".join(search_terms)

        params = {
            'q': search_term,
            'limit': random.choice([50, 100]),
            'offset': random.randint(0, 500)
        }

        self.client.get("/api/search", params=params)

class MobileUser(BlogAPIUser):
    """Mobile users with different usage patterns"""
    wait_time = between(2, 5)

    @task(50)
    def mobile_browsing(self):
        """Mobile users typically browse smaller chunks"""
        params = {
            'limit': random.choice([5, 10, 15]),
            'offset': random.randint(0, 100)
        }

        # Mobile users less likely to use complex filters
        if random.random() < 0.2:
            params['category'] = random.choice(self.categories)

        self.client.get("/api/posts", params=params)

    @task(30)
    def quick_search(self):
        """Mobile users do quick searches"""
        search_term = random.choice(self.search_terms)

        params = {
            'q': search_term,
            'limit': random.choice([5, 10]),
            'offset': 0
        }

        self.client.get("/api/search", params=params)

class APIClient(BlogAPIUser):
    """API clients that make systematic requests"""
    wait_time = between(0.1, 0.5)

    @task(30)
    def systematic_data_fetch(self):
        """API clients fetch data systematically"""
        limit = 100
        for offset in range(0, random.randint(500, 2000), limit):
            params = {
                'limit': limit,
                'offset': offset
            }

            response = self.client.get("/api/posts", params=params)
            if response.status_code != 200:
                break
            time.sleep(0.05)

    @task(20)
    def analytics_queries(self):
        """API clients run analytics queries"""
        for limit in [10, 20, 50]:
            self.client.get("/api/heavy-query", params={'limit': limit})
            time.sleep(0.1)