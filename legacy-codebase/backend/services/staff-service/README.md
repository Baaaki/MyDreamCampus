# Staff Service

Staff yönetimi servisi - Öğretmen ve personel CRUD operasyonları

---

## 🚀 Başlangıç (Bilgisayar Açıldıktan Sonra)

### Terminal 1: Infrastructure'ı Başlat
```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/infrastructure
make up        # PostgreSQL, RabbitMQ, Redis'i başlatır
make health    # Servislerin hazır olduğunu kontrol et
```

### Terminal 2: Staff Service'i Başlat
```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/services/staff-service
make dev       # Migration + Hot-reload ile servisi başlatır
```

### Test Et
```bash
curl http://localhost:8002/health
# Beklenen: {"service":"staff-service","status":"healthy"}
```

---

## 📁 Dosya Yapısı

```
staff-service/
├── cmd/
│   └── main.go                        # Entry point (server setup, dependency injection)
│
├── config/
│   └── config.go                      # Viper ile .env okuma ve validation
│
├── internal/
│   ├── db/                            # sqlc tarafından generate edilen kod
│   │   ├── db.go                      # DBTX interface
│   │   ├── models.go                  # Go struct'ları (Staff, OutboxEvent)
│   │   ├── staff.sql.go               # Staff CRUD sorguları
│   │   └── outbox.sql.go              # Outbox sorguları
│   │
│   ├── dto/                           # Request/Response DTO'ları
│   │   ├── common_dto.go              # Pagination, Response wrappers
│   │   └── staff_dto.go               # Staff DTO'ları (Create, Update, List)
│   │
│   ├── handler/                       # HTTP handlers (Gin)
│   │   └── staff_handler.go           # POST, GET, PUT, DELETE endpoint'leri
│   │
│   ├── repository/                    # Data access layer (sqlc wrapper)
│   │   ├── staff_repository.go        # Staff CRUD + transaction logic
│   │   └── outbox_repository.go       # Outbox event CRUD
│   │
│   ├── service/                       # Business logic
│   │   └── staff_service.go           # Staff service + outbox publishing
│   │
│   └── worker/                        # Background workers
│       └── outbox_worker.go           # RabbitMQ event publisher (outbox pattern)
│
├── sql/
│   ├── migrations/                    # goose migrations
│   │   ├── 00001_create_staff_table.sql
│   │   └── 00002_create_outbox_events_table.sql
│   │
│   └── queries/                       # SQL sorguları (sqlc input)
│       ├── staff.sql                  # Staff CRUD sorguları
│       └── outbox.sql                 # Outbox sorguları
│
├── .air.toml                          # Air hot-reload config
├── .env                               # Environment variables
├── Makefile                           # Build komutları
├── sqlc.yaml                          # sqlc configuration
├── go.mod                             # Go dependencies
└── README.md                          # Bu dosya
```

---

## 🛠️ Makefile Komutları

```bash
make dev              # Infrastructure check + migrate + run (hot-reload)
make run              # Air ile servisi başlat (hot-reload)
make build            # Binary oluştur (bin/staff-service)
make migrate-up       # Migration'ları çalıştır
make migrate-down     # Son migration'ı geri al
make migrate-status   # Migration durumunu göster
make migrate-create   # Yeni migration oluştur (name=... ile)
make sqlc             # sqlc kod generate et
make test             # Unit testleri çalıştır
```

---

## 🧪 API Endpoints

**Base URL:** `http://localhost:8002`

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/health` | Health check |
| POST | `/api/staff` | Yeni staff oluştur |
| GET | `/api/staff` | Staff listele (pagination) |
| GET | `/api/staff/:id` | Staff detayını getir |
| PUT | `/api/staff/:id` | Staff güncelle |
| DELETE | `/api/staff/:id` | Staff sil (soft delete) |

### Örnek Request (Staff Oluştur)
```bash
curl -X POST http://localhost:8002/api/staff \
  -H "Content-Type: application/json" \
  -d '{
    "email": "ahmet.yilmaz@okul.com",
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "role": "teacher",
    "department": "Matematik",
    "phone": "05551234567",
    "office_location": "A Blok 201"
  }'
```

---

## 🔧 Teknoloji Stack

- **Go** 1.23+
- **Gin** 1.10+ (HTTP framework)
- **PostgreSQL** 18 (UUID v7 native support)
- **sqlc** 1.27+ (Type-safe SQL → Go)
- **pgx/v5** (PostgreSQL driver)
- **goose** 3.22+ (Migration tool)
- **RabbitMQ** 4.0 (Message broker)
- **Zap** 1.27+ (Structured logging)
- **Air** 1.52+ (Hot reload)
- **Viper** 1.19+ (Config management)

---

## 📊 Database Schema

**Database Name:** `mydreamcampus_staff`

### staff table
```sql
id              UUID PRIMARY KEY DEFAULT uuidv7()
email           VARCHAR(255) UNIQUE NOT NULL
first_name      VARCHAR(100) NOT NULL
last_name       VARCHAR(100) NOT NULL
role            VARCHAR(50) NOT NULL DEFAULT 'teacher'
department      VARCHAR(100)
phone           VARCHAR(20)
office_location VARCHAR(100)
is_active       BOOLEAN NOT NULL DEFAULT true
deleted_at      TIMESTAMP
created_at      TIMESTAMP NOT NULL DEFAULT NOW()
updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
```

### outbox_events table
```sql
id          UUID PRIMARY KEY DEFAULT uuidv7()
event_type  VARCHAR(100) NOT NULL
payload     JSONB NOT NULL
published   BOOLEAN NOT NULL DEFAULT false
created_at  TIMESTAMP NOT NULL DEFAULT NOW()
```

---

## 🐛 Troubleshooting

### Servis Başlamıyor
```bash
# Infrastructure'ın çalıştığını kontrol et
cd infrastructure && make health

# PostgreSQL hazır mı?
docker exec -it mydreamcampus-postgres pg_isready -U postgres
```

### Migration Hatası
```bash
# Migration durumunu kontrol et
make migrate-status

# Gerekirse sıfırdan başlat
cd infrastructure && make clean  # ⚠️ Tüm verileri siler!
make up
cd ../services/staff-service && make migrate-up
```

### RabbitMQ Connection Error
```bash
# RabbitMQ healthy mi?
docker exec mydreamcampus-rabbitmq rabbitmq-diagnostics ping

# Restart gerekirse
cd infrastructure && docker compose restart rabbitmq
```

---

## 📝 Notlar

- **UUID v7:** Timestamp-based, sortable UUID'ler (PostgreSQL 18 native)
- **Soft Delete:** `deleted_at` ile silinmiş kayıtlar korunur
- **Outbox Pattern:** DB transaction + RabbitMQ event publishing (atomicity)
- **Hot Reload:** Air sayesinde kod değişikliğinde otomatik restart
- **Structured Logging:** Zap ile JSON formatında loglar
- **Event-Driven:** RabbitMQ ile diğer servislere event gönderimi

---

## 🔍 Advanced Logging Features

Staff service'de 3 advanced Zap logging özelliği kullanılıyor:

### 1. Log Sampling (Production)

Production'da aynı log'dan çok fazla basılırsa **sampling** yapılır (disk tasarrufu):

```go
// İlk 3 log yazılır, sonra her 100 logdan 1 tanesi yazılır
config.Sampling = &zap.SamplingConfig{
    Initial:    3,
    Thereafter: 100,
}
```

**Fayda**: Error storm durumunda disk dolması önlenir (%98 tasarruf)

---

### 2. Request ID Tracking

Her HTTP request için **unique ID** oluşturulur ve tüm log'lara otomatik eklenir:

```json
{
  "level": "info",
  "request_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "endpoint": "CreateStaff",
  "msg": "staff created successfully"
}
```

**Kullanım (Grafana Loki)**:
```logql
{service="staff-service"} | json | request_id="a1b2c3d4-..."
```

**Fayda**: Production'da bir request'in tüm log'larını uçtan uca görebilirsin!

---

### 3. Child Loggers (Context-Aware)

Handler ve service layer'da **child logger** kullanılıyor. Ortak field'lar bir kez tanımlanıyor, her log'da otomatik ekleniyor:

**Handler Örneği:**
```go
// Child logger oluştur (request_id + endpoint + handler otomatik eklenir)
reqLogger := logger.WithContextAndFields(ctx,
    zap.String("endpoint", "CreateStaff"),
    zap.String("handler", "StaffHandler"),
)

// Her log'da endpoint & handler OTOMATIK ekleniyor!
reqLogger.Info("creating staff", zap.String("email", req.Email))
reqLogger.Error("failed to create staff", zap.Error(err))
```

**Service Örneği:**
```go
// Service için child logger
serviceLogger := logger.WithContextAndFields(ctx,
    zap.String("service", "StaffService"),
    zap.String("method", "CreateStaff"),
    zap.String("email", req.Email),
)

// Her log'da service + method + email OTOMATIK ekleniyor!
serviceLogger.Info("staff created successfully in database")
```

**Fayda**: Kod tekrarı azalır, log'lar daha tutarlı olur

---

### Production Log Örneği

**CreateStaff Request Flow:**
```json
// 1. Middleware log
{"level":"info","request_id":"a1b2c3d4","method":"POST","path":"/api/staff","msg":"http request","status":201,"duration":"45ms"}

// 2. Handler log
{"level":"info","request_id":"a1b2c3d4","endpoint":"CreateStaff","handler":"StaffHandler","msg":"creating staff","email":"john@example.com"}

// 3. Service log
{"level":"info","request_id":"a1b2c3d4","service":"StaffService","method":"CreateStaff","email":"john@example.com","msg":"staff created successfully in database","staff_id":"123e4567"}
```

**Grafana'da request_id ile filtrelersen**: Bu 3 log'u birlikte görebilirsin! 🎯

---

### Detaylı Dokümantasyon

Tüm advanced logging özelliklerinin detaylı anlatımı için:

📚 [`ADVANCED_LOGGING_FEATURES.md`](../../ADVANCED_LOGGING_FEATURES.md)

---

## ⚠️ Dikkat Edilmesi Gerekenler

### Dosya İzinleri
```bash
# Config dosyaları 644 olmalı (600 değil!)
chmod 644 .env
```

### Infrastructure Önceliği
```bash
# ❌ YANLIŞ: Önce staff service'i başlatma
cd services/staff-service && make dev

# ✅ DOĞRU: Önce infrastructure'ı başlat
cd infrastructure && make up && make health
cd ../services/staff-service && make dev
```

### Port Kontrolü
```bash
# 8002 portu kullanımda mı?
sudo lsof -i :8002

# Gerekirse öldür
sudo kill -9 <PID>
```
