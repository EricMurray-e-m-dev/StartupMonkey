# BarberBook - Barbershop Booking System

A simple booking system for barbershops, intentionally designed with performance issues for StartupMonkey to detect and fix.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- StartupMonkey (for monitoring)

### Run the Application
```bash
# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Access the Application

- **Frontend**: http://localhost:8080
- **Backend API**: http://localhost:3001
- **Database**: localhost:5433

### Test Accounts

No authentication required - just enter any phone number to view bookings.

## Intentional Performance Issues

This application was built with several performance issues that StartupMonkey will detect:

1. **Missing Indexes** - No indexes on frequently queried columns
2. **Connection Pool Exhaustion** - Creates new DB connection for each request
3. **High Latency** - Inefficient queries that load all data then filter in JavaScript
4. **Cache Misses** - No caching for frequently accessed aggregations

## API Endpoints

### Barbers
- `GET /api/barbers` - List all barbers
- `GET /api/popular-barbers` - Get most popular barbers (expensive query)
- `GET /api/barbers/:id/bookings` - Get barber's bookings (inefficient)
- `GET /api/barbers/:id/available-slots?date=YYYY-MM-DD` - Get available time slots

### Bookings
- `GET /api/bookings/today` - Today's bookings (no index on date)
- `GET /api/bookings/search?phone=XXX` - Search by phone (no index)
- `POST /api/bookings` - Create booking

### Services
- `GET /api/services` - List all services

## Database Schema

See `database/init.sql` for the complete schema with 50,000 test bookings.

## Load Testing

Use Locust or ApacheBench to generate traffic:
```bash
# Example with ApacheBench
ab -n 1000 -c 50 http://localhost:3001/api/popular-barbers
```

## Monitoring with StartupMonkey

1. Configure StartupMonkey to monitor `barberbook` database on port 5433
2. Generate load on the application
3. Watch StartupMonkey detect and fix performance issues
4. Observe improved response times

## Architecture
```
┌─────────────┐      ┌─────────────┐      ┌──────────────┐
│   Frontend  │─────▶│   Backend   │─────▶│  PostgreSQL  │
│  (Nginx)    │      │  (Express)  │      │   (5433)     │
│   :8080     │      │   :3001     │      │              │
└─────────────┘      └─────────────┘      └──────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ StartupMonkey│
                    │  Monitoring  │
                    └──────────────┘
```

## Development

### Backend
```bash
cd backend
npm install
npm run dev
```

### Frontend
Just open `frontend/index.html` in a browser, or use a simple HTTP server:
```bash
cd frontend
python3 -m http.server 8080
```

## License

MIT