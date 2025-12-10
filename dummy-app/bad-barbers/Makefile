.PHONY: up down logs restart clean

# Start all services
up:
	docker-compose up -d
	@echo "BarberBook is starting..."
	@echo "Frontend: http://localhost:8080"
	@echo "Backend:  http://localhost:3001"

# Stop all services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Restart services
restart:
	docker-compose restart

# Clean everything (including volumes)
clean:
	docker-compose down -v
	@echo "All data removed!"

# Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:3001/health | json_pp
	@curl -s http://localhost:8080 > /dev/null && echo "Frontend: OK" || echo "Frontend: DOWN"