# Code Review - 2026-03-31

Tum backend kodu (3 servis + shared paketler + infrastructure) 5 paralel inceleme ile taranmistir.

---

## KRITIK - Hemen duzeltilmeli

### 1. RabbitMQ data race - mutex yok
**Dosya:** `backend/shared/rabbitmq/connection.go`

`conn`, `channel`, `connected` alanlari hicbir senkronizasyon olmadan hem `handleReconnect` goroutine'i hem de `Channel()`/`IsConnected()` tarafindan okunuyor/yaziliyor. Go race detector bunu yakalayacaktir.

**Duzeltme:** `sync.RWMutex` ekle. `Channel()` ve `IsConnected()` okurken `RLock`, `connect()` yazarken `Lock` kullan.

---

### 2. RabbitMQ reconnect sonrasi consumer sessizce oluyor
**Dosya:** `backend/shared/rabbitmq/consumer.go:52-96`

`Consume` ve `ConsumeWithDLQ` baglanti kopup yeniden kuruldugunda eski `msgs` channel'i kapaniyor, goroutine cikiyor, yeni channel'a abone olunmuyor. Servis event'leri almayi durdurur, log bile basmaz.

**Duzeltme:** Consumer goroutine'i channel kapanmasini algilamali ve yeni channel'a re-subscribe olmali.

---

### 3. RabbitMQ handleReconnect goroutine leak
**Dosya:** `backend/shared/rabbitmq/connection.go:90-112`

`Close()` cagirildiginda goroutine durdurulamaz. Sonsuz dongude kalir.

**Duzeltme:** `done chan struct{}` ekle, `Close()` bunu kapatsin, `handleReconnect` icinde `select` ile dinle.

---

### 4. Refresh token, access token olarak kullanilabiliyor
**Dosya:** `backend/shared/middleware/auth.go:62-148`

`JWTAuth` middleware'i `token_type` claim'ini kontrol etmiyor. Refresh token `Bearer` header'inda gonderilirse gecerli kabul ediliyor (role bos string olur). `RequireRole` olmayan tum endpoint'ler refresh token ile erisilebilir.

**Duzeltme:** Middleware'e `claims.TokenType == "access"` kontrolu ekle.

---

### 5. Hardcoded internal secret fallback
**Dosya:** `backend/shared/middleware/auth.go:199-202`

`INTERNAL_SERVICE_SECRET` env var'i set edilmemisse `"changeme_internal_secret"` kullaniliyor. Bu deger bilindigi icin herhangi biri `X-User-ID`, `X-User-Role` header'larini spoof edebilir.

**Duzeltme:** Env var bossa tum internal header'lari sor kosutsuz strip et. Fallback secret kullanma.

---

### 6. Nil pointer dereference - logout endpoint
**Dosya:** `backend/services/auth-service/internal/service/auth_service.go:736-746`

`parseRefreshTokenWithoutValidation` JWT parse hatasini `_` ile goruyor, ama `token` nil olabilir. Bozuk bir refresh_token cookie'si ile goroutine panic atar.

**Duzeltme:** `token` nil kontrolu ekle veya `ValidateTokenIgnoreExpiry` kullan.

---

### 7. Event contract mismatch - staff.deactivated
**Dosyalar:**
- `backend/services/staff-service/internal/service/staff_service.go:266-268` — `"staff_id"` gonderiyor
- `backend/services/auth-service/internal/dto/event_dto.go:62-63` — `"id"` bekliyor

Sonuc: Staff deactivate edildiginde auth-service'te kullanici HICBIR ZAMAN deactivate edilmez. Buyuk guvenlik acigi.

**Duzeltme:** Staff-service event payload'inda `"staff_id"` yerine `"id"` kullan (veya auth-service DTO'sunu guncelle).

---

### 8. postgres:18-alpine - olmayan Docker image
**Dosya:** `backend/infrastructure/docker-compose.yml:16` (ve tum postgres tanimlari)

PostgreSQL 18 henuz yayinlanmadi. Tum 8 postgres container'i baslatilAmaz.

**Duzeltme:** `postgres:17-alpine` veya `postgres:16-alpine` kullan.

---

## YUKSEK - Yakin zamanda duzeltilmeli

### 9. MaxBytesReader dosya okunduktan sonra ayarlaniyor
**Dosya:** `backend/services/student-service/internal/handler/student_handler.go:493-507`

`FormFile()` body'yi zaten parse etmis, ardindan `MaxBytesReader` set ediliyor. 10MB limit hicbir zaman uygulanmaz. Sinirsiz dosya yukleme mumkun.

**Duzeltme:** `MaxBytesReader`'i `FormFile()`'dan once cagir.

---

### 10. Poison message sonsuz dongusu
**Dosya:** `backend/shared/rabbitmq/consumer.go:86`

`Consume()` (DLQ'suz) hata durumunda `msg.Nack(false, true)` ile mesaji suresiz olarak requeue eder. Parse edilemeyen bir mesaj CPU'yu tuketir.

**Duzeltme:** Tum consumer'larda `ConsumeWithDLQ` kullan, veya `Consume` icinde retry limiti ekle.

---

### 11. CSV import JSON injection
**Dosya:** `backend/services/student-service/internal/service/import_service.go:173-175`

`fmt.Appendf` ile elle JSON olusturuluyor, kullanici kontrollü CSV hucreleri icerebilir. Malformed veya injected JSON uretebilir.

**Duzeltme:** `encoding/json.Marshal` kullan.

---

### 12. URL injection - staff_client.go
**Dosya:** `backend/services/student-service/internal/service/staff_client.go:109`

`department` parametresi `url.QueryEscape` olmadan URL'e interpolate ediliyor. `&` veya `=` karakterleri query string'i bozabilir.

**Duzeltme:** `url.QueryEscape(department)` kullan.

---

### 13. CleanupOldProcessedEvents hicbir zaman calismiyor
**Dosya:** `backend/services/auth-service/internal/repository/event_repository.go:47-58`

`pgtype.Interval{}` sifir degerli ve `Valid: false`. `NOW() - NULL = NULL` oldugundan WHERE kosulu hicbir zaman true olmaz. Temizlik saatlik calisiyor ama hic bir sey silmiyor.

**Duzeltme:** `olderThan` string'ini parse edip `pgtype.Interval{Microseconds: ..., Valid: true}` olustur.

---

### 14. Redis sifre uyumsuzlugu - healthcheck basarisiz
**Dosyalar:**
- `backend/infrastructure/redis/redis.conf:34` — `requirepass` yorum satiri (sifresiz)
- `backend/infrastructure/docker-compose.yml:273` — healthcheck `-a` flag ile sifre gonderiyor

Container surekli `unhealthy` kalir.

**Duzeltme:** `redis.conf`'ta `requirepass` satirini uncomment et veya healthcheck'ten `-a` flag'ini kaldir.

---

### 15. x-retry-count type assertion sessizce basarisiz
**Dosya:** `backend/shared/rabbitmq/consumer.go:165`

Sadece `int32` kontrol ediliyor, RabbitMQ farkli int tipleri gonderebilir. Retry counter hep 0 kalir, `maxRetries` etkisiz, mesajlar DLQ'ya ulasamaz.

**Duzeltme:** Type switch ile `int32`, `int64`, `int` tiplerini kontrol et.

---

### 16. Unsafe type assertion panikleri
**Dosya:** `backend/services/student-service/internal/handler/student_handler.go:575-576`

`userRole.(string)` nil context degerinde panic atar. Ayni pattern `GetImportJobStatus`'ta da var (satir 521, 599, 629).

**Duzeltme:** `userRole, exists := c.Get("role")` ile kontrol et, `exists` false ise 401 don.

---

## ORTA - Planli sprint'te duzeltilmeli

### 17. pgtype struct literal kullanimi (CLAUDE.md ihlali)
`shared/utils/pgtype_helpers.go` helper'lari var ama kullanilmiyor:

| Dosya | Satir | Yanlis | Dogru |
|-------|-------|--------|-------|
| `auth-service/internal/service/auth_service.go` | 103, 144, 430, 524 | `pgtype.Timestamp{Time: t, Valid: true}` | `utils.TimeToPgTimestamp(t)` |
| `staff-service/internal/service/staff_service.go` | 184-187 | `pgtype.UUID{Bytes: id, Valid: true}` | `utils.UUIDToPgtype(id)` |
| `student-service/internal/repository/student_repository.go` | 490 | `pgtype.Text{String: s, Valid: true}` | `utils.StringToPgText(s)` |

---

### 18. HTTP call yerine event kullanilmali (skills.md ihlali)
**Dosya:** `backend/services/student-service/internal/service/staff_client.go`

Student service, staff-service'e dogrudan HTTP cagrisi yapiyor. `backend/skills.md` bunu acikca yasakliyor: "HTTP call YAPMA — RabbitMQ ile event-driven iletisim kullan."

**Duzeltme:** Advisor bilgisini event ile senkronize tut veya student DB'sinde snapshot sakla.

---

### 19. err == pgx.ErrNoRows - errors.Is kullanilmali
**Dosyalar:**
- `auth-service/internal/repository/auth_repository.go` — satirlar 33, 45, 77, 89, 101, 143, 167
- `auth-service/internal/repository/session_repository.go` — satirlar 41, 62, 74, 89
- `staff-service/internal/repository/staff_repository.go` — satirlar 90, 103, 125, 162

`==` ile karsilastirma wrapped error'larda calismaz. `errors.Is(err, pgx.ErrNoRows)` kullanilmali.

---

### 20. CountStudents filtre uyumsuzlugu
**Dosya:** `backend/services/student-service/internal/service/student_service.go:469-477`

Filtreli listeleme yaparken filtresiz `CountStudents` cagiriliyor. Pagination `total` degeri yanlis.

**Duzeltme:** `CountStudentsFiltered` query'si ekle, ayni filtreleri uygula.

---

### 21. Recovery middleware panic detaylarini expose ediyor
**Dosya:** `backend/shared/middleware/recovery.go:33`

Panic mesajlari HTTP response'ta gosteriliyor (dosya yollari, bellek adresleri). Information disclosure.

**Duzeltme:** Generic "Internal server error" don, detayi sadece logla.

---

### 22. Session rotation atomik degil
**Dosya:** `backend/services/auth-service/internal/service/auth_service.go:420-438`

Eski session silinip yeni olusturuluyor, transaction yok. `CreateSession` basarisiz olursa kullanici sessionsiz kalir.

**Duzeltme:** Her iki islemi tek transaction icine al.

---

### 23. Staff outbox'ta retry/failure tracking yok
**Dosyalar:**
- `backend/services/staff-service/sql/migrations/00002_create_outbox_events_table.sql`
- `backend/services/staff-service/internal/worker/outbox_worker.go`

Sadece boolean `processed` var, `retry_count`/`max_retries`/`error_message` yok. Basarisiz mesajlar sonsuza kadar denenir.

**Duzeltme:** Student-service'teki outbox schema'sini (status enum, retry_count, max_retries) staff-service'e de uygula.

---

### 24. rand.Read hatasi gormezden geliniyor
**Dosya:** `backend/shared/middleware/csrf.go:67`

`crypto/rand.Read` entropy tukenmesinde hata donebilir. Hata gormezden gelinirse CSRF token `"000...000"` olabilir.

**Duzeltme:** Hatayi kontrol et, basarisizsa 500 don.

---

### 25. Access token suresi varsayilan 600 dakika (10 saat)
**Dosya:** `backend/services/auth-service/config/config.go:41`

Tipik guvenli varsayilan 15-30 dakikadir. 10 saat cok uzun.

**Duzeltme:** Varsayilani 15 veya 30 dakikaya dusur.

---

## DUSUK - Iyilestirme

| # | Dosya | Konu |
|---|-------|------|
| 26 | `student-service/sql/migrations/00001_create_students_table.sql:13` | `status` kolonu `NOT NULL` constraint eksik |
| 27 | `staff-service/internal/worker/outbox_worker.go:85` | `event_id` UUID degil, integer serial — global uniqueness yok |
| 28 | Tum servislerin outbox query'leri | `FOR UPDATE SKIP LOCKED` yok — multi-instance'da duplicate publish riski |
| 29 | `infrastructure/traefik/dynamic.yml:364` | CORS `maxAge: 100` cok kisa, 86400 olmali |
| 30 | `shared/middleware/auth.go:151-189` | `OptionalJWTAuth` blacklist/token version kontrolu yapmiyor |
| 31 | `staff-service/cmd/main.go:123`, `student-service/cmd/main.go:134` | Worker goroutine'leri shutdown'da beklenmiyor (`sync.WaitGroup` ekle) |
| 32 | `infrastructure/Makefile:24,165` | Traefik dashboard URL'i yanlis (port 8080 expose edilmiyor) |
| 33 | `shared/utils/pgtype_helpers.go:44-51,262` | `PgtypeToUUID`/`PgUUIDToUUID` Valid flag'i kontrol etmiyor, sessizce `uuid.Nil` donuyor |
| 34 | `shared/utils/pgtype_helpers.go:203-208` | `Float64ToPgNumeric` Scan hatasini gormezden geliyor |
| 35 | `auth-service/config/config.go:44` | `ADMIN_INITIAL_PASSWORD` varsayilani `"Admin123!"` — production'da tehlikeli |
| 36 | `student-service/internal/repository/student_repository.go:279-283` | Constraint name mismatch — unique index violation dogru yakalanmiyor |
| 37 | Tum migration'lar | `TIMESTAMP` yerine `TIMESTAMPTZ` kullanilmali (timezone farkliliklarinda veri bozulmasi) |

---

## Ozet Tablosu

| Kategori | Kritik | Yuksek | Orta | Dusuk | Toplam |
|----------|--------|--------|------|-------|--------|
| Guvenlik | 3 | 3 | 2 | 1 | 9 |
| RabbitMQ / Event | 3 | 2 | 2 | 1 | 8 |
| Infrastructure | 2 | 1 | 0 | 3 | 6 |
| CLAUDE.md uyumu | 0 | 0 | 3 | 0 | 3 |
| Diger bug'lar | 0 | 2 | 2 | 7 | 11 |
| **Toplam** | **8** | **8** | **9** | **12** | **37** |

---

## Onerilen Oncelik Sirasi

1. **Ilk sprint:** Kritik 1-8 (guvenlik + altyapi)
2. **Ikinci sprint:** Yuksek 9-16 (input validation + consumer)
3. **Ucuncu sprint:** Orta 17-25 (CLAUDE.md uyumu + kod kalitesi)
4. **Backlog:** Dusuk 26-37 (iyilestirme)
