# Uçtan Uca İstek İzlenebilirliği

> Faz dosyası değil, **tek bir gereksinimin uçtan uca tarifi**. Faz 3 ve Faz 4
> bu dosyaya atıf yapıyor.

**Gereksinim:** Client'tan gelen bir isteğin, dokunduğu **tüm** servislerde ve
tetiklediği **tüm** event zincirinde tek bir kimlikle takip edilebilmesi.

Bu, OWASP A09'un (Security Logging Failures) karşılığı ve sonradan eklenmesi
en pahalı şeylerden biri: zincirin her halkasına dokunmayı gerektirir.

---

## Zincir ve Halkaların Durumu

```
  Client
    │  X-Request-ID (yoksa üretilir)
    ▼
  CADDY ───────────────────────────────────────── [2] EKLENECEK
    │
    ▼
  Servis A  ── RequestLogger: header'ı al/üret, ctx'e koy ── [1] ZATEN VAR
    │            logger.WithContext(ctx) → log'da request_id  ── ZATEN VAR
    │            response'a X-Request-ID yaz                  ── ZATEN VAR
    │
    ├──HTTP──► Servis B ── base.go giden header  ───────────── [3] EKLENECEK
    │                       (Faz 3'te planlandı)
    │
    └──outbox──► outbox_events satırı ──────────────────────── [4] EKLENECEK
                    │
                    ▼
                 OutboxWorker → envelope.correlation_id ────── [5] EKLENECEK
                    │
                    ▼
                 RabbitMQ
                    │
                    ▼
                 Servis C consumer → ctx'e geri koy ────────── [6] EKLENECEK
                    │
                    └──outbox──► zinciri taşımaya devam ────── [7] EKLENECEK
```

**İyi haber:** [1] tamamen hazır.
[middleware/logger.go](new-backend/monolith/internal/platform/middleware/logger.go)
gelen `X-Request-ID`'yi onurlandırıyor, yoksa UUID üretiyor, context'e koyuyor,
response header'ına yazıyor. `logger.WithContext(ctx)` de log satırına
`request_id` alanını otomatik ekliyor. Mikroservis döneminden kalma altyapı.

Eksik olan 6 halka aşağıda.

---

## [2] Caddy — kenarda ID üret

**Nerede:** `frontend/Caddyfile` (Faz 5)

```caddyfile
	# Gerçek kenar burası. ID'yi Caddy üretirse Caddy access log'u ile servis
	# logları aynı ID'yi paylaşır; servise bırakılırsa kenardaki hop görünmez.
	request_header X-Request-ID {http.request.uuid}
```

Caddy yalnızca **yoksa** üretmeli — client kendi ID'sini gönderiyorsa
(mobil uygulama debug modu) o korunmalı. Caddy'nin `{http.request.uuid}`
placeholder'ı her istek için üretiliyor; mevcut header'ı ezmemek için
`header_up` yerine koşullu matcher kullan:

```caddyfile
	@no_request_id not header X-Request-ID *
	request_header @no_request_id X-Request-ID {http.request.uuid}
```

---

## [3] Servisler arası HTTP — giden header

**Nerede:** `shared/client/base.go` (Faz 3, adım 1)

```go
if rid := logger.GetRequestID(ctx); rid != "" {
    req.Header.Set("X-Request-ID", rid)
}
```

Karşı serviste `RequestLogger` bunu okuyup kendi context'ine koyar — halka [1]
zaten çalıştığı için karşı taraf ek iş gerektirmez.

---

## [4] Outbox satırında ID'yi sakla

**Nerede:** 8 modülün outbox migration'ı (Faz 3'te yaz — modüller hâlâ
monolith'te, `make migrate-create-<module>` tek yerden çalışıyor, Faz 4'ten
sonra 8 ayrı serviste yapmaktan kolay)

Mevcut tablo (`staff` örneği) şu kolonlara sahip:
`id, event_type, routing_key, payload, status, retry_count, max_retries,
error_message, created_at, processed_at` — **korelasyon alanı yok.**

```sql
-- +goose Up
-- Event'i tetikleyen HTTP isteğinin kimliği. "Bu istek hangi event'leri
-- doğurdu" sorusunun SQL'den cevaplanabilmesi için ayrı kolon; payload
-- JSON'una gömülse sorgulanamazdı.
ALTER TABLE staff.outbox_events ADD COLUMN correlation_id UUID;
CREATE INDEX idx_staff_outbox_correlation ON staff.outbox_events(correlation_id)
  WHERE correlation_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS staff.idx_staff_outbox_correlation;
ALTER TABLE staff.outbox_events DROP COLUMN correlation_id;
```

Sekiz modül için tekrarla: `auth`, `staff`, `student`, `course_catalog`,
`enrollment`, `attendance`, `grades`, `meal`. (payment'ın DB'si yok — envelope'a
doğrudan yazar.)

**Nullable** olmalı: worker/scheduler kaynaklı event'lerin (SessionExpiry,
ReservationWorker) tetikleyen bir HTTP isteği yoktur.

Sonra `make sqlc-<module>` — outbox insert query'sine kolonu ekle.

Servis katmanında outbox'a yazarken `logger.GetRequestID(ctx)` değerini geçir.

---

## [5] Envelope'a `correlation_id`

**Önce keşfet:** Envelope (`{event_id, event_type, timestamp, data}`) nerede
kuruluyor tespit et — `shared/platform/rabbitmq` publisher'ı ya da
`shared/eventbus/outbox_worker.go`. `shared/events/events.go` sadece sabit
tutuyor, struct orada değil.

```go
type Envelope struct {
    EventID       string    `json:"event_id"`
    EventType     string    `json:"event_type"`
    Timestamp     time.Time `json:"timestamp"`
    CorrelationID string    `json:"correlation_id,omitempty"`
    Data          any       `json:"data"`
}
```

**Bu bir event şeması değişikliği** (CLAUDE.md §6 normalde onay ister).
Kullanıcı uçtan uca izlenebilirliği talep ettiği için **onaylı sayılır**.
Geriye uyumlu: `omitempty` + mevcut consumer'lar bilinmeyen alanı yok sayıyor,
eski ve yeni event'ler bir arada akabilir.

---

## [6] Consumer — ID'yi context'e geri koy

**Nerede:** her `worker/*_consumer.go` (Faz 4, servis çıkarılırken)

```go
// Zincir burada devam ediyor: event'i doğuran HTTP isteğinin ID'si
// consumer'ın kendi log satırlarına da düşsün.
ctx = logger.WithRequestIDValue(ctx, envelope.CorrelationID)
```

`WithRequestIDValue` boş değer gelirse yeni UUID üretiyor — worker kaynaklı
event'lerde de log'da bir ID olur, zincir başlar.

Bunu her consumer'a tek tek yazmak yerine ortak consumer sarmalayıcısına
(`shared/platform/rabbitmq` içindeki `Consume` helper'ı) koy — bir yerde
çözülür, 9 serviste kazanılır.

---

## [7] Zinciri ileri taşı

Consumer bir event'i işlerken kendi outbox'ına yazıyorsa (örnek: grades'in
`attendance.semester.failed`'i işleyip `grade.finalized` yayınlaması),
`correlation_id` aynen aktarılmalı. [6] context'e koyduğu için [4]'teki
`logger.GetRequestID(ctx)` çağrısı bunu otomatik alır — ek kod gerekmez.

---

## Doğrulama (Faz 7)

```bash
# 1. Bir istek at, ID'yi yakala
RID=$(curl -si localhost/api/auth/login -d '{...}' | grep -i "^x-request-id:" | tr -d '\r' | awk '{print $2}')
echo "$RID"

# 2. Tüm servislerin logunda aynı ID görünüyor mu
for c in $(sudo docker ps --format '{{.Names}}' | grep mydreamcampus-); do
  n=$(sudo docker logs "$c" 2>&1 | grep -c "$RID")
  [ "$n" -gt 0 ] && echo "$c: $n satır"
done

# 3. Event zinciri: o isteğin doğurduğu event'ler
sudo docker exec mydreamcampus-postgres psql -U postgres -d staff -c \
  "SELECT event_type, routing_key, created_at FROM staff.outbox_events
   WHERE correlation_id = '$RID' ORDER BY created_at;"
```

**Başarı kriteri:** Öğrenci ekleme gibi çok servisli bir akışta (student →
event → auth + attendance + grades + meal), tek `X-Request-ID` ile **beş
servisin** logu ve outbox satırları bulunabilmeli.

---

## Bu Dosyanın Kapsamadığı (bilinçli)

- **OpenTelemetry / distributed tracing** — span, parent-child ilişkisi,
  süre ölçümü. Correlation ID bunun yerine geçmez, ama OTel'e geçildiğinde
  `request_id` → `trace_id` eşlemesi kolay olur.
- **Prometheus metrikleri** — `00-BASLANGIC.md` gözlemlenebilirlik tablosuna bak.
- **Log toplama (Loki)** — bu dosya ID'nin **taşınmasını** sağlar; toplamak
  ayrı iş. Taşınmıyorsa Loki de işe yaramaz, sırası bu.
