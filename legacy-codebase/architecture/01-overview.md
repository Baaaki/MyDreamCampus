> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 1. Hedef ve Motivasyon

**Hedef:** 9 ayrı Go servisi + 9 PostgreSQL instance + RabbitMQ + Traefik yapısını **tek monolith binary + 1 ayrı notification servisi** haline getirmek.

**Neden:**
- Proje CV/portfolio amaçlı, tek geliştirici. 9 servisin operasyonel yükü ve geliştirme sürtünmesi gereksiz.
- Modüler monolith iterasyon hızını kat kat artırır, demo yapması kolaylaşır.
- "Neyi ayır, neyi bırak" yargısını gösteren bir mimari, "her şeyi servis yapan junior" sinyalinden daha güçlü mühendislik sinyali verir.
- Modül sınırları doğru çizilirse, ileride gerçek ihtiyaç doğduğunda modülü servis olarak ayırmak ucuz.

**Neden notification baştan ayrı servis:**
- Fire-and-forget pattern: kullanıcı T+30ms'de HTTP yanıtı alır, email T+800ms'de gönderilir — kullanıcı SMTP'yi beklemez.
- External I/O bağımlılıkları: SMTP (~500ms), FCM (rate-limited), SMS (Twilio). Bu I/O'lar monolith request thread'ini bloklarsa request latency patlamış olur.
- Failure izolasyonu: SMTP server down olduğunda kullanıcı hâlâ kayıt olabilmeli; email biriksin, sonra gönderilsin.
- Bu kararla geçici "in-process dispatcher" yazıp sonra atmaktan kurtuluyoruz; mimari baştan final.

**Bilinen yük profili (somut sayılar):**
- **attendance modülü zirve yükü:** 2 saatte 100.000 yoklama isteği (~14 RPS ortalama, kısa pencerede ~50-100 RPS burst). Modül-modül haritasında en yoğun trafik buradadır. Ders saatleri başında batch'ler beklenir.
- **meal ↔ payment iletişimi:** meal modülü **sadece** payment ile event üzerinden iletişim kurar (yemek kredisi ödeme akışı). Başka modüle event veya read lookup yoktur.
- **diğer modüller:** Düşük-orta trafik (admin operasyonları, dönem başı kayıt akışları). Sayısal eşik henüz ölçülmedi; ihtiyaç doğunca Bölüm 13'e ([Açık Sorular](08-rules.md)) eklenir.

---

## 2. Mimari Kararlar (Özet)

| Konu | Karar | Gerekçe |
|---|---|---|
| Monolith | `backend/monolith/` — auth, staff, student, course_catalog, enrollment, attendance, grades, meal, payment **modül** olarak | Tek binary, tek deploy, hızlı geliştirme |
| Notification | `backend/services/notification/` — ayrı binary, ayrı DB, RabbitMQ consumer | Doğal servis adayı (yukarıda) |
| DB | Monolith: tek PostgreSQL + **schema-per-module**. Notification: ayrı PostgreSQL instance | Ops kolaylığı + izolasyon disiplini |
| Cross-schema FK | **YASAK** | İleride modül ayırma maliyetini sıfıra indirir |
| Cross-schema SELECT/JOIN | **YASAK** | Aynı sebep; teknik mümkün ama mimari yasak |
| Event bus | RabbitMQ (baştan) + **outbox pattern** zorunlu | Atomicity + at-least-once + audit; broker baştan var |
| Inter-module (read) | Diğer modülün public Go `Service` interface'i (in-process call) | Network'süz, tip-güvenli; servis ayrılınca HTTP RPC'ye dönüşür |
| Module → Notification | Outbox → RabbitMQ event (sync HTTP RPC yok) | Notification ayrı servis, async leaf |
| HTTP framework | Tek Gin app, modül başına route group (`/api/<modul>/*`) | Tek port (8080), tek router |
| Reverse proxy | **YOK** — monolith Gin tek binary'de hem `/api/*` hem static frontend'i servis eder | Tek backend için gateway gereksiz; modüler monolith'in ruhuna uygun, ihtiyaç doğarsa sonra eklenir |
| Auth | JWT HS256 (mevcut), her modül kendi middleware'i ile doğrular — auth modülüne RPC yok | Mevcut mimariden değişmez |
| Migration tool | goose (mevcut) | Değişmez |
| ORM/Query | sqlc + pgx/v5 (mevcut) | Değişmez |

---

## 3. Klasör Yapısı

```
backend/
├── monolith/
│   ├── cmd/
│   │   └── main.go                          # Tek entry point
│   ├── config/
│   │   └── config.go
│   ├── internal/
│   │   ├── modules/
│   │   │   ├── auth/
│   │   │   │   ├── service/                 # Public + internal service
│   │   │   │   ├── repository/
│   │   │   │   ├── handler/
│   │   │   │   ├── dto/
│   │   │   │   ├── db/                      # sqlc generated (auth schema)
│   │   │   │   ├── sql/
│   │   │   │   │   ├── migrations/
│   │   │   │   │   └── queries/
│   │   │   │   └── module.go                # Public Service interface + Register()
│   │   │   ├── staff/
│   │   │   ├── student/
│   │   │   ├── course_catalog/              # Go package adi: coursecatalog
│   │   │   ├── enrollment/
│   │   │   ├── attendance/
│   │   │   ├── grades/
│   │   │   ├── meal/
│   │   │   └── payment/
│   │   ├── eventbus/
│   │   │   ├── publisher.go                 # rabbitmq.Publisher wrapper (mevcut shared/rabbitmq kullanir)
│   │   │   ├── outbox_worker.go             # Outbox -> RabbitMQ relay goroutine
│   │   │   └── outbox_worker_test.go
│   │   ├── platform/                        # Mevcut shared/* paketleri buraya tasinir
│   │   │   ├── database/                    # pgxpool config (mevcut shared/database)
│   │   │   ├── logger/                      # zap setup (mevcut shared/logger)
│   │   │   ├── errors/                      # AppError + Wrap/Is (mevcut shared/errors)
│   │   │   ├── middleware/                  # JWT, CORS, RateLimit (mevcut shared/middleware)
│   │   │   ├── rabbitmq/                    # Connection + Publisher (mevcut shared/rabbitmq)
│   │   │   ├── redis/                       # Redis client (mevcut shared/redis)
│   │   │   └── utils/                       # PgType helpers (mevcut shared/utils)
│   │   └── http/
│   │       └── server.go                    # Gin setup, route registration
│   ├── test/
│   │   ├── testdb/                          # Shared Postgres test fixture
│   │   ├── testfixtures/
│   │   └── e2e/                             # Monolith + notification + RabbitMQ
│   ├── Makefile
│   └── go.mod
│
├── services/
│   └── notification/
│       ├── cmd/main.go
│       ├── internal/
│       │   ├── consumer/                    # RabbitMQ consumer (mevcut shared/rabbitmq.Consumer kullanir)
│       │   ├── delivery/                    # SMTP, FCM adapters
│       │   ├── repository/                  # delivery_log + processed_events
│       │   ├── service/
│       │   ├── db/                          # sqlc generated
│       │   └── sql/
│       └── go.mod
│
├── shared/                                   # Sadece monolith + notification ortak event kontratlari
│   └── events/                              # Event ad sabitleri + routing keys (mevcut shared/events.go)
│
└── infrastructure/
    ├── docker-compose.yml                    # Postgres, RabbitMQ, Redis (Traefik YOK)
    └── ...
```

**Modül sınırı kuralı:** `internal/modules/<modul>/` içinden başka modülün `internal/...` paketine import **yasak**. Sadece `module.go` dosyasındaki public `Service` interface ve public DTO'lar import edilebilir.

**Mevcut koddan taşıma:** Mevcut `backend/shared/{database,logger,errors,middleware,rabbitmq,redis,utils}` paketleri **olduğu gibi** `backend/monolith/internal/platform/` altına taşınır. Davranış değişmez, sadece import path güncellenir. `backend/shared/events/` paketi hem monolith hem notification tarafından import edildiği için **`backend/shared/events/` olarak kalır**.

---

