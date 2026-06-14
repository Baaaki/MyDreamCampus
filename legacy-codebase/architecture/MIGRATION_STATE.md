# Modüler Monolith Migration — Durum

> **Bu dosyayı oturum başında oku.** Hangi modüller bitti, hangileri kaldı, hangi pattern hangi dosyada — buradan devam edersin.

Son güncelleme: 2026-05-08
Mimari plan: [README.md](README.md)
Hedef: 9 mikroservisi `new-backend/monolith` altında modüler monolith'e + `services/notification`'a taşı. `backend/` klasörüne dokunulmuyor.

---

## Tamamlananlar

| Faz | Modül | Durum | Test sayısı | Atomic commit |
|---|---|---|---|---|
| 0 | Iskelet (cmd, config, eventbus, http, platform, shared/events, infra) | ✓ | — | bekliyor |
| 1 | auth | ✓ | 49 | bekliyor |
| 2.1 | staff | ✓ | 40 | bekliyor |
| 2.2 | student | ✓ | 84 | bekliyor |
| 2.3 | course_catalog | ✓ | ~165 | bekliyor |
| 2.4 | enrollment | ✓ | 48 | bekliyor |
| 2.5 | attendance | ✓ | 65 | bekliyor |
| 2.6 | grades | ✓ | ? | bekliyor |
| 2.7 | meal | ✓ | ? | bekliyor |
| 2.8 | payment | ✓ | ? | bekliyor |
| 3 | notification | ✓ | ? | bekliyor |

**Toplam mevcut:** `go build ./monolith/... ✓` — `go test` 615 test, 60 pakette geçiyor — `go vet` temiz.

> **Not:** Henüz hiç commit atılmadı (kullanıcı isteği). Tüm değişiklikler unstaged, ilk gerçek commit için tek seferde tüm faz 0-2.4 birleşip atılacak.

---

## Sıradakiler (öncelik sırası)

* **Phase 4: Frontend / Mobile proxy entegrasyonu ve Testler** - PENDING
   - **ÖNEMLİ:** `backend/` klasörü ASLA silinmeyecek, referans olarak korunacaktır. Eski dosyaların temizliği yapılmayacaktır.
   - Frontend `vite.config.ts` ve Mobile `.env` yönlendirmelerinin yeni monolith'e (port 8080) göre ayarlanması
   - Uçtan uca sistem testlerinin koşturulması

---

## Yerleşik Pattern'ler (her modül için tekrarlanan)

### A. Modül klasör yapısı
```
internal/modules/<modul>/
├── module.go              # Module interface impl + Bootstrap (varsa)
├── sqlc.yaml              # rename map: <modul>_<table> → <Table>
├── db/                    # sqlc-generated (gitignored — regenerate)
├── dto/
├── errors/
├── handler/
├── repository/
│   ├── <main>_repository.go
│   ├── outbox_repository.go
│   └── outbox_store.go    # eventbus.OutboxStore adapter
├── service/
│   ├── <main>_service.go
│   ├── event_payloads.go  # map[string]any builder'lar
│   └── staff_client.go    # in-process adapter (HTTP client'in yerine)
├── worker/                # event_consumer.go (varsa) — outbox_worker.go SİLİNDİ
└── sql/
    ├── migrations/
    │   ├── 00000_create_schema.sql       # CREATE SCHEMA <modul>
    │   ├── 00001_create_<main>_table.sql # tablolar <modul>.<table>
    │   ├── 00002_create_outbox_events_table.sql
    │   └── ...
    └── queries/
```

### B. Schema-prefix kuralı (kritik)
- Tüm migration ve query SQL'lerinde tablolar **schema-qualified**: `<modul>.<table>` (örn. `student.students`)
- Ana enum'lar da: `<modul>.<enum_name>` (örn. `student.outbox_status_enum`)
- Cross-schema FK / JOIN **YASAK** (plan Bölüm 4.2)

### C. sqlc rename (her modülün sqlc.yaml'ında)
```yaml
gen:
  go:
    rename:
      <modul>_<table>: "<Table>"        # student_student → Student
      <modul>_<table_2>: "<Table2>"
      <modul>_<enum>: "<Enum>"
```
Sebep: schema-qualified tablo `<modul>.<table>` → sqlc default'ta `<Modul><Table>` üretir; rename ile orijinal repo kodu uyumlu kalsın.

**Dikkat — enum value constants:** sqlc `rename` sadece TIPI yeniden adlandırır. `<modul>_<enum>_value` constant'ları otomatik rename OLMUYOR, tam ad kullanılır. Örnek: `db.CourseCatalogCourseCatalogStatusEnumActive` (rename: type → `CourseCatalogStatusEnum` ama constant kalır). Repo kodunda bu uzun adlara sed ile çevrildi.

### D. Migration konsolidasyonu (yeni DB için)
Mevcut servislerin migration history'sinde ALTER TABLE / DROP COLUMN serileri varsa (örn. CC 00006 drop column, student 00005 add advisor_name), monolith fresh DB için:
- Birleştir (yeni final state'i 00001'e yaz, ALTER'ları sil) — student'ta yapıldı
- VEYA olduğu gibi bırak (CC'de yapıldı — 13 migration sırayla çalışıyor)

### E. OutboxStore adapter pattern (her publishing modül)
`repository/outbox_store.go`: sqlc tipini `eventbus.OutboxEvent` value type'ına çeviren adapter. `eventbus.OutboxStore` interface'ini implement eder. Generic `eventbus.OutboxWorker` bunu çalıştırır (per-module goroutine, plan Bölüm 5.5.2 Seçenek A).

CC modülünde `ResetFailedOutboxEvent` query'si yoktu — eklendi (eventbus.OutboxWorker reset'e ihtiyaç duyuyor).

### F. Cross-module HTTP client → in-process adapter
Eski `staff_client.go` HTTP çağrıları (`/internal/staff/...`) yapıyordu. Monolith'te bu file'lar yeniden yazıldı:
- `service.NewInProcessStaffClient(staffSvc *staffService.StaffService)` — staff modülünün public servisini doğrudan çağırır
- Aynı interface (`StaffClient`, `StaffServiceInterface` vb.) korunuyor — `StudentService` / `SemesterService` constructor'ları aynı imza
- Plan Bölüm 8 strateji 1: cross-module read **public Service interface üzerinden in-process call**

### G. Module interface
`internal/http/server.go`:
```go
type Module interface {
    Name() string                          // /api/<name> URL slug
    RegisterRoutes(rg *gin.RouterGroup)    // /api/<name>/* mount
}

// Optional — for routes outside /api/<name>
type PublicRoutesProvider interface {
    RegisterPublicRoutes(r *gin.Engine)
}
```

### H. Module.Name() vs schema name
- Schema/Go-package: snake_case `course_catalog`
- URL slug: legacy frontend yolu → `catalog` (CC), `students` (plural - student frontend), `staff`, `auth`

### I. Downstream RabbitMQ binding'leri
`cmd/main.go` startup'ta `eventbus.DeclareDownstreamBindings`:
- `auth_events_queue` ← staff/student events
- `student.staff_events` ← staff.deactivated

Tüm modüller migrate olunca bu RabbitMQ köprüleri in-process pubsub'a refactor edilebilir (ama bu Faz 4 sonrası iş — şu an çalışıyor).

### J. Fresh dependency: `internal/platform/...` paketleri
İhtiyaç doğdukça `backend/shared/` 'dan kopyalanır:
- Faz 0: database, logger, errors, middleware, rabbitmq, redis, utils, clock, audit, handler/health, dto/time
- Faz 1+: handler/time_handler.go (staff için)
- Faz 2.3: repository (period), rules (grading/period/semester), handler/simple_period_handler.go, dto/period_dto.go

`backend/shared/client/` (HTTP client) **kopyalanmadı** — in-process call'a dönüşüyor, gerek kalmadı.

---

## Yapılan Mimari Karar Notları

1. **Modül başına outbox worker (plan Bölüm 5.5.2 Seçenek A)** — generic `eventbus.OutboxWorker`, her modül kendi `OutboxStore`'unu publish eder
2. **Modül başına sqlc.yaml** (kullanıcı tercihi) — her modül kendi schema/query/db klasörü
3. **PostgreSQL role enforcement: yok** — convention + PR review (plan default)
4. **Code migration** (sıfırdan yazım değil) — mevcut servis kalıbı 1:1 taşınıyor
5. **Migration tool: goose**, **ORM: sqlc + pgx/v5**, **HTTP: Gin**, **JWT: HS256 + Argon2id + Redis blacklist** (mevcut, değişmiyor)
6. **CC SemesterStatusHandler ServiceURLs**: hepsi `http://localhost:<port>` loopback — period distribution flow şu an HTTP üzerinden kendi monolithine atıyor; modüller migrate olunca refactor edilebilir
7. **CC `Module.Name()` "catalog"** — frontend `/api/catalog/*` çağırıyor, korunuyor (plan'daki `course_catalog` schema/package adıyla farklı)

---

## Dosya Konumları (hızlı referans)

- Mimari plan: [00-ai-rules.md](00-ai-rules.md), [01-overview.md](01-overview.md), [02-data.md](02-data.md), [03-events.md](03-events.md)
- Yapılan iş: [new-backend/monolith/](../new-backend/monolith/)
- Eski referans: [backend/services/](../backend/services/) ← DOKUNMA, sadece kopyalama kaynağı
- Schema-prefix script (CC için yazıldı, başka modüllerde de kullanılabilir): [/tmp/schema_prefix.py](/tmp/schema_prefix.py)
- Build/test: `cd new-backend && go build ./monolith/... && go test ./monolith/... ./shared/...`

---

## Hatırlatmalar (faz başında dikkat)

- [ ] `cd /home/nautilus/...` chdir bash'i kalıcı değiştirir — absolute path kullan
- [ ] sqlc enum value constant'ları auto-rename olmuyor — repo kodu sed ile düzelt
- [ ] CC'nin `OutboxRepository.GetFailedEventsForRetry` ismi farklı (staff/student `GetFailedEvents`) — adapter'da farklı method çağırılıyor
- [ ] CC'de `OutboxEvent.RetryCount` `pgtype.Int2` (nullable kolon, sqlc öyle üretti) — adapter'da `.Int16` field erişimi
- [ ] Auth henüz outbox kullanmıyor (orijinal auth-service'te de yoktu) — Faz 3 notification gelince eklenmeli (plan Bölüm 5.9: `user.registered`, `user.password_reset_requested`)
- [ ] Mevcut staff outbox migration UUID id (collapsed 00002) — student da aynı pattern
- [ ] CC migrationları as-is taşındı (13 migration, ALTER'lar dahil) — fresh DB'de fazlasıyla çalışır ama temiz değil
- [ ] sed kullanırken `\b` POSIX'te zayıf — Python regex (negative lookbehind `(?<![\.\w])`) daha güvenli, [/tmp/schema_prefix.py](/tmp/schema_prefix.py) referans
