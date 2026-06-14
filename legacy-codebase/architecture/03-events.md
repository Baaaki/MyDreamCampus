> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 5. Event Bus: RabbitMQ + Outbox

### 5.1 Akış

```
[Modul Service]
    │
    ├── tx.Begin()
    ├── tx.Insert(domain_table)        # ornek: auth.users
    ├── tx.Insert(public.outbox_events) # ayni transaction
    └── tx.Commit()                    # atomik garanti
            │
            ▼
[Outbox Relay Goroutine] (monolith icinde, baslangicta calisir)
    │
    ├── public.outbox_events tablosundan UNPUBLISHED kayitlari oku
    ├── RabbitMQ exchange'e publish et
    ├── basariliysa: published_at = NOW()
    └── basarisizsa: retry sayisi++ (exponential backoff)
            │
            ▼
[RabbitMQ Exchange: domain_events]
    │
    ▼
[Notification Service Consumer]
    └── handler calisir (welcome email, push notification, vb.)
```

### 5.2 Outbox tablo şeması

> **Mevcut staff_service'in kalıbına uyumlu** (`backend/services/staff_service/sql/migrations/00002_create_outbox_events_table.sql`). Her modül kendi schema'sında `outbox_events` tutar.

```sql
-- backend/monolith/internal/modules/staff/sql/migrations/00002_create_outbox_events_table.sql
-- (Aynisi her modul icin tekrarlanir, schema adi degisir)

CREATE TABLE IF NOT EXISTS staff.outbox_events (
    id              SERIAL PRIMARY KEY,
    event_type      VARCHAR(100) NOT NULL,         -- "staff.created", "staff.updated"
    routing_key     VARCHAR(100) NOT NULL,         -- "staff.created"
    payload         JSONB NOT NULL,                 -- {"id": "...", "email": "...", ...}
    processed       BOOLEAN NOT NULL DEFAULT false,
    retry_count     SMALLINT NOT NULL DEFAULT 0,
    max_retries     SMALLINT NOT NULL DEFAULT 5,
    error_message   TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMP
);

CREATE INDEX idx_staff_outbox_processed ON staff.outbox_events(processed);
CREATE INDEX idx_staff_outbox_created_at ON staff.outbox_events(created_at);
```

**Önemli detaylar (mevcut koddan):**
- `id SERIAL` — UUID değil. Outbox PK doğal sayı, payload içindeki `id` (varsa) UUID olabilir.
- `processed BOOLEAN` + `processed_at TIMESTAMP` — `published_at` değil. "processed" → "RabbitMQ'ya teslim edildi".
- `routing_key` ayrı kolon — `event_type` ile aynı olabilir ama event_type domain dili, routing_key transport dili.
- `retry_count` + `max_retries` — failed event'ler retry edilir, max'a ulaşınca durur (DLQ değil, manuel inceleme).

**Outbox bazı modüllerde değil, herkeste var.** Schema-per-module dolayısıyla her modül kendi outbox tablosuna yazar. Outbox worker tüm modüllerin outbox tablolarını paralel poll eder (Bölüm 5.5.2).

### 5.3 Event payload self-contained olur

Notification servisi event payload'undan **her şeyi** alabilmeli, monolith DB'sine geri dönüş yok.

**Mevcut kalıp** (`backend/services/staff_service/internal/service/event_payloads.go`): payload `map[string]any` döndüren builder fonksiyonlar. Service katmanında builder çağrılır, repository payload'ı outbox'a yazar.

```go
// monolith/internal/modules/auth/service/event_payloads.go
// (Mevcut staff_service'in event_payloads.go kalibinin aynisi)

// buildUserRegisteredPayload: auth.user.registered icin outbox payload'i.
// Notification welcome email icin email + first_name kullanir.
func buildUserRegisteredPayload(req dto.RegisterRequest) map[string]any {
    return map[string]any{
        "id":         nil,                 // CreateUserWithEvent tx icinde overwrite eder
        "email":      req.Email,
        "first_name": req.FirstName,
        "last_name":  req.LastName,
        "role":       req.Role,
    }
}
```

**Neden typed struct DEĞİL, `map[string]any`:** Mevcut tüm servisler (`staff_service`, `auth_service`, `student_service` dahil hepsi) bu kalıbı kullanıyor. Outbox worker, RabbitMQ publisher, consumer hepsi `map[string]any` üzerinden çalışıyor (`shared/rabbitmq/publisher.go` payload `any` alıyor, içinde `json.Marshal` yapıyor). Mimari tutarlılık için bu kalıp korunur.

**Wire contract uyarısı:** Event payload'larındaki alan adları **wire contract**'tir. Sessizce yeniden adlandırma notification'ı bozar. `event_payloads.go` dosyalarında her builder'ın üstüne wire contract yorumu yazılır (mevcut staff_service'teki gibi).

### 5.4 Event kontratları

`backend/shared/events/events.go` paketinde sadece **string sabitleri** (event_type ve routing_key). Hem monolith hem notification import eder. Bu paket **wire contract** — sabit silmek veya yeniden adlandırmak breaking change'dir; yeni sabit eklemek OK.

**Mevcut kalıp** (`backend/shared/events/events.go`):

```go
// backend/shared/events/events.go
package events

// Auth Module Events
const (
    EventUserRegistered          = "user.registered"
    EventUserPasswordResetReq    = "user.password_reset_requested"
)

// Staff Module Events (zaten mevcut)
const (
    EventStaffCreated     = "staff.created"
    EventStaffUpdated     = "staff.updated"
    EventStaffDeactivated = "staff.deactivated"
)

// Payment Module Events
const (
    EventPaymentSucceeded = "payment.succeeded"
    EventPaymentFailed    = "payment.failed"
)

// Meal Module Events (sadece payment ile etkilesim)
const (
    EventMealCreditPurchaseRequested = "meal.credit_purchase_requested"
)

// Notification Queue Names (notification servisinde kullanilir)
const (
    QueueNotificationEvents = "notification.events"
)

// Routing Keys (event_type ile ayni format, ama bagimsiz sabit)
const (
    RoutingKeyUserRegistered       = "user.registered"
    RoutingKeyUserPasswordResetReq = "user.password_reset_requested"
    RoutingKeyStaffCreated         = "staff.created"
    RoutingKeyStaffUpdated         = "staff.updated"
    RoutingKeyStaffDeactivated     = "staff.deactivated"
    RoutingKeyPaymentSucceeded     = "payment.succeeded"
    RoutingKeyPaymentFailed        = "payment.failed"
    RoutingKeyMealCreditPurchase   = "meal.credit_purchase_requested"
)
```

**Payload struct YOK** — payload her zaman `map[string]any` (Bölüm 5.3). Bu sayede yeni alan eklemek schema migration'sız mümkün, eski consumer'lar yeni alanları görmezden gelir.

**Tam event listesi Bölüm 5.9'da** (Notification servisinin consume edeceği event kataloğu).

### 5.5 Outbox implementasyonu

> **Mevcut kalıp aynen korunur.** `backend/services/staff_service/internal/repository/staff_repository.go` ve `internal/worker/outbox_worker.go` referansdır. Yeni helper veya pattern uydurulmaz.

#### 5.5.1 Repository-with-Event pattern

Her modülün repository'sinde **`CreateXxxWithEvent` / `UpdateXxxWithEvent` / `SoftDeleteXxxWithEvent`** method'ları olur. Bu method'lar transaction'ı kendi içinde açar, domain insert + outbox insert'ı atomik yapar, commit eder.

**Mevcut staff_service kalıbı** (uyarlanmış — schema referansları eklenmiş):

```go
// monolith/internal/modules/staff/repository/staff_repository.go
package repository

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

    "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/db"
    serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/errors"
    "github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
    "github.com/baaaki/mydreamcampus/shared/events"
    sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

type StaffRepository struct {
    queries *db.Queries
    pool    *pgxpool.Pool
}

func NewStaffRepository(pool *pgxpool.Pool) *StaffRepository {
    return &StaffRepository{queries: db.New(pool), pool: pool}
}

// CreateStaffWithEvent: staff insert + outbox insert atomik
func (r *StaffRepository) CreateStaffWithEvent(
    ctx context.Context,
    params db.CreateStaffParams,
    eventPayload map[string]any,
) (db.Staff, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return db.Staff{}, fmt.Errorf("%w: failed to begin transaction: %v",
            sharedErrors.ErrTransactionFailed, err)
    }
    defer tx.Rollback(ctx)

    qtx := r.queries.WithTx(tx)

    // 1) Staff insert
    staff, err := qtx.CreateStaff(ctx, params)
    if err != nil {
        var pgxErr *pgconn.PgError
        if errors.As(err, &pgxErr) && pgxErr.Code == "23505" {
            return db.Staff{}, fmt.Errorf("%w: email already exists",
                serviceErrors.ErrStaffExistsRepo)
        }
        return db.Staff{}, fmt.Errorf("%w: failed to create staff: %v",
            sharedErrors.ErrQueryFailed, err)
    }

    // 2) Payload icine generated id'yi yaz
    eventPayload["id"] = utils.PgtypeToUUIDString(staff.ID)

    payload, err := json.Marshal(eventPayload)
    if err != nil {
        return db.Staff{}, fmt.Errorf("%w: failed to marshal event payload: %v",
            sharedErrors.ErrQueryFailed, err)
    }

    // 3) Outbox insert (ayni tx)
    _, err = qtx.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
        EventType:  events.EventStaffCreated,
        RoutingKey: events.RoutingKeyStaffCreated,
        Payload:    payload,
    })
    if err != nil {
        return db.Staff{}, fmt.Errorf("%w: failed to create outbox event: %v",
            sharedErrors.ErrQueryFailed, err)
    }

    if err := tx.Commit(ctx); err != nil {
        return db.Staff{}, fmt.Errorf("%w: failed to commit transaction: %v",
            sharedErrors.ErrTransactionFailed, err)
    }
    return staff, nil
}
```

**Service katmanı kullanımı** (mevcut `staff_service.go` kalıbı):

```go
// monolith/internal/modules/staff/service/staff_service.go
func (s *StaffService) CreateStaff(ctx context.Context, req dto.CreateStaffRequest) (dto.StaffResponse, error) {
    // ... existence check, validations ...

    params := db.CreateStaffParams{ /* ... */ }
    eventPayload := buildStaffCreatedPayload(req)

    staff, err := s.staffRepo.CreateStaffWithEvent(ctx, params, eventPayload)
    if err != nil {
        // mevcut error wrapping pattern
    }

    return s.toStaffResponse(staff), nil
}
```

**Kritik kurallar:**
- Domain insert + outbox insert **aynı `qtx`** üzerinde (yani aynı transaction).
- `defer tx.Rollback(ctx)` — `Commit()` başarılıysa Rollback no-op olur.
- Hata durumunda: `sharedErrors.Wrap` ile sarmalanır, service katmanı `sharedErrors.Is` ile kontrol eder.
- `eventPayload` map'i, generated UUID'yi `qtx.CreateStaff` döndüğünde içine yazılır (`payload["id"] = ...`).

#### 5.5.2 Outbox Worker (per-module veya merged)

Mevcut staff_service'te her servisin **kendi outbox worker'ı** var (5sn polling, batch 10). Monolith'te iki seçenek:

**Seçenek A — Modül başına ayrı worker (mevcut kalıp tıpkısı):**
- 9 outbox worker, hepsi 5sn polling, batch 10.
- Her biri kendi modülünün outbox tablosunu okur, kendi exchange'ine publish eder.
- Mevcut kodun direkt taşınması.

**Seçenek B — Tek "Multi-Module Outbox Worker":**
- Tek goroutine, tüm modüllerin outbox tablolarını sırayla okur.
- Daha az goroutine ama daha karmaşık koordinasyon.

**Karar (Bölüm 13 — [Açık Sorular](08-rules.md)'a referans):** Faz 0'da **Seçenek A** ile başla (mevcut kod aynen taşınır, risk düşük). Ölçüm sonrası gerekirse B'ye geçilir.

**Mevcut worker kalıbı** (`backend/services/staff_service/internal/worker/outbox_worker.go`):

```go
// monolith/internal/modules/staff/worker/outbox_worker.go
// (Mevcut staff_service/internal/worker/outbox_worker.go AYNEN tasinir,
// sadece import path'leri guncellenir.)

type OutboxWorker struct {
    outboxRepo *repository.OutboxRepository
    publisher  *rabbitmq.Publisher
    interval   time.Duration  // 5 * time.Second
    batchSize  int32          // 10
}

func (w *OutboxWorker) Start(ctx context.Context) {
    log := logger.WithContextAndFields(ctx, zap.String("worker", "OutboxWorker"))
    log.Info("starting outbox worker",
        zap.Duration("interval", w.interval),
        zap.Int32("batch_size", w.batchSize),
    )

    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    w.processEvents(ctx)  // immediate first run

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processEvents(ctx)
        }
    }
}

func (w *OutboxWorker) processEvents(ctx context.Context) {
    pendingEvents, err := w.outboxRepo.GetPendingEvents(ctx, w.batchSize)
    if err != nil { /* log */ return }

    for _, event := range pendingEvents {
        var payload map[string]any
        if err := json.Unmarshal(event.Payload, &payload); err != nil {
            w.outboxRepo.MarkEventFailed(ctx, /*id*/, err.Error())
            continue
        }

        // Event metadata wrap (mevcut auth-service consumer'in bekledigi format)
        eventMessage := map[string]any{
            "event_id":   utils.PgtypeToUUIDString(event.ID),
            "event_type": event.EventType,
            "timestamp":  event.CreatedAt.Time,
            "data":       payload,
        }

        err := w.publisher.Publish(ctx, "staff.events", event.RoutingKey, eventMessage)
        if err != nil {
            w.outboxRepo.MarkEventFailed(ctx, /*id*/, err.Error())
            continue
        }
        w.outboxRepo.MarkEventProcessed(ctx, /*id*/)
    }

    // Failed events retry (max_retries kontrolu ile)
    w.processFailedEvents(ctx)
}
```

**Startup wiring** (`monolith/cmd/main.go`):
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Her modul icin bir worker
go staffOutboxWorker.Start(ctx)
go authOutboxWorker.Start(ctx)
go studentOutboxWorker.Start(ctx)
// ... diger 6 modul
```

**Polling 5 saniye** (mevcut staff_service değeri). 100ms önerilmedi — staff_service'in kanıtlanmış değeri korunur. Latency hassas event yok (notification email zaten async).

### 5.6 RabbitMQ Topology

> **Mevcut kalıp aynen korunur — per-modül exchange.** `backend/services/staff_service/cmd/main.go` referansdır.

#### 5.6.1 Per-modül exchange (mevcut kalıp)

Her modül kendi exchange'ine publish eder. Tek shared `domain.events` değil, modül başına bir tane:

```go
// monolith/cmd/main.go startup'ta her modul icin:
exchanges := []string{
    "auth.events",
    "staff.events",
    "student.events",
    "course_catalog.events",
    "enrollment.events",
    "attendance.events",
    "grades.events",
    "meal.events",
    "payment.events",
}
for _, ex := range exchanges {
    if err := channel.ExchangeDeclare(
        ex,        // name
        "topic",   // type
        true,      // durable
        false,     // auto-deleted
        false,     // internal
        false,     // no-wait
        nil,
    ); err != nil {
        logger.Fatal("failed to declare exchange", zap.String("exchange", ex), zap.Error(err))
    }
}
```

**Neden per-modül exchange (mevcut karar):**
- Modül izolasyonu → her modülün event topology'si bağımsız.
- İleride bir modül servise ayrılırsa exchange zaten ayrı, taşıma kolay.
- Routing key namespace çakışması imkansız (her exchange ayrı namespace).
- Mevcut microservices kodu bu kalıbı kullanıyor, tutarlılık.

#### 5.6.2 Notification consumer queue + binding

Notification servisi tek queue (`notification.events`) açar, **birden fazla exchange'den** routing key'lere bind eder:

```go
// services/notification/internal/consumer/setup.go
// Mevcut shared/rabbitmq.Publisher.DeclareAndBindQueue helper'ini kullanir.

func SetupTopology(publisher *rabbitmq.Publisher) error {
    queueName := events.QueueNotificationEvents  // "notification.events"

    // Notification'in dinledigi event'ler — her satir (exchange, routing_key) cifti.
    bindings := []struct {
        exchange   string
        routingKey string
    }{
        {"auth.events", events.RoutingKeyUserRegistered},
        {"auth.events", events.RoutingKeyUserPasswordResetReq},
        {"payment.events", events.RoutingKeyPaymentSucceeded},
        {"payment.events", events.RoutingKeyPaymentFailed},
        // ... Bolum 5.9 (Event katalogu) eksiksiz listeyi tutar
    }

    for _, b := range bindings {
        if err := publisher.DeclareAndBindQueue(queueName, b.exchange, b.routingKey); err != nil {
            return fmt.Errorf("bind %s -> %s: %w", b.exchange, b.routingKey, err)
        }
    }
    return nil
}
```

`DeclareAndBindQueue` mevcut `shared/rabbitmq/publisher.go` helper'ı (durable queue declare + bind). Yeniden yazılmaz.

#### 5.6.3 Downstream queue pre-declaration (mevcut staff_service kalıbı)

**Önemli detay** (mevcut `staff_service/cmd/main.go` line 273-291): Publisher (monolith) startup'ta consumer queue'larını **publish öncesi** declare ediyor. Sebep: notification offline iken event publish edilse bile mesaj queue'da birikir, online olunca işlenir.

```go
// monolith/cmd/main.go
publisher := rabbitmq.NewPublisher(rabbitConn)

// Notification queue'sunu HER MODUL EXCHANGE'INE bind et (notification offline iken mesaj kaybolmasin)
downstreamBindings := []struct {
    queue, exchange, routingKey string
}{
    {"notification.events", "auth.events", "user.registered"},
    {"notification.events", "auth.events", "user.password_reset_requested"},
    {"notification.events", "payment.events", "payment.succeeded"},
    {"notification.events", "payment.events", "payment.failed"},
    // ... Bolum 5.9 listesi
}
for _, b := range downstreamBindings {
    if err := publisher.DeclareAndBindQueue(b.queue, b.exchange, b.routingKey); err != nil {
        logger.Fatal("failed to declare downstream queue", zap.Error(err))
    }
}
```

#### 5.6.4 DLQ (Dead Letter Queue)

> Mevcut kod `shared/rabbitmq/dlq.go` dosyasında DLQ helper barındırıyor. Notification servisi bunu kullanır.

DLQ exchange + queue: `notification.dlq.events` exchange + `notification.dlq` queue. Notification kuyruğunda `x-dead-letter-exchange` argümanı ile bağlanır. Failed event'ler manuel inceleme + replay için tutulur, otomatik temizleme yok.

### 5.7 Hata yönetimi: Retry + DLQ

> Consumer side. `shared/rabbitmq/consumer.go` mevcut helper'ları kullanılır.

Notification consumer handler. Outbox worker payload'ı `{event_id, event_type, timestamp, data}` zarfıyla yolluyor — handler bu zarfı parse eder:

```go
// services/notification/internal/consumer/handlers.go
func (c *Consumer) handle(ctx context.Context, msg amqp.Delivery) {
    // 1) Outbox worker'in zarfini parse et
    var envelope struct {
        EventID   string         `json:"event_id"`
        EventType string         `json:"event_type"`
        Timestamp time.Time      `json:"timestamp"`
        Data      map[string]any `json:"data"`
    }
    if err := json.Unmarshal(msg.Body, &envelope); err != nil {
        c.log.Error("malformed event envelope", zap.Error(err))
        msg.Ack(false)  // parse edilemeyen mesaji kuyruktan dusur, DLQ'ya bile gonderme
        return
    }

    // 2) Idempotency check
    eventID, _ := uuid.Parse(envelope.EventID)
    if processed, _ := c.repo.IsProcessed(ctx, eventID); processed {
        msg.Ack(false)  // tekrar geldi, skip et
        return
    }

    // 3) Routing key'e gore handler dispatch
    var err error
    switch envelope.EventType {
    case events.EventUserRegistered:
        err = c.svc.SendWelcomeEmail(ctx, envelope.Data)
    case events.EventPaymentSucceeded:
        err = c.svc.SendPaymentReceipt(ctx, envelope.Data)
    case events.EventPaymentFailed:
        err = c.svc.SendPaymentFailedEmail(ctx, envelope.Data)
    // ... Bolum 5.9'daki listenin tamami
    default:
        c.log.Warn("unknown event type",
            zap.String("type", envelope.EventType),
            zap.String("event_id", envelope.EventID),
        )
        msg.Ack(false)
        return
    }

    // 4) Sonuc handling
    if err != nil {
        retryCount := getRetryCount(msg.Headers)
        if retryCount < 3 {
            msg.Nack(false, true)   // requeue=true → tekrar denenir
        } else {
            msg.Nack(false, false)  // requeue=false → DLQ'ya gider
        }
        return
    }

    // 5) Basarili: idempotency tablosuna yaz, ACK
    c.repo.MarkProcessed(ctx, eventID, envelope.EventType)
    msg.Ack(false)
}
```

**Manuel ACK kritik:** `auto-ack=false` verilir consumer setup'ta. Crash olursa mesaj kaybolmaz, RabbitMQ tekrar dağıtır (at-least-once).

### 5.8 Idempotency

RabbitMQ at-least-once garanti ediyor — aynı mesaj N kere gelebilir. Çift email gönderimi engellemek için notification kendi DB'sinde işlenmiş event'leri tutar:

```sql
-- notification servisinin kendi DB'sinde
CREATE TABLE notification.processed_events (
    event_id     UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`event_id` zinciri: monolith outbox PK → RabbitMQ `MessageId` header → notification consumer okur → `processed_events`'e PK olarak yazar. Çakışırsa (`ON CONFLICT DO NOTHING`) skip et + ACK.

### 5.9 Event kataloğu (kim publish eder, notification ne yapar)

> **Bu liste eksiksizdir.** Yeni event eklemek için ÖNCE kullanıcıya sor (Bölüm 0.1 AI Kuralları).

> Mevcut kalıba göre tüm event_type sabitleri `backend/shared/events/events.go`'da, her modül için **kendi event'leri zaten orada tanımlı**. Plan bu sabitlerin üstüne yeni event'ler önerir; mevcut sabitler **silinmez** (hâlihazırda diğer servislerce kullanılıyor olabilir).

| event_type | routing_key | Publisher modül | Exchange | Payload alanları (`map[string]any`) | Notification aksiyonu |
|---|---|---|---|---|---|
| `user.registered` | `user.registered` | auth | `auth.events` | `id`, `email`, `first_name`, `last_name`, `role` | Welcome email |
| `user.password_reset_requested` | `user.password_reset_requested` | auth | `auth.events` | `user_id`, `email`, `reset_token`, `expires_at` | Reset link email |
| `staff.created` | `staff.created` | staff | `staff.events` | `id`, `email`, `first_name`, `last_name`, `role`, `department` | (notification consume etmiyor — auth tüketiyor) |
| `staff.updated` | `staff.updated` | staff | `staff.events` | `staff_id`, `department`, `phone`, `office_location` | (notification consume etmiyor) |
| `staff.deactivated` | `staff.deactivated` | staff | `staff.events` | `staff_id` | (notification consume etmiyor) |
| `payment.succeeded` | `payment.succeeded` | payment | `payment.events` | `transaction_id`, `user_id`, `email`, `amount`, `currency`, `description`, `paid_at` | Makbuz email |
| `payment.failed` | `payment.failed` | payment | `payment.events` | `user_id`, `email`, `amount`, `currency`, `reason` | "Ödeme alınamadı" email |
| `meal.credit_purchase_requested` | `meal.credit_purchase_requested` | meal | `meal.events` | `user_id`, `credit_amount`, `price`, `currency` | (sadece payment consume eder, notification DEĞİL — Bölüm 9'daki meal-payment izolasyonu) |

**Naming garantileri:**
- `event_type` = `routing_key` formatı: `<modul>.<aksiyon>`. Tek istisna yok.
- Exchange formatı: `<modul>.events`. Tek istisna yok.
- Payload alan adları **snake_case** (`first_name`, `paid_at`).
- Tüm timestamp'ler `time.Time` (JSON: ISO8601 string).
- Tüm UUID'ler string formatında JSON'a yazılır (`utils.PgtypeToUUIDString`).

**Yeni event eklemek için akış:**
1. **Kullanıcıya sor.** Plan otoritedir.
2. `backend/shared/events/events.go`'a `EventXxxYyy` ve `RoutingKeyXxxYyy` sabitlerini ekle.
3. Publisher modülün `internal/service/event_payloads.go`'sına `buildXxxYyyPayload` builder fonksiyonu ekle.
4. Publisher repository'sinin `CreateXxxWithEvent` benzeri method'unda outbox insert'ı tetikle.
5. Notification servisinin consumer binding listesine ekle (Bölüm 5.6.2).
6. Notification servisinin handler'ına case ekle (Bölüm 5.7'deki switch).
7. Bu tabloyu güncelle.

### 5.10 End-to-end timeline (somut örnek)

Kullanıcı `Ali (ali@example.com)` kayıt oluyor:

```
T+0ms      Frontend → POST /api/auth/register {email, password, fullName}
T+5ms      Monolith Gin → auth.RegisterUser handler
T+10ms     ├── tx.Begin (auth modulu repository'sinde, pool.Begin(ctx))
T+15ms     ├── INSERT auth.users (id=550e..., email=ali@..., first_name=Ali)
T+20ms     ├── INSERT auth.outbox_events (
                 id=42 (SERIAL), event_type="user.registered",
                 routing_key="user.registered",
                 payload={"id":"550e...","email":"ali@...","first_name":"Ali",...},
                 processed=false)
T+25ms     ├── tx.Commit              ◄── ATOMIK
T+30ms     └── HTTP 201 Created → Frontend  (kullanici "hosgeldin" sayfasini gordu)

T+5000ms   Auth Outbox Worker tick (5sn polling)
T+5005ms   ├── SELECT * FROM auth.outbox_events WHERE processed=false LIMIT 10 → 1 kayit
T+5010ms   ├── Envelope wrap: {event_id:"42", event_type:"user.registered",
                                timestamp:..., data:<payload>}
T+5015ms   ├── RabbitMQ basic.publish(
                 exchange="auth.events",
                 routing_key="user.registered",
                 body=<envelope JSON>, persistent=true)
T+5020ms   ├── RabbitMQ → topic match → notification.events queue
T+5025ms   └── UPDATE auth.outbox_events SET processed=true, processed_at=NOW() WHERE id=42

T+5030ms   Notification Consumer aldi (basic.deliver)
T+5035ms   ├── Envelope unmarshal → event_id="42", event_type="user.registered", data={...}
T+5040ms   ├── IsProcessed("42")? → NO
T+5045ms   ├── handler: sendWelcomeEmail(data)
T+5700ms   │   └── SMTP gonderim tamamlandi (~655ms surdu)
T+5705ms   ├── INSERT notification.delivery_log (event_id="42", channel="email", status="sent")
T+5710ms   ├── INSERT notification.processed_events (event_id="42") ON CONFLICT DO NOTHING
T+5715ms   └── basic.ack(deliveryTag) → RabbitMQ kuyruktan siler

Toplam: kullanici T+30ms'de yanit aldi (anlik), email ~T+5.7sn'de teslim (5sn polling tick + ~700ms SMTP).
```

**Polling dezavantajı:** En kötü durumda 5sn gecikme (worker tick'leri arası). Welcome email için kabul edilebilir, ödeme bildirimi için de yeterli (kullanıcı zaten ödeme onay sayfasında bekliyor değil). İhtiyaç varsa `LISTEN/NOTIFY` ile push-based tetikleme — Bölüm 13 (Açık Sorular) maddesi.

---

