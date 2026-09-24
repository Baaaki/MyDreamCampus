# Referans Mimari — Ortak Tablolar

> **Projenin mimari kaynağı bu dosyadır.** Bir **sözlük** olarak yazıldı;
> baştan sona okuma, ihtiyacın olan tabloyu bul.

> **Asıl kaynak koddur.** Buradaki tablolar `cmd/main.go`, servislerin
> `internal/module.go` dosyaları, `worker/` consumer'ları ve migration
> `.sql`'lerinden çıkarıldı. Bir davranış sorusunda önce koda bak; bu dosya
> nereye bakacağını söyler.

---

## 1. Servis Tablosu

| Servis | Dizin | Port | DB adı | Schema | Route prefix |
|---|---|---|---|---|---|
| auth | `services/auth-service/` | 8081 | `auth` | `auth` | `/api/auth` |
| staff | `services/staff-service/` | 8082 | `staff` | `staff` | `/api/staff`, `/api/admin-staff` |
| student | `services/student-service/` | 8083 | `student` | `student` | `/api/students` |
| catalog | `services/catalog-service/` | 8084 | `catalog` | `course_catalog` | `/api/catalog`, `/api/semesters` |
| enrollment | `services/enrollment-service/` | 8085 | `enrollment` | `enrollment` | `/api/enrollment` |
| attendance | `services/attendance-service/` | 8086 | `attendance` | `attendance` | `/api/attendance` |
| grades | `services/grades-service/` | 8087 | `grades` | `grades` | `/api/grades` |
| meal | `services/meal-service/` | 8088 | `meal` | `meal` | `/api/meals` |
| payment | `services/payment-service/` | 8089 | `payment` | `payment` | `/api/payments` |
| notification | `services/notification-service/` | 9090 | `notification` | `public` | — (sadece consumer) |

**Dikkat:** catalog servisinin DB adı `catalog`, schema adı `course_catalog`.
Schema adları migration dosyalarındaki haliyle korunuyor — hiçbir `.sql`
dosyası değişmiyor.

Go modül adları: `github.com/baaaki/mydreamcampus/<servis>` (örn. `.../auth`).

---

## 2. Sync Bağımlılık Haritası (internal REST)

Ok yönü = çağıran → çağrılan.

```
catalog     ──► staff       (instructor doğrulama)
student     ──► staff       (danışman doğrulama)
enrollment  ──► student     (öğrenci doğrulama)
enrollment  ──► catalog     (ders listesi / ders doğrulama)
attendance  ──► catalog     (SemesterInfo)
grades      ──► catalog     (SemesterInfo)
meal        ──► payment     (ödeme başlatma / iade)
```

`academic_periods` **sync değil** — event ile lokal projeksiyon.
Audit log **sync değil** — event ile catalog'a yazılır.

Tüm internal çağrılar `X-Internal-Secret` header'ı taşır; doğrulama
`shared/platform/middleware/internal.go` içindeki mevcut middleware ile yapılır.

### Internal endpoint kontratları

| Sağlayan | Endpoint | Dönen |
|---|---|---|
| staff | `GET /internal/staff/:id` | id, first_name, last_name, department, status |
| staff | `GET /internal/staff?department=X&role=instructor` | yukarıdakinin listesi |
| student | `GET /internal/students/:id` | StudentResponse |
| student | `GET /internal/students?advisor_id=X` | StudentResponse listesi |
| catalog | `GET /internal/semesters/:name` | name, status, hard_deadline, is_past_deadline |
| catalog | `GET /internal/semester-courses?semester=&department=&class_level=` | SemesterCourseListItem listesi |
| catalog | `GET /internal/semester-courses/:id?semester=` | SemesterCourseResponse |
| payment | `POST /internal/payments/initiate` | InitiatePaymentResponse |
| payment | `POST /internal/payments/refund` | RefundResponse |

Bu endpoint'ler `/api/*` altında **değil** — Caddy bunları dışarı proxy'lemez,
sadece compose network'ünden erişilir.

---

## 3. Event Haritası (RabbitMQ)

**Exchange'ler (topic, durable):** `auth.events`, `staff.events`,
`student.events`, `course_catalog.events`, `enrollment.events`,
`attendance.events`, `grades.events`, `meal.events`, `payment.events`

| Kuyruk | Kaynak exchange | Routing key | Tüketen servis |
|---|---|---|---|
| `auth_events_queue` | staff.events, student.events | `staff.*`, `student.*` (created/updated/deactivated) | auth |
| `student.staff_events` | staff.events | `staff.deactivated` | student |
| `attendance.sync_events` | student, course_catalog, enrollment | `student.*`, `course.semester.created`, `enrollment.program.approved` | attendance |
| `grades.sync_events` | student, course_catalog, enrollment, attendance | yukarıdakiler + `attendance.semester.failed` | grades |
| `grades.finalize_requested` | grades.events | `grade.finalize.requested` | grades (self-loop) |
| `enrollment.sync_events` | grades.events | `grade.student.prerequisite.passed` | enrollment |
| `meal.payment_completed_queue` | payment.events | `payment.completed` | meal |
| `meal.payment_failed_queue` | payment.events | `payment.failed` | meal |
| `meal.student_*_queue` | student.events | `student.*` | meal |
| `notification_events_queue` | auth.events | `user.registered`, `user.password_reset_requested` | notification |
| `enrollment.period_events` | course_catalog.events | `course_catalog.period.enrollment.*` | enrollment |
| `grades.period_events` | course_catalog.events | `course_catalog.period.grading.*` | grades |
| `attendance.period_events` | course_catalog.events | `course_catalog.period.attendance.*` | attendance |
| `catalog.audit_events` | grades.events, meal.events | `audit.entry.created` | catalog |

Dönem kuyruklarında routing key sonu `created` / `updated` / `deleted`.
Filtreleme broker'da yapılır — her tüketici sadece kendi dönem tipini alır.

**Envelope:** `{event_id, event_type, timestamp, data}` — `event_id`
idempotency anahtarı, `processed_events` tablosuyla kontrol edilir.

**Kural:** Her publish outbox üzerinden (istisna: payment — DB'si yok).
Her consumer idempotent.

### 3.1 Kuyruk Declare Sorumluluğu

1. Kuyruğu **tüketen servis** declare eder — kendi `cmd/main.go`'sundaki
   `DeclareQueues` listesinde. Publisher asla başkasının kuyruğunu tanımlamaz;
   aksi halde tüketici hiç ayağa kalkmasa bile kuyruk var olur ve sahipliği
   koddan okunamaz hale gelir.
2. Kayıp riskine karşı topoloji ayrıca **`infrastructure/rabbitmq/definitions.json`**
   ile RabbitMQ boot'unda yüklenir. Böylece bir tüketici hiç başlamamış olsa
   bile binding vardır ve topic exchange mesajı atmak yerine kuyrukta biriktirir.
3. Servis içi declare'ler idempotent — tek servis `definitions.json` olmadan
   da dev ortamında çalışabilir.

**Bakım notu:** Yeni bir kuyruk **iki yere birden** eklenir (1 ve 2). Sadece
birine eklemek, tüketici hiç ayağa kalkmadığı senaryoda mesajın sessizce
kaybolması demektir.

---

## 4. Konteyner Haritası

| Konteyner | Image | Host portu | Tahmini RAM |
|---|---|---|---|
| `mydreamcampus-postgres` | postgres:18 | 127.0.0.1:5432 (standalone) | ~300 MB |
| `mydreamcampus-rabbitmq` | rabbitmq:4.3-management | 127.0.0.1:5672/15672 (standalone) | ~130 MB |
| `mydreamcampus-redis` | redis:8.10-alpine | 127.0.0.1:6379 (standalone) | ~15 MB |
| `mydreamcampus-mailhog` | mailhog/mailhog | 127.0.0.1:1025/8025 (standalone) | ~10 MB |
| `mydreamcampus-migrate` | one-shot | — | — |
| `mydreamcampus-seed` | one-shot | — | — |
| `mydreamcampus-auth` … `-payment` (9 adet) | kendi image'ları | expose only | ~30 MB × 9 |
| `mydreamcampus-notification` | kendi image'ı | expose only | ~20 MB |
| `mydreamcampus-caddy` | frontend/Dockerfile | **80** (base) / 443 (standalone) | ~15 MB |

**Toplam: 16 konteyner, ~760 MB RAM.**

`notification-postgres` konteyneri **kaldırılıyor** — DB'si tek Postgres'e
katlanıyor.

**Compose dosya bölünmesi korunuyor** (CLAUDE.md §13): `docker-compose.yml`
sadece Caddy `:80`'i publish eder, `docker-compose.standalone.yml` infra
portlarını ekler. Yeni host portu **standalone dosyasına** eklenir.

---

## 5. Repo Yapısı

```
new-backend/
├── go.work
├── shared/
│   ├── go.mod                       (modül: .../shared)
│   ├── events/                      (mevcut)
│   ├── platform/                    (tüm servislerin ortak altyapısı)
│   │   ├── audit/ clock/ clocksync/ database/ dto/ errors/ handler/
│   │   ├── logger/ middleware/ rabbitmq/ redis/ repository/
│   │   └── rules/ semester/ utils/
│   ├── client/                      (internal REST client'ları)
│   └── httpserver/                  (ortak Gin bootstrap)
├── services/
│   ├── auth-service/  staff-service/  student-service/
│   ├── catalog-service/  enrollment-service/  attendance-service/
│   ├── grades-service/  meal-service/  payment-service/
│   └── notification-service/        (RabbitMQ consumer, HTTP route'u yok)
└── infrastructure/
    ├── docker-compose.yml
    ├── docker-compose.standalone.yml
    ├── postgres/init-databases.sh
    ├── migrate/
    └── seed/
```

Her servis içi yapı:

```
services/<x>-service/
├── go.mod  go.sum  Dockerfile  sqlc.yaml
├── cmd/main.go
├── config/config.go
└── internal/
    ├── handler/ service/ repository/ db/ dto/ errors/ worker/
    └── sql/{migrations,queries}/
```

---

## 6. Environment Değişkenleri (servis başına)

Ortak (hepsinde):

```
ENVIRONMENT       production
PORT              <servis portu>
DB_URL            postgres://<svc>_svc:<pw>@postgres:5432/<db>?sslmode=disable
RABBITMQ_URL      amqp://<user>:<pw>@rabbitmq:5672/
JWT_SECRET        <ortak — tüm servisler aynı, HS256 doğrulaması için>
INTERNAL_SERVICE_SECRET  <ortak — internal REST çağrıları için>
CORS_ALLOWED_ORIGINS     ${PUBLIC_ORIGIN}
```

Servise özel:

| Servis | Ek değişken |
|---|---|
| auth | `REDIS_ADDR`, `REDIS_PASSWORD`, `ADMIN_EMAIL`, `ADMIN_INITIAL_PASSWORD` |
| attendance | `REDIS_ADDR`, `REDIS_PASSWORD`, `QR_SECRET`, `CATALOG_SERVICE_URL` |
| meal | `REDIS_ADDR`, `REDIS_PASSWORD`, `QR_SECRET`, `PAYMENT_SERVICE_URL` |
| grades | `CATALOG_SERVICE_URL` |
| enrollment | `STUDENT_SERVICE_URL`, `CATALOG_SERVICE_URL` |
| catalog | `STAFF_SERVICE_URL` |
| student | `STAFF_SERVICE_URL` |
| tüm servisler (rate limit) | `REDIS_ADDR`, `REDIS_PASSWORD` |

**Not:** Rate limiting Redis tabanlı ve global middleware zincirinde. Tüm
servisler Redis'e bağlanır. `auth` için Redis erişilemezliği **fatal**
(fail-closed login), diğerlerinde fail-open.

**Zaman makinesi:** İş kuralı saati (`platform/clock`) gerçek saat + ofset;
ofset Redis'te `clock:state` anahtarında, değişiklik `clock:changed`
kanalıyla duyurulur ve her servis 10 sn'de bir anahtarı yeniden okur
(`platform/clocksync`). Redis yoksa servis gerçek saatte kalır. Token,
oturum, rate limit, idempotency, outbox ve Redis TTL'leri her zaman gerçek
saattedir. Her servis `GET /api/<prefix>/admin/time/status` (JWT + admin)
sunar; ayar yalnız catalog'da:
`POST /api/catalog/admin/time/simulate` (`{"time": "<RFC3339>"}`, en fazla
±2 yıl) ve `POST /api/catalog/admin/time/reset`; ikisi de catalog audit
log'una yazılır.

Service URL formatı: `http://<servis-adı>:<port>` (compose DNS).

---

## 7. Bilinen Açıklar / Kapsam Dışı

Mimarinin taşıdığı, henüz kapatılmamış açıklar:

- payment mock (gerçek sağlayıcı entegrasyonu yok, outbox kullanmıyor)
- İlk şifre = email (`force_password_change`)
- Notification'daki iskelet handler'lar (`grades.entered`, `student.graduated`)
- Mobil push gönderimi iskelet (`delivery/push` yalnızca logluyor; FCM entegrasyonu yok)
- İdari personel rehberinde (`/api/admin-staff`) kayıt oluşturma yalnızca API ve seed üzerinden; arayüzde oluşturma formu yok
- Grafana/Loki/Promtail compose'a ekli değil

Bunlar ayrı iş kalemleri — hiçbiri mikroservis bölünmesinden kaynaklanmıyor.
