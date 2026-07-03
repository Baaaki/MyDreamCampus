# MyDreamCampus — System Design

> Bu dokuman `new-backend/` mimarisini anlatir. Eski mikroservis kod tabani `legacy-codebase/` altindadir ve artik gelistirilmemektedir.

---

## 1. Genel Bakis

MyDreamCampus, universite yonetim sistemi. Mimari: **modular monolith** + tek bagimsiz mikroservis (notification).

- **Monolith** (`new-backend/monolith/`): 9 is modulu tek Go process'inde calisir (port **8080**). Her modul kendi handler/service/repository/db katmanina ve kendi Postgres **schema**'sina sahiptir.
- **Notification service** (`new-backend/services/notification/`): Ayri process, ayri Postgres instance. RabbitMQ'dan event tuketir, e-posta (SMTP/MailHog) ve push (FCM stub) gonderir.
- **Frontend**: React + Vite (dev: 3000), prod'da monolith SPA fallback ile servis edebilir (`FRONTEND_STATIC_ENABLED`).
- **Mobile**: React Native (Expo), ayni `/api/*` endpointlerini kullanir.

```
                        ┌──────────────┐   ┌──────────────┐
                        │ Web (Vite)   │   │ Mobile (Expo)│
                        └──────┬───────┘   └──────┬───────┘
                               │  HTTP /api/*     │
                               ▼                  ▼
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
                                     │               └──────────────┘
                                     ▼
                        ┌────────────────────────┐
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

**Route seviyesi:** `JWTAuth` → `CSRFProtection` → `UserRateLimit` → `RequireRole(...)`. Her modul JWT'yi kendi middleware'inde dogrular; auth modulune RPC yapilmaz (JWT HS256, ortak secret).

**Health:** `/health` (liveness), `/ready` (DB + RabbitMQ + Redis ping).

**Internal route'lar:** Modul-icinden-modul-icine HTTP fan-out gereken yerlerde (or. catalog → meal kapali gun dagitimi) `/api/<modul>/internal/*` + `X-Internal-Secret` header kullanilir.

---

## 3. Moduller ve Sorumluluklar

| Modul | Sorumluluk | Onemli akislar |
|---|---|---|
| **auth** | Login/logout/refresh, sifre degistirme/sifirlama, session yonetimi, admin seed | Argon2id, timing-safe dummy verify, hesap kilitleme (5 deneme → 30 dk), token rotation, Redis blacklist + token_version |
| **staff** | Personel CRUD, ogretim uyesi sorgulari | `staff.created/updated/deactivated` event'leri → auth user projection |
| **student** | Ogrenci CRUD, danisman atama (tekil + bulk), CSV import | `student.*` event'leri → auth; `staff.deactivated` tuketir (danisman dusurme) |
| **course_catalog** | Ders katalogu, donem (semester) yasam dongusu, donem dersi acma, ders programi, kayit/not/yoklama period tanimlari | Donem durumu: `planned → active → ...` — aktiflesince ders yapisi donar; instructor cakisma kontrolu |
| **enrollment** | Ders secimi (program taslagi), danisman onay/red, kontenjan | Kapasite kontrolu advisory lock + tek transaction; pending program auto-replace; siki period kilidi (admin dahil kimse period disinda degisemez) |
| **attendance** | QR ile yoklama, manuel yoklama, oturum yonetimi, devamsizlik finalize | QR: HMAC imzali statik payload; taramalar Redis buffer'a SADD ile dedup edilerek yazilir, BufferFlusher 5 sn'de bir toplu DB flush eder; esikler acilan oturum sayisina oranlanir (10/14, 11/14) |
| **grades** | Not girisi (tekil/bulk), assessment kilitleme, otomatik finalize (bagil/mutlak), transkript, itiraz | Bagil sistem: z-score, sinif istatistikleri donduruluyor (itirazda frozen stats ile yeniden hesap) |
| **meal** | Yemekhane, aylik menu, rezervasyon (tekil/batch), QR ile kullanim, iptal/iade | Rezervasyon `pending` yazilir → mock payment `payment.completed` publish eder → consumer `confirmed`'a cevirir; suresi dolan pending'leri worker expire eder |
| **payment** | Mock odeme saglayici (in-process) | `InitiatePayment` 2 sn gecikmeli `payment.completed` yayinlar; refund her zaman basarili |

**Notification service:** `auth.events` exchange'inden `user.registered` (hosgeldin maili) ve `user.password_reset_requested` (sifre sifirlama maili) tuketir. Teslimatlar kendi DB'sine loglanir.

---

## 4. Veri Mimarisi

- **Tek PostgreSQL instance** (:5432, DB `mydreamcampus`), **modul basina schema**: `auth`, `staff`, `student`, `course_catalog`, `enrollment`, `attendance`, `grades`, `meal`.
- Notification icin **ayri Postgres instance** (:5433) — gercek mikroservis izolasyonu sergilemek icin.
- Erisim: **sqlc + pgx/v5** (raw SQL, ORM yok). Migration: **goose**.
- **Cache/projection tablolari:** Moduller birbirinin tablosunu okumaz. Ihtiyac duyulan veri event ile senkronlanan lokal "view" tablolarinda tutulur (or. `attendance.students_view`, `grades.courses_view`, `meal.students_cache`). Bu, monolith icinde mikroservis sinirlarini korur — ileride tekrar servis ayirmayi kolaylastirir.
- Kimlik: UUID; ilk admin sabit UUID ile seed edilir.

---

## 5. Event Mimarisi

**Akis:** Is islemi + event **ayni DB transaction'inda** yazilir (outbox pattern) → modul basina bir `OutboxWorker` goroutine'i outbox tablosunu poll'lar → RabbitMQ topic exchange'ine publish eder → consumer'lar `processed_events` tablosuyla **idempotent** islem yapar. Basarisiz event'ler `retry_count`/`max_retries` ile yeniden denenir; DLQ altyapisi mevcut.

**Exchange'ler (topic, durable):** `auth.events`, `staff.events`, `student.events`, `course_catalog.events`, `enrollment.events`, `attendance.events`, `grades.events`, `meal.events`, `payment.events`.

**Aktif tuketiciler:**

| Kuyruk | Kaynak exchange | Routing key | Tuketen |
|---|---|---|---|
| `auth_events_queue` | staff.events, student.events | staff/student `.created/.updated/.deactivated` | auth (user projection) |
| `student.staff_events` | staff.events | `staff.deactivated` | student (danisman dusurme) |
| `meal.payment_completed_queue` | payment.events | `payment.completed` | meal (rezervasyon confirm) |
| `meal.payment_failed_queue` | payment.events | `payment.failed` | meal (rezervasyon expire) |
| meal student kuyrugu | student.events | `student.*` | meal (students_cache sync) |
| `attendance.sync_events` | student.events, course_catalog.events, enrollment.events | `student.*`, `course.semester.created`, `enrollment.program.approved` | attendance (cache sync) |
| `grades.sync_events` | student.events, course_catalog.events, enrollment.events, attendance.events | `student.*`, `course.semester.created`, `enrollment.program.approved`, `attendance.semester.failed` | grades (cache + registration sync, devamsizlik isaretleme) |
| `grades.finalize_requested` | grades.events | `grade.finalize.requested` | grades (kendi finalize self-loop'u — AutoFinalize request path disinda calisir) |
| `notification_events_queue` | auth.events | `user.registered`, `user.password_reset_requested` | notification service |

**Envelope:** `{event_id, event_type, timestamp, data}` — `event_id` idempotency anahtaridir.

---

## 6. Redis Kullanim Haritasi

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

## 7. Donem Kural Motoru (`platform/rules`)

Uc katmanli zaman kilidi:

1. **Hard deadline** (donem bitisi) — *hic kimse* (admin dahil) sonrasinda islem yapamaz.
2. **Period penceresi** (kayit/not/yoklama donemleri, catalog'da tanimlanir) — normal kullanicilar pencere disinda islem yapamaz.
3. **Admin override** — admin period'u asabilir ama hard deadline'i asamaz. **Istisna: enrollment** — siki kilit, admin de period disinda ogrenci programina dokunamaz (ders kaydi ogrencinin sorumlulugu).

Katalog erisilemezse kontroller "graceful degradation" ile atlanir (fail-open) — bkz. bilinen eksikler.

---

## 8. Guvenlik Ozeti

- JWT **HS256**, access (dakika) + refresh (saat) ayrimi, refresh rotation, JTI tabanli session takibi
- Sifre: **Argon2id**; email enumeration'a karsi timing-safe dummy verify; deaktif hesap = "invalid credentials"
- 5 basarisiz giris → 30 dk hesap kilidi
- CSRF token (cookie tabanli akis icin), CSP + security headers, CORS whitelist, request body limit
- QR imzalari (yoklama + yemekhane) HMAC-SHA256; yemekhane QR'i donen zaman penceresi ile (leak blast radius kucuk)
- Infra portlari compose'da `127.0.0.1`'e bind (VPS senaryosu icin), RabbitMQ/Redis parolali
- Audit log: kritik admin aksiyonlari (`platform/audit`) ayrica loglanir

---

## 9. Altyapi ve Portlar

`new-backend/infrastructure/docker-compose.yml`:

| Bilesen | Port | Not |
|---|---|---|
| Monolith (host process) | 8080 | `PORT` env, compose'da degil |
| PostgreSQL (monolith) | 127.0.0.1:5432 | init script schema'lari olusturur |
| PostgreSQL (notification) | 127.0.0.1:5433 | ayri instance |
| RabbitMQ | 127.0.0.1:5672 / 15672 (UI) | user/pass env'den |
| Redis | 127.0.0.1:6379 | requirepass zorunlu |
| MailHog | 127.0.0.1:1025 (SMTP) / 8025 (UI) | notification mail cikisi |
| Frontend dev (Vite) | 3000 | `/api` proxy → 8080 |

Grafana/Loki/Promtail config dizinleri hazir ancak henuz compose'a ekli degil.

---

## 10. Bilinen Eksikler / Yol Haritasi Notlari

- Enrollment on-kosul (prerequisite) kontrolu hala bypass durumda (TODO): grades `grade.student.prerequisite.passed` event'ini publish ediyor ama enrollment henuz tuketmiyor.
- `payment` mock — gercek saglayici entegrasyonu kapsam disi. Odeme her zaman basarili; `payment.failed` akisi pratikte tetiklenmez.
- Notification'daki `grades.entered`, `student.graduated` vb. event handler'lari iskelet durumunda (kuyruk binding'i yok).
- Ilk sifre = email (`force_password_change` ile). Demo akisi icin bilincli tercih; gercek kurulumda davetiye/rastgele sifre akisi gerekir.
- Grafana/Loki/Promtail config'leri hazir ama compose'a ekli degil.
