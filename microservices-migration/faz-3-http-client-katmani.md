# Faz 3 — `shared/client/` Internal REST Katmanı

**Ön koşul:** Faz 0 ve Faz 2 tamamlandı
**Risk:** Orta
**Monolith durumu:** Faz sonunda **hâlâ çalışır** — hem in-process hem HTTP modunda

---

## Amaç

Modüller arası 7 sync bağımlılığın in-process implementasyonlarının yanına
HTTP implementasyonlarını yazmak. Interface'ler zaten var; sadece ikinci bir
implementasyon ekliyoruz.

**Kilit avantaj:** Monolith bu fazın sonunda **kendi kendine HTTP atarak**
çalışabilir hale gelir (`INTERNAL_CLIENT_MODE=http`). Yani servisleri
bölmeden önce client katmanını gerçek trafikle test edebiliyoruz. Faz 4'e
girerken client'ların çalıştığını biliyor oluyoruz.

---

## Kapsam

| # | Interface | Nerede tanımlı | Yeni implementasyon |
|---|---|---|---|
| 1 | `StaffClient` | `course_catalog/service/staff_client.go` | `HTTPStaffClient` |
| 2 | `StudentClient` | `enrollment/service/clients.go` | `HTTPStudentClient` |
| 3 | `CourseCatalogClient` | `enrollment/service/clients.go` | `HTTPCourseCatalogClient` |
| 4 | `SemesterClient` | `attendance/service/semester_client.go` | `HTTPSemesterClient` |
| 5 | `SemesterClient` | `grades/service/semester_client.go` | `HTTPSemesterClient` |
| 6 | `PaymentClient` | `meal/service/payment_client.go` | `HTTPPaymentClient` |
| 7 | **interface yok** | student → staff | **önce interface çıkar**, sonra HTTP impl |
| 8 | `audit.Logger` | `platform/audit` | `EventAuditLogger` (HTTP değil, **event**) |

**7 numaraya dikkat:** `student.New(pool, rabbitConn, staffModule.StaffService())`
— student modülü staff'ın **somut tipini** alıyor, interface yok. Önce
`student/service/staff_client.go` içinde interface çıkarılmalı (catalog'un
`StaffClient`'ı şablon).

---

## Adımlar

### 1. Ortak HTTP client altyapısı

Yeni dosya: `shared/client/base.go`

```go
// Servisler arası çağrılar için ortak taşıyıcı. X-Internal-Secret'ı her
// istekte ekler; doğrulama karşı tarafta platform/middleware.InternalAuth
// ile yapılır.
type Base struct {
    baseURL string
    secret  string
    http    *http.Client
}

func NewBase(baseURL, secret string) *Base {
    return &Base{baseURL: baseURL, secret: secret,
        http: &http.Client{Timeout: 10 * time.Second}}
}

func (b *Base) Get(ctx context.Context, path string, out any) error
func (b *Base) Post(ctx context.Context, path string, in, out any) error
```

Kurallar:
- Timeout **zorunlu** — timeout'suz client bir servisin yavaşlamasını tüm
  sisteme yayar
- Retry **yok** — çağrılar zaten request path'inde, kullanıcı hatayı görsün
- 404 → `ErrNotFound` sentinel'i döner; çağıran taraf modül hatasına map'ler
- 5xx → hata olarak döner, fail-open yapma
- **`X-Request-ID` ileri taşınır** — aşağıya bak

#### `X-Request-ID` propagasyonu

```go
// Aynı istek zincirinin farklı servislerdeki log satırlarını tek bir ID ile
// birleştirir. Gelen yön middleware'de kurulu; burası olmazsa zincir servis
// sınırında kopar.
if rid := logger.GetRequestID(ctx); rid != "" {
    req.Header.Set("X-Request-ID", rid)
}
```

Bu, uçtan uca izlenebilirlik zincirinin **3. halkası**. Zincirin tamamı ve
diğer halkalar: `03-IZLENEBILIRLIK.md`.

#### Circuit breaker

`Base`, hedef servis başına bir breaker taşır. Kütüphane **onaylandı**:

```bash
cd new-backend/shared && go get github.com/sony/gobreaker/v2
```

Sürüm sabitlenmedi, `go get` çözsün. v2 generic API kullanıyor
(`NewCircuitBreaker[*http.Response]`), v1 kullanmıyor — hangisi geldiyse ona
göre yaz. `shared/go.mod`'a giriyor, servisler `shared` üzerinden alıyor.
`go mod tidy` sonrası `go.sum` tek satır büyümeli; daha fazlaysa yanlış paket.

Ayarlar, eşikler ve **"neyin hata sayıldığı"** kuralı: `05-DAYANIKLILIK.md`
Bölüm B.

En kritik detay oradan: **404 ve diğer 4xx hata sayılmaz.** "Öğrenci
bulunamadı" geçerli bir iş cevabıdır; hata sayılırsa normal kullanımda breaker
açılır ve çalışan servisi ölü ilan eder. Sadece `err != nil` ve `5xx` hatadır.

#### Güvenlik

`/internal/*` route'ları bu fazda doğuyor — `02-GUVENLIK.md` A01 bölümündeki
kuralları uygula: her route `InternalAuth` taşır, secret boşsa servis başlamaz.

Breaker'ın güvenlikle etkileşimi var: açıkken `SemesterInfo` alınamıyorsa
"hard deadline yok" **varsayma, isteği reddet** (`05-DAYANIKLILIK.md` Bölüm B,
"Güvenlikle etkileşim").

`shared/platform/middleware/internal.go` (Faz 0'da taşındı) karşı taraf
doğrulaması için hazır — yeniden yazma.

### 2. Sağlayıcı servislerde `/internal/*` route'ları

Kontratlar `01-REFERANS-MIMARI.md` §2'de. Her sağlayan modülün `module.go`
`RegisterRoutes`'una `InternalAuth` middleware'i ile korunmuş bir grup ekle:

```go
internal := rg.Group("/internal", platformMiddleware.InternalAuth(secret))
internal.GET("/staff/:id", h.GetStaffInternal)
```

**Önemli:** Bu route'lar modülün `/api/<name>` grubunun altına düşüyor
(`/api/staff/internal/staff/:id`). Caddy `/api/*`'ı dışarı proxy'lediği için
bunlar **dışarıdan erişilebilir** olur — `InternalAuth` bu yüzden zorunlu.

Faz 4'te servisler ayrılınca bu grup `/api/*` dışına, kök altına
(`/internal/*`) taşınacak ve Caddy hiç proxy'lemeyecek. Şimdilik `/api` altında
kalması normal.

Mevcut örnek: meal'in `/internal/closed-days/batch` handler'ı aynen bu desende
([meal/handler/closed_days_handler.go:245](new-backend/monolith/internal/modules/meal/handler/closed_days_handler.go#L245)) — şablon olarak kullan.

### 3. HTTP client implementasyonlarını yaz

Her biri karşılık geldiği in-process adapter'ın **hata map'leme davranışını
birebir kopyalamalı**. Örnek — `InProcessStaffClient.GetInstructor`:

- staff bulunamadı → `catalogErrors.ErrInstructorNotFound`
- `resp.Status != "active"` → `catalogErrors.ErrInstructorNotActive`
- `resp.Department != department` → `catalogErrors.ErrInstructorNotInDepartment`

HTTP versiyonu 404 aldığında da `ErrInstructorNotFound` dönmeli. Bu map'leme
kaybolursa handler'lar yanlış status code üretir.

Her dosyanın sonuna compile-time assertion koy (mevcut desen):

```go
var _ StaffClient = (*HTTPStaffClient)(nil)
```

**Nerede duracaklar:** Client'lar tükettikleri modülün DTO'larına bağımlı
(`catalogDTO.SemesterCourseListItem` vb.). Bu yüzden `shared/client/` yerine
**tüketen modülün `service/` dizininde** dursunlar — in-process adapter'ların
yanında. `shared/client/` sadece `base.go` ve DTO'suz ortak parçaları tutar.

Faz 4'te servisler ayrılınca paylaşılan DTO'lar `shared/`'a taşınacak; şimdi
o refactor'a girme.

### 4. Audit'i event'e çevir

`grades` ve `meal`, catalog'un `AuditRepo`'suna in-process yazıyor
(`catalogService.NewDirectAuditLogger`). Bu bir **side-effect**, sync olmasına
gerek yok.

- `shared/platform/audit` içine `EventAuditLogger` ekle: `audit.AuditEvent`'i
  çağıran modülün **kendi outbox'ına** `audit.entry.created` routing key'iyle
  yazar.
- catalog'a consumer ekle: `catalog.audit_events` kuyruğu →
  `course_catalog.audit_log` tablosuna yazar (idempotent, `event_id`).
- Binding'leri `cmd/main.go`'ya ekle:

```go
{Queue: "catalog.audit_events", Exchange: "grades.events", RoutingKey: "audit.entry.created"},
{Queue: "catalog.audit_events", Exchange: "meal.events",   RoutingKey: "audit.entry.created"},
```

- `catalogModule.AuditRepo()` accessor'ını **sil**. `DirectAuditLogger` sadece
  catalog'un kendi içinde kalır.

Audit yazımı artık asenkron — `audit.Logger` interface'i hata dönmeye devam
etsin ama outbox yazımı iş transaction'ının içinde olduğu için kayıp yok.

### 5. Catalog → meal closed-days fan-out'unu client'a çek

`semester_status_handler.go` içindeki `distributeClosedDays`, meal'e ham
`http.Client` ile POST atıyor. Bunu `shared/client/base.go` üzerinden geçen
bir `MealClient`'a taşı. Davranış aynı, sadece ortak taşıyıcıyı kullanıyor
(timeout, secret, hata map'leme tek yerden).

`ServiceURLs` struct'ı bu fazın sonunda sadece `Meal` alanını taşıyor olmalı
(diğerleri Faz 2'de silindi).

### 6. Mod anahtarı ve wiring

`monolith/config/config.go`'ya ekle:

```go
// inprocess = modüller birbirini doğrudan çağırır (monolith varsayılanı)
// http      = modüller birbirine HTTP atar; monolith kendi kendine çağrı
//             yapar. Faz 4'e geçmeden client katmanını gerçek trafikle
//             sınamak için.
InternalClientMode string  // INTERNAL_CLIENT_MODE, default "inprocess"
ServiceURLs map[string]string
```

`cmd/main.go`'da seçim:

```go
if cfg.InternalClientMode == "http" {
    enrollmentStudentClient = enrollmentService.NewHTTPStudentClient(...)
} else {
    enrollmentStudentClient = enrollmentService.NewInProcessStudentClient(studentModule.StudentService())
}
```

`http` modunda tüm servis URL'leri monolith'in kendisini gösterir
(`http://localhost:8080`).

### 7. Testler

Her HTTP client için `httptest.NewServer` ile birim testi:
- Başarılı yanıt → doğru DTO
- 404 → doğru sentinel hata
- 500 → hata döner, panic yok
- `X-Internal-Secret` header'ı gönderiliyor

Test isimlendirme: `TestHTTPStaffClient_GetInstructor_NotFoundMapsToSentinel`
(CLAUDE.md §6).

### 8. `new-backend/skills.md`'deki çelişkiyi kapat

`skills.md` §1 hâlâ şunu diyor:

> **Moduller arasi cagri**: In-process client interface — modul modulu HTTP ile CAGIRMAZ.

Bu satır artık yanlış ve `skills.md` her backend görevinde zorunlu okuma
(CLAUDE.md §2) — düzeltilmezse sonraki oturumlar planla çelişen talimat okur.

O maddeyi şununla değiştir:

```markdown
- **Moduller arasi cagri**: Sync okuma/dogrulama icin **internal REST**
  (`shared/client`, `X-Internal-Secret`). In-process client adapter'lari
  mikroservis migrasyonunda kaldiriliyor — bkz.
  `microservices-migration/00-BASLANGIC.md`. Side-effect/notify icin
  RabbitMQ event (degismedi).
```

Dosyanın tamamının yeniden yazımı Faz 8'de. Burada **sadece bu satır**.

### 9. Outbox'a `correlation_id` ekle (izlenebilirlik halkaları 4-5)

Tam tarif: `03-IZLENEBILIRLIK.md` [4] ve [5].

Özet:
- 8 modülün `outbox_events` tablosuna nullable `correlation_id UUID` kolonu +
  kısmi index (migration yaz, çalıştırmayı kullanıcıya sor)
- `make sqlc-<module>` → outbox insert query'sine kolonu ekle
- Servis katmanı outbox'a yazarken `logger.GetRequestID(ctx)` değerini geçir
- Event envelope'una `CorrelationID string \`json:"correlation_id,omitempty"\`` ekle

**Neden Faz 4'te değil, burada:** modüller hâlâ monolith'te, migration'lar tek
Makefile'dan (`make migrate-create-<module>`) yazılıyor. Faz 4'ten sonra aynı iş
8 ayrı serviste tekrarlanır.

**Neden 4. adımla birlikte yapılmalı:** audit event'i de outbox'tan geçiyor;
ikisi aynı sqlc regenerate turunda halledilir.

### 10. HTTP idempotency middleware'i

Tam tarif: `05-DAYANIKLILIK.md` Bölüm A.

Yeni dosya: `shared/platform/middleware/idempotency.go`. `Idempotency-Key`
header'ı + Redis (`idem:<servis>:<user_id>:<key>`, TTL 24 saat).

Bu fazda **sadece middleware yazılır**; route'lara bağlanması Faz 4'te (her
servisin `module.go`'sunda, işaretli endpoint'lerde).

**Client tarafına dokunulmuyor.** Middleware header yoksa isteği aynen
geçiriyor, yani "Frontend / Mobil hiç değişmiyor" kararı korunuyor. Client'ın
header göndermesi ayrı bir iş kalemi.

Redis erişilemezse **fail-open** — rate limit'in aksine. Idempotency bir
güvenlik kontrolü değil; Redis düştü diye rezervasyon almayı durdurmak daha
kötü.

---

## Bitiş Kriteri

```bash
# 1. Derleme + testler
cd new-backend/shared && go build ./... && go test ./...
cd ../monolith && go build ./... && go test ./...

# 2. Tüm interface'lerin iki implementasyonu var
grep -rn "var _ StaffClient\|var _ StudentClient\|var _ CourseCatalogClient\|var _ SemesterClient\|var _ PaymentClient" \
  --include="*.go" new-backend/monolith
#    → her interface için en az 2 satır (InProcess + HTTP)

# 3. Audit artık in-process değil
grep -rn "AuditRepo()" --include="*.go" new-backend/monolith   # boş dönmeli
```

**Asıl doğrulama — HTTP modunda golden path** (kullanıcı çalıştırır):

```bash
# Monolith'i http modunda başlat
INTERNAL_CLIENT_MODE=http make backend

# Sonra tarayıcıda / curl ile:
#  - admin login
#  - ders açma (catalog → staff HTTP çağrısı)
#  - öğrenci ders seçimi (enrollment → student + catalog HTTP çağrıları)
#  - yemek rezervasyonu (meal → payment HTTP çağrısı)
```

Üçü de `inprocess` modundaki davranışla aynı sonucu vermeli. Fark varsa
client'ın hata map'lemesi eksiktir.

---

## Commit

```
feat(shared): add internal REST client transport with secret propagation
feat(shared): add HTTP implementations for cross-module service clients
refactor(shared): publish audit entries as events instead of in-process writes
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-3-http-client-katmani-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 3 satırını `[x]` yap
3. "Sıradaki faz" satırını **4** yap
