> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 6. Notification Service — Setup ve Implementasyon

### 6.1 Klasör yapısı

```
backend/services/notification/
├── cmd/
│   └── main.go                 # Entry point
├── config/
│   └── config.go               # Env config (DB, RabbitMQ, SMTP, FCM)
├── internal/
│   ├── consumer/
│   │   ├── consumer.go         # RabbitMQ consume loop
│   │   ├── setup.go            # Exchange/queue/binding declare
│   │   └── handlers.go         # event_type → handler dispatch
│   ├── delivery/
│   │   ├── email/              # SMTP adapter
│   │   │   └── smtp.go
│   │   ├── push/               # FCM adapter
│   │   │   └── fcm.go
│   │   └── sms/                # Twilio/benzeri (ileride)
│   ├── service/
│   │   └── notification_service.go  # Business logic (template, retry karari)
│   ├── repository/
│   │   ├── delivery_log.go
│   │   └── processed_events.go
│   ├── db/                     # sqlc generated
│   ├── sql/
│   │   ├── migrations/
│   │   │   ├── 00001_create_delivery_log.sql
│   │   │   └── 00002_create_processed_events.sql
│   │   └── queries/
│   └── templates/              # Email/push template'leri (Go html/template)
│       ├── welcome.html
│       ├── payment_receipt.html
│       └── ...
├── Makefile
├── Dockerfile
├── go.mod
└── go.sum
```

### 6.2 Notification DB şeması

Kendi PostgreSQL instance'ı (monolith DB'sinden ayrı):

```sql
-- 00001_create_delivery_log.sql
CREATE TABLE delivery_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL,
    event_type   TEXT NOT NULL,
    channel      TEXT NOT NULL,           -- email, push, sms
    recipient    TEXT NOT NULL,           -- email adresi, telefon, vs.
    template     TEXT NOT NULL,
    status       TEXT NOT NULL,           -- pending, sent, failed
    error        TEXT,
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_delivery_log_event ON delivery_log (event_id);
CREATE INDEX idx_delivery_log_status ON delivery_log (status, created_at);

-- 00002_create_processed_events.sql
CREATE TABLE processed_events (
    event_id     UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 6.3 Bağımlılıklar (go.mod özeti)

```
github.com/rabbitmq/amqp091-go    // AMQP
github.com/jackc/pgx/v5           // Postgres
github.com/jordan-wright/email    // SMTP (alternatif: net/smtp + mime)
firebase.google.com/go/v4         // FCM push (ileride)
go.uber.org/zap                   // Logging
github.com/spf13/viper            // Config
github.com/google/uuid
```

### 6.4 main.go iskeleti

```go
// services/notification/cmd/main.go
func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)

    // DB
    pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
    must(err)
    defer pool.Close()

    // RabbitMQ
    conn, err := amqp.Dial(cfg.RabbitMQURL)
    must(err)
    defer conn.Close()
    ch, err := conn.Channel()
    must(err)
    defer ch.Close()

    // Topology setup (idempotent)
    if err := consumer.SetupTopology(ch); err != nil {
        log.Fatal("topology setup failed", zap.Error(err))
    }

    // Adapters
    smtp := email.NewSMTPSender(cfg.SMTP)

    // Repository + Service
    repo := repository.New(pool)
    svc := service.New(repo, smtp, log)

    // Consumer
    cons := consumer.New(ch, svc, log)
    go cons.Start(ctx)

    // Health endpoint (Docker healthcheck icin)
    go startHealthServer(":9090")

    // Graceful shutdown
    waitForSignal()
}
```

### 6.5 Email template örneği

```html
<!-- internal/templates/welcome.html -->
<!DOCTYPE html>
<html>
<body>
  <h1>Hoş geldin {{.first_name}}!</h1>
  <p>MyDreamCampus'a kayıt oldun. Hesabını aktive etmek için aşağıdaki bağlantıya tıkla:</p>
  <a href="{{.activation_url}}">Hesabımı aktive et</a>
</body>
</html>
```

Service layer template render eder. Payload `map[string]any` formatında geldiği için (Bölüm 5.3), template aynı snake_case alan adlarını kullanır:

```go
// services/notification/internal/service/notification_service.go
func (s *Service) SendWelcomeEmail(ctx context.Context, data map[string]any) error {
    email, ok := data["email"].(string)
    if !ok || email == "" {
        return fmt.Errorf("missing or invalid email in payload")
    }
    firstName, _ := data["first_name"].(string)
    userID, _ := data["id"].(string)

    body, err := s.tpl.Render("welcome.html", map[string]any{
        "first_name":     firstName,
        "activation_url": s.cfg.AppURL + "/activate?u=" + userID,
    })
    if err != nil {
        return err
    }
    return s.email.Send(ctx, email.Message{
        To:      email,
        Subject: "MyDreamCampus'a hoş geldin!",
        HTML:    body,
    })
}
```

### 6.6 Dev environment

Docker compose'a eklenir:

```yaml
services:
  notification:
    build: ./backend/services/notification
    environment:
      DATABASE_URL: postgres://postgres:postgres@notification-db:5432/notification
      RABBITMQ_URL: amqp://guest:guest@rabbitmq:5672/
      SMTP_HOST: mailhog
      SMTP_PORT: 1025
    depends_on:
      - notification-db
      - rabbitmq
      - mailhog

  notification-db:
    image: postgres:18
    environment:
      POSTGRES_DB: notification
      POSTGRES_PASSWORD: postgres
    volumes:
      - notification-db-data:/var/lib/postgresql/data

  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"  # Management UI

  mailhog:
    image: mailhog/mailhog
    ports:
      - "1025:1025"   # SMTP
      - "8025:8025"   # Web UI — gonderilen email'leri gor
```

**Dev avantajı:** MailHog tüm email'leri yakalar, web UI'dan görürsün — gerçek SMTP'ye gerek yok dev'de.

### 6.7 Test stratejisi

| Test türü | Yaklaşım |
|---|---|
| Unit (service) | Saf Go testi, SMTP/DB mock |
| Repository (DB) | testcontainers ile gerçek Postgres |
| Consumer (entegrasyon) | testcontainers ile RabbitMQ + Postgres + MailHog; event publish et, MailHog API'sinden mesaj geldi mi kontrol et |
| Idempotency | Aynı `event_id`'li mesajı 2 kere publish et, sadece 1 email düşmesini doğrula |
| Retry | SMTP'i bilerek 503 dönmeye ayarla, requeue + retry sayacı çalıştığını doğrula |
| DLQ | 4. retry'da DLQ'ya düştüğünü doğrula |

### 6.8 Notification servisi servis sınırı kuralları

| Yasak | Sebep |
|---|---|
| Monolith DB'sine SQL atmak | Servis sınırı net olmalı; payload self-contained |
| Monolith'e HTTP RPC | Tek bağ RabbitMQ; sync bağımlılık eklemek izolasyonu kırar |
| Notification → başka modül event publish | Notification leaf consumer; plan'da kendi event'i tanımlı değil (Bölüm 5.9 katalog eksiksiz) |
| Email/SMS template'i monolith'te tutmak | Notification kendi template'lerini sahiplenir |

---

