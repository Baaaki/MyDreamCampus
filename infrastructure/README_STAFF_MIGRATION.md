# Staff Service → Infrastructure Migration ✅

Bu dokümantasyon, staff-service'in infrastructure klasörüne başarılı bir şekilde taşınmasını açıklar.

## ✅ Yapılan Değişiklikler

### 1. Infrastructure Setup

#### [infrastructure/postgres/init-dbs.sql](postgres/init-dbs.sql)
```sql
-- mydreamcampus_staff database oluşturuldu
-- uuid-ossp ve pgcrypto extension'ları aktif edildi
```

#### [infrastructure/docker-compose.yml](docker-compose.yml)
- PostgreSQL (port 5432)
- RabbitMQ (port 5672, 15672)
- Redis (port 6379)
- Traefik (port 80, 8080)

### 2. Staff Service Değişiklikleri

#### Silinen Dosyalar
- ❌ `services/staff-service/docker-compose.yml` (artık gerekmiyor)

#### Güncellenen Dosyalar

**[services/staff-service/.env](../services/staff-service/.env)**
```bash
# BEFORE
DB_URL=postgres://user:password@localhost:5432/mydreamcampus_staff
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# AFTER (infrastructure credentials)
DB_URL=postgres://postgres:postgres@localhost:5432/mydreamcampus_staff
RABBITMQ_URL=amqp://rabbitmq:rabbitmq@localhost:5672/
```

**[services/staff-service/Makefile](../services/staff-service/Makefile)**
- ❌ `docker-up`, `docker-down`, `docker-clean` komutları kaldırıldı
- ✅ `check-infra` komutu eklendi (infrastructure kontrolü)
- ✅ `DB_URL` güncellendi (postgres:postgres)
- ✅ `dev` workflow güncellendi: `check-infra → migrate-up → run`

## 🚀 Yeni Kullanım

### Adım 1: Infrastructure'ı Başlat (İlk Kez)

```bash
cd infrastructure
docker-compose up -d
```

**Beklenen Çıktı**:
```
✅ mydreamcampus-postgres (localhost:5432)
   └─ mydreamcampus_staff database
✅ mydreamcampus-rabbitmq (localhost:5672, localhost:15672)
✅ mydreamcampus-redis (localhost:6379)
✅ mydreamcampus-traefik (localhost:80, localhost:8080)
```

### Adım 2: Infrastructure Durumunu Kontrol Et

```bash
docker ps --filter "name=mydreamcampus"
```

**Beklenen Çıktı**:
```
CONTAINER ID   IMAGE                            STATUS         PORTS
abc123...      postgres:17-alpine               Up 10 seconds  0.0.0.0:5432->5432/tcp
def456...      rabbitmq:4.0-management-alpine   Up 10 seconds  0.0.0.0:5672->5672/tcp, 0.0.0.0:15672->15672/tcp
ghi789...      redis:7.2-alpine                 Up 10 seconds  0.0.0.0:6379->6379/tcp
jkl012...      traefik:v3.2                     Up 10 seconds  0.0.0.0:80->80/tcp, 0.0.0.0:8080->8080/tcp
```

### Adım 3: Database'i Kontrol Et

```bash
docker exec -it mydreamcampus-postgres psql -U postgres -c "\l"
```

**Beklenen Çıktı**:
```
                                   List of databases
         Name          |  Owner   | Encoding
-----------------------+----------+----------
 mydreamcampus_staff   | postgres | UTF8     ← ✅ STAFF DATABASE
 postgres              | postgres | UTF8
```

### Adım 4: Staff Service'i Çalıştır

```bash
cd services/staff-service
make dev
```

**Beklenen Çıktı**:
```
Checking if infrastructure is running...
✅ Infrastructure is running
goose -dir sql/migrations postgres "postgres://postgres:postgres@localhost:5432/mydreamcampus_staff?sslmode=disable" up
OK    00001_create_staff_table.sql
OK    00002_create_outbox_table.sql
air
  __    _   ___
 / /\  | | | |_)
/_/--\ |_| |_| \_ 1.52.0

watching...
building...
running...

INFO    starting staff-service    {"environment": "development", "port": "8002"}
INFO    database connection established
INFO    RabbitMQ connection established
INFO    RabbitMQ exchange declared    {"exchange": "staff_exchange"}
INFO    server starting    {"port": "8002"}
```

### Adım 5: Test Et

```bash
# Health check
curl http://localhost:8002/health

# List staff
curl http://localhost:8002/api/staff
```

## 🔍 Connection Details

### PostgreSQL
- **Host**: `localhost:5432`
- **User**: `postgres`
- **Password**: `postgres`
- **Database**: `mydreamcampus_staff`
- **Connection String**: `postgres://postgres:postgres@localhost:5432/mydreamcampus_staff?sslmode=disable`

### RabbitMQ
- **AMQP**: `localhost:5672`
- **Management UI**: `http://localhost:15672`
- **User**: `rabbitmq`
- **Password**: `rabbitmq`
- **Connection String**: `amqp://rabbitmq:rabbitmq@localhost:5672/`

### Redis
- **Host**: `localhost:6379`
- **Password**: (none)
- **DB**: `0`

### Traefik Dashboard
- **URL**: `http://localhost:8080/dashboard/`

## 🛠️ Troubleshooting

### Infrastructure Çalışmıyor
```bash
cd infrastructure
docker-compose up -d
docker-compose ps
docker-compose logs -f
```

### Port Çakışması
Eğer başka bir PostgreSQL/RabbitMQ/Redis çalışıyorsa:
```bash
# Hangi process 5432'yi kullanıyor?
sudo lsof -i :5432

# Varsa durdur
sudo systemctl stop postgresql
```

### Database Bağlantısı Başarısız
```bash
# PostgreSQL ready mi?
docker exec -it mydreamcampus-postgres pg_isready -U postgres

# Database var mı?
docker exec -it mydreamcampus-postgres psql -U postgres -c "\l" | grep staff
```

### RabbitMQ Bağlantısı Başarısız
```bash
# RabbitMQ ready mi?
docker exec -it mydreamcampus-rabbitmq rabbitmq-diagnostics ping

# Management UI'a gir
open http://localhost:15672
# Login: rabbitmq / rabbitmq
```

## 📊 Öncesi vs Sonrası

### ❌ Öncesi (Her Servis Kendi Infrastructure'ı)
```
services/staff-service/
├── docker-compose.yml        ← PostgreSQL + RabbitMQ burada
├── Makefile                  ← docker-up/down komutları
└── .env                      ← guest:guest credentials

services/auth-service/
├── docker-compose.yml        ← AYNI ŞEYLER TEKRAR (port çakışması!)
```

### ✅ Sonrası (Merkezi Infrastructure)
```
infrastructure/
├── docker-compose.yml        ← TEK MERKEZ (tüm servisler için)
├── postgres/init-dbs.sql     ← Database init
├── rabbitmq/rabbitmq.conf    ← RabbitMQ config
└── redis/redis.conf          ← Redis config

services/staff-service/
├── Makefile                  ← Sadece migrate + run
└── .env                      ← Infrastructure'a bağlanır
```

## 🎯 Avantajlar

1. ✅ **Port çakışması yok** - Tek PostgreSQL instance
2. ✅ **Kaynak tasarrufu** - Paylaşımlı infrastructure
3. ✅ **Kolay setup** - `docker-compose up -d` bir kere yeterli
4. ✅ **Gerçek mikroservis** - Shared infrastructure, isolated data
5. ✅ **DRY principle** - Infrastructure kodu tekrarı yok

## 📝 Notlar

- Staff service artık **sadece uygulama kodu** içeriyor
- Infrastructure **tek seferlik** başlatılır, tüm servisler kullanır
- Diğer servisler (auth, student) henüz taşınmadı
- Production'da mutlaka credentials değiştirilmeli

---

**Migrasyon Tamamlandı**: 2025-12-18 ✅
