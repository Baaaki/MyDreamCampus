# Referans Mimari — Ortak Tablolar

> Bu bir faz dosyası değil, **sözlük**. Faz dosyaları buraya atıf yapar.
> Baştan sona okuma — ihtiyacın olan tabloyu bul.

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
| payment | `services/payment-service/` | 8089 | — (stateless) | — | `/api/payments` |
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

`academic_periods` **sync değil** — event ile lokal projeksiyon (Faz 2).
Audit log **sync değil** — event ile catalog'a yazılır (Faz 3).

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

## 3. Event Haritası (RabbitMQ — değişmiyor)

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

**Faz 2'de eklenecek yeni kuyruklar:**

| Kuyruk | Kaynak | Routing key | Tüketen |
|---|---|---|---|
| `enrollment.period_events` | course_catalog.events | `course_catalog.period.enrollment.*` | enrollment |
| `grades.period_events` | course_catalog.events | `course_catalog.period.grading.*` | grades |
| `attendance.period_events` | course_catalog.events | `course_catalog.period.attendance.*` | attendance |

Routing key sonu `created` / `updated` / `deleted`. Filtreleme broker'da
yapılır — her tüketici sadece kendi dönem tipini alır.

**Faz 3'te eklenecek:**

| Kuyruk | Kaynak | Routing key | Tüketen |
|---|---|---|---|
| `catalog.audit_events` | grades.events, meal.events | `audit.entry.created` | catalog |

**Envelope:** `{event_id, event_type, timestamp, data}` — `event_id`
idempotency anahtarı, `processed_events` tablosuyla kontrol edilir.

**Kural:** Her publish outbox üzerinden (istisna: payment — DB'si yok).
Her consumer idempotent.

---

## 4. Konteyner Haritası (hedef)

| Konteyner | Image | Host portu | Tahmini RAM |
|---|---|---|---|
| `mydreamcampus-postgres` | postgres:18 | 127.0.0.1:5432 (standalone) | ~300 MB |
| `mydreamcampus-rabbitmq` | rabbitmq:3-management | 127.0.0.1:5672/15672 (standalone) | ~130 MB |
| `mydreamcampus-redis` | redis:7-alpine | 127.0.0.1:6379 (standalone) | ~15 MB |
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

## 5. Repo Yapısı (hedef)

```
new-backend/
├── go.work                          ← Faz 4
├── shared/
│   ├── go.mod                       (modül: .../shared)
│   ├── events/                      (mevcut)
│   ├── platform/                    ← Faz 0: monolith/internal/platform/ buradan gelir
│   │   ├── audit/ clock/ database/ dto/ errors/ handler/
│   │   ├── logger/ middleware/ rabbitmq/ redis/ repository/
│   │   └── rules/ semester/ utils/
│   ├── client/                      ← Faz 3
│   └── httpserver/                  ← Faz 4 (ortak Gin bootstrap)
├── services/
│   ├── auth-service/  staff-service/  student-service/
│   ├── catalog-service/  enrollment-service/  attendance-service/
│   ├── grades-service/  meal-service/  payment-service/
│   └── notification-service/        ← mevcut notification/ buraya taşınır
└── infrastructure/
    ├── docker-compose.yml
    ├── docker-compose.standalone.yml
    ├── postgres/init-databases.sh   ← Faz 1
    ├── migrate/                     ← Faz 1'de güncellenir
    └── seed/
```

Her servis içi yapı (monolith'teki modül şablonunun aynısı):

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

Service URL formatı: `http://<servis-adı>:<port>` (compose DNS).

---

## 7. Blokerler ve Hangi Fazda Çözüldükleri

| Bloker | Faz |
|---|---|
| `monolith/internal/platform/` — Go `internal/` kısıtı servisler arası paylaşımı engelliyor | 0 |
| Tek `pgxpool` 9 modüle dağıtılıyor | 1 + 4 |
| `SimplePeriodRepository` → `course_catalog.academic_periods` cross-service okuma | 2 |
| 7 adet in-process sync client | 3 |
| catalog → grades/meal audit yazımı (in-process `DirectAuditLogger`) | 3 |
| Tek `cmd/main.go`, tek binary | 4 |
| Caddy tek upstream'e proxy'liyor | 5 |
| HTTP loopback kalıntısı (`X-Internal-Secret` + `/internal/*`) | 3'te normalleşir |

---

## 8. Taşınmayan / Kapsam Dışı

Bu migrasyon **davranış değiştirmez**. Mevcut açıklar taşınır, çözülmez:

- Prerequisite kontrolü bypass (enrollment `grade.student.prerequisite.passed` tüketmiyor)
- payment mock (gerçek sağlayıcı entegrasyonu yok, outbox kullanmıyor)
- İlk şifre = email (`force_password_change`)
- Notification'daki iskelet handler'lar (`grades.entered`, `student.graduated`)
- Grafana/Loki/Promtail compose'a ekli değil

Bunlar ayrı iş kalemleri. Migrasyon sırasında "bu arada şunu da düzelteyim"
yapma — faz sınırlarını kirletir.
