.PHONY: up down restart logs ps clean help

# Start all infrastructure services
up:
	@echo "🚀 Starting infrastructure services..."
	sudo docker compose up -d
	@echo "✅ Infrastructure services started"
	@echo ""
	@echo "Services running:"
	@docker ps --filter "name=mydreamcampus" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "Access points:"
	@echo "  PostgreSQL:        localhost:5432 (postgres:postgres)"
	@echo "  RabbitMQ AMQP:     localhost:5672 (rabbitmq:rabbitmq)"
	@echo "  RabbitMQ Management: http://localhost:15672 (rabbitmq:rabbitmq)"
	@echo "  Redis:             localhost:6379"
	@echo "  Traefik Dashboard: http://localhost:8080"

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

# Show logs for specific service (usage: make logs-postgres, make logs-rabbitmq, etc.)
logs-postgres:
	sudo docker compose logs -f postgres

logs-rabbitmq:
	sudo docker compose logs -f rabbitmq

logs-redis:
	sudo docker compose logs -f redis

logs-traefik:
	sudo docker compose logs -f traefik

logs-loki:
	sudo docker compose logs -f loki

logs-promtail:
	sudo docker compose logs -f promtail

logs-grafana:
	sudo docker compose logs -f grafana

# Show running containers
ps:
	@echo "Infrastructure services status:"
	@docker ps --filter "name=mydreamcampus" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check health of services
health:
	@echo "Checking infrastructure health..."
	@echo ""
	@echo "PostgreSQL:"
	@docker exec mydreamcampus-postgres pg_isready -U postgres || echo "  ❌ Not ready"
	@echo ""
	@echo "RabbitMQ:"
	@docker exec mydreamcampus-rabbitmq rabbitmq-diagnostics ping || echo "  ❌ Not ready"
	@echo ""
	@echo "Redis:"
	@docker exec mydreamcampus-redis redis-cli ping || echo "  ❌ Not ready"
	@echo ""
	@echo "✅ Health check completed"

# Stop and remove all containers, networks, and volumes (DESTRUCTIVE!)
clean:
	@echo "⚠️  WARNING: This will remove all containers, networks, and volumes!"
	@echo "⚠️  All data will be lost!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	sudo docker compose down -v
	@echo "✅ Infrastructure cleaned (all data removed)"

# Show databases
db-list:
	@echo "Databases in PostgreSQL:"
	@docker exec -it mydreamcampus-postgres psql -U postgres -c "\l"

# Connect to PostgreSQL
db-connect:
	@docker exec -it mydreamcampus-postgres psql -U postgres

# Connect to specific database (usage: make db-connect-staff)
db-connect-staff:
	@docker exec -it mydreamcampus-postgres psql -U postgres -d mydreamcampus_staff

db-connect-auth:
	@docker exec -it mydreamcampus-postgres psql -U postgres -d mydreamcampus_auth

db-connect-student:
	@docker exec -it mydreamcampus-postgres psql -U postgres -d mydreamcampus_student

# Connect to Redis CLI
redis-cli:
	@docker exec -it mydreamcampus-redis redis-cli

# Open RabbitMQ Management UI
rabbitmq-ui:
	@echo "Opening RabbitMQ Management UI..."
	@xdg-open http://localhost:15672 2>/dev/null || open http://localhost:15672 2>/dev/null || echo "Open http://localhost:15672 in your browser"
	@echo "Login: rabbitmq / rabbitmq"

# Open Traefik Dashboard
traefik-ui:
	@echo "Opening Traefik Dashboard..."
	@xdg-open http://localhost:8080 2>/dev/null || open http://localhost:8080 2>/dev/null || echo "Open http://localhost:8080 in your browser"

# Open Grafana (when enabled)
grafana-ui:
	@echo "Opening Grafana..."
	@xdg-open http://localhost:3000 2>/dev/null || open http://localhost:3000 2>/dev/null || echo "Open http://localhost:3000 in your browser"
	@echo "Login: admin / admin"

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
	@echo "  make logs-postgres   Show PostgreSQL logs"
	@echo "  make logs-rabbitmq   Show RabbitMQ logs"
	@echo "  make logs-redis      Show Redis logs"
	@echo "  make logs-traefik    Show Traefik logs"
	@echo ""
	@echo "Database commands:"
	@echo "  make db-list         List all databases"
	@echo "  make db-connect      Connect to PostgreSQL (default database)"
	@echo "  make db-connect-staff    Connect to staff database"
	@echo "  make db-connect-auth     Connect to auth database"
	@echo "  make db-connect-student  Connect to student database"
	@echo ""
	@echo "UI commands:"
	@echo "  make rabbitmq-ui     Open RabbitMQ Management UI"
	@echo "  make traefik-ui      Open Traefik Dashboard"
	@echo "  make grafana-ui      Open Grafana (when enabled)"
	@echo ""
	@echo "Redis commands:"
	@echo "  make redis-cli       Connect to Redis CLI"
	@echo ""
	@echo "For more information, see README.md"
