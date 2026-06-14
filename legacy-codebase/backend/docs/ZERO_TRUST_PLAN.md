# Zero Trust Dönem Güvenliği & Audit Log Sistemi — Uygulama Planı

> **Tarih**: 20 Şubat 2026
> **Durum**: Onaylandı — uygulamaya hazır
> **Tahmini etkilenen servisler**: catalog, enrollment, grades, meal + shared + frontend

---

## İçindekiler

1. [Genel Bakış](#1-genel-bakış)
2. [Özellik 1: Semester State Machine](#2-özellik-1-semester-state-machine)
3. [Özellik 2: Period CRUD Kilidi](#3-özellik-2-period-crud-kilidi)
4. [Özellik 3: Hard Deadline ile Otomatik Kilit](#4-özellik-3-hard-deadline-ile-otomatik-kilit)
5. [Özellik 4: Audit Log (PostgreSQL)](#5-özellik-4-audit-log-postgresql)
6. [Özellik 5: Appeal (İtiraz) İyileştirmesi](#6-özellik-5-appeal-i̇tiraz-i̇yileştirmesi)
7. [Frontend Güncellemeleri](#7-frontend-güncellemeleri)
8. [Uygulama Sırası](#8-uygulama-sırası)
9. [Dosya Değişiklik Özeti](#9-dosya-değişiklik-özeti)
10. [Doğrulama Adımları](#10-doğrulama-adımları)

---

## 1. Genel Bakış

### Problem
Mevcut sistemde:
- Admin hesabı ele geçirilirse geçmiş dönemlerin deadline'ları değiştirilebilir
- Gelecek dönemlere sahte deadline eklenebilir
- Hiçbir kritik işlemin değiştirilemez kaydı tutulmuyor

### Güvenlik Modeli

| Dönem Durumu | Deadline CRUD | Not Girişi | Not İtirazı |
|---|---|---|---|
| `planned` (açılmamış) | ❌ Yasak | ❌ Yasak | ❌ Yasak |
| `active` (açık) | ✅ Serbest | ✅ Deadline'a bağlı | ✅ Serbest |
| `completed` (bitmiş) | ❌ Yasak | ❌ Yasak | ✅ Serbest (audit ile) |

### Prensipler
- **Geri dönüşsüz geçişler**: `completed` → `active` asla mümkün değil (DB seviyesinde)
- **Otomatik kilit**: `hard_deadline` tarihi geçince admin unutsa bile dönem kapanır
- **Merkezi audit log**: PostgreSQL tablosu + DB trigger ile tüm kritik işlemler değiştirilemez şekilde loglanır
- **Deadline'dan bağımsız itiraz**: Geçmiş dönemlere not düzeltme ayrı mekanizma ile yapılır

---

## 2. Özellik 1: Semester State Machine

Şu an catalog servisinde ayrı bir `semesters` tablosu yok. Semester bilgisi sadece
`semester_courses.semester` kolonunda VARCHAR olarak tutuluyor (`2025-2026-Fall` formatında).

### 2.1 Migration Dosyası

**Dosya**: `backend/services/course-catalog-service/sql/migrations/00010_create_semesters_table.sql`

```sql
-- +goose Up

-- Dönem durumu enum'ı
CREATE TYPE semester_status AS ENUM ('planned', 'active', 'completed');

-- Dönem tablosu
CREATE TABLE semesters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,               -- "2025-2026-Fall"
    status semester_status NOT NULL DEFAULT 'planned',
    hard_deadline TIMESTAMPTZ NOT NULL,              -- Bu tarihten sonra otomatik completed
    activated_at TIMESTAMPTZ,                        -- planned → active geçiş zamanı
    completed_at TIMESTAMPTZ,                        -- active → completed geçiş zamanı
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_semesters_status ON semesters(status);
CREATE INDEX idx_semesters_name ON semesters(name);

-- GERİ DÖNÜŞÜ ENGELLEME: completed → active veya planned geçişini DB seviyesinde engelle
-- Bu trigger, uygulama katmanı bypass edilse bile (SQL injection vb.) güvenlik sağlar
CREATE OR REPLACE FUNCTION prevent_semester_reactivation()
RETURNS TRIGGER AS $$
BEGIN
    -- completed olan dönem başka bir duruma geçemez
    IF OLD.status = 'completed' THEN
        RAISE EXCEPTION 'Cannot change status of a completed semester (id: %)', OLD.id;
    END IF;

    -- planned sadece active'e geçebilir
    IF OLD.status = 'planned' AND NEW.status != 'active' THEN
        RAISE EXCEPTION 'Planned semester can only transition to active (id: %)', OLD.id;
    END IF;

    -- active sadece completed'e geçebilir
    IF OLD.status = 'active' AND NEW.status != 'completed' THEN
        RAISE EXCEPTION 'Active semester can only transition to completed (id: %)', OLD.id;
    END IF;

    -- Otomatik timestamp
    IF OLD.status = 'planned' AND NEW.status = 'active' THEN
        NEW.activated_at = NOW();
    END IF;

    IF OLD.status = 'active' AND NEW.status = 'completed' THEN
        NEW.completed_at = NOW();
    END IF;

    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_semester_status_change
    BEFORE UPDATE OF status ON semesters
    FOR EACH ROW
    EXECUTE FUNCTION prevent_semester_reactivation();

-- +goose Down
DROP TRIGGER IF EXISTS trg_semester_status_change ON semesters;
DROP FUNCTION IF EXISTS prevent_semester_reactivation();
DROP TABLE IF EXISTS semesters;
DROP TYPE IF EXISTS semester_status;
```

### 2.2 SQL Queries

**Dosya**: `backend/services/course-catalog-service/sql/queries/semesters.sql`

```sql
-- name: CreateSemester :one
INSERT INTO semesters (name, hard_deadline)
VALUES ($1, $2)
RETURNING *;

-- name: GetSemesterByName :one
SELECT * FROM semesters WHERE name = $1;

-- name: GetActiveSemester :one
SELECT * FROM semesters WHERE status = 'active' LIMIT 1;

-- name: ListSemesters :many
SELECT * FROM semesters ORDER BY created_at DESC;

-- name: ActivateSemester :one
UPDATE semesters SET status = 'active' WHERE id = $1 AND status = 'planned'
RETURNING *;

-- name: CompleteSemester :one
UPDATE semesters SET status = 'completed' WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: AutoCompleteSemester :exec
UPDATE semesters SET status = 'completed'
WHERE name = $1 AND status = 'active' AND hard_deadline < NOW();
```

### 2.3 Repository

**Dosya**: `backend/services/course-catalog-service/internal/repository/semester_status_repository.go`

```go
package repository

// SemesterStatusRepository — semesters tablosu CRUD işlemleri
// sqlc tarafından generate edilen db.Queries'i wrap eder

type SemesterStatusRepository struct {
    queries *db.Queries
}

func NewSemesterStatusRepository(queries *db.Queries) *SemesterStatusRepository { ... }

// Temel metodlar:
// - CreateSemester(ctx, name, hardDeadline) → Semester
// - GetSemesterByName(ctx, name) → Semester
// - GetActiveSemester(ctx) → Semester
// - ListSemesters(ctx) → []Semester
// - ActivateSemester(ctx, id) → Semester  (planned → active)
// - CompleteSemester(ctx, id) → Semester  (active → completed)
// - IsSemesterActive(ctx, name) → bool    (status + hard_deadline kontrolü)
```

**`IsSemesterActive` metodunun detayı** (hard deadline kontrolü ile birlikte):
```go
func (r *SemesterStatusRepository) IsSemesterActive(ctx context.Context, name string) (bool, error) {
    semester, err := r.queries.GetSemesterByName(ctx, name)
    if err != nil {
        return false, err // semester bulunamadı → aktif değil
    }

    if semester.Status == "completed" {
        return false, nil
    }
    if semester.Status == "planned" {
        return false, nil
    }

    // status == "active" — ama hard_deadline geçmiş mi?
    if time.Now().After(semester.HardDeadline) {
        // Otomatik complete et
        r.queries.AutoCompleteSemester(ctx, name)
        return false, nil
    }

    return true, nil
}
```

### 2.4 Handler

**Dosya**: `backend/services/course-catalog-service/internal/handler/semester_status_handler.go`

```go
package handler

// SemesterStatusHandler — Admin semester yönetim endpoint'leri

// Endpoint'ler:
// POST   /api/catalog/admin/semesters           → CreateSemester
// GET    /api/catalog/admin/semesters           → ListSemesters
// GET    /api/catalog/admin/semesters/active    → GetActiveSemester
// PUT    /api/catalog/admin/semesters/:id/activate  → ActivateSemester (planned → active)
// PUT    /api/catalog/admin/semesters/:id/complete   → CompleteSemester (active → completed)
// GET    /api/catalog/admin/semesters/:name/status   → IsSemesterActive (diğer servisler bunu çağırır)
```

**CreateSemester request body:**
```json
{
    "name": "2025-2026-Fall",
    "hard_deadline": "2026-02-15T23:59:59Z"
}
```

Validasyon:
- `name` formatı regex ile kontrol: `^\d{4}-\d{4}-(Fall|Spring)$`
- `hard_deadline` gelecekte olmalı
- Aynı isimde semester zaten varsa → 409 Conflict

### 2.5 Mevcut Dosyalardaki Değişiklikler

**`backend/services/course-catalog-service/internal/service/semester_service.go`**:
- `CreateSemesterCourse` fonksiyonunda mevcut period kontrolünden ÖNCE semester status kontrolü eklenir:
```go
// Yeni eklenen kontrol (mevcut period kontrolünden önce)
isActive, err := s.semesterStatusRepo.IsSemesterActive(ctx, semester)
if err != nil || !isActive {
    return dto.SemesterCourseResponse{}, catalogErrors.ErrSemesterNotActive
}
// ... mevcut period kontrolü devam eder
```

**`backend/services/course-catalog-service/internal/errors/errors.go`**:
- Yeni hata eklenir:
```go
var ErrSemesterNotActive = sharedErrors.New("SEMESTER_NOT_ACTIVE",
    "This semester is not active — courses cannot be created", http.StatusForbidden)
```

**`backend/services/course-catalog-service/cmd/main.go`**:
- `SemesterStatusRepository` oluştur
- `SemesterStatusHandler` oluştur
- Route'ları kaydet: `admin.Group("/semesters")` altına
- `SemesterService`'e `semesterStatusRepo` inject et

### 2.6 Diğer Servislerin Semester Durumunu Öğrenmesi

Enrollment ve grades servisleri, catalog servisine HTTP çağrısı yaparak semester durumunu kontrol eder.

**Shared interface** — `backend/shared/semester/checker.go` (yeni dosya):
```go
package semester

import "context"

// Checker verifies if a semester is currently active.
// Catalog service implements this via direct DB access.
// Other services implement this via HTTP call to catalog service.
type Checker interface {
    IsSemesterActive(ctx context.Context, semester string) (bool, error)
}
```

**HTTP implementasyonu** — `backend/shared/semester/http_checker.go` (yeni dosya):
```go
package semester

// HTTPChecker calls catalog service to verify semester status.
// Used by enrollment, grades, and meal services.
//
// Endpoint: GET /api/catalog/admin/semesters/{name}/status
//
// Kullanım:
//   checker := semester.NewHTTPChecker("http://localhost:8004")
//   active, err := checker.IsSemesterActive(ctx, "2025-2026-Fall")

type HTTPChecker struct {
    catalogBaseURL string  // örn: "http://localhost:8004"
    httpClient     *http.Client
}
```

**NOT**: Bu endpoint'e Traefik'ten erişim için `catalog-admin` router'ı zaten
`forward-auth` middleware'i kullanıyor. Servisler arası iletişimde ise direkt
`host.docker.internal:8004` üzerinden erişim yapılır (Traefik bypass).

---

## 3. Özellik 2: Period CRUD Kilidi

Period handler'lar (hem `PeriodHandler` hem `SimplePeriodHandler`) her Create/Update/Delete
işleminde semester'ın aktif olup olmadığını kontrol edecek.

### 3.1 Handler Değişiklikleri

**`backend/shared/handler/period_handler.go`** — değişiklikler:

```go
// Mevcut struct:
type PeriodHandler struct {
    repo *repository.PeriodRepository
}

// Yeni struct (SemesterChecker ekleniyor):
type PeriodHandler struct {
    repo            *repository.PeriodRepository
    semesterChecker semester.Checker  // yeni alan
}

// Constructor değişir:
func NewPeriodHandler(repo *repository.PeriodRepository, checker semester.Checker) *PeriodHandler {
    return &PeriodHandler{repo: repo, semesterChecker: checker}
}
```

**CreatePeriod metoduna eklenen kontrol:**
```go
func (h *PeriodHandler) CreatePeriod(c *gin.Context) {
    // ... mevcut request parse ...

    // YENİ: Semester aktif mi kontrol et
    active, err := h.semesterChecker.IsSemesterActive(c.Request.Context(), req.Semester)
    if err != nil || !active {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "SEMESTER_NOT_ACTIVE",
            "message": "Cannot modify periods for a non-active semester",
        })
        return
    }

    // ... mevcut CRUD devam ...
}
```

Aynı kontrol `UpdatePeriod` ve `DeletePeriod`'a da eklenir (period'un semester'ını
DB'den çekip kontrol eder).

**`backend/shared/handler/simple_period_handler.go`** — aynı değişiklikler uygulanır.

### 3.2 Meal Service Closed Days Handler

**`backend/services/meal-service/internal/handler/closed_days_handler.go`** — değişiklikler:

Closed days semester'a bağlı değil (tarih bazlı), ama yine de audit log'a yazılacak.
Semester kilidi closed days için geçerli DEĞİL — çünkü tatil günleri dönemden bağımsız.

### 3.3 Etkilenen cmd/main.go Dosyaları

Her serviste `NewPeriodHandler` veya `NewSimplePeriodHandler` çağrısına
`semesterChecker` parametresi eklenir:

- `backend/services/course-catalog-service/cmd/main.go`:
  ```go
  // Catalog kendi DB'sine direkt bakabilir
  checker := semesterStatusRepo  // (SemesterStatusRepository zaten Checker interface'ini karşılar)
  periodHandler := sharedHandler.NewSimplePeriodHandler(periodRepo, checker)
  ```

- `backend/services/enrollment-service/cmd/main.go`:
  ```go
  checker := semester.NewHTTPChecker("http://localhost:8004")
  periodHandler := sharedHandler.NewSimplePeriodHandler(periodRepo, checker)
  ```

- `backend/services/grades-service/cmd/main.go`:
  ```go
  checker := semester.NewHTTPChecker("http://localhost:8004")
  periodHandler := sharedHandler.NewPeriodHandler(periodRepo, checker)
  ```

---

## 4. Özellik 3: Hard Deadline ile Otomatik Kilit

Bu özellik Özellik 1 içinde zaten tanımlandı. Ek bir dosya gerektirmiyor.

### Nasıl çalışır:

1. Admin semester oluştururken `hard_deadline` tarihi verir
2. `IsSemesterActive()` her çağrıldığında:
   - `status == "active"` VE `hard_deadline > NOW()` → aktif
   - `status == "active"` VE `hard_deadline <= NOW()` → otomatik `completed`'e güncelle, aktif değil
3. DB trigger geri dönüşü engelliyor (completed → active asla mümkün değil)

### Örnek senaryo:
```
Semester: 2025-2026-Spring
Hard Deadline: 2026-07-01T00:00:00Z
Status: active

→ 30 Haziran 2026'da admin deadline ekleyebilir ✅
→ 1 Temmuz 2026'da IsSemesterActive() çağrılınca:
  - hard_deadline geçmiş → otomatik completed
  - Admin artık bu döneme dokunAMAZ ❌
```

---

## 5. Özellik 4: Audit Log (PostgreSQL)

### 5.1 Neden PostgreSQL?

- Zaten mevcut altyapıda var — ek bağımlılık yok
- SQL ile doğrudan sorgulama (WHERE, JOIN, GROUP BY, sayfalama)
- DB trigger ile UPDATE/DELETE engellenir → değiştirilemez log
- pg_dump ile zaten yedekleniyor
- Frontend'e sunmak için basit bir GET endpoint yeterli

### 5.2 Migration Dosyası

**Dosya**: `backend/services/course-catalog-service/sql/migrations/00011_create_audit_log_table.sql`

```sql
-- +goose Up

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    service VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    actor_role VARCHAR(20) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB
);

CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_service ON audit_log(service);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_id);

-- DEĞİŞTİRİLEMEZLİK: UPDATE ve DELETE'i DB seviyesinde engelle
-- Uygulama katmanı bypass edilse bile (SQL injection vb.) audit kayıtları korunur
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_no_update
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- +goose Down
DROP TRIGGER IF EXISTS trg_audit_no_update ON audit_log;
DROP FUNCTION IF EXISTS prevent_audit_modification();
DROP TABLE IF EXISTS audit_log;
```

### 5.3 SQL Queries

**Dosya**: `backend/services/course-catalog-service/sql/queries/audit_log.sql`

```sql
-- name: InsertAuditLog :one
INSERT INTO audit_log (service, actor_id, actor_role, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAuditLog :many
SELECT * FROM audit_log
WHERE
    (sqlc.narg('service')::VARCHAR IS NULL OR service = sqlc.narg('service')) AND
    (sqlc.narg('action')::VARCHAR IS NULL OR action = sqlc.narg('action')) AND
    (sqlc.narg('actor_id')::UUID IS NULL OR actor_id = sqlc.narg('actor_id'))
ORDER BY timestamp DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLog :one
SELECT COUNT(*) FROM audit_log
WHERE
    (sqlc.narg('service')::VARCHAR IS NULL OR service = sqlc.narg('service')) AND
    (sqlc.narg('action')::VARCHAR IS NULL OR action = sqlc.narg('action')) AND
    (sqlc.narg('actor_id')::UUID IS NULL OR actor_id = sqlc.narg('actor_id'));
```

### 5.4 Repository & Logger

**Dosya**: `backend/services/course-catalog-service/internal/repository/audit_repository.go`

```go
package repository

// AuditRepository — audit_log tablosu CRUD işlemleri
type AuditRepository struct {
    queries *db.Queries
}

// Temel metodlar:
// - InsertAuditLog(ctx, params) → AuditLog
// - ListAuditLog(ctx, params) → []AuditLog
// - CountAuditLog(ctx, params) → int64
```

**Dosya**: `backend/shared/audit/logger.go` (yeni dosya)

```go
package audit

import "context"

// AuditEvent represents an immutable audit log entry.
type AuditEvent struct {
    ActorID      string         `json:"actor_id"`
    ActorRole    string         `json:"actor_role"`
    Action       string         `json:"action"`
    ResourceType string         `json:"resource_type"`
    ResourceID   string         `json:"resource_id"`
    Details      map[string]any `json:"details"`
}

// Logger interface — catalog servisi doğrudan DB'ye yazar,
// diğer servisler catalog'un HTTP endpoint'ini çağırır.
type Logger interface {
    Log(ctx context.Context, event AuditEvent) error
}
```

**Dosya**: `backend/shared/audit/http_logger.go` (yeni dosya)

```go
package audit

// HTTPLogger calls catalog service to write audit log entries.
// Used by enrollment, grades, and meal services.
//
// Endpoint: POST /api/catalog/internal/audit-log
//
// Kullanım:
//   logger := audit.NewHTTPLogger("http://localhost:8004")
//   logger.Log(ctx, audit.AuditEvent{...})

type HTTPLogger struct {
    catalogBaseURL string
    httpClient     *http.Client
    serviceName    string
}
```

### 5.5 Handler

**Dosya**: `backend/services/course-catalog-service/internal/handler/audit_handler.go`

```go
package handler

// AuditHandler — Audit log endpoint'leri

// Endpoint'ler:
// GET  /api/catalog/admin/audit-log           → ListAuditLog (admin dashboard için)
// POST /api/catalog/internal/audit-log        → InsertAuditLog (diğer servislerden gelen yazma)
```

**ListAuditLog query parametreleri:**
- `service` — filtreleme (opsiyonel)
- `action` — filtreleme (opsiyonel)
- `actor_id` — filtreleme (opsiyonel)
- `limit` — sayfalama (default: 50)
- `offset` — sayfalama (default: 0)

### 5.6 Hangi İşlemler Loglanır

| Action | Service | Tetiklendiği Yer |
|---|---|---|
| `semester.created` | catalog | SemesterStatusHandler.CreateSemester |
| `semester.activated` | catalog | SemesterStatusHandler.ActivateSemester |
| `semester.completed` | catalog | SemesterStatusHandler.CompleteSemester |
| `semester.auto_completed` | catalog | SemesterStatusRepository.IsSemesterActive (hard_deadline) |
| `period.created` | catalog, enrollment, grades | PeriodHandler.CreatePeriod / SimplePeriodHandler.CreatePeriod |
| `period.updated` | catalog, enrollment, grades | PeriodHandler.UpdatePeriod / SimplePeriodHandler.UpdatePeriod |
| `period.deleted` | catalog, enrollment, grades | PeriodHandler.DeletePeriod / SimplePeriodHandler.DeletePeriod |
| `closed_day.created` | meal | ClosedDaysHandler.CreateClosedDay |
| `closed_day.deleted` | meal | ClosedDaysHandler.DeleteClosedDay |
| `grade.appeal_processed` | grades | GradeService.ProcessAppeal |

### 5.7 Audit Event Örnekleri

**Semester aktivasyonu:**
```json
{
    "id": "550e8400-...",
    "timestamp": "2026-02-20T10:30:00Z",
    "service": "catalog",
    "actor_id": "00000000-0000-0000-0000-000000000001",
    "actor_role": "admin",
    "action": "semester.activated",
    "resource_type": "semester",
    "resource_id": "a1b2c3d4-...",
    "details": {
        "semester_name": "2025-2026-Spring",
        "hard_deadline": "2026-07-01T00:00:00Z"
    }
}
```

**Period silme:**
```json
{
    "id": "660e8400-...",
    "timestamp": "2026-03-15T14:20:00Z",
    "service": "grades",
    "actor_id": "00000000-0000-0000-0000-000000000001",
    "actor_role": "admin",
    "action": "period.deleted",
    "resource_type": "academic_period",
    "resource_id": "b2c3d4e5-...",
    "details": {
        "semester": "2025-2026-Spring",
        "period_start": "2026-02-01T00:00:00Z",
        "period_end": "2026-06-30T23:59:59Z",
        "course_id": null
    }
}
```

**Not itirazı:**
```json
{
    "id": "770e8400-...",
    "timestamp": "2026-09-01T09:00:00Z",
    "service": "grades",
    "actor_id": "00000000-0000-0000-0000-000000000001",
    "actor_role": "admin",
    "action": "grade.appeal_processed",
    "resource_type": "grade",
    "resource_id": "c3d4e5f6-...",
    "details": {
        "student_id": "d4e5f6g7-...",
        "course_code": "CS101",
        "semester": "2024-2025-Spring",
        "slug": "final",
        "old_score": 45.0,
        "new_score": 65.0,
        "old_grade_point": "FF",
        "new_grade_point": "CB",
        "reason": "Sınav kağıdı yeniden değerlendirildi"
    }
}
```

### 5.8 Diğer Servislerin Audit Log Yazması

Enrollment, grades ve meal servisleri catalog servisinin internal endpoint'ini çağırarak audit log yazar:

```
POST /api/catalog/internal/audit-log
```

Bu endpoint Traefik'ten geçmez — servisler arası direkt iletişim (`host.docker.internal:8004`).

---

## 6. Özellik 5: Appeal (İtiraz) İyileştirmesi

### 6.1 DTO Değişikliği

**`backend/services/grades-service/internal/dto/grade_dto.go`**:

```go
// Mevcut:
type AppealScoreRequest struct {
    StudentID uuid.UUID `json:"student_id" binding:"required"`
    CourseID  uuid.UUID `json:"course_id" binding:"required"`
    Slug      string    `json:"slug" binding:"required"`
    NewScore  float64   `json:"new_score" binding:"required,min=0,max=100"`
}

// Yeni (reason alanı ekleniyor):
type AppealScoreRequest struct {
    StudentID uuid.UUID `json:"student_id" binding:"required"`
    CourseID  uuid.UUID `json:"course_id" binding:"required"`
    Slug      string    `json:"slug" binding:"required"`
    NewScore  float64   `json:"new_score" binding:"required,min=0,max=100"`
    Reason    string    `json:"reason" binding:"required,min=10"` // minimum 10 karakter gerekçe
}
```

### 6.2 Service Değişikliği

**`backend/services/grades-service/internal/service/grade_service.go`** — `ProcessAppeal` fonksiyonuna:

```go
func (s *GradeService) ProcessAppeal(ctx context.Context, req dto.AppealScoreRequest) (*dto.AppealScoreResponse, error) {
    // ... mevcut appeal mantığı ...

    // YENİ: İşlem bittikten sonra audit log yaz
    s.auditLogger.Log(ctx, audit.AuditEvent{
        ActorID:      getActorIDFromContext(ctx), // middleware'den gelen user_id
        ActorRole:    "admin",
        Action:       "grade.appeal_processed",
        ResourceType: "grade",
        ResourceID:   req.CourseID.String(),
        Details: map[string]any{
            "student_id":      req.StudentID,
            "course_code":     completedCourse.CourseCode,
            "semester":        completedCourse.Semester,
            "slug":            req.Slug,
            "old_score":       oldScore,
            "new_score":       req.NewScore,
            "old_grade_point": oldGradePoint,
            "new_grade_point": newGradePoint,
            "reason":          req.Reason,     // İtiraz gerekçesi
        },
    })

    return response, nil
}
```

---

## 7. Frontend Güncellemeleri

### 7.1 Yeni TypeScript Tipleri

**`frontend/lib/types.ts`** — eklenecekler:

```typescript
// Semester State Machine
export type SemesterStatus = 'planned' | 'active' | 'completed';

export interface Semester {
    id: string;
    name: string;                // "2025-2026-Fall"
    status: SemesterStatus;
    hard_deadline: string;       // ISO 8601
    activated_at?: string;
    completed_at?: string;
    created_at: string;
    updated_at: string;
}

export interface CreateSemesterRequest {
    name: string;
    hard_deadline: string;
}

// Audit Log
export interface AuditLogEntry {
    id: string;
    timestamp: string;
    service: string;
    actor_id: string;
    actor_role: string;
    action: string;
    resource_type: string;
    resource_id: string;
    details: Record<string, any>;
}
```

### 7.2 Yeni Servis Fonksiyonları

**`frontend/lib/services/system-service.ts`** — eklenecekler:

```typescript
// Semester Management
export async function listSemesters(): Promise<Semester[]> { ... }
export async function createSemester(data: CreateSemesterRequest): Promise<Semester> { ... }
export async function activateSemester(id: string): Promise<Semester> { ... }
export async function completeSemester(id: string): Promise<Semester> { ... }

// Audit Log
export async function listAuditLog(params?: {
    service?: string;
    action?: string;
    limit?: number;
}): Promise<AuditLogEntry[]> { ... }
```

### 7.3 Yeni UI Sayfaları/Bileşenleri

**`frontend/app/(admin)/system/periods/page.tsx`** — mevcut sayfaya ekleme:
- Sayfanın üst kısmına **aktif semester bilgisi** banner'ı eklenir
- Eğer aktif semester yoksa → "Önce bir dönem aktifleştirin" uyarısı gösterilir
- Period CRUD butonları aktif semester yoksa disabled olur

**`frontend/app/(admin)/system/semesters/page.tsx`** — yeni sayfa:
- Semester listesi (planned / active / completed badge'leri ile)
- "Yeni Dönem Oluştur" butonu ve formu (name + hard_deadline)
- "Aktifleştir" butonu (planned dönemlerde) — onay dialogu ile
- "Tamamla" butonu (active dönemlerde) — onay dialogu ile
- Geri dönüş butonu YOK (tasarım gereği)
- Hard deadline yaklaşıyorsa uyarı banner'ı

**`frontend/app/(admin)/system/audit/page.tsx`** — yeni sayfa:
- Audit log tablosu (tarih, servis, işlem, aktör, detay)
- Filtreleme: servise göre, işleme göre, tarihe göre
- Detay modal'ı: JSON details alanını güzel formatlayarak gösterir
- Sayfalama (limit/offset)

### 7.4 Sidebar Güncelleme

Sistem menüsüne iki yeni link eklenir:
- Dönem Yönetimi → `/system/semesters`
- Audit Log → `/system/audit`

---

## 8. Uygulama Sırası

Adımlar arasındaki bağımlılıklar dikkate alınarak sıralama:

### Adım 1: Semester State Machine (Catalog Servisinde)
1. Migration dosyası oluştur (semesters tablosu + DB trigger)
2. sqlc queries yaz + `sqlc generate`
3. `SemesterStatusRepository` oluştur
4. `SemesterStatusHandler` oluştur
5. `cmd/main.go`'ya entegre et
6. `semester_service.go`'ya aktif semester kontrolü ekle
7. Migration uygula + `go build` ile doğrula

### Adım 2: Shared Semester Checker
1. `backend/shared/semester/checker.go` — interface
2. `backend/shared/semester/http_checker.go` — HTTP implementasyonu
3. Catalog repo'nun Checker interface'ini karşıladığını doğrula

### Adım 3: Period CRUD Kilidi
1. `PeriodHandler`'a `SemesterChecker` inject et
2. `SimplePeriodHandler`'a `SemesterChecker` inject et
3. Create/Update/Delete'e semester kontrolü ekle
4. Her servisin `cmd/main.go`'suna checker'ı geç
5. Tüm servisleri build et

### Adım 4: Audit Log (PostgreSQL)
1. Migration dosyası oluştur (audit_log tablosu + immutability trigger)
2. sqlc queries yaz + `sqlc generate`
3. `AuditRepository` oluştur (catalog servisinde)
4. `AuditHandler` oluştur (GET + POST endpoint'ler)
5. `backend/shared/audit/logger.go` — Logger interface
6. `backend/shared/audit/http_logger.go` — HTTP implementasyonu (diğer servisler için)
7. Her serviste audit logger inject et ve kritik işlemlere log ekle
8. Migration uygula + `go build` ile doğrula

### Adım 5: Appeal İyileştirmesi
1. `AppealScoreRequest`'e `reason` alanı ekle
2. `ProcessAppeal`'a audit log yaz
3. `go build` ile doğrula

### Adım 6: Frontend
1. TypeScript tipleri ekle
2. Servis fonksiyonları ekle
3. Semester yönetimi sayfası
4. Periods sayfasına aktif semester banner'ı
5. Audit log sayfası
6. Sidebar güncelle
7. `bun tsc` ile doğrula

---

## 9. Dosya Değişiklik Özeti

### Yeni Dosyalar
| Dosya | Açıklama |
|---|---|
| `backend/services/course-catalog-service/sql/migrations/00010_create_semesters_table.sql` | Semester tablosu + DB trigger |
| `backend/services/course-catalog-service/sql/queries/semesters.sql` | Semester SQL sorguları |
| `backend/services/course-catalog-service/internal/repository/semester_status_repository.go` | Semester repository |
| `backend/services/course-catalog-service/internal/handler/semester_status_handler.go` | Semester admin endpoint'leri |
| `backend/shared/semester/checker.go` | SemesterChecker interface |
| `backend/shared/semester/http_checker.go` | HTTP bazlı semester checker |
| `backend/services/course-catalog-service/sql/migrations/00011_create_audit_log_table.sql` | Audit log tablosu + immutability trigger |
| `backend/services/course-catalog-service/sql/queries/audit_log.sql` | Audit log SQL sorguları |
| `backend/services/course-catalog-service/internal/repository/audit_repository.go` | Audit log repository |
| `backend/services/course-catalog-service/internal/handler/audit_handler.go` | Audit log endpoint'leri |
| `backend/shared/audit/logger.go` | Audit Logger interface |
| `backend/shared/audit/http_logger.go` | HTTP bazlı audit logger (diğer servisler için) |
| `frontend/app/(admin)/system/semesters/page.tsx` | Semester yönetimi UI |
| `frontend/app/(admin)/system/audit/page.tsx` | Audit log görüntüleme UI |

### Değiştirilecek Dosyalar
| Dosya | Değişiklik |
|---|---|
| `backend/shared/handler/period_handler.go` | SemesterChecker inject + kontrol |
| `backend/shared/handler/simple_period_handler.go` | SemesterChecker inject + kontrol |
| `backend/services/course-catalog-service/cmd/main.go` | Semester repo/handler ekleme |
| `backend/services/course-catalog-service/internal/service/semester_service.go` | Aktif semester kontrolü |
| `backend/services/course-catalog-service/internal/errors/errors.go` | ErrSemesterNotActive |
| `backend/services/enrollment-service/cmd/main.go` | HTTPChecker + HTTPAuditLogger ekleme |
| `backend/services/grades-service/cmd/main.go` | HTTPChecker + HTTPAuditLogger ekleme |
| `backend/services/grades-service/internal/dto/grade_dto.go` | AppealScoreRequest.Reason |
| `backend/services/grades-service/internal/service/grade_service.go` | Appeal audit log |
| `backend/services/meal-service/cmd/main.go` | HTTPAuditLogger ekleme |
| `backend/services/meal-service/internal/handler/closed_days_handler.go` | Audit log |
| `frontend/lib/types.ts` | Semester + AuditLogEntry tipleri |
| `frontend/lib/services/system-service.ts` | Semester + audit servis fonksiyonları |
| `frontend/app/(admin)/system/periods/page.tsx` | Aktif semester banner |

### sqlc Regenerate Gereken Servisler
- `backend/services/course-catalog-service/` (yeni semesters + audit_log queries)

---

## 10. Doğrulama Adımları

### Backend
```bash
# 1. Migration uygula
cd backend/services/course-catalog-service && goose -dir sql/migrations postgres "$DB_URL" up

# 2. DB trigger testi — completed semester geri aktif edilememeli
psql -c "UPDATE semesters SET status = 'active' WHERE status = 'completed';"
# → HATA beklenir: "Cannot change status of a completed semester"

# 3. DB trigger testi — audit log değiştirilemez olmalı
psql -c "DELETE FROM audit_log WHERE id = '...';"
# → HATA beklenir: "Audit log entries cannot be modified or deleted"

# 4. Tüm servisleri build et
cd backend/services/course-catalog-service && go build ./...
cd backend/services/enrollment-service && go build ./...
cd backend/services/grades-service && go build ./...
cd backend/services/meal-service && go build ./...

# 5. Period oluşturma testi — inactive semester için
curl -X POST /api/catalog/periods -d '{"semester":"2020-2021-Fall",...}'
# → 403 SEMESTER_NOT_ACTIVE beklenir

# 6. Audit log'a kayıt geldi mi?
psql -c "SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 5;"
```

### Frontend
```bash
cd frontend && bun tsc --noEmit
```

### Manuel Test Senaryoları
1. Yeni semester oluştur (planned) → deadline eklemeyi dene → ❌ reddedilmeli
2. Semester'ı aktifleştir → deadline ekle → ✅ başarılı
3. Semester'ı tamamla → deadline silmeyi dene → ❌ reddedilmeli
4. Tamamlanmış semester'ı geri aktifleştirmeyi dene → ❌ DB trigger hatası
5. Hard deadline geçmiş active semester → otomatik completed olmalı
6. Not itirazı yap → audit log'da görünmeli (reason ile birlikte)
7. Audit log sayfasını aç → tüm işlemler listelenmiş olmalı
