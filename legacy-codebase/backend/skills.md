# Backend — Go Mikroservisler (AI Talimati)

Go 1.26 + Gin + sqlc + pgx + RabbitMQ + Redis. `backend/services/**` veya `backend/shared/**` icinde calisirken bu dosya zorunlu okumadir.

---

## 1. Sert Kurallar (asla ihlal etme)

- **pgtype**: `shared/utils/pgtype_helpers.go` fonksiyonlarini kullan. Struct literal (`pgtype.Text{...}`) ve `AssignTo()` YAPMA.
- **Inter-service**: **Tercih RabbitMQ event-driven** (notify, side-effect, eventual consistency). **Sync HTTP kabul edilir** sadece su uc durum icin: (a) read-only lookup ("bu advisor var mi?"), (b) anlik validasyon (event ile yapilamaz — yarış/asynchrony), (c) fan-out orkestrasyon (semester wizard gibi). Bu durumlarda **`internal/...` prefix'li route + `X-Internal-Secret` header** zorunlu, public route'a HTTP cagrisi YASAK. JWT validasyonu her servisin kendi `JWTAuth` middleware'inde — auth servise HTTP RPC yok.
- **DB izolasyonu**: Her servisin ayri DB'si. Baska servisin tablosuna SELECT YAPMA.
- **Frontend erisim**: Sadece Traefik uzerinden (`http://localhost/api/{service}`). Servis portuna direkt erisim YAPMA.
- **Generated kod**: `internal/db/*.go` sqlc cikti — manuel DUZENLEME YAPMA. Query degisirse `sql/queries/*.sql` duzenle, sonra `make sqlc`.
- **Error karsilastirma**: `errors.Is(err, target)` ve `errors.As(err, &t)` kullan. Ciplak `==` veya `.(*T)` type-assert YAPMA — wrapped error'u kacirir.
- **Error olusturma**: Sentinel/AppError tanimlari `errors.New(...)` veya `sharedErrors.New(...)`. Wrap'lerken: 
  - Sentinel ekleyerek (genel iclerden): `fmt.Errorf("%w: context", err)` 
  - AppError'a sarip HTTP'ye cevirmek icin (service -> handler): `sharedErrors.Wrap(sharedErrors.ErrInternal, err)`
- **Logger**: Her handler/service metodunda `logger.WithContextAndFields(ctx, ...)` ile child logger uret. Global `logger.Info` direkt cagirma.

---

## 2. Servis Iskeletinin Sabit Yapisi

```
service-name/
├── cmd/main.go              # Entry point (logger init, pool init, router)
├── config/config.go         # Viper config
├── internal/
│   ├── db/                  # sqlc generated — DOKUNMA
│   ├── repository/          # sqlc query'lerini saran katman
│   ├── service/             # Business logic (pgtype donusumleri burada)
│   ├── handler/             # Gin HTTP handler'lar
│   ├── dto/                 # Request/Response struct'lari
│   ├── worker/              # outbox + event consumer
│   └── errors/              # Servise ozel sentinel error'lar
├── sql/
│   ├── migrations/          # goose .sql dosyalari
│   └── queries/             # sqlc input
├── sqlc.yaml
├── Makefile
├── .env.example
└── go.mod
```

---

## 3. Yeni Endpoint Workflow (sira zorunlu)

```
1. Migration:        make migrate-create name=create_xxx
2. Schema yaz:       sql/migrations/NNNN_create_xxx.sql (-- +goose Up/Down)
3. Migrate calistir: make migrate-up   (KULLANICI calistirir, sen calistirma)
4. Query yaz:        sql/queries/xxx.sql (sqlc annotation)
5. Generate:         make sqlc
6. Repository:       internal/repository/xxx_repository.go (Bolum 6 sablonu)
7. DTO:              internal/dto/xxx_dto.go (Bolum 7 sablonu)
8. Service:          internal/service/xxx_service.go (Bolum 8 sablonu)
9. Handler:          internal/handler/xxx_handler.go (Bolum 9 sablonu)
10. Route bagla:     cmd/main.go (router.POST/GET)
11. Test:            go test ./internal/...
12. Commit:          feat(xxx): add yyy endpoint
```

**YAPMA:** Sira atlama. Repository sqlc'siz yazma, service repository'siz yazma.

---

## 4. Migration Yazma

### Goose `$$` kurali (kritik)
PL/pgSQL fonksiyon/trigger icin **zorunlu** olarak `StatementBegin/End` sar:

```sql
-- DOGRU
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_updated_at();
-- +goose StatementEnd
```

### Migration kurallari
- `Up` ve `Down` her zaman dolu olsun (rollback test edilebilmeli)
- `NOT NULL` eklerken once `DEFAULT` ile backfill, sonra `NOT NULL` (ayri migration)
- `DROP COLUMN` yazma — kullaniciya sor (data loss)
- Index ekleme: `CREATE INDEX CONCURRENTLY` (prod'da lock yapmasin)
- Index ismi: `idx_{table}_{column}` veya `idx_{table}_{col1}_{col2}`

### Migration dosya adi
Format: `NNNNN_<verb>_<object>.sql` — `make migrate-create name=<verb>_<object>` otomatik 5-haneli numara verir.

| Verb | Ne zaman |
|---|---|
| `create` | Yeni tablo / index / function |
| `add` | Mevcut tabloya kolon |
| `drop` | Kolon/tablo sil (ONAY GEREKIR) |
| `alter` | Kolon tipi/constraint degistir |
| `backfill` | Veri doldurma migration'i |

Ornek: `00012_add_session_type_to_schedule_sessions.sql`, `00013_create_idx_attendance_student_date.sql`.

### UUID uretimi (kim uretir?)
- **DB-side default**: Birincil anahtarlar her zaman `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`. Service kod `id` gondermez, INSERT sonrasi `RETURNING id` ile alir.
- **Code-side `uuid.New()`**: Sadece event payload'lari (event ID), idempotency key, transaction ID gibi DB-disi kullanim icin.
- **Asla**: Service'te `uuid.New()` ile ID uret, sonra INSERT'e ekle. Kosulu race ve duplicate riski.

---

## 5. sqlc Query Yazma

```sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND is_active = true LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, role, created_at;

-- name: ListActiveUsers :many
SELECT * FROM users WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1;
```

**Annotation kurallari:**
- `:one` — tek satir doner, `pgx.ErrNoRows` handle edilir
- `:many` — slice doner
- `:exec` — sadece etki, return yok
- `:execrows` — etki + RowsAffected

**YAPMA:** `SELECT *` production-critical query'de — sutun ekleyince beklenmedik davranis. Ad list ver.

---

## 6. Repository Sablonu (kopyala-uyarla)

```go
package repository

import (
    "context"
    "errors"
    "fmt"

    "github.com/baaaki/mydreamcampus/{service}/internal/db"
    serviceErrors "github.com/baaaki/mydreamcampus/{service}/internal/errors"
    sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
    "github.com/baaaki/mydreamcampus/shared/utils"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

type XxxRepository struct {
    queries *db.Queries
    pool    *pgxpool.Pool
}

func NewXxxRepository(pool *pgxpool.Pool) *XxxRepository {
    return &XxxRepository{
        queries: db.New(pool),
        pool:    pool,
    }
}

func (r *XxxRepository) GetByID(ctx context.Context, id uuid.UUID) (db.Xxx, error) {
    row, err := r.queries.GetXxxByID(ctx, utils.UUIDToPgtype(id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return db.Xxx{}, fmt.Errorf("%w: xxx with id %s not found", serviceErrors.ErrXxxNotFoundRepo, id)
        }
        return db.Xxx{}, fmt.Errorf("%w: get xxx: %v", sharedErrors.ErrQueryFailed, err)
    }
    return row, nil
}

func (r *XxxRepository) Create(ctx context.Context, params db.CreateXxxParams) (db.Xxx, error) {
    row, err := r.queries.CreateXxx(ctx, params)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) {
            switch pgErr.Code {
            case "23505": // unique violation
                return db.Xxx{}, fmt.Errorf("%w: duplicate", serviceErrors.ErrXxxExistsRepo)
            case "23503": // foreign key violation
                return db.Xxx{}, fmt.Errorf("%w: fk %s", sharedErrors.ErrForeignKeyViolation, pgErr.ConstraintName)
            }
        }
        return db.Xxx{}, fmt.Errorf("%w: create xxx: %v", sharedErrors.ErrQueryFailed, err)
    }
    return row, nil
}
```

> **Mevcut kod uyarisi**: Bazi servislerde (ornek: `auth-service/internal/repository/auth_repository.go`) hala `if err == pgx.ErrNoRows` ve `err.(*pgconn.PgError)` pattern'i kullaniliyor. **Yeni kod yazarken yukaridaki sablonu** kullan. Eski kodu refactor etmek icin kullanici onayi gerekir (3+ dosya degisikligi — bkz. CLAUDE.md Bolum 6).

**Repository kurallari:**
- Sadece DB I/O yapar — business logic YOK
- `pgx.ErrNoRows` her zaman repo-specific sentinel'e cevrilir (`ErrXxxNotFoundRepo`)
- `23505` (unique violation) → `ErrXxxExistsRepo`
- `23503` (FK violation) → `ErrForeignKeyViolation`
- pgtype donusumlerini repository'de YAPMA, service'te yap (input parametreleri pgtype gelir, output pgtype doner)

---

## 7. DTO Sablonu

```go
package dto

import "time"

// Request — Gin binding tag'leri zorunlu
type CreateXxxRequest struct {
    Name  string `json:"name" binding:"required,min=2,max=100"`
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age" binding:"required,gte=0,lte=150"`
}

// Response — sadece JSON tag, optional alanlar pointer
type XxxResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Avatar    *string   `json:"avatar,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

// Liste response her zaman wrapper struct icinde
type XxxListResponse struct {
    Items []XxxResponse `json:"items"`
    Total int           `json:"total"`
}
```

**DTO kurallari:**
- Request struct'lari `binding:` ile validate (Gin `ShouldBindJSON` kontrol eder)
- Response'da `omitempty` sadece optional alanlarda
- ID string (UUID), client'a `pgtype` veya `uuid.UUID` SIZDIRMA
- Tarih `time.Time` — RFC3339 JSON serialize otomatik
- DB struct'ini direkt response yapma; her zaman ayri DTO

---

## 8. Service Sablonu

```go
package service

import (
    "context"
    "fmt"

    "github.com/baaaki/mydreamcampus/{service}/internal/db"
    "github.com/baaaki/mydreamcampus/{service}/internal/dto"
    serviceErrors "github.com/baaaki/mydreamcampus/{service}/internal/errors"
    "github.com/baaaki/mydreamcampus/{service}/internal/repository"
    sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
    "github.com/baaaki/mydreamcampus/shared/logger"
    "github.com/baaaki/mydreamcampus/shared/utils"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

type XxxService struct {
    xxxRepo *repository.XxxRepository
}

func NewXxxService(xxxRepo *repository.XxxRepository) *XxxService {
    return &XxxService{xxxRepo: xxxRepo}
}

func (s *XxxService) Create(ctx context.Context, req dto.CreateXxxRequest) (dto.XxxResponse, error) {
    log := logger.WithContextAndFields(ctx,
        zap.String("service", "XxxService"),
        zap.String("method", "Create"),
    )

    params := db.CreateXxxParams{
        Name:  req.Name,
        Email: req.Email,
    }

    row, err := s.xxxRepo.Create(ctx, params)
    if err != nil {
        if sharedErrors.Is(err, serviceErrors.ErrXxxExistsRepo) {
            return dto.XxxResponse{}, serviceErrors.ErrXxxExists
        }
        log.Error("create xxx failed", zap.Error(err))
        return dto.XxxResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
    }

    return dto.XxxResponse{
        ID:        utils.PgUUIDToString(row.ID),
        Name:      row.Name,
        Email:     row.Email,
        CreatedAt: row.CreatedAt.Time,
    }, nil
}

func (s *XxxService) GetByID(ctx context.Context, id string) (dto.XxxResponse, error) {
    parsedID, err := uuid.Parse(id)
    if err != nil {
        return dto.XxxResponse{}, fmt.Errorf("%w: invalid id format", sharedErrors.ErrBadRequest)
    }

    row, err := s.xxxRepo.GetByID(ctx, parsedID)
    if err != nil {
        if sharedErrors.Is(err, serviceErrors.ErrXxxNotFoundRepo) {
            return dto.XxxResponse{}, serviceErrors.ErrXxxNotFound
        }
        return dto.XxxResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
    }

    return dto.XxxResponse{
        ID:    utils.PgUUIDToString(row.ID),
        Name:  row.Name,
        Email: row.Email,
    }, nil
}
```

**Service kurallari:**
- Business logic burada (validation, kosullu mantik, event publish)
- pgtype <-> Go donusumleri burada yapilir (`utils.PgUUIDToString`, `utils.StringToPgText` vs.)
- Repository sentinel'lerini service-level error'a cevir
- Beklenmedik error'lari `sharedErrors.Wrap(ErrInternal, err)` ile sar
- Logger her metodun basinda `WithContextAndFields` ile uret

---

## 9. Handler Sablonu

```go
package handler

import (
    "context"
    "net/http"
    "time"

    "github.com/baaaki/mydreamcampus/{service}/internal/dto"
    serviceErrors "github.com/baaaki/mydreamcampus/{service}/internal/errors"
    "github.com/baaaki/mydreamcampus/{service}/internal/service"
    sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
    "github.com/baaaki/mydreamcampus/shared/logger"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

const requestTimeout = 10 * time.Second

type XxxHandler struct {
    xxxService *service.XxxService
}

func NewXxxHandler(xxxService *service.XxxService) *XxxHandler {
    return &XxxHandler{xxxService: xxxService}
}

func (h *XxxHandler) Create(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
    defer cancel()

    log := logger.WithContextAndFields(ctx,
        zap.String("endpoint", "Create"),
        zap.String("handler", "XxxHandler"),
    )

    var req dto.CreateXxxRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, dto.ErrorResponse{
            Error:   "VALIDATION_ERROR",
            Message: err.Error(),
        })
        return
    }

    resp, err := h.xxxService.Create(ctx, req)
    if err != nil {
        log.Error("create failed", zap.Error(err))

        if sharedErrors.Is(err, serviceErrors.ErrXxxExists) {
            c.JSON(http.StatusConflict, dto.ErrorResponse{
                Error:   "XXX_EXISTS",
                Message: "Bu kayit zaten mevcut",
            })
            return
        }

        c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
            Error:   "INTERNAL_ERROR",
            Message: "Bir hata olustu",
        })
        return
    }

    c.JSON(http.StatusCreated, resp)
}

func (h *XxxHandler) GetByID(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
    defer cancel()

    log := logger.WithContextAndFields(ctx, zap.String("endpoint", "GetByID"))

    id := c.Param("id")
    resp, err := h.xxxService.GetByID(ctx, id)
    if err != nil {
        if sharedErrors.Is(err, serviceErrors.ErrXxxNotFound) {
            c.JSON(http.StatusNotFound, dto.ErrorResponse{
                Error:   "XXX_NOT_FOUND",
                Message: "Kayit bulunamadi",
            })
            return
        }
        log.Error("get failed", zap.Error(err))
        c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
            Error: "INTERNAL_ERROR",
        })
        return
    }

    c.JSON(http.StatusOK, resp)
}
```

**Handler kurallari:**
- `context.WithTimeout` zorunlu (10s default)
- `ShouldBindJSON` validation hatasi → 400 `VALIDATION_ERROR`
- Service error'lari sentinel kontrolu ile HTTP status'a cevrilir
- Kullanici mesaji **Turkce**, error code **UPPERCASE_SNAKE**
- Internal error log'lanir, kullaniciya generic mesaj
- Business logic YOK, sadece request/response orchestration

### HTTP Status Eslestirme

| Servis Error | HTTP Status | Error Code |
|---|---|---|
| Validation/binding fail | 400 | `VALIDATION_ERROR` |
| Auth gerekiyor | 401 | `UNAUTHORIZED` |
| Yetki yok | 403 | `FORBIDDEN` |
| Bulunamadi | 404 | `XXX_NOT_FOUND` |
| Cakisma (duplicate) | 409 | `XXX_EXISTS` |
| Rate limit | 429 | `RATE_LIMIT_EXCEEDED` |
| Beklenmedik | 500 | `INTERNAL_ERROR` |

---

## 10. Servis-Ozel Errors

```go
package errors

import (
    "net/http"
    sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
)

var (
    // Public error'lar (handler'da HTTP'ye cevrilir)
    ErrXxxNotFound = sharedErrors.New("XXX_NOT_FOUND", "Kayit bulunamadi", http.StatusNotFound)
    ErrXxxExists   = sharedErrors.New("XXX_EXISTS", "Kayit zaten var", http.StatusConflict)

    // Repository sentinel (service icinde sharedErrors.Is ile kontrol edilir)
    ErrXxxNotFoundRepo = sharedErrors.ErrNotFoundRepo
    ErrXxxExistsRepo   = sharedErrors.ErrAlreadyExistsRepo
)
```

**Naming:**
- Public: `ErrXxx{Action}` (`ErrUserNotFound`, `ErrEmailExists`)
- Repo sentinel: `ErrXxx{Action}Repo` (sadece servis icinde gorulur)
- Error code: `UPPERCASE_SNAKE_CASE`
- Mesaj: Turkce (kullaniciya gosterilecek)

---

## 11. pgtype Hizli Referans

| pgtype | Go'ya | pgtype'a |
|---|---|---|
| `pgtype.Text` | `utils.PgTextToString(v)` | `utils.StringToPgText(s)` |
| `pgtype.Bool` | `utils.PgBoolToBool(v)` | `utils.BoolToPgBool(b)` |
| `pgtype.Numeric` | `utils.PgNumericToFloat64(v)` | `utils.Float64ToPgNumeric(f)` |
| `pgtype.Int2` | `int16(v.Int16)` | `utils.Int16ToPgInt2(i)` |
| `pgtype.Timestamp` | `utils.PgTimestampToTime(v)` | `utils.TimeToPgTimestamp(t)` |
| `pgtype.UUID` | `utils.PgUUIDToString(v)` | `utils.UUIDToPgtype(uid)` |

### pgtype YAPILMAZ Listesi

```go
// 1. Struct literal — YASAK
params.Email = pgtype.Text{String: "x", Valid: true}  // ❌
params.Email = utils.StringToPgText("x")              // ✅

// 2. Boolean check pgtype.Bool uzerinde — YASAK
if user.IsActive {              // ❌ struct, true degerlendirilmez
if utils.PgBoolToBool(user.IsActive) {  // ✅

// 3. AssignTo — YASAK (pgx v5'te kaldirildi)
result.Score.AssignTo(&score)   // ❌
score, err := utils.PgNumericToFloat64(result.Score)  // ✅

// 4. Aggregate sonucu interface{} type assert
var total float64
if t, ok := result.Total.(float64); ok {  // ✅
    total = t
}
```

---

## 12. Transaction Pattern

Birden fazla insert/update **atomic** olmali ise (en sik: outbox + business write). `pool.Begin(ctx)` + `defer tx.Rollback(ctx)` + `tx.Commit(ctx)`:

```go
func (r *XxxRepository) CreateWithEvent(ctx context.Context, params db.CreateXxxParams, event []byte) (db.Xxx, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return db.Xxx{}, fmt.Errorf("%w: begin tx: %v", sharedErrors.ErrTransactionFailed, err)
    }
    defer tx.Rollback(ctx) // Commit basarili olursa no-op

    qtx := r.queries.WithTx(tx)

    row, err := qtx.CreateXxx(ctx, params)
    if err != nil {
        return db.Xxx{}, fmt.Errorf("%w: create xxx: %v", sharedErrors.ErrQueryFailed, err)
    }

    if err := qtx.InsertOutbox(ctx, db.InsertOutboxParams{
        EventType: "xxx.created",
        Payload:   event,
    }); err != nil {
        return db.Xxx{}, fmt.Errorf("%w: outbox insert: %v", sharedErrors.ErrQueryFailed, err)
    }

    if err := tx.Commit(ctx); err != nil {
        return db.Xxx{}, fmt.Errorf("%w: commit: %v", sharedErrors.ErrTransactionFailed, err)
    }
    return row, nil
}
```

**Kurallar:**
- `defer tx.Rollback(ctx)` her zaman ilk satir — `Commit` basariliysa pgx no-op yapar, error path'inde garanti rollback
- `r.queries.WithTx(tx)` ile sqlc query'leri tx uzerinde calistir; `r.queries` kullanma (yeni connection alir, transaction'a girmez)
- Transaction icinde RabbitMQ publish **YAPMA** — outbox tablosuna yaz, `OutboxWorker` async publish eder. Aksi takdirde commit basarisiz olursa event gitmis olur.
- Transaction icinde uzun is **YAPMA** (HTTP cagrisi, file IO) — connection pool tutulur
- Nested transaction yok; servis metodu seviyesinde basla, ic metodlar `qtx` (`*db.Queries`) parametresi alsin

---

## 13. Outbox + Event Publish

### Yeni event eklerken
```
1. Event sabitini ekle:    shared/events/events.go (EventXxxCreated = "xxx.created")
2. Queue sabitini ekle:    shared/events/events.go (QueueAaaXxxEvents = "aaa.xxx.events")
3. Publisher tarafi:       service'te outbox tablosuna yaz
4. Tum consumer service'lerin DTO'larina event payload struct'ini ekle
5. Routing handler:        switch case'e ekle (event_consumer.go)
```

**YAPMA:**
- Event publish'i ayri transaction'da yapma — outbox tablosuna ayni transaction'da yaz
- Event'i RabbitMQ'ya dogrudan publish etme — `OutboxWorker` halleder
- Field rename — geriye uyumsuz, eski consumer kirilir

### Consumer kurallari
- Auto-ack KAPALI, manual ack
- Idempotent: `processed_events` tablosuna event_id yaz, duplicate ignore
- Max retry sonrasi DLQ
- Hata durumunda nack + retry counter artir

### Event DTO bilinen tuzaklari
- Catalog → `course.semester` event'i `semester_course_id` ve `instructor_fullname` gonderir (course_id/instructor_name DEGIL)
- Yeni event eklerken **tum** consumer service'lerin DTO'larini guncelle

---

## 14. Auth & RBAC

```go
// Route gruplari
authMiddleware := sharedMiddleware.JWTAuth(jwtSecret)
adminOnly := sharedMiddleware.RequireRole("admin")
teacherOrAdmin := sharedMiddleware.RequireRole("teacher", "admin")

router.Group("/api/xxx").
    Use(authMiddleware).
    POST("/", handler.Create)

router.Group("/api/admin/xxx").
    Use(authMiddleware, adminOnly).
    DELETE("/:id", handler.Delete)
```

**Sabitler:**
- Roller: `admin`, `teacher`, `student`
- Access token: 15 dk
- Refresh token: 24 saat
- JWT claims: `UserID`, `Role`, `Department`, `TokenVersion`
- Header'lardan kullanici cikar: `c.GetString("user_id")`, `c.GetString("user_role")`

---

## 15. Rate Limit

```go
// auth gibi kritik endpoint'ler icin FailClosed: true
"login":   {Limit: 5, Window: 1*time.Minute, FailClosed: true},

// genel endpoint'ler FailClosed: false (Redis down -> izin ver)
"default": {Limit: 100, Window: 1*time.Minute, FailClosed: false},
```

**Kural:** Brute-force riski olan endpoint'lerde `FailClosed: true`. Diger her yerde `FailClosed: false`.

---

## 16. Make Komutlari (her serviste ayni)

| Komut | Ne yapar |
|---|---|
| `make sqlc` | SQL'den Go kodu uretir |
| `make migrate-create name=xxx` | Yeni migration dosyasi olusturur |
| `make migrate-up` | Bekleyen migration'lari calistirir (kullanici calistirir) |
| `make migrate-down` | Son migration'i geri alir |
| `make migrate-status` | Migration durumu |
| `make build` | Servisi derler |

---

## 17. Test Yazma

### Isimlendirme
```go
func TestUserService_Login_ReturnsTokenForValidCredentials(t *testing.T) { ... }
func TestUserService_Login_ReturnsErrInvalidCredentialsForWrongPassword(t *testing.T) { ... }
func TestUserService_Login_LocksAccountAfter5FailedAttempts(t *testing.T) { ... }
```

Format: `Test{Type}_{Method}_{Scenario_ExpectedResult}`

### Test piramidi (bu projede)
| Katman | Bagimlilik | Hedef |
|---|---|---|
| Repository | Real Postgres (testpool) | Sorgu + sqlc + sentinel cevirimi dogru mu |
| Service | Repository **mock** (mockery) | Business logic + error mapping |
| Handler | Service mock + `httptest` + Gin | HTTP status + response shape |

### Table-driven sablon
```go
func TestUserService_Login_Variants(t *testing.T) {
    cases := []struct {
        name      string
        email     string
        password  string
        repoSetup func(*mocks.UserRepository)
        wantErr   error
    }{
        {
            name:     "valid credentials returns token",
            email:    "a@b.c",
            password: "Pass1234!",
            repoSetup: func(m *mocks.UserRepository) {
                m.On("GetByEmail", mock.Anything, "a@b.c").Return(validUser, nil)
            },
            wantErr: nil,
        },
        {
            name:     "wrong password returns ErrInvalidCredentials",
            email:    "a@b.c",
            password: "wrong",
            repoSetup: func(m *mocks.UserRepository) {
                m.On("GetByEmail", mock.Anything, "a@b.c").Return(validUser, nil)
            },
            wantErr: serviceErrors.ErrInvalidCredentials,
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            repo := mocks.NewUserRepository(t)
            tc.repoSetup(repo)
            svc := service.NewUserService(repo)

            _, err := svc.Login(context.Background(), tc.email, tc.password)
            if tc.wantErr != nil {
                require.ErrorIs(t, err, tc.wantErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Mockery
```bash
# Kurulum (her servis kendi)
go install github.com/vektra/mockery/v2@latest

# Repository interface'lerinden mock uret
mockery --name=UserRepository --dir=internal/repository --output=internal/mocks
```
Servis-bazli `.mockery.yaml` ile config: `cd backend/services/auth-service && mockery`.

### Repository test (real Postgres)
- `shared/database` paketinden testpool yardimcisi: `pool := database.NewTestPool(t)` — her test icin schema sifirla
- CI yoksa lokal: `sudo docker compose up -d postgres-{servis}` (kullaniciya komut ver)

### Kurallar
- Table-driven test tercih (3+ vaka varsa)
- `testify/assert` ve `testify/require` kullan; error karsilastirma `require.ErrorIs(t, err, target)` (ciplak `==` YAPMA)
- Handler test = `httptest.NewRecorder` + `gin.SetMode(gin.TestMode)`

**YAPMA:** Test'i `t.Skip()` ile atla, `time.Sleep()` ile flaky yap, gercek RabbitMQ baglantisi kur.

---

## 18. Graceful Shutdown

`cmd/main.go` her servis icin ayni iskelet:

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}
go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Fatal("http server", zap.Error(err))
    }
}()

<-ctx.Done()
logger.Info("shutdown signal received")

shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
defer shutdownCancel()

// Sira: HTTP -> consumer -> outbox worker -> rabbitmq -> redis -> pgxpool
_ = srv.Shutdown(shutdownCtx)
consumer.Stop()
outboxWorker.Stop()
rabbit.Close()
redis.Close()
pool.Close()
```

**Kurallar:**
- Shutdown sirasi: **request alimini durdur**, sonra **isleyen workers**, en son **kaynaklar**
- 15 saniye timeout (longest-running request + safety)
- `errors.Is(err, http.ErrServerClosed)` ile normal kapanis ayrilir, log'a `Fatal` gitmesin
- RabbitMQ consumer manual ack: shutdown'da in-flight mesaji nack ettirme — broker yeniden teslim eder

---

## 19. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| sqlc generate fail | SQL syntax kontrol et, sutun isimleri ile annotation eslesiyor mu bak | `internal/db/*.go` manuel duzenle |
| Migration fail | Hatayi oku, kullaniciya `migrate-down` oner | Migration tablosunu DELETE et, manuel DROP |
| Test fail | Hatayi oku, fix et veya rapor et | `t.Skip()`, test'i sil |
| Pre-commit hook fail | Sebep kontrol et, fix et, yeni commit at | `--no-verify` ile bypass |
| Event consumer DLQ'ya dustu | `processed_events` tablosunu kontrol et, payload'u logla | DLQ'yu temizle ve unutmayi sec |
| pgx connection refused | Postgres up mi kontrol et | Pool'u nil yap, hata yutma |
| RabbitMQ baglanti yok | Kullaniciya `sudo docker compose up -d rabbitmq` oner | RabbitMQ olmadan publish'i atla |

---

## 20. Shared Paketler (referans)

| Paket | Ne icin |
|---|---|
| `database/` | `pgxpool.Pool` olusturma |
| `middleware/` | `JWTAuth`, `RequireRole`, `RateLimit`, `Recovery`, `Logger`, `CORS` |
| `rabbitmq/` | `Connection`, `Publisher`, `Consumer`, DLQ |
| `redis/` | Client, blacklist, session |
| `utils/` | pgtype helpers, JWT, password (Argon2id), validation |
| `logger/` | Zap wrapper, `WithContext`, `WithContextAndFields`, request ID |
| `errors/` | `AppError` (HTTP status + code + message), `Wrap`, `Is` |
| `clock/` | `Now()` — test'te `SimulatedClock` |
| `events/` | Event type + queue isim sabitleri |
| `audit/` | Security audit log (login, password change vs.) |
| `handler/` | `LivenessHandler`, `ReadinessHandler` |

**Kural:** Yeni utility yazmadan once `shared/`'da var mi bak.
