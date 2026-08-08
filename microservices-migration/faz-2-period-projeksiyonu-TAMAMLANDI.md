# Faz 2 — `academic_periods` Cross-Service Okumasını Kaldır

**Ön koşul:** Faz 0 tamamlandı (Faz 1'den bağımsız, sırayla yapılması yeterli)
**Risk:** **Yüksek** — migrasyonun en kritik fazı
**Monolith durumu:** Faz sonunda **hâlâ çalışır** (tek DB içinde, schema'lar arası)

---

## Amaç

Bu, projedeki **tek gerçek servis sınırı ihlali**. enrollment, attendance ve
grades modülleri, catalog'un `course_catalog.academic_periods` tablosunu
**doğrudan** okuyor. Servisler ayrı DB'lere geçince bu fiziksel olarak
imkânsız hale gelecek.

Çözüm: catalog dönem tanımlarını event ile yayınlar, üç tüketici kendi
schema'sındaki lokal `academic_periods` kopyasında tutar.

---

## Mevcut Durumun Tespiti (doğrulanmış)

Kodu okuduğumuzda çıkan tablo — bunu bilmek fazı çok kolaylaştırıyor:

1. **`platform/repository/period_repository.go` ölü kod.** `NewPeriodRepository`
   hiçbir yerden çağrılmıyor. Silinecek.

2. **`SimplePeriodRepository` tek örnek olarak yaratılıyor**
   ([course_catalog/module.go:62](new-backend/monolith/internal/modules/course_catalog/module.go#L62))
   ve `catalogModule.PeriodRepo()` ile enrollment, attendance, grades'e
   dağıtılıyor. Üçü de catalog'un tablosuna yazıyor/okuyor.

3. **`attendance.academic_periods` tablosu ZATEN VAR**
   (`attendance/sql/migrations/00001`) ve kolonları catalog'unkiyle birebir
   aynı: `id, semester, period_start, period_end, is_active, created_at,
   updated_at` + `semester` üzerinde unique index. Kullanılmıyor — bu faz için
   hazır bekliyor.

4. **HTTP fan-out'u şu an BOZUK.** `distributePeriods`
   ([semester_status_handler.go:227](new-backend/monolith/internal/modules/course_catalog/handler/semester_status_handler.go#L227))
   enrollment/grades/attendance'a `POST <url>/internal/periods` atıyor, ama
   **bu üç modülde böyle bir route yok** → 404 → hata `period_errors` olarak
   response'a ekleniyor, dönem yine de oluşuyor. Yani dönem satırları sadece
   catalog'un tablosuna düşüyor, diğer üçü de oradan okuduğu için sistem
   "çalışıyor" görünüyor.

Bu faz 4 numaralı bozuk push'u event ile değiştiriyor ve 2 numaralı ihlali
kaldırıyor.

---

## Hedef Tasarım

```
                    catalog-service
              ┌────────────────────────────┐
              │ course_catalog.academic_    │  ← 4 satır / dönem:
              │   periods (+ period_type)   │    catalog, enrollment,
              │        SOURCE OF TRUTH      │    grading, attendance
              └──────────┬─────────────────┘
                         │ outbox → course_catalog.events
       ┌─────────────────┼─────────────────┐
       │ routing key:    │                 │
       │ ...period.      │ ...period.      │ ...period.
       │ enrollment.*    │ grading.*       │ attendance.*
       ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ enrollment.  │  │ grades.      │  │ attendance.  │
│ academic_    │  │ academic_    │  │ academic_    │
│ periods      │  │ periods      │  │ periods      │
│ (1 satır/dönem)│ (1 satır/dönem)│ │ (ZATEN VAR)  │
└──────────────┘  └──────────────┘  └──────────────┘
```

Catalog kendi period'unu lokal yazar (event yok). Diğer üçü için event yayınlar.
Her tüketici sadece kendi routing key'ini bind eder — filtreleme RabbitMQ'da.

---

## Adımlar

### 1. Ölü kodu sil

```bash
rm new-backend/shared/platform/repository/period_repository.go
```

Kullanan yok, `go build ./...` ile doğrula.

### 2. Catalog tablosuna `period_type` ekle

Yeni migration: `course_catalog/sql/migrations/000NN_add_period_type.sql`
(NN = mevcut en yüksek numara + 1 — `ls` ile kontrol et, 00013 vardı).

```sql
-- +goose Up
ALTER TABLE course_catalog.academic_periods
  ADD COLUMN period_type VARCHAR(20) NOT NULL DEFAULT 'catalog';

-- Dönem başına tek satır kısıtı artık tip bazında.
DROP INDEX IF EXISTS course_catalog.idx_academic_periods_unique_semester;
CREATE UNIQUE INDEX idx_academic_periods_unique_semester_type
  ON course_catalog.academic_periods(semester, period_type);

ALTER TABLE course_catalog.academic_periods
  ADD CONSTRAINT chk_period_type
  CHECK (period_type IN ('catalog','enrollment','grading','attendance'));

-- +goose Down
ALTER TABLE course_catalog.academic_periods DROP CONSTRAINT IF EXISTS chk_period_type;
DROP INDEX IF EXISTS course_catalog.idx_academic_periods_unique_semester_type;
CREATE UNIQUE INDEX idx_academic_periods_unique_semester
  ON course_catalog.academic_periods(semester);
ALTER TABLE course_catalog.academic_periods DROP COLUMN period_type;
```

`DEFAULT 'catalog'` mevcut satırları bozmaz. Sonra `make sqlc-course_catalog`.

**Migration'ı çalıştırma, kullanıcıya sor** (CLAUDE.md §6).

### 3. enrollment ve grades'e lokal tablo ekle

`enrollment/sql/migrations/000NN_create_academic_periods.sql` ve
`grades/sql/migrations/000NN_create_academic_periods.sql`.
İkisi de `attendance.academic_periods`'ın birebir kopyası, sadece schema adı
değişik:

```sql
-- +goose Up
-- catalog'dan event ile senkronlanan lokal projeksiyon. Kaynak:
-- course_catalog.academic_periods (period_type='enrollment'). Buraya
-- servis DIŞINDAN yazılmaz — sadece event consumer yazar.
CREATE TABLE enrollment.academic_periods (
    id UUID PRIMARY KEY,
    semester VARCHAR(50) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_enrollment_academic_periods_semester
  ON enrollment.academic_periods(semester);

-- +goose Down
DROP TABLE IF EXISTS enrollment.academic_periods;
```

**Dikkat:** `id` burada `DEFAULT gen_random_uuid()` **değil** — catalog'daki
id'yi aynen taşıyoruz ki idempotent upsert yapılabilsin.

`attendance.academic_periods` zaten var ama `id` kolonu
`DEFAULT gen_random_uuid()`. Sorun değil (biz her zaman explicit id yazacağız),
dokunmaya gerek yok.

### 4. `SimplePeriodRepository`'yi schema-parametrik yap

`shared/platform/repository/simple_period_repository.go`:

```go
// Her servis kendi schema'sındaki academic_periods tablosuna bakar. Schema
// adı config'ten gelen sabit bir değer, kullanıcı girdisi değil — yine de
// SQL'e gömülmeden önce whitelist'ten geçirilir.
var allowedPeriodSchemas = map[string]bool{
    "course_catalog": true, "enrollment": true,
    "attendance": true, "grades": true,
}

func NewSimplePeriodRepository(pool *pgxpool.Pool, schema string) *SimplePeriodRepository {
    if !allowedPeriodSchemas[schema] {
        panic("invalid period schema: " + schema)
    }
    return &SimplePeriodRepository{pool: pool, table: schema + ".academic_periods"}
}
```

Tüm sorgulardaki `course_catalog.academic_periods` sabitini `r.table` ile
değiştir (`fmt.Sprintf`). Metot imzaları **değişmiyor** — çağıran kod
etkilenmiyor.

Yeni metot ekle (consumer'ların upsert'ü için):

```go
// UpsertPeriod — event consumer'ı için idempotent yazma. Aynı event iki kez
// gelirse ikincisi no-op olur.
func (r *SimplePeriodRepository) UpsertPeriod(ctx context.Context, p SimplePeriod) error
// ON CONFLICT (semester) DO UPDATE SET period_start=..., period_end=..., is_active=..., updated_at=NOW()
```

Catalog'un instance'ı için `period_type` destekli metotlar gerekecek
(`CreatePeriodTyped`, `GetPeriodsBySemesterAndType`, ...). Bunları
`allowedPeriodSchemas` içinde sadece `course_catalog` için anlamlı olacak
şekilde ekle.

### 5. Catalog: event yayınla

`distributePeriods` içindeki `createRemotePeriod` çağrısını sil. Yerine:
**4 satırın hepsini lokal yaz** (kendi `period_type`'ıyla) + 3 tanesi için
outbox'a event yaz — hepsi **aynı transaction'da**.

```go
// catalog + enrollment + grading + attendance: dördü de catalog'un tablosuna
// yazılır (source of truth). Üçü için outbox'a event düşer; catalog kendi
// period'unu zaten lokal okuyor, event'e gerek yok.
```

Event payload:

```json
{
  "id": "uuid", "semester": "2024-2025-guz", "period_type": "enrollment",
  "period_start": "2024-09-01T00:00:00Z", "period_end": "2024-09-15T23:59:59Z",
  "is_active": true
}
```

Routing key'ler (`course_catalog.events` exchange'i üzerinde):

| Olay | Routing key |
|---|---|
| Dönem oluşturuldu | `course_catalog.period.<type>.created` |
| Dönem güncellendi | `course_catalog.period.<type>.updated` |
| Dönem silindi | `course_catalog.period.<type>.deleted` |

`<type>` ∈ {`enrollment`, `grading`, `attendance`}.

Dönem silme/güncelleme yollarını da (`DeletePeriodBySemester`,
`UpdatePeriodBySemester` çağıran handler'lar) aynı şekilde event yayınlar hale
getir — yoksa projeksiyonlar bayatlar.

`createRemotePeriod` fonksiyonunu ve `ServiceURLs`'ün
`Enrollment/Grades/Attendance` alanlarını sil. `ServiceURLs.Meal` **kalıyor**
(closed-days fan-out'u ayrı iş, Faz 3'te ele alınacak).

### 6. Kuyruk binding'lerini ekle

`monolith/cmd/main.go` içindeki `downstreamBindings` listesine:

```go
// catalog dönem tanımlarını yayınlar; üç tüketici sadece kendi tipini bind
// eder — filtreleme broker'da yapılır, consumer'da değil.
{Queue: "enrollment.period_events", Exchange: "course_catalog.events", RoutingKey: "course_catalog.period.enrollment.*"},
{Queue: "grades.period_events",     Exchange: "course_catalog.events", RoutingKey: "course_catalog.period.grading.*"},
{Queue: "attendance.period_events", Exchange: "course_catalog.events", RoutingKey: "course_catalog.period.attendance.*"},
```

`DeclareDownstreamBindings` topic exchange kullanıyor, `*` wildcard çalışır.

### 7. Consumer'ları yaz

Üç modülün `worker/` dizinine `period_consumer.go`. Mevcut
`event_consumer.go`'ları şablon al — aynı idempotency deseni:

```
event al → processed_events'te event_id var mı? → yoksa:
  created/updated → UpsertPeriod
  deleted         → DeletePeriodBySemester
  → processed_events'e event_id yaz (aynı tx)
→ ack
```

enrollment ve grades'te `processed_events` tablosu var. attendance'ta da var.
Yeni tablo gerekmiyor.

Worker'ları modülün `Bootstrap()`'ında başlat (mevcut consumer'ların yanında).

### 8. Wiring'i değiştir

`cmd/main.go` — her modül artık **kendi** period repo'sunu alır:

```go
catalogModule := coursecatalog.New(cfg, pool, staffModule.StaffService())
// ÖNCE: enrollment.New(..., catalogModule.PeriodRepo())
// SONRA:
enrollmentPeriodRepo := platformRepo.NewSimplePeriodRepository(pool, "enrollment")
attendancePeriodRepo := platformRepo.NewSimplePeriodRepository(pool, "attendance")
gradesPeriodRepo     := platformRepo.NewSimplePeriodRepository(pool, "grades")
```

`catalogModule.PeriodRepo()` accessor'ını **sil** — artık kimse kullanmamalı.
Silince derleme hatası alacağın her yer, düzeltilmesi gereken bir sızıntıdır.

Catalog kendi repo'sunu `"course_catalog"` ile kurar.

### 9. Backfill endpoint'i

Mevcut dönemler için event hiç yayınlanmadı → tüketicilerin tabloları boş
kalır. Catalog'a ekle:

```
POST /internal/periods/republish   (X-Internal-Secret korumalı)
```

Tüm `academic_periods` satırlarını (catalog tipi hariç) outbox'a `*.updated`
olarak yeniden yazar. Idempotent — istediğin kadar çağırabilirsin.

**Soğuk projeksiyon uyarısı:** Tüketicinin tablosu boşken `platform/rules`
period kontrolü "graceful degradation" ile **fail-open** davranır — yani dönem
kilidi sessizce devre dışı kalır. Bu mevcut davranış, yeni bir açık değil, ama
Faz 7'de bunu doğrulaman gerekiyor: yeni kurulumda seed dönemi oluşturunca
event akışı kendiliğinden başlar; mevcut kurulumda republish'i **bir kez**
çağırman şart.

---

## Bitiş Kriteri

```bash
# 1. Ölü kod ve sızıntı kalmadı — üçü de boş dönmeli
grep -rn "NewPeriodRepository" --include="*.go" new-backend
grep -rn "PeriodRepo()" --include="*.go" new-backend/monolith
grep -rn "createRemotePeriod" --include="*.go" new-backend

# 2. course_catalog.academic_periods'a sadece catalog modülü erişiyor
grep -rn "course_catalog.academic_periods" --include="*.go" new-backend
#    → sadece shared/platform/repository/simple_period_repository.go'daki
#      whitelist satırında geçmeli, hardcoded SQL'de DEĞİL

# 3. Derleme + test
cd new-backend/shared && go build ./... && go test ./...
cd ../monolith && go build ./... && go test ./...
```

Fonksiyonel doğrulama (migration'lar uygulandıktan sonra, kullanıcı çalıştırır):

```bash
# Dönem oluştur (admin token ile), sonra üç tabloyu da kontrol et:
sudo docker exec mydreamcampus-postgres psql -U postgres -d mydreamcampus -c \
  "SELECT 'enrollment' src, semester, period_start FROM enrollment.academic_periods
   UNION ALL SELECT 'grades', semester, period_start FROM grades.academic_periods
   UNION ALL SELECT 'attendance', semester, period_start FROM attendance.academic_periods;"
```

Üç satır da dolmuşsa projeksiyon çalışıyor. Boşsa: outbox worker loglarına ve
RabbitMQ management UI'daki kuyruk derinliklerine bak.

---

## Commit

Bu faz iki commit'e bölünebilir:

```
feat(catalog): publish academic period events per consumer service
feat(shared): project academic periods into enrollment, attendance and grades
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-2-period-projeksiyonu-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 2 satırını `[x]` yap
3. "Sıradaki faz" satırını **3** yap

---

## Uygulama Notları (plandan sapmalar — Faz 4 bunları bilmek zorunda)

**1. Consumer'larda `processed_events` kullanılmadı.** Adım 7 idempotency için
`processed_events` tablosunu şart koşuyordu; üç consumer da yerine doğal
idempotency kullanıyor (`UpsertPeriod` semester üzerinde ON CONFLICT,
`DeletePeriodBySemester` semester üzerinde). Nedenleri:

- **grades'te `processed_events` tablosu yok** — plan yanlış varsayıyordu.
  Doğrulama: `monolith/internal/modules/grades/worker/event_consumer.go:35`
  bunu açıkça belirtiyor. Eklemek yeni migration + sqlc + repo demekti.
- Mevcut kod tabanında emsal var: enrollment'ın `EventConsumer`'ı da aynı
  gerekçeyle dedup yapmıyor (upsert doğal idempotent).
- Plan "aynı tx'te işaretle" diyordu; bu üç modülde bu tablolar için
  tx-aware repo yok, yani dedup zaten atomik olmayacaktı.

Faz 4'te consumer'lar servislere taşınırken bu karar aynen korunabilir.

**2. Catalog instance'ı `period_type='catalog'` ile scope'lanıyor.** Adım 4
sadece "typed metotlar ekle" diyordu, ama bir semester'da artık 4 satır var:
`GetActivePeriodBySemester` type filtresi olmadan rastgele bir servisin
dönemini döndürebilirdi. `SimplePeriodRepository`'nin catalog handle'ında
**tüm eski metotlar** `period_type='catalog'`'a scope'lanıyor. Sonuç: admin
period CRUD'u (`/api/catalog/admin/periods`) ve frontend'in gördüğü liste Faz
2 öncesiyle bire bir aynı — projeksiyon satırları o yüzeye sızmıyor. Bir
consumer'ın dönemini değiştirmenin tek yolu `PUT /admin/semesters/:id`
(event yayınlayan yol).

**3. `make sqlc-<modul>` tüm modüllerde kırık — Faz 2'den önce de kırıktı.**
`schema "<x>" does not exist`: schema'lar migration'larda değil
`infrastructure/postgres/init-databases.sh` içinde yaratılıyor, sqlc ise
sadece `sql/migrations` dizinini okuyor. Bu yüzden catalog'un generated
`db/` kodu `period_type` kolonunu görmüyor. Sorun değil: el yazımı kod
generated period sorgularından **sadece** `DeletePeriodsBySemester`'ı
kullanıyor (semester'ın tüm tiplerini siler — istenen davranış). Kullanılmayan
diğerlerine `sql/queries/periods.sql` başında uyarı düşüldü. Faz 4 servis
başına sqlc kuracağı için bu tooling kırığı orada çözülmeli.

**4. Fan-out artık transactional.** Adım 5'in istediği gibi 4 satır + 3 outbox
event tek transaction'da. `DeletePlannedSemester` de aynı tx'te 3 `deleted`
event yazıyor. `UpdatePlannedSemester` benzer şekilde 4 satırı güncelleyip 3
`updated` event yazıyor (plan bunu "aynı şekilde yap" diye geçiyordu).
