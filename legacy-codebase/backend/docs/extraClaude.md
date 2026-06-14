# Extra Configuration Details

Bu dosya CLAUDE.md'den taşınan detaylı konfigürasyon dosyalarını içerir.

---

## 📊 Loki Merkezi Log Toplama - Detaylı Konfigürasyon

### 1. Loki Configuration (`infrastructure/loki/loki-config.yml`)
```yaml
auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096

common:
  instance_addr: 127.0.0.1
  path_prefix: /tmp/loki
  storage:
    filesystem:
      chunks_directory: /tmp/loki/chunks
      rules_directory: /tmp/loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2024-01-01
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h

limits_config:
  retention_period: 168h  # 7 days

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 100
```

### 2. Promtail Configuration (`infrastructure/promtail/promtail-config.yml`)
```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  # Docker container logs scraping
  - job_name: docker
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
        filters:
          - name: label
            values: ["logging=promtail"]
    relabel_configs:
      # Service name label
      - source_labels: ['__meta_docker_container_label_com_docker_compose_service']
        target_label: 'service'
      # Container name
      - source_labels: ['__meta_docker_container_name']
        target_label: 'container'
      # Environment (dev/prod)
      - source_labels: ['__meta_docker_container_label_environment']
        target_label: 'environment'
    pipeline_stages:
      # JSON log parsing (Zap structured logs)
      - json:
          expressions:
            level: level
            ts: ts
            msg: msg
            caller: caller
      # Set log level
      - labels:
          level:
      # Timestamp parsing
      - timestamp:
          source: ts
          format: Unix
```

### 3. Grafana Datasource (`infrastructure/grafana/datasources.yml`)
```yaml
apiVersion: 1

datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: true
    editable: true
```

### 4. Docker Compose Loki Stack
`infrastructure/docker-compose.yml` dosyasına ekle:

```yaml
  # Grafana Loki - Log aggregation
  loki:
    image: grafana/loki:3.2.0
    container_name: mydreamcampus-loki
    ports:
      - "3100:3100"
    volumes:
      - ./loki/loki-config.yml:/etc/loki/local-config.yaml
      - loki-data:/tmp/loki
    command: -config.file=/etc/loki/local-config.yaml
    networks:
      - mydreamcampus
    restart: unless-stopped

  # Promtail - Log shipper
  promtail:
    image: grafana/promtail:3.2.0
    container_name: mydreamcampus-promtail
    volumes:
      - ./promtail/promtail-config.yml:/etc/promtail/config.yml
      - /var/run/docker.sock:/var/run/docker.sock
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
    command: -config.file=/etc/promtail/config.yml
    networks:
      - mydreamcampus
    depends_on:
      - loki
    restart: unless-stopped

  # Grafana - Log visualization
  grafana:
    image: grafana/grafana:11.4.0
    container_name: mydreamcampus-grafana
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    networks:
      - mydreamcampus
    depends_on:
      - loki
    restart: unless-stopped

volumes:
  loki-data:
  grafana-data:
```

### 5. LogQL Query Örnekleri
```logql
# Tüm auth-service logları
{service="auth-service"}

# Error level loglar (tüm servisler)
{service=~".+"} |= "error"

# Belirli serviste belirli user'ın logları
{service="auth-service"} |= "email" |= "john@example.com"

# Son 5 dakikada 500 hatası alan request'ler
{service="auth-service"} |= "status=500" [5m]

# JSON field'larına göre filtreleme
{service="auth-service"} | json | level="error"
```

---

## 🔧 Servis Konfigürasyon Dosyaları

### sqlc.yaml (Her Serviste)
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./sql/queries"
    schema: "./sql/migrations"
    gen:
      go:
        package: "db"
        out: "./internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
```

### .air.toml (Hot Reload)
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ./cmd/main.go"
  bin = "tmp/main"
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_dir = ["tmp", "vendor"]
  delay = 1000
```

### Makefile (Her Servis İçin)
```makefile
# Variables
GOOSE_DRIVER=postgres
DATABASE_URL=postgresql://user:password@localhost:5432/mydreamcampus_auth?sslmode=disable
MIGRATIONS_DIR=./sql/migrations
BINARY_NAME=auth-service

# Colors for output
GREEN=\033[0;32m
NC=\033[0m # No Color

.PHONY: help migrate-create migrate-up migrate-down migrate-status migrate-reset sqlc build docker-build install-tools

help: ## Show this help
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-15s${NC} %s\n", $$1, $$2}'

migrate-create: ## Create a new migration (usage: make migrate-create name=create_users)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter required. Usage: make migrate-create name=create_users"; \
		exit 1; \
	fi
	@echo "${GREEN}Creating migration: $(name)${NC}"
	goose -dir $(MIGRATIONS_DIR) create $(name) sql
	@echo "${GREEN}Migration created in $(MIGRATIONS_DIR)${NC}"

migrate-up: ## Run all migrations
	@echo "${GREEN}Running migrations up...${NC}"
	goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" up
	@echo "${GREEN}Migrations applied!${NC}"

migrate-down: ## Rollback last migration
	@echo "${GREEN}Rolling back last migration...${NC}"
	goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" down
	@echo "${GREEN}Migration rolled back!${NC}"

migrate-status: ## Show migration status
	@echo "${GREEN}Migration status:${NC}"
	goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" status

migrate-reset: ## Reset database (down all + up all)
	@echo "${GREEN}Resetting database...${NC}"
	goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DATABASE_URL)" reset
	@echo "${GREEN}Database reset complete!${NC}"

sqlc: ## Generate sqlc code
	@echo "${GREEN}Generating sqlc code...${NC}"
	sqlc generate
	@echo "${GREEN}sqlc code generated!${NC}"

build: ## Build production binary
	@echo "${GREEN}Building $(BINARY_NAME)...${NC}"
	go build -o bin/$(BINARY_NAME) cmd/main.go
	@echo "${GREEN}Build complete: bin/$(BINARY_NAME)${NC}"

docker-build: ## Build docker image
	@echo "${GREEN}Building Docker image...${NC}"
	docker build -t $(BINARY_NAME):latest .
	@echo "${GREEN}Docker image built!${NC}"

install-tools: ## Install required tools (goose, sqlc, air)
	@echo "${GREEN}Installing development tools...${NC}"
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/cosmtrek/air@latest
	@echo "${GREEN}Tools installed!${NC}"

# Default target
.DEFAULT_GOAL := help
```

### .env Dosyası Şablonu
```bash
# Server
PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=mydreamcampus_auth
DB_SSLMODE=disable

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=mydreamcampus

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-key-change-in-production
JWT_EXPIRATION_HOUR=24
```

---

## 🚀 Hızlı Setup Komutları

### Faz 0: Foundation Setup
```bash
# 1. Workspace oluştur
go work init

# 2. Shared package setup
mkdir -p shared/{database,logger,middleware,rabbitmq,redis,models,dto,errors,utils,config}
cd shared && go mod init github.com/yourusername/mydreamcampus/shared
go work use ./shared

# 3. Infrastructure setup
cd infrastructure
docker-compose up -d
```

### İlk Servis (auth-service) Setup
```bash
# 1. Servis klasörü oluştur
mkdir -p services/auth-service/{sql/{migrations,queries},internal/{db,repository,service,handler,dto},cmd,config}
cd services/auth-service

# 2. Go module init
go mod init github.com/yourusername/mydreamcampus/auth-service
cd ../..
go work use ./services/auth-service

# 3. Makefile, sqlc.yaml, .air.toml, .env oluştur
cd services/auth-service
# (yukarıdaki şablonları kullan)

# 4. Tool'ları yükle
make install-tools

# 5. Dependencies yükle
go mod tidy
```

### Geliştirme Döngüsü
```bash
# 1. Migration oluştur
make migrate-create name=create_users

# 2. Migration dosyasını doldur
# sql/migrations/XXXXXX_create_users.sql

# 3. Migration'ı çalıştır
make migrate-up

# 4. Query dosyası oluştur ve doldur
# sql/queries/users.sql

# 5. sqlc generate
make sqlc

# 6. Repository, Service, Handler yaz
# internal/repository/user_repository.go
# internal/service/user_service.go
# internal/handler/user_handler.go

# 7. Development server başlat
air

# 8. Manuel test (Postman/curl)

# 9. Commit
git add .
git commit -m "feat(auth): add user management"
```

### Loki Setup (Faz 1 Bitiminde)
```bash
# 1. Config dosyalarını oluştur
cd infrastructure
mkdir -p loki promtail grafana
# (yukarıdaki config'leri kopyala)

# 2. Docker compose güncelle ve restart
docker-compose down
docker-compose up -d

# 3. Grafana'ya giriş
# Browser: http://localhost:3000
# Username: admin / Password: admin

# 4. Logları görüntüle
# Grafana → Explore → Query: {service="auth-service"}
```
