.PHONY: up down restart logs ps clean help

# Start all infrastructure services
up:
	@echo "🚀 Starting infrastructure services..."
	sudo docker compose up -d
	@echo "✅ Infrastructure services started"
	@echo ""
	@echo "Services running:"
	@sudo docker ps --filter "name=mydreamcampus" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "Access points:"
	@echo "  PostgreSQL Auth:     localhost:5432"
	@echo "  PostgreSQL Staff:    localhost:5433"
	@echo "  PostgreSQL Student:  localhost:5434"
	@echo "  PostgreSQL Catalog:  localhost:5435"
	@echo "  PostgreSQL Enrollment: localhost:5436"
	@echo "  PostgreSQL Attendance: localhost:5437"
	@echo "  PostgreSQL Grades:   localhost:5438"
	@echo "  PostgreSQL Meal:     localhost:5439"
	@echo "  RabbitMQ AMQP:       localhost:5672"
	@echo "  RabbitMQ Management: http://localhost:15672"
	@echo "  Redis:               localhost:6379"
	@echo "  Traefik Dashboard:   http://localhost:8080"

# Stop all infrastructure services
down:
	@echo "🛑 Stopping infrastructure services..."
	sudo docker compose down
	@echo "✅ Infrastructure services stopped"

# Restart all infrastructure services
restart:
	@echo "🔄 Restarting infrastructure services..."
	sudo docker compose restart
	@echo "✅ Infrastructure services restarted"

# Show logs for all services
logs:
	sudo docker compose logs -f

# Service-specific logs
logs-postgres-auth:
	sudo docker compose logs -f postgres-auth

logs-postgres-staff:
	sudo docker compose logs -f postgres-staff

logs-postgres-student:
	sudo docker compose logs -f postgres-student

logs-postgres-catalog:
	sudo docker compose logs -f postgres-catalog

logs-postgres-enrollment:
	sudo docker compose logs -f postgres-enrollment

logs-postgres-attendance:
	sudo docker compose logs -f postgres-attendance

logs-postgres-grades:
	sudo docker compose logs -f postgres-grades

logs-postgres-meal:
	sudo docker compose logs -f postgres-meal

logs-postgres:
	sudo docker compose logs -f postgres-auth postgres-staff postgres-student postgres-catalog postgres-enrollment postgres-attendance postgres-grades postgres-meal

logs-rabbitmq:
	sudo docker compose logs -f rabbitmq

logs-redis:
	sudo docker compose logs -f redis

logs-traefik:
	sudo docker compose logs -f traefik

# Show running containers
ps:
	@echo "Infrastructure services status:"
	@docker ps --filter "name=mydreamcampus" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check health of services
health:
	@echo "Checking infrastructure health..."
	@echo ""
	@echo "PostgreSQL (auth):"
	@sudo docker exec mydreamcampus-postgres-auth pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (staff):"
	@sudo docker exec mydreamcampus-postgres-staff pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (student):"
	@sudo docker exec mydreamcampus-postgres-student pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (catalog):"
	@sudo docker exec mydreamcampus-postgres-catalog pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (enrollment):"
	@sudo docker exec mydreamcampus-postgres-enrollment pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (attendance):"
	@sudo docker exec mydreamcampus-postgres-attendance pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (grades):"
	@sudo docker exec mydreamcampus-postgres-grades pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "PostgreSQL (meal):"
	@sudo docker exec mydreamcampus-postgres-meal pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "RabbitMQ:"
	@sudo docker exec mydreamcampus-rabbitmq rabbitmq-diagnostics ping || echo "  ❌ Not ready"
	@echo ""
	@echo "Redis:"
	@sudo docker exec mydreamcampus-redis redis-cli ping || echo "  ❌ Not ready"
	@echo ""
	@echo "✅ Health check completed"

# Stop and remove all containers, networks, and volumes (DESTRUCTIVE!)
clean:
	@echo "⚠️  WARNING: This will remove all containers, networks, and volumes!"
	@echo "⚠️  All data will be lost!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	sudo docker compose down -v
	@echo "✅ Infrastructure cleaned (all data removed)"

# Database connections
db-connect-auth:
	@docker exec -it mydreamcampus-postgres-auth psql -U postgres -d mydreamcampus_auth

db-connect-staff:
	@docker exec -it mydreamcampus-postgres-staff psql -U postgres -d mydreamcampus_staff

db-connect-student:
	@docker exec -it mydreamcampus-postgres-student psql -U postgres -d mydreamcampus_student

db-connect-catalog:
	@docker exec -it mydreamcampus-postgres-catalog psql -U postgres -d mydreamcampus_catalog

db-connect-enrollment:
	@docker exec -it mydreamcampus-postgres-enrollment psql -U postgres -d mydreamcampus_enrollment

db-connect-attendance:
	@docker exec -it mydreamcampus-postgres-attendance psql -U postgres -d mydreamcampus_attendance

db-connect-grades:
	@docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades

db-connect-meal:
	@docker exec -it mydreamcampus-postgres-meal psql -U postgres -d mydreamcampus_meal

# Connect to Redis CLI
redis-cli:
	@docker exec -it mydreamcampus-redis redis-cli

# Open RabbitMQ Management UI
rabbitmq-ui:
	@echo "Opening RabbitMQ Management UI..."
	@xdg-open http://localhost:15672 2>/dev/null || open http://localhost:15672 2>/dev/null || echo "Open http://localhost:15672 in your browser"

# Open Traefik Dashboard
traefik-ui:
	@echo "Opening Traefik Dashboard..."
	@xdg-open http://localhost:8080 2>/dev/null || open http://localhost:8080 2>/dev/null || echo "Open http://localhost:8080 in your browser"

# Show help
help:
	@echo "Infrastructure Makefile Commands"
	@echo ""
	@echo "Main commands:"
	@echo "  make up              Start all infrastructure services"
	@echo "  make down            Stop all infrastructure services"
	@echo "  make restart         Restart all infrastructure services"
	@echo "  make ps              Show running containers"
	@echo "  make logs            Show logs for all services (follow mode)"
	@echo "  make health          Check health of all services"
	@echo "  make clean           Remove all containers, networks, and volumes (DESTRUCTIVE!)"
	@echo ""
	@echo "Service-specific logs:"
	@echo "  make logs-postgres          All PostgreSQL logs"
	@echo "  make logs-postgres-auth     Auth DB logs"
	@echo "  make logs-postgres-staff    Staff DB logs"
	@echo "  make logs-postgres-student  Student DB logs"
	@echo "  make logs-rabbitmq          RabbitMQ logs"
	@echo "  make logs-redis             Redis logs"
	@echo "  make logs-traefik           Traefik logs"
	@echo ""
	@echo "Database commands:"
	@echo "  make db-connect-auth        Connect to auth database"
	@echo "  make db-connect-staff       Connect to staff database"
	@echo "  make db-connect-student     Connect to student database"
	@echo "  make db-connect-catalog     Connect to catalog database"
	@echo "  make db-connect-enrollment  Connect to enrollment database"
	@echo "  make db-connect-attendance  Connect to attendance database"
	@echo "  make db-connect-grades      Connect to grades database"
	@echo "  make db-connect-meal        Connect to meal database"
	@echo ""
	@echo "UI commands:"
	@echo "  make rabbitmq-ui     Open RabbitMQ Management UI"
	@echo "  make traefik-ui      Open Traefik Dashboard"
	@echo ""
	@echo "Redis commands:"
	@echo "  make redis-cli       Connect to Redis CLI"
