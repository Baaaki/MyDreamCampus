# Student Service

Öğrenci yönetimi servisi. CRUD, CSV bulk import, event-driven architecture ile danışman atama.

## Özellikler

- Student CRUD + soft delete
- CSV bulk import (PostgreSQL COPY - high performance)
- Background job processing
- Auto advisor assignment (round-robin)
- Event publishing/consuming (RabbitMQ)
- Pagination & filtering

**Tech Stack:** Go 1.23+, Gin, PostgreSQL 18, pgx, sqlc, goose, RabbitMQ, Air

---

## Hızlı Başlangıç

### 1. Infrastructure Başlat

```bash
# PostgreSQL + RabbitMQ container'ları
cd /home/nautilus/Desktop/Playground/mydreamcampus/infrastructure
sudo docker compose up -d

# Kontrol
sudo docker compose ps
```

### 2. Staff Service Başlat (Dependency)

```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/services/staff-service
make dev
```

### 3. Student Service Başlat

```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/services/student-service
make dev
```

**`make dev` ne yapar?**
- Infrastructure kontrolü
- Database migration çalıştırır
- Air ile hot reload başlatır (port 8003)

### 4. Durdur

```bash
# Student service'i durdur (Ctrl+C)

# Infrastructure'ı durdur
cd /home/nautilus/Desktop/Playground/mydreamcampus/infrastructure
sudo docker compose down
```

---

## Makefile Komutları

```bash
# Development
make dev                 # check-infra + migrate-up + air

# Database
make migrate-up          # Migration'ları uygula
make migrate-down        # Son migration'ı geri al
make migrate-status      # Migration durumunu göster

# Code generation
make sqlc                # sqlc kod üret

# Build
make build               # Binary oluştur (bin/student-service)

# Test
make test                # Quick test (curl health check)
```

---

## Test

Integration testler tüm endpoint'leri otomatik test eder.

**Gereksinim:** Infrastructure + Staff Service + Student Service çalışıyor olmalı

```bash
# Tüm testleri çalıştır
go test -v ./tests/

# Coverage
go test -coverprofile=coverage.out ./tests/
go tool cover -html=coverage.out
```

**11 test:** Health, CRUD, List, Bulk Import, Job Tracking, Soft Delete (~6s)

---

## API Endpoints

**Base URL:** `http://localhost:8003`

### Student CRUD
- `GET /health` - Health check
- `POST /api/v1/students` - Create student
- `GET /api/v1/students` - List (pagination, filters)
- `GET /api/v1/students/{id}` - Get by ID
- `PUT /api/v1/students/{id}` - Update (class_level, advisor_id, status)
- `DELETE /api/v1/students/{id}` - Soft delete

### Bulk Import
- `POST /api/v1/students/bulk-import` - Upload CSV
- `GET /api/v1/students/bulk-import/{job_id}` - Job status
- `GET /api/v1/students/bulk-import` - List jobs

**CSV Format:**
```csv
student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level
2021001,Ahmet,Yılmaz,ahmet.yilmaz@university.edu.tr,Engineering,Computer Engineering,2021,3
```

---

## Mimari

### Layered Architecture
```
cmd/main.go              → Entry point
internal/handler/        → HTTP handlers (Gin)
internal/service/        → Business logic
internal/repository/     → Data access (sqlc)
internal/dto/            → Request/Response DTOs
internal/db/             → sqlc generated code
internal/worker/         → Background workers (event consumer, outbox)
sql/migrations/          → goose migrations
sql/queries/             → sqlc queries
```

### Event-Driven
**Published Events:** `student.created`, `student.updated`, `student.deleted`
**Consumed Events:** `staff.deactivated` (nullify advisor_id)
**Patterns:** Outbox pattern, event idempotency

### Database Schema
- `students` - id, student_number, email (unique), advisor_id (FK), deleted_at
- `import_jobs` - job tracking for bulk import
- `outbox_events` - reliable event publishing
- `processed_events` - event idempotency

---

## Environment Variables

```env
PORT=8003
DB_URL=postgres://postgres:postgres@localhost:5434/mydreamcampus_student?sslmode=disable
RABBITMQ_URL=amqp://rabbitmq:rabbitmq@localhost:5672/
STAFF_SERVICE_URL=http://localhost:8002
JWT_SECRET=your-secret-key
```

---

## Troubleshooting

**Port kullanımda:**
```bash
lsof -i :8003
kill -9 <PID>
```

**Database connection error:**
```bash
sudo docker logs mydreamcampus-postgres-student
cd infrastructure && sudo docker compose restart postgres-student
```

**RabbitMQ error:**
```bash
sudo docker logs mydreamcampus-rabbitmq
# Management UI: http://localhost:15672 (rabbitmq/rabbitmq)
```

---
