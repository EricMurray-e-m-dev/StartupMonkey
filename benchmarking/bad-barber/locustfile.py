"""
Locust load test for Bad Barbers application.
Simulates realistic user behavior with intentional performance issues.
"""

from locust import HttpUser, task, between, SequentialTaskSet
import random
from datetime import datetime, timedelta


class BadBarbersUserBehavior(SequentialTaskSet):
    """Sequential user tasks simulating real booking flow."""
    
    def on_start(self):
        """Setup - runs once per user when they start."""
        # Generate fake customer data
        self.customer_phone = f"+353 87 {random.randint(1000000, 9999999)}"
        self.customer_name = random.choice([
            "John Smith", "Mary Jones", "Pat Murphy", "Sarah Kelly",
            "Tom Brady", "Emma Wilson", "James Brown", "Lisa Davis",
            "Michael Ryan", "Anna Walsh", "David Moore", "Claire O'Brien"
        ])
    
    @task
    def view_homepage(self):
        """
        User visits homepage.
        TRIGGERS: Cache miss detection (popular_barbers query on every request)
        """
        with self.client.get("/", catch_response=True) as response:
            if response.status_code != 200:
                response.failure("Homepage failed to load")
    
    @task
    def load_popular_barbers(self):
        """
        Homepage loads popular barbers via API.
        TRIGGERS: Cache miss detection (expensive aggregation query)
        """
        with self.client.get("/api/popular-barbers", catch_response=True, name="/api/popular-barbers") as response:
            if response.status_code != 200:
                response.failure("Failed to load popular barbers")
            elif response.elapsed.total_seconds() > 2.0:
                response.failure("Popular barbers query too slow (>2s)")
    
    @task
    def view_booking_page(self):
        """User navigates to booking page."""
        self.client.get("/booking.html")
    
    @task
    def load_barbers_list(self):
        """
        Booking page loads barbers list.
        TRIGGERS: Missing index detection (if shop_id join is slow)
        """
        with self.client.get("/api/barbers", catch_response=True, name="/api/barbers") as response:
            if response.status_code == 200:
                self.barbers = response.json()
            else:
                response.failure("Failed to load barbers")
    
    @task
    def load_services(self):
        """Booking page loads services list."""
        self.client.get("/api/services", name="/api/services")
    
    @task
    def check_available_slots(self):
        """
        User checks available slots for a barber.
        TRIGGERS: Connection pool exhaustion (new connection per request)
        """
        if not hasattr(self, 'barbers') or not self.barbers:
            return
        
        barber = random.choice(self.barbers)
        tomorrow = (datetime.now() + timedelta(days=1)).strftime('%Y-%m-%d')
        
        with self.client.get(
            f"/api/barbers/{barber['barber_id']}/available-slots?date={tomorrow}",
            catch_response=True,
            name="/api/barbers/[id]/available-slots"
        ) as response:
            if response.status_code != 200:
                response.failure("Failed to load available slots")
    
    @task
    def create_booking(self):
        """
        User creates a booking.
        TRIGGERS: Connection pool exhaustion + missing index on booking_date
        """
        if not hasattr(self, 'barbers') or not self.barbers:
            return
        
        barber = random.choice(self.barbers)
        tomorrow = (datetime.now() + timedelta(days=1)).strftime('%Y-%m-%d')
        
        booking_data = {
            "customer_name": self.customer_name,
            "customer_phone": self.customer_phone,
            "customer_email": f"{self.customer_name.lower().replace(' ', '.')}@email.com",
            "barber_id": barber['barber_id'],
            "booking_date": tomorrow,
            "booking_time": random.choice([
                "10:00:00", "10:30:00", "11:00:00", "11:30:00",
                "12:00:00", "14:00:00", "14:30:00", "15:00:00"
            ]),
            "service": random.choice([
                "Haircut", "Haircut & Beard Trim", "Beard Trim", "Hot Towel Shave"
            ]),
            "price": random.choice([25.00, 35.00, 15.00, 30.00])
        }
        
        with self.client.post(
            "/api/bookings",
            json=booking_data,
            catch_response=True,
            name="/api/bookings [POST]"
        ) as response:
            if response.status_code == 201:
                # Booking created successfully
                pass
            else:
                response.failure(f"Failed to create booking: {response.status_code}")
    
    @task
    def search_my_bookings(self):
        """
        User searches for their bookings by phone.
        TRIGGERS: Missing index detection (customer_phone has no index - full table scan!)
        """
        with self.client.get(
            f"/api/bookings/search?phone={self.customer_phone}",
            catch_response=True,
            name="/api/bookings/search"
        ) as response:
            if response.status_code != 200:
                response.failure("Failed to search bookings")
            elif response.elapsed.total_seconds() > 3.0:
                response.failure("Booking search too slow (>3s - sequential scan!)")
    
    @task
    def view_admin_panel(self):
        """
        Admin checks today's bookings.
        TRIGGERS: Missing index on booking_date (sequential scan)
        """
        with self.client.get("/api/bookings/today", catch_response=True, name="/api/bookings/today") as response:
            if response.status_code != 200:
                response.failure("Failed to load today's bookings")
            elif response.elapsed.total_seconds() > 2.0:
                response.failure("Today's bookings query too slow (>2s)")


class BadBarbersUser(HttpUser):
    """
    Simulated user for Bad Barbers application.
    
    User behavior:
    - Views homepage (triggers cache miss)
    - Browses barbers and services
    - Checks available slots (triggers connection exhaustion)
    - Creates bookings (triggers missing indexes)
    - Searches past bookings (triggers sequential scan)
    """
    
    tasks = [BadBarbersUserBehavior]
    
    # Wait between 1-3 seconds between tasks (realistic user behavior)
    wait_time = between(1, 3)
    
    # User session settings
    host = "http://localhost:3020"  # Will be overridden by --host flag


class AggressiveUser(HttpUser):
    """
    Aggressive user that hammers specific slow endpoints.
    Use this to quickly trigger detections during testing.
    """
    
    wait_time = between(0.5, 1.5)
    
    @task(3)
    def popular_barbers(self):
        """Hammer the expensive aggregation query."""
        self.client.get("/api/popular-barbers")
    
    @task(3)
    def search_bookings(self):
        """Hammer the sequential scan endpoint."""
        phone = f"+353 87 {random.randint(1000000, 9999999)}"
        self.client.get(f"/api/bookings/search?phone={phone}")
    
    @task(2)
    def today_bookings(self):
        """Hammer today's bookings (no index on date)."""
        self.client.get("/api/bookings/today")
    
    @task(1)
    def available_slots(self):
        """Hammer available slots (connection pool exhaustion)."""
        barber_id = random.randint(1, 4)
        tomorrow = (datetime.now() + timedelta(days=1)).strftime('%Y-%m-%d')
        self.client.get(f"/api/barbers/{barber_id}/available-slots?date={tomorrow}")


class ReadOnlyUser(HttpUser):
    """
    Read-only user for baseline testing.
    Only performs GET requests to measure query performance.
    """
    
    wait_time = between(1, 2)
    
    @task(4)
    def popular_barbers(self):
        self.client.get("/api/popular-barbers")
    
    @task(3)
    def barbers_list(self):
        self.client.get("/api/barbers")
    
    @task(2)
    def services(self):
        self.client.get("/api/services")
    
    @task(1)
    def today_bookings(self):
        self.client.get("/api/bookings/today")