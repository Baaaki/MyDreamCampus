# MyDreamCampus — System Design

> Bu dokuman `new-backend/` mimarisini anlatir. Eski mikroservis kod tabani `legacy-codebase/` altindadir ve artik gelistirilmemektedir.

---

## 1. Genel Bakis

MyDreamCampus, universite yonetim sistemi. Mimari: **modular monolith** + tek bagimsiz mikroservis (notification). Tum stack **docker compose** ile containerize; tek public giris noktasi **Caddy** (80/443).

- **Monolith** (`new-backend/monolith/`): 9 is modulu tek Go process'inde calisir (container ici :8080). Her modul kendi handler/service/repository/db katmanina ve kendi Postgres **schema**'sina sahiptir.
- **Notification service** (`new-backend/services/notification/`): Ayri container, ayri Postgres instance. RabbitMQ'dan event tuketir, HTML template'li e-posta (SMTP/MailHog) ve push (FCM stub) gonderir.
- **Caddy edge** (`frontend/Dockerfile` + `Caddyfile`): SPA build'ini servis eder, `/api/*` ve `/health`'i monolith'e proxy'ler. Ayni origin oldugu icin CORS/absolute URL derdi yok; `SITE_ADDRESS` icin otomatik Let's Encrypt sertifikasi ceker (domain'siz HTTPS icin sslip.io — bkz. `DEPLOY.md`).
- **Frontend**: React + Vite (dev: 3000, `/api` proxy → 8080). Prod'da Caddy image'inin icinde build edilir (`FRONTEND_STATIC_ENABLED=false`, monolith SPA servis etmez).
- **Mobile**: React Native (Expo), ayni `/api/*` endpointlerini kullanir.
- **One-shot container'lar**: `migrate` (tum modullerin goose migration'larini uygular, monolith bunu bekler) ve `seed` (admin API uzerinden idempotent demo verisi: ogretmen/ders/ogrenci/kayit/not/yoklama/yemek; `SEED_DEMO=false` ile kapatilir).

```
                            Internet (80/443)
                                   │
                        ┌──────────▼──────────┐
                        │   CADDY (edge)      │  SPA static + TLS
                        │  /api/* , /health ──┼────────┐
                        └─────────────────────┘        │
      Web (react-router SPA)  /  Mobile (Expo) ────────┤
                                                       ▼
                    ┌─────────────────────────────────────┐
                    │        MONOLITH (Go/Gin, :8080)     │
                    │                                     │
                    │  auth │ staff │ student │ catalog   │
                    │  enrollment │ attendance │ grades   │
                    │  meal │ payment(mock)               │
                    │                                     │
                    │  platform: middleware, redis,       │
                    │  rabbitmq, rules, audit, outbox     │
                    └──────┬──────────┬──────────┬────────┘
                           │          │          │
              ┌────────────┘          │          └───────────┐
              ▼                       ▼                      ▼
      ┌──────────────┐        ┌──────────────┐       ┌──────────────┐
      │ PostgreSQL   │        │   RabbitMQ   │       │    Redis     │
      │ :5432        │        │ :5672/:15672 │       │    :6379     │
      │ (schema per  │        │ (topic       │       │ (blacklist,  │
      │  module)     │        │  exchanges)  │       │  rate limit, │
      └──────────────┘        └──────┬───────┘       │  yoklama buf)│
              ▲                      │               └──────────────┘
   migrate ───┘ (one-shot)           ▼
   seed ──► monolith API   ┌────────────────────────┐
                           │ NOTIFICATION SERVICE   │──► MailHog :1025/:8025
                           │ (ayri Postgres :5433)  │──► FCM (stub)
                           └────────────────────────┘
```

---

## 2. HTTP Katmani

`monolith/internal/http/server.go` — her modul `Module` interface'ini (`Name()`, `RegisterRoutes()`) implemente eder ve `/api/<modul-adi>` altina mount edilir.

**Global middleware zinciri (sirali):**
1. `Recovery` — panic yakalama
2. `SecurityHeaders` — CSP, X-Frame-Options vb.
3. `CORS`
4. `BodySizeLimit`
5. `RequestLogger` (Zap)
6. `IPRateLimit` (Redis tabanli)
7. `SetCSRFToken`

**Route seviyesi:** `JWTAuth` → `CSRFProtection` → `UserRateLimit` → `RequireRole(...)`. Her modul JWT'yi platform middleware'i ile kendi route'larinda dogrular; auth modulune RPC yapilmaz (JWT HS256, ortak secret).

**Health:** `/health` (liveness), `/ready` (DB + RabbitMQ + Redis ping). Caddy `/health`'i de proxy'ler.

**Moduller arasi sync iletisim:** Kural olarak **in-process client interface** (bkz. §4). HTTP loopback + `X-Internal-Secret` yalnizca mikroservis doneminden kalan birkac akista durur (catalog'un donem yasam dongusu fan-out'u: period dagitimi, meal kapali gun batch'i, audit-log yazimi) — bilinen sadelestirme adayi, bkz. §11.

---

## 3. Moduller ve Sorumluluklar

| Modul | Sorumluluk | Onemli akislar |
|---|---|---|
| **auth** | Login/logout/refresh, sifre degistirme/sifirlama, session yonetimi, admin seed | Argon2id, timing-safe dummy verify, hesap kilitleme (5 deneme → 30 dk), token rotation, Redis blacklist + token_version |
| **staff** | Personel CRUD, ogretim uyesi sorgulari, teacher profilleri | `staff.created/updated/deactivated` event'leri → auth user projection |
| **student** | Ogrenci CRUD, danisman atama (tekil + bulk), CSV import | `student.*` event'leri → auth; `staff.deactivated` tuketir (danisman dusurme) |
| **course_catalog** | Ders katalogu, donem (semester) yasam dongusu, donem dersi acma, ders programi, kayit/not/yoklama period tanimlari, audit log deposu | Donem durumu: `planned → active → ...` — aktiflesince ders yapisi donar; instructor cakisma + tek aktif donem DB kisitlari |
| **enrollment** | Ders secimi (program taslagi), danisman onay/red, kontenjan | Kapasite kontrolu advisory lock + tek transaction; pending program auto-replace; siki period kilidi (admin dahil kimse period disinda degisemez) |
| **attendance** | QR ile yoklama, manuel yoklama, oturum yonetimi, devamsizlik finalize | QR: HMAC imzali statik payload; taramalar Redis buffer'a SADD ile dedup edilerek yazilir, BufferFlusher 5 sn'de bir toplu DB flush eder; esikler acilan oturum sayisina oranlanir (10/14, 11/14) |
| **grades** | Not girisi (tekil/bulk), assessment kilitleme, otomatik finalize (bagil/mutlak), transkript, itiraz | Bagil sistem: z-score, sinif istatistikleri donduruluyor (itirazda frozen stats ile yeniden hesap); finalize event self-loop ile request path disinda |
| **meal** | Yemekhane, aylik menu, rezervasyon (tekil/batch), QR ile kullanim, iptal/iade | Rezervasyon `pending` yazilir → mock payment `payment.completed` publish eder → consumer `confirmed`'a cevirir; suresi dolan pending'leri worker expire eder |
| **payment** | Mock odeme saglayici (in-process, stateless) | `InitiatePayment` 2 sn gecikmeli `payment.completed` yayinlar; refund her zaman basarili |

**Notification service:** `auth.events` exchange'inden `user.registered` (hosgeldin maili) ve `user.password_reset_requested` (sifre sifirlama maili) tuketir. HTML template'ler (`templates/welcome.html`, `password_reset.html`) image'a gomulu. Teslimatlar kendi DB'sine loglanir.

---

## 4. Modul Ic Mimarisi (standart sablon + modul basina tasarim)

### 4.1 Standart modul sablonu

Her modul ayni katmanli yapiyi izler — yeni modul de bu sablona uyar:

```
modules/<modul>/
├── module.go        ← composition root: bagimliliklari kurar, RegisterRoutes, worker'lari baslatir
├── handler/         ← Gin handler: bind + validate + rol kontrolu. Is kurali YOK.
├── service/         ← is kurallari, transaction sinirlari, outbox'a event yazimi
├── repository/      ← sqlc cagrilarini saran veri erisim katmani
├── db/              ← sqlc GENERATED — elle dokunulmaz (make sqlc-<modul>)
├── dto/             ← request/response + event payload tipleri
├── errors/          ← modul AppError tanimlari
├── worker/          ← (varsa) RabbitMQ consumer'lar + arka plan isleri
├── sql/migrations/  ← goose (make migrate-up-<modul>)
├── sql/queries/     ← sqlc kaynak SQL
└── sqlc.yaml
```

**Veri akislari:**

```
HTTP istegi : Caddy → Gin middleware → handler → service → repository → db(sqlc) → Postgres(<modul> schema)
Event cikisi: service (is verisi + outbox kaydi AYNI tx) → OutboxWorker (modul basina goroutine) → RabbitMQ <modul>.events
Event girisi: worker/event_consumer → processed_events idempotency → repository → lokal view/cache tablosu
```

**Sinir kurallari (her modul icin gecerli):**

1. Modul, baska modulun schema'sina/tablosuna **dokunmaz**. Ihtiyac:
   - **Sync okuma/validasyon** → in-process client interface (`service.XxxClient`); somut implementasyon `main.go`'da diger modulun service'i ile baglanir. HTTP yok, secret yok.
   - **Side-effect/bildirim** → event (outbox → RabbitMQ) + tuketen tarafta lokal `*_view` tablosu.
2. Her event publish **outbox uzerinden** (istisna: payment — DB'si yok, dogrudan publish).
3. Her consumer **idempotent** (`processed_events` tablosu, `event_id` anahtar).
4. JWT dogrulama platform middleware'de; auth'a RPC yapilmaz.
5. Zaman kilitleri `platform/rules` + catalog'un `PeriodRepo`'su uzerinden (bkz. §8).

### 4.2 Modul basina tasarim

#### auth
- **Veri:** `auth.users`, `auth.sessions`, `auth.processed_events`, `auth.outbox_events`
- **Sync bagimlilik:** yok — kimse auth'a sync gitmez, auth kimseye gitmez
- **Publish:** `user.registered`, `user.password_reset_requested` → `auth.events`
- **Consume:** `staff.*` + `student.*` → user projection (personel/ogrenci olusunca login hesabi acilir/kapanir)
- **Worker:** event consumer + outbox worker
- **Redis:** access blacklist (JTI), refresh store, token_version, password reset token (1 saat TTL)
- **Tasarim notlari:** Argon2id; email enumeration'a karsi timing-safe dummy verify; 5 basarisiz giris → 30 dk kilit; refresh rotation; ilk admin bootstrap'ta sabit UUID ile seed. Login/refresh/password rate limit **fail-closed**.

#### staff
- **Veri:** `staff.staff`, `staff.teacher_profiles`, `staff.outbox_events`
- **Sync bagimlilik:** yok. **In-process sunar:** `StaffService` → student (danisman dogrulama) ve catalog (instructor dogrulama)
- **Publish:** `staff.created/updated/deactivated`
- **Consume:** yok — en yalin modul, event consumer'i yok
- **Tasarim notlari:** salt CRUD + sorgu; is kurali agirligi dusuk, diger modullerin referans verisi kaynagi.

#### student
- **Veri:** `student.students`, `student.import_jobs`, `student.processed_events`, `student.outbox_events`
- **Sync bagimlilik:** staff (`StaffService`, danisman gecerliligi). **In-process sunar:** `StudentService` → enrollment
- **Publish:** `student.created/updated/deactivated`
- **Consume:** `staff.deactivated` → danismani dusur
- **Worker:** event consumer + outbox worker; CSV import `import_jobs` uzerinden async islenir
- **Tasarim notlari:** bulk danisman atama ve CSV import idempotent tasarlanmali (tekrar calistirma guvenli).

#### course_catalog
- **Veri:** `course_catalog.course_catalog`, `semester_courses`, `course_schedule_sessions`, `academic_periods`, `semesters`, `audit_log`, `outbox_events`
- **DB kisitlari:** tek aktif donem (partial unique), instructor program cakismasi (constraint) — is kurali DB seviyesinde de korunur
- **Sync bagimlilik:** staff (instructor dogrulama). **In-process sunar:** `SemesterService` (enrollment/attendance/grades donem kontrolu), `PeriodRepo` (tum period kontrolleri), `AuditRepo` + `DirectAuditLogger` (grades/meal audit yazimi)
- **Publish:** `course.semester.created`
- **Consume:** yok
- **Tasarim notlari:** Monolith'in "referans veri + zaman otoritesi". Donem aktiflesince ders yapisi donar. Legacy: donem yasam dongusu handler'inda HTTP loopback fan-out (`/internal/periods`, meal `closed-days`, `X-Internal-Secret`) hala durur — hedef tamamen in-process'e cekmek.

#### enrollment
- **Veri:** `enrollment.enrollment_programs`, `enrollment_program_courses`, `enrollment_rejection_logs`, `processed_events`, `outbox_events`
- **Sync bagimlilik:** `StudentClient` (ogrenci dogrulama), `CourseCatalogClient` (donem/ders dogrulama) — ikisi de in-process adapter
- **Publish:** `enrollment.program.approved` / `enrollment.program.rejected`
- **Consume:** yok (su an; prerequisite event'i icin bkz. §11)
- **Tasarim notlari:** kontenjan = advisory lock + tek transaction (yaris kosulu yok); pending program auto-replace; **siki period kilidi** — admin dahi period disinda ogrenci programina dokunamaz (rules motorundaki tek istisna).

#### attendance
- **Veri:** `attendance.attendance_sessions`, `attendance_records`, `academic_periods` (lokal kopya), `students_view`, `courses_view`, `enrollments_view`, `processed_events`, `outbox_events`
- **Sync bagimlilik:** catalog (`SemesterService`, `PeriodRepo`)
- **Publish:** `attendance.semester.failed` (devamsizliktan kalma)
- **Consume:** `student.*`, `course.semester.created`, `enrollment.program.approved` → view tablolari sync
- **Worker:** `EventConsumer` (cache sync), `BufferFlusher` (Redis buffer → DB, 5 sn), `SessionExpiry` (acik oturumlari kapatir) + outbox worker
- **Redis:** QR tarama buffer (`SADD` atomik dedup — cift okutma tek sayilir), session metadata + enrolled ogrenci seti cache
- **Tasarim notlari:** yazma yolu yuksek hacimli (sinifin tamami ayni dakikada QR okutur) → once Redis, sonra toplu DB flush. Devamsizlik esigi acilan oturum sayisina oranlanir; QR payload HMAC-SHA256 imzali.

#### grades
- **Veri:** `grades.student_course_registrations`, `student_assessment_scores`, `student_completed_courses`, `students_view`, `courses_view`, `prerequisite_courses_view`, `outbox_events`
- **Sync bagimlilik:** catalog (`PeriodRepo`, `SemesterService`, `DirectAuditLogger`)
- **Publish:** `grade.submitted`, `grade.finalized`, `grade.appeal_processed`, `grade.student.prerequisite.passed`, `grade.finalize.requested` (self-loop)
- **Consume:** `student.*`, `course.semester.created`, `enrollment.program.approved`, `attendance.semester.failed` (devamsizlik → FF isaretleme) + kendi `grade.finalize.requested`'i
- **Worker:** `EventConsumer` (sync), `FinalizeConsumer` (self-loop: finalize istegi event olarak kuyruklanir, agir hesap request path disinda calisir) + outbox worker
- **Tasarim notlari:** bagil (z-score) / mutlak finalize; sinif istatistikleri finalize aninda dondurulur, itiraz yeniden hesabi frozen stats ile yapilir (itiraz digerlerinin harfini degistirmez).

#### meal
- **Veri:** `meal.cafeterias`, `monthly_menus`, `reservations`, `closed_days`, `students_view`, `processed_events`, `outbox_events`
- **Sync bagimlilik:** payment (`PaymentAdapter` → in-process `PaymentService`), catalog (`DirectAuditLogger`)
- **Publish:** `meal.reservation.created` / `meal.reservation.cancelled`
- **Consume:** `payment.completed` (rezervasyon confirm), `payment.failed` (expire), `student.*` (view sync)
- **Worker:** `PaymentConsumer`, `StudentConsumer`, `ReservationWorker` (suresi dolan pending'leri expire eder) + outbox worker
- **Tasarim notlari:** rezervasyon bir mini-saga: `pending` yaz → odeme baslat → event ile `confirmed`/expire. Iade akisi once iptal sonra refund sirasiyla (cifte iade yok). Yemekhane QR'i donen zaman penceresi ile HMAC imzali (leak blast radius kucuk). Catalog'un kapali gun dagitimi icin `/internal/closed-days/*` route'lari (loopback) mevcut.

#### payment (mock)
- **Veri:** yok — schema'siz, migration'siz, stateless
- **Sync sunar:** `PaymentService` → meal (`PaymentAdapter` uzerinden)
- **Publish:** `payment.completed` / `payment.failed` — DB'si olmadigi icin **outbox kullanmaz**, dogrudan publish eder (mock icin bilincli taviz)
- **Tasarim notlari:** `InitiatePayment` 2 sn gecikmeyle `payment.completed` yayinlar, refund hep basarili. Gercek saglayici entegrasyonu kapsam disi; interface sinirlari korunursa yerine gercek adapter takilabilir.

#### notification (ayri servis)
- **Yapi:** `consumer/` (RabbitMQ setup + handler dispatch) → `service/` → `delivery/email` (SMTP) + `delivery/push` (FCM stub) → `repository/db` (teslimat loglari), `templates/` (HTML)
- **Veri:** kendi Postgres instance'i (:5433, DB `notification`) — gercek mikroservis izolasyonu
- **Consume:** `auth.events` → `user.registered`, `user.password_reset_requested`
- **Tasarim notlari:** monolith'e hicbir sync bagimliligi yok; sadece exchange kontratina bagli. Yeni bildirim tipi = yeni binding + handler + template.

---

## 5. Veri Mimarisi

- **Tek PostgreSQL instance** (:5432, DB `mydreamcampus`), **modul basina schema**: `auth`, `staff`, `student`, `course_catalog`, `enrollment`, `attendance`, `grades`, `meal` (payment stateless — schema'si yok).
- Notification icin **ayri Postgres instance** (:5433) — gercek mikroservis izolasyonu sergilemek icin.
- Erisim: **sqlc + pgx/v5** (raw SQL, ORM yok). Migration: **goose**, compose'da one-shot `migrate` container'i uygular; modul basina ayri version tablosu.
- **View/projection tablolari:** Moduller birbirinin tablosunu okumaz. Ihtiyac duyulan veri event ile senkronlanan lokal "view" tablolarinda tutulur: `attendance.students_view/courses_view/enrollments_view`, `grades.students_view/courses_view/prerequisite_courses_view`, `meal.students_view`. Bu, monolith icinde mikroservis sinirlarini korur — ileride tekrar servis ayirmayi kolaylastirir.
- Kimlik: UUID; ilk admin sabit UUID ile seed edilir.
- Demo verisi: `seed` container'i admin API uzerinden yukler (DB'ye dogrudan yazmaz) — API kontratlari ayni zamanda seed ile test edilmis olur.

---

## 6. Event Mimarisi

**Akis:** Is islemi + event **ayni DB transaction'inda** yazilir (outbox pattern) → modul basina bir `OutboxWorker` goroutine'i outbox tablosunu poll'lar → RabbitMQ topic exchange'ine publish eder → consumer'lar `processed_events` tablosuyla **idempotent** islem yapar. Basarisiz event'ler `retry_count`/`max_retries` ile yeniden denenir; DLQ altyapisi mevcut. Kuyruk binding'leri `main.go`'da **pre-declare** edilir — consumer offline olsa da mesaj birikmeye devam eder.

**Exchange'ler (topic, durable):** `auth.events`, `staff.events`, `student.events`, `course_catalog.events`, `enrollment.events`, `attendance.events`, `grades.events`, `meal.events`, `payment.events`.

**Aktif tuketiciler:**

| Kuyruk | Kaynak exchange | Routing key | Tuketen |
|---|---|---|---|
| `auth_events_queue` | staff.events, student.events | staff/student `.created/.updated/.deactivated` | auth (user projection) |
| `student.staff_events` | staff.events | `staff.deactivated` | student (danisman dusurme) |
| `meal.payment_completed_queue` | payment.events | `payment.completed` | meal (rezervasyon confirm) |
| `meal.payment_failed_queue` | payment.events | `payment.failed` | meal (rezervasyon expire) |
| `meal.student_created/updated/deactivated_queue` | student.events | `student.*` | meal (students_view sync) |
| `attendance.sync_events` | student.events, course_catalog.events, enrollment.events | `student.*`, `course.semester.created`, `enrollment.program.approved` | attendance (view sync) |
| `grades.sync_events` | student.events, course_catalog.events, enrollment.events, attendance.events | `student.*`, `course.semester.created`, `enrollment.program.approved`, `attendance.semester.failed` | grades (view + registration sync, devamsizlik isaretleme) |
| `grades.finalize_requested` | grades.events | `grade.finalize.requested` | grades (finalize self-loop — request path disinda) |
| `notification_events_queue` | auth.events | `user.registered`, `user.password_reset_requested` | notification service |

**Envelope:** `{event_id, event_type, timestamp, data}` — `event_id` idempotency anahtaridir.

---

## 7. Redis Kullanim Haritasi

| Kullanim | Modul | Detay |
|---|---|---|
| Access token blacklist | auth (+tum JWTAuth) | Logout'ta JTI kalan TTL ile blacklist'e yazilir |
| Refresh token store | auth | JTI → user; DB session source-of-truth, Redis hizlandirici |
| Token version | auth | `logout-all`/sifre degisiminde versiyon artar, eski tokenlar gecersiz |
| Password reset token | auth | 1 saat TTL |
| Rate limiting | platform | IP + user limitleri; login/refresh/password endpointleri **fail-closed** (Redis dususe 503), digerleri fail-open |
| Yoklama buffer | attendance | `SADD` ile atomik dedup + hash buffer; worker toplu DB flush (5 sn) |
| Yoklama session cache | attendance | Session metadata + enrolled ogrenci seti |

Redis erisilemezse monolith **acilmaz** (main.go'da fatal) — auth guvenligi Redis'e bagli oldugu icin bilincli tercih.

---

## 8. Donem Kural Motoru (`platform/rules`)

Period tanimlari `course_catalog.academic_periods`'ta tutulur; moduller catalog'un `PeriodRepo`'sunu in-process kullanir. Uc katmanli zaman kilidi:

1. **Hard deadline** (donem bitisi) — *hic kimse* (admin dahil) sonrasinda islem yapamaz.
2. **Period penceresi** (kayit/not/yoklama donemleri, catalog'da tanimlanir) — normal kullanicilar pencere disinda islem yapamaz.
3. **Admin override** — admin period'u asabilir ama hard deadline'i asamaz. **Istisna: enrollment** — siki kilit, admin de period disinda ogrenci programina dokunamaz (ders kaydi ogrencinin sorumlulugu).

Katalog erisilemezse kontroller "graceful degradation" ile atlanir (fail-open) — bkz. bilinen eksikler.

---

## 9. Guvenlik Ozeti

- JWT **HS256**, access (dakika) + refresh (saat) ayrimi, refresh rotation, JTI tabanli session takibi
- Sifre: **Argon2id**; email enumeration'a karsi timing-safe dummy verify; deaktif hesap = "invalid credentials"
- 5 basarisiz giris → 30 dk hesap kilidi
- CSRF token (cookie tabanli akis icin), CSP + security headers, CORS whitelist, request body limit
- QR imzalari (yoklama + yemekhane) HMAC-SHA256; yemekhane QR'i donen zaman penceresi ile (leak blast radius kucuk)
- Infra portlari compose'da `127.0.0.1`'e bind — disaridan yalniz Caddy (80/443) erisilir; RabbitMQ/Redis parolali, compose secret'lari `.env`'den zorunlu (`:?` — default parola yok)
- TLS: Caddy otomatik Let's Encrypt (`SITE_ADDRESS`); localhost'ta internal CA
- Audit log: kritik admin aksiyonlari (`platform/audit`) `course_catalog.audit_log`'a yazilir
- CI: `gosec` + `govulncheck` security workflow'da calisir (`.github/`)

---

## 10. Altyapi ve Deploy

`new-backend/infrastructure/docker-compose.yml` — tum stack tek compose:

| Servis | Port | Not |
|---|---|---|
| caddy | **80/443 (public)** | SPA static + `/api` ve `/health` proxy; tek public giris |
| monolith | expose 8080 (host'a acik degil) | `migrate` bitmeden baslamaz; `ENVIRONMENT=production` |
| notification | expose 9090 | ayri container, `migrate`'i bekler |
| migrate | one-shot | tum modullerin goose migration'lari + notification DB |
| seed | one-shot | admin API ile idempotent demo verisi (`SEED_DEMO` ile kontrol) |
| postgres (monolith) | 127.0.0.1:5432 | init script schema'lari olusturur |
| postgres (notification) | 127.0.0.1:5433 | ayri instance |
| rabbitmq | 127.0.0.1:5672 / 15672 (UI) | user/pass `.env`'den zorunlu |
| redis | 127.0.0.1:6379 | requirepass zorunlu |
| mailhog | 127.0.0.1:1025 (SMTP) / 8025 (UI) | notification mail cikisi |

- **Prod deploy:** tek DigitalOcean droplet + `docker compose` + sslip.io HTTPS — adim adim rehber: `DEPLOY.md`.
- **Dev:** infra compose'dan, monolith host'ta (`make` hedefleri), frontend Vite :3000 (`/api` proxy → 8080).
- Grafana/Loki/Promtail config dizinleri hazir ancak henuz compose'a ekli degil.

---

## 11. Bilinen Eksikler / Yol Haritasi Notlari

- **Prerequisite kontrolu bypass:** grades `grade.student.prerequisite.passed` publish ediyor ama enrollment tuketmiyor; enrollment'taki kontrol su an gecerken `prerequisite_courses_view`/cache kaldirildigi icin bypass ediliyor (kodda TODO).
- **HTTP loopback kalintisi:** catalog'un donem yasam dongusu fan-out'u (`/internal/periods`, meal `closed-days`, audit `/internal/audit-log`) hala `X-Internal-Secret` + loopback HTTP kullaniyor. Hedef mimari (CLAUDE.md §12) tamamen in-process — bu path'lerin client interface'e cekilmesi bekliyor. `meal/proto/` altindaki gRPC dosyalari da mikroservis doneminden kalinti, kullanilmiyor.
- **payment mock** — gercek saglayici entegrasyonu kapsam disi. Odeme her zaman basarili; `payment.failed` akisi pratikte tetiklenmez. Outbox kullanmaz (DB'si yok).
- Notification'daki `grades.entered`, `student.graduated` vb. handler'lar iskelet durumunda (kuyruk binding'i yok).
- Ilk sifre = email (`force_password_change` ile). Demo akisi icin bilincli tercih; gercek kurulumda davetiye/rastgele sifre akisi gerekir.
- Grafana/Loki/Promtail config'leri hazir ama compose'a ekli degil.
