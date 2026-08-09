# Faz 4 — 9 Servisi Ayır

**Ön koşul:** Faz 0, 1, 2, 3 tamamlandı
**Risk:** Orta (hacim yüksek, sürpriz düşük — Faz 0-3 riskleri temizledi)
**Monolith durumu:** Bu fazın sonunda **monolith ölür**

> **Beklenti yönetimi:** Faz 4 bitiminde sistem uçtan uca **çalışmaz** — Caddy
> hâlâ tek upstream'e bakıyor (Faz 5) ve compose'da servisler yok (Faz 6).
> Bu normaldir. Faz 4'ün doğrulaması "her servis tek başına derlenir, ayağa
> kalkar ve `/health` döner" seviyesindedir.

---

## Bu Faz Neden Büyük Görünüp Aslında Mekanik

Faz 0-3 gerçek işi bitirdi: platform paylaşılabilir, DB'ler ayrı, cross-service
tablo okuması yok, HTTP client'lar yazıldı ve gerçek trafikle test edildi. Bu
fazda **yeni mimari kararı yok** — dosya taşıma, `go.mod` yazma ve `main.go`
çoğaltma var.

---

## A Bölümü — Kalan Ortak Kodu `shared/`'a Taşı

### A1. `eventbus` → `shared/eventbus/`

```bash
cd new-backend
git mv monolith/internal/eventbus shared/eventbus
```

3 dosya: `outbox_worker.go`, `topology.go`, `types.go`.

`topology.go` içindeki `ModuleExchanges` listesi ve `DeclareDownstreamBindings`
artık **her servis** tarafından çağrılacak.

**Exchange'ler:** Her servis `DeclareModuleExchanges` ile **9 exchange'in
hepsini** declare eder, sadece kendi yayınladığını değil. Declare idempotent.
Bunu daraltma — grades, `attendance.events`'ten tüketiyor ve attendance
servisi henüz ayağa kalkmamışsa exchange yoktur, binding patlar ve grades
başlangıçta çöker. Hepsini declare etmek bu başlangıç yarışını tamamen
ortadan kaldırır.

**Binding'ler:** Her servis **sadece KENDİ tükettiği** kuyrukları declare
eder. `cmd/main.go`'daki tek büyük `downstreamBindings` listesi 9 parçaya
bölünür (referans: `01-REFERANS-MIMARI.md` §3 tablosu — hangi kuyruk hangi
servisin).

**Yanlış sahipliği düzelt:** `payment/service/payment_service.go:76-79`
`meal.payment_completed_queue` ve `meal.payment_failed_queue`'yu declare edip
bind ediyor — yani **publisher, consumer'ın kuyruğunu tanımlıyor**. Bu satırları
payment'tan **sil**; meal zaten aynı kuyrukları
`meal/worker/event_consumer.go`'da declare ediyor.

### A1b. `definitions.json` — pre-declare garantisini koru

Monolith'te binding'ler `main.go`'da merkezi olarak pre-declare ediliyordu ve
yorumu şunu söylüyordu: *"consumer offline olsa da mesaj birikmeye devam
eder"*. Servisler ayrılınca bu garanti kaybolur — meal-service hiç ayağa
kalkmadıysa `payment.completed` mesajları exchange'e düşer ve **sessizce
kaybolur** (binding yok).

Çözüm hazır bekliyor: `infrastructure/rabbitmq/rabbitmq.conf` içinde şu satır
**yorumlu** duruyor:

```
# management.load_definitions = /etc/rabbitmq/definitions.json
```

Yap:

1. `infrastructure/rabbitmq/definitions.json` oluştur — 9 exchange + tüm
   kuyruklar + tüm binding'ler (`01-REFERANS-MIMARI.md` §3 tabloları).
   Mevcut topolojiyi çalışan bir kurulumdan çıkarabilirsin:
   `curl -u user:pass http://localhost:15672/api/definitions > definitions.json`

   **Dosya git'e girmeden `users`, `permissions`, `policies` bloklarını SİL.**
   RabbitMQ'nun definitions çıktısı **parola hash'lerini içerir**; sadece
   `exchanges`, `queues`, `bindings` kalmalı (`02-GUVENLIK.md` A02/A08).
2. `rabbitmq.conf`'taki satırın yorumunu kaldır.
3. Compose'da mount et (Faz 6):
   `- ./rabbitmq/definitions.json:/etc/rabbitmq/definitions.json:ro`

Servis içi declare'leri **silme** — idempotent, ve tek servisi
`definitions.json` olmadan lokal çalıştırabilmeyi sağlıyor. `definitions.json`
emniyet kemeri, tek kaynak değil.

**Bakım notu:** Yeni kuyruk eklerken hem servisin declare koduna hem
`definitions.json`'a eklenmeli. Bunu Faz 8'de `skills.md`'ye kural olarak yaz.

### A2. `internal/http/server.go` → `shared/httpserver/`

```bash
git mv monolith/internal/http shared/httpserver
```

`Module` interface'i (`Name()`, `RegisterRoutes()`) ve `PublicRoutesProvider`
korunuyor. Servis başına tek modül kayıt edilecek — `RegisterModules` tek
elemanlı çağrılır. Bu, modül kodunu **hiç değiştirmeden** taşımayı sağlar.

SPA static serving kodu (`FRONTEND_STATIC_ENABLED`) artık hiçbir serviste
kullanılmıyor (Caddy servis ediyor) — silinebilir, ama Faz 8'e bırak, şimdi
kapsamı şişirme.

### A3. `monolith/config/` → `shared/config/`

```bash
git mv monolith/config shared/config
```

295 satırlık tek `Config` struct'ı **bölünmüyor**. Her servis aynı struct'ı
okur, ihtiyacı olmayan alanlar boş kalır. Viper env'den okuduğu için ekstra iş
yok.

**Tek değişiklik:** `Validate()` fonksiyonu. Şu an `DB_URL` zorunlu olabilir —
payment servisinin DB'si yok. Zorunluluğu opsiyonel yap:

```go
// payment servisi stateless — DB_URL'siz de geçerli bir config.
func (c *Config) Validate(opts ...ValidateOption) error
```

veya servis kendi zorunlu alanlarını `main.go`'da kontrol etsin. İkincisi daha
basit, onu tercih et.

### A4. Servisler arası DTO'ları `shared/contracts/`'a çıkar

**Bu adım atlanırsa Faz 4 derlenmez.** Faz 3'te yazılan HTTP client'lar hâlâ
karşı modülün DTO paketini import ediyor:

```go
// enrollment/service/clients.go
catalogDTO "github.com/.../monolith/internal/modules/course_catalog/dto"
studentDTO "github.com/.../monolith/internal/modules/student/dto"
```

Servisler ayrı Go modülü olunca bu import bir **modüller arası bağımlılık**
haline gelir — teknik olarak `go.mod require` ile mümkün ama tam da kaldırmaya
çalıştığımız coupling'i geri getirir.

Servis sınırını geçen tipleri bul:

```bash
cd new-backend/monolith/internal/modules
grep -rn "modules/[a-z_]*/dto\"" --include="*.go" . | \
  grep -vE "modules/([a-z_]+)/.*modules/\1/dto"
```

Çıkan her tipi `shared/contracts/` altına taşı:

```
shared/contracts/
├── student.go   # StudentResponse
├── catalog.go   # SemesterCourseListItem, SemesterCourseResponse, SemesterInfo
└── payment.go   # InitiatePaymentRequest/Response, RefundRequest/Response
```

Kurallar:
- **Sadece sınırı geçen tipler** taşınır. Modülün iç DTO'ları yerinde kalır —
  `shared/contracts/` bir çöplük değil, servisler arası **kontrat**tır.
- Sağlayan servis kendi handler'ında `contracts.X` döner, tüketen servis
  `contracts.X` okur. İki taraf da aynı tipi görür.
- Bir alan eklemek geriye uyumlu; alan silmek/yeniden adlandırmak **breaking
  change** — CLAUDE.md §6 gereği kullanıcıya sorulur.

**Mevcut contract testlerini koru.** Repoda `enrollment/dto/event_contract_test.go`,
`attendance/dto/event_contract_test.go`, `grades/dto/event_dto_test.go` var ve
yorumları eski mikroservis servislerine atıfta bulunuyor — bunlar tam da bu
sınırı korumak için yazılmıştı. Servisler ayrılınca **yeniden değer kazanıyorlar**.
Sil me, servisle birlikte taşı, yorumlardaki eski yolları yeni servis
yollarıyla güncelle.

### A5. Import path'lerini güncelle

```bash
cd new-backend
grep -rl "monolith/internal/eventbus\|monolith/internal/http\|monolith/config" \
  --include="*.go" monolith shared \
  | xargs sed -i \
    -e 's|github.com/baaaki/mydreamcampus/monolith/internal/eventbus|github.com/baaaki/mydreamcampus/shared/eventbus|g' \
    -e 's|github.com/baaaki/mydreamcampus/monolith/internal/http|github.com/baaaki/mydreamcampus/shared/httpserver|g' \
    -e 's|github.com/baaaki/mydreamcampus/monolith/config|github.com/baaaki/mydreamcampus/shared/config|g'

cd shared && go mod tidy && go build ./...
cd ../monolith && go mod tidy && go build ./...
```

**Bu noktada commit at** — monolith hâlâ çalışıyor olmalı. Servis bölmeden
önceki son sağlam nokta.

```
refactor(shared): move eventbus, http server and config into the shared module
```

---

## B Bölümü — `go.work`

`new-backend/go.work`:

```
go 1.26

use (
	./shared
	./services/auth-service
	./services/staff-service
	./services/student-service
	./services/catalog-service
	./services/enrollment-service
	./services/attendance-service
	./services/grades-service
	./services/meal-service
	./services/payment-service
	./services/notification-service
)
```

Servisleri ekledikçe satır ekle — hepsi baştan yazılırsa `go build` hata verir.

**Dockerfile'lar `GOWORK=off` ile derliyor** — bu bilinçli, `go.work` sadece
lokal geliştirme içindir. Docker build'lerinde `go.mod` + `replace` kullanılır.
Bu ayarı bozma.

---

## C Bölümü — Tek Servis Çıkarma Tarifi

Aşağıdaki 10 adım **her servis için** tekrarlanır. `<x>` = servis adı,
`<modul>` = monolith'teki modül dizin adı (ikisi farklı olabilir: `catalog`
servisi ↔ `course_catalog` modülü).

Port ve DB adları: `01-REFERANS-MIMARI.md` §1.

### 1. Dizini taşı

```bash
cd new-backend
mkdir -p services/<x>-service
git mv monolith/internal/modules/<modul> services/<x>-service/internal
```

### 2. `go.mod` yaz

```
module github.com/baaaki/mydreamcampus/<x>

go 1.26

replace github.com/baaaki/mydreamcampus/shared => ../../shared

require github.com/baaaki/mydreamcampus/shared v0.0.0-00010101000000-000000000000
```

Kalan bağımlılıkları `go mod tidy` çözer.

### 3. Import path'lerini düzelt

```bash
grep -rl "monolith/internal/modules/<modul>" --include="*.go" . \
  | xargs sed -i 's|github.com/baaaki/mydreamcampus/monolith/internal/modules/<modul>|github.com/baaaki/mydreamcampus/<x>/internal|g'
```

Dikkat: **diğer modüller de** bu path'i import ediyor olabilir (client
adapter'ları). Faz 3'te HTTP client'lara geçtiğimiz için in-process
adapter'ların artık silinmesi gerekiyor — bkz. adım 7.

### 4. `cmd/main.go` yaz

Şablon: `services/notification-service/cmd/main.go` (mevcut, çalışan örnek).
İçerik `monolith/cmd/main.go`'nun o modüle ait dilimi:

```
config.Load()
utils.InitJWTSecret / logger.Init / audit.InitSecurity
pool := database.NewPostgresPool(cfg.Database.URL)      // payment'ta YOK
redisClient := ...                                      // ihtiyacı olanlarda
rabbitConn + publisher
eventbus.DeclareModuleExchanges(publisher)
eventbus.DeclareDownstreamBindings(publisher, <SADECE BU SERVİSİN kuyrukları>)
module := <x>.New(...)                                  // HTTP client'larla
module.Bootstrap(ctx)
go eventbus.NewOutboxWorker("<x>", "<x>.events", module.OutboxStore(), ...)
server := httpserver.NewServer(cfg)
server.RegisterHealthCheck(...)
server.RegisterModules(module)
server.Run() + graceful shutdown
```

Sırayı `monolith/cmd/main.go`'dan kopyala — özellikle Redis'in auth için
**fatal** olduğu kısım ve 30 sn'lik graceful shutdown korunmalı (outbox
worker'ı ve consumer'lar temiz kapanmalı, yoksa restart'ta yarım işlenmiş
event kalır).

**Log'a `service` alanı ekle** — gelecekte Loki'de `{service="grades"}`
sorgusunun dayanağı. Her `main.go`'da logger init'ten hemen sonra:

```go
// 10 servisin logu tek akışa düşecek; hangi satırın kimden geldiği alandan
// okunmalı. Sonradan eklemek 10 dosyaya tek tek dokunmak demek.
logger.Log = logger.Log.With(zap.String("service", "grades"))
```

`logger.Init` imzasına servis adı parametresi eklemek de olur — hangisi
`shared/platform/logger`'ın mevcut yapısına daha temiz oturuyorsa onu seç.

**Rate limit kovası — `02-GUVENLIK.md` A07, atlanırsa güvenlik regresyonu:**

```go
// Global IP/user limitleri TÜM servislerde AYNI Redis kovasını paylaşmalı.
// Buraya servis adı yazılırsa her servis kendi kovasını açar ve saldırgan
// isteği 9 servise yayarak efektif limiti 9 katına çıkarır.
ServiceName: "public"   // sabit — servis adı DEĞİL
```

**Consumer'da izlenebilirlik zincirini sürdür** (`03-IZLENEBILIRLIK.md` [6]):
`shared/platform/rabbitmq`'daki ortak `Consume` sarmalayıcısına
`ctx = logger.WithRequestIDValue(ctx, envelope.CorrelationID)` ekle — bir
yerde çözülür, 9 serviste kazanılır. Consumer'lara tek tek yazma.

**`audit.InitSecurity(cfg.Server.Environment)` her serviste korunmalı**
(`02-GUVENLIK.md` A09) — monolith `main.go`'sunda var, kopyalarken düşürme.

**DLQ'yu bağla** (`04-PROD-HAZIRLIK.md` §1 — kritik). `ConsumeWithDLQ` ve
`SetupDLQ` `shared/platform/rabbitmq`'da yazılı ama **hiçbir yerden
çağrılmıyor**; tüm consumer'lar düz `Consume` kullanıyor ve hatada
`Nack(requeue=true)` yapıyor — yani işlenemeyen bir mesaj sonsuza kadar
kuyruğa geri dönüyor. Servis çıkarırken consumer'ları `ConsumeWithDLQ`'ya
çevir, her kuyruk için `SetupDLQ` çağır, DLQ exchange/queue'larını
`definitions.json`'a ekle.

**Retention worker'ı başlat** (`04-PROD-HAZIRLIK.md` §2). `outbox_events`
satırları `processed` işaretlenip **hiç silinmiyor**; `processed_events`
sadece auth'ta temizleniyor. `shared/eventbus`'a ortak bir retention worker
yaz, her `main.go`'da başlat:

```go
go eventbus.NewRetentionWorker(store, cfg.Timeout.ProcessedEventsRetentionDays,
    cfg.Timeout.CleanupSchedulerIntervalHours).Start(ctx)
```

`03-IZLENEBILIRLIK.md`'nin eklediği `correlation_id` index'i bu tabloların
büyümesini daha pahalı hale getiriyor — retention onunla birlikte gelmeli.

**Idempotency middleware'ini route'lara bağla** (`05-DAYANIKLILIK.md` Bölüm A).
Faz 3'te yazılan middleware, işaretli endpoint'lere `module.go`
`RegisterRoutes` içinde eklenir — **global değil**, her isteğe Redis
round-trip'i gereksiz. Bağlanacak endpoint'ler:

| Servis | Endpoint |
|---|---|
| meal | rezervasyon oluştur / iptal / iade |
| enrollment | program gönder, danışman onay / red |
| grades | not girişi (tekil + bulk) |
| attendance | manuel yoklama |
| student, staff | oluşturma |

QR tarama **bağlanmaz** — Redis `SADD` ile zaten atomik dedup var.

### 5. `/internal/*` route'larını kök altına al

Faz 3'te `/api/<x>/internal/...` altındaydılar. Artık `RegisterPublicRoutes`
ile kök `/internal/...` altına taşı. Caddy sadece `/api/*` proxy'lediği için
bu route'lar **dışarıdan tamamen erişilemez** hale gelir.

`InternalAuth` middleware'ini yine de bırak — derinlemesine savunma.

Karşılık gelen client `baseURL`'lerini de güncelle (`http://staff-service:8082`
+ `/internal/staff/:id`).

### 6. `sqlc.yaml` + `Makefile`

`sqlc.yaml`'ı modülden taşı, içindeki yolları servis köküne göre düzelt
(`internal/sql/queries`, `internal/db`).

Servis kökünde küçük bir `Makefile`:

```make
DB_URL ?= postgres://<x>_svc:$(SERVICE_DB_PASSWORD)@localhost:5432/<db>?sslmode=disable

sqlc:        ; sqlc generate
migrate-up:  ; goose -dir internal/sql/migrations -table goose_db_version_<modul> postgres "$(DB_URL)" up
migrate-down:; goose -dir internal/sql/migrations -table goose_db_version_<modul> postgres "$(DB_URL)" down
build:       ; go build -o bin/<x> ./cmd
run:         ; go run ./cmd
```

goose tablo adı **modül adıyla** kalsın (`goose_db_version_course_catalog`),
DB adıyla değil — Faz 1'de uygulanan migration geçmişiyle uyumlu kalır.

### 7. In-process adapter'ları sil

Faz 3'te HTTP karşılıkları yazıldı. Artık `InProcessStaffClient`,
`InProcessStudentClient`, `InProcessCourseCatalogClient`,
`InProcessSemesterClient`, `PaymentAdapter` ve `DirectAuditLogger`'ın
modül-dışı kullanımları **silinir** — başka servisin somut tipine bağımlılık
bırakırsan `go build` zaten hata verir.

`INTERNAL_CLIENT_MODE` config alanı da bu noktada anlamsızlaşır — sil.

### 8. `Dockerfile`

`services/notification-service/Dockerfile` birebir şablon. Değişenler:
servis adı, port, `COPY` yolları. Build context **repo kökü** (Openship
gereksinimi — yorumu koru).

```dockerfile
FROM golang:1.26 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src
COPY new-backend/shared/ ./shared/
COPY new-backend/services/<x>-service/go.mod new-backend/services/<x>-service/go.sum ./services/<x>-service/
WORKDIR /src/services/<x>-service
RUN go mod download
COPY new-backend/services/<x>-service/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/<x> ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/<x> /app/<x>
EXPOSE <port>
USER nonroot:nonroot
ENTRYPOINT ["/app/<x>"]
```

### 9. `go.work`'e ekle + derle

```bash
cd new-backend/services/<x>-service && go mod tidy && go build ./... && go test ./...
```

### 10. Ayağa kaldır ve doğrula

```bash
cd new-backend/services/<x>-service
DB_URL=... RABBITMQ_URL=... JWT_SECRET=... go run ./cmd &
curl -s localhost:<port>/health
curl -s localhost:<port>/ready
```

`/ready` DB + RabbitMQ + Redis ping'i döner — yeşilse servis sağlam.

**Her servis kendi commit'i:**

```
refactor(<scope>): extract <x> into a standalone service
```

---

## D Bölümü — Çıkarma Sırası

Sıra **bağımlılık yönüne göre**: kimseye sync çağrı yapmayan servisler önce.
Böylece her adımda derleme hatası sayısı minimum kalır.

| # | Servis | Modül dizini | Port | Sync bağımlılığı | Durum |
|---|---|---|---|---|---|
| 1 | staff | `staff` | 8082 | yok | [ ] |
| 2 | payment | `payment` | 8089 | yok (DB'si de yok) | [ ] |
| 3 | auth | `auth` | 8081 | yok | [ ] |
| 4 | student | `student` | 8083 | staff | [ ] |
| 5 | catalog | `course_catalog` | 8084 | staff | [ ] |
| 6 | enrollment | `enrollment` | 8085 | student, catalog | [ ] |
| 7 | attendance | `attendance` | 8086 | catalog | [ ] |
| 8 | grades | `grades` | 8087 | catalog | [ ] |
| 9 | meal | `meal` | 8088 | payment, catalog | [ ] |

Servis çıkardıkça `monolith/cmd/main.go`'dan o modülün wiring'ini sil. 9. adım
sonrası `main.go` boşalır.

### E — notification'ı yeniden adlandır

```bash
git mv new-backend/services/notification new-backend/services/notification-service
```

`go.mod`'daki `replace ... => ../../shared` derinliği aynı kalır (değişiklik
gerekmez). `Dockerfile` içindeki `COPY new-backend/services/notification/`
yollarını güncelle. `docker-compose.yml`'deki `dockerfile:` yolunu güncelle
(Faz 6'da da dokunacaksın, ikisini karıştırma).

### F — monolith'i sil

Dokuz servis de çıktıktan sonra `monolith/` içinde sadece boş `cmd/`, `test/`
ve `go.mod` kalır.

**Bu fazda silme.** Faz 8'de siliniyor — arada bir şeye bakmak gerekirse
elinin altında dursun. `go.work`'ten çıkar ki derlemeyi bozmasın.

### G — migrate Dockerfile'ını güncelle

Migration dosyaları taşındı. `infrastructure/migrate/Dockerfile`:

```dockerfile
COPY new-backend/services/auth-service/internal/sql/migrations       /migrations/modules/auth
COPY new-backend/services/staff-service/internal/sql/migrations      /migrations/modules/staff
COPY new-backend/services/student-service/internal/sql/migrations    /migrations/modules/student
COPY new-backend/services/catalog-service/internal/sql/migrations    /migrations/modules/course_catalog
COPY new-backend/services/enrollment-service/internal/sql/migrations /migrations/modules/enrollment
COPY new-backend/services/attendance-service/internal/sql/migrations /migrations/modules/attendance
COPY new-backend/services/grades-service/internal/sql/migrations     /migrations/modules/grades
COPY new-backend/services/meal-service/internal/sql/migrations       /migrations/modules/meal
COPY new-backend/services/notification-service/sql/migrations        /migrations/notification
```

`/migrations/modules/<ad>` hedefleri **değişmiyor** — `entrypoint.sh`'daki
eşleştirme tablosu bozulmasın.

### H — Kök `Makefile`'ı güncelle

`test-backend` hedefi `./monolith/...` diyor. Değiştir:

```make
test-backend:
	@cd new-backend && go test -race -count=1 ./shared/... ./services/...
```

`backend:` ve `notification:` hedefleri de artık geçersiz — Faz 6'da compose
hedefleriyle değiştirilecek. Şimdilik bozuk bırakabilirsin, Faz 6'da düzelt.

---

## Bitiş Kriteri

```bash
cd new-backend

# 1. 10 servis dizini var
ls services/     # auth-service ... payment-service, notification-service

# 2. Hepsi derleniyor ve testleri geçiyor
go build ./services/... ./shared/...
go test -count=1 ./services/... ./shared/...

# 3. Servisler arası somut tip bağımlılığı kalmadı — boş dönmeli
grep -rn "monolith/internal/modules" --include="*.go" .

# 4. Bir servis diğerinin Go paketini import etmiyor — boş dönmeli
grep -rEn "mydreamcampus/(auth|staff|student|catalog|enrollment|attendance|grades|meal|payment)/internal" \
  --include="*.go" services/ | grep -v "^services/\([a-z]*\)-service/.*mydreamcampus/\1/"

# 5. Her servisin Dockerfile'ı var
ls services/*/Dockerfile | wc -l    # 10
```

3 ve 4 boş dönmüyorsa servis sınırı sızıntısı var — kapatmadan devam etme.

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-4-servis-iskeletleri-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 4 satırını `[x]` yap
3. "Sıradaki faz" satırını **5** yap

---

## Uygulama Notları (faz kapanışında eklendi)

Plandan sapılan veya planda olmayan noktalar:

- **`shared/bootstrap/`** eklendi. Plan her `main.go`'nun monolith'ten
  kopyalanmasını söylüyordu; ortak açılış sırası 9 kez kopyalanmak yerine tek
  pakette toplandı. Gerekçe: `ServiceName: "public"`, `audit.InitSecurity` ve
  boş `INTERNAL_SERVICE_SECRET` kontrolü kopyalandıkça düşen türden
  güvenlik adımları. Servis `main.go`'ları 25-50 satır kaldı.
- **`Consume` yerine `ConsumeEnvelope`.** DLQ bütçesi ve correlation-id
  aktarımı ([6]) tek sarmalayıcıda birleştirildi; her consumer'a ayrı ayrı
  yazılmadı. `MaxDeliveryAttempts = 3` sabit — servis başına ayar yok.
- **Kuyruk argümanları tek kaynaktan** (`rabbitmq.WorkQueueArgs`). RabbitMQ
  farklı argümanla yeniden declare'i reddedip kanalı kapattığı için
  publisher, consumer ve `definitions.json` birebir aynı tabloyu vermeli.
  Bu, **mevcut RabbitMQ volume'unun sıfırlanmasını gerektirir** — eski
  kuyruklar `x-dead-letter-exchange` argümanı olmadan yaratıldı. Faz 6'da
  volume zaten siliniyor.
- **attendance ve meal'de Redis fatal.** `01-REFERANS-MIMARI.md` §6 sadece
  auth'u fatal sayıyor. İkisi Redis'i rate limit değil **veri deposu** olarak
  kullanıyor (QR tampon / QR dedup); nil client ilk taramada panic olurdu.
- **`config.Validate()`'ten `DB_URL` zorunluluğu kaldırıldı** (A3'ün ikinci
  seçeneği): payment'ın şeması yok, kontrolü `bootstrap` yapıyor.
- **`shared/contracts` alias'la bağlandı.** Sağlayıcı modüllerin `dto`
  paketleri taşınan tipleri `type X = contracts.X` ile yeniden ihraç ediyor,
  böylece modül içi 60+ referans ve testler dokunulmadan kaldı.
- **Retention worker sqlc üzerinden.** Ham SQL yasağı (Kod Yazım Kuralları)
  gereği 8 modüle `DeleteProcessedOutboxEvents`, 5 modüle
  `DeleteOldProcessedEvents` query'si eklenip `make sqlc-<m>` çalıştırıldı.
  auth'un `StartCleanupScheduler`'ı yalnızca session temizliğini sürdürüyor.
- **`monolith/` go.work'ten çıkarıldı**, silinmedi (F bölümü). `cmd/` içi
  boşaldı; `test/` ve `go.mod` duruyor.

### Bu fazda doğrulanamayanlar

C bölümü adım 10 (`go run ./cmd` + `curl /health`) çalıştırılmadı: Postgres,
RabbitMQ ve Redis `sudo docker` gerektiriyor (CLAUDE.md §5). Derleme, testler
ve sınır sızıntısı kontrolleri geçti; ayağa kalkma doğrulaması Faz 6/7'de
compose ile yapılacak.
