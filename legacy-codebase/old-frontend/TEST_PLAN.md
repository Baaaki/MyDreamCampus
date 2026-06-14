# Test Coverage — Yapılanlar + Eksikler

Bu doküman test altyapısının mevcut durumunu ve **kalan işleri** detaylı listeler.
Yeni bir context window'da kaldığı yerden devam etmek için tasarlandı.

> Son güncelleme: 2026-04-27
> Mevcut: 63 test dosyası, ~480 test geçiyor (`make test`)

---

## ✅ Yapıldı (referans için)

### Backend
- **shared/** — utils (password, jwt, pgtype, validation), clock, errors, rules
  (grading/period/semester), middleware (auth, cors, csrf, recovery,
  security_headers, rbac, ratelimit), logger/context, handler/health,
  audit/security, config (helpers, ports), events, semester/http_checker,
  client/semester_client → **224+ test, 17 paket**
- **auth-service** — errors, dto (auth + event), handler/verify, mevcut
  service+handler testleri → **45 test**
- **staff-service** — errors, dto (staff, common, teacher_profile) → **29 test**
- **student-service** — errors, dto (student, import, event) → **42 test**
  (integration_test.go `INTEGRATION_TESTS=true` ile gate'li)
- **course-catalog-service** — errors (incl. ScheduleConflictError),
  dto (catalog, common, semester) → **37 test**
- **enrollment-service** — errors, dto → **13 test**
- **attendance-service** — errors, dto → **13 test**
- **grades-service** — errors, dto, **service/grading_helpers** (z-score,
  absolute mapping, weighted average, class statistics) → **41 test**
- **meal-service** — errors (24 kod), dto (batch validation, meal time/menu)
  → **16 test**
- **payment-service** — gRPC server (mock payment + refund + clock injection)
  → **5 test**

### Frontend (Vitest + RTL)
- **lib/** — password-policy (cross-platform mirror), utils, constants,
  api-client (CSRF + 401/refresh single-flight + retry-loop guard)
- **components/** — error-boundary (catch + reset), auth-guard (role-based
  redirects, corrupt JSON)
- → **42 test, 6 dosya**

### Mobile (Jest + babel-jest, Node env, RN/Expo bypass)
- **lib/password-policy** (backend + frontend ile aynı sözleşme)
- **services/api.ts** — axios interceptor (Bearer attach, 401 fail-paths,
  refresh+retry, no-loop, /auth/login skip)
- **services/authService.ts** — tüm public API
- → **34 test, 3 dosya**

### Altyapı
- `frontend/vitest.config.ts` + `src/test/setup.ts`
- `mobile/jest.setup.js` + `__mocks__/react-native.js` + babel-jest config
  (jest-expo'da scope hatası nedeniyle Node env'e geçildi)
- Tüm Go modüllerine `go mod tidy` + testify direkt bağımlılık
- `Makefile`: `test`, `test-backend`, `test-frontend`, `test-mobile`,
  `test-coverage`
- `.github/workflows/ci.yml`: frontend `bun run test` + mobile `npm test --ci`
  adımları eklendi

---

## ❌ Eksik — Öncelik sırasına göre

### Tier 1 — Yüksek değer / orta-yüksek efor

#### 1.1 Backend service-layer integration tests (testcontainers)

**Sorun**: Servislerin `internal/service/*.go` katmanı concrete `*pgxpool.Pool`
ve `*repository.XxxRepository` bağımlılığı taşıyor. Mock zor; gerçek DB+Redis+
RabbitMQ container gerek.

**Çözüm**: `github.com/testcontainers/testcontainers-go` ile suite kurulumu.
Auth-service'in mevcut `tests/integration_test.go`'su (INTEGRATION_TESTS=true
gate'li) bir başlangıç — onu testcontainers kullanacak şekilde refactor et,
sonra 8 servise yay.

**Test edilmesi gereken kritik flow'lar:**

- **auth-service** `auth_service.go`:
  - `Login`: account lock (5 fail), failed_attempts increment+reset,
    deactivated path, locked_until check, force_password_change response
  - `RefreshAccessToken`: token version mismatch → revoke, session not found,
    new JTI rotation
  - `LogoutAll`: token version increment + Redis SetTokenVersion +
    BlacklistAllUserTokens + DeleteAllUserSessions
  - `ChangePassword`: old password verify, policy validation, token version
    increment, session cleanup
  - `SeedAdmin`: idempotency, force_password_change=true on admin

- **enrollment-service** `enrollment_service.go`:
  - `CreateEnrollmentProgram`: prereq check, capacity check (transaction),
    department mismatch, class level mismatch, duplicate course in request,
    schedule conflict (within new + with existing approved), auto-replace
    pending program, MaxCoursesPerEnrollment limit
  - `CancelMyEnrollment`: with capacity decrement
  - period lock (CanEnrollInSemester) — strict, no admin bypass

- **course-catalog-service**:
  - `SemesterService` — schedule conflict detection logic
  - `CatalogService` — prerequisite validation (class level less than course)
  - Semester course state machine: planned → active → frozen
  - Theory/lab hours mismatch with schedule sessions

- **attendance-service**:
  - QR generation/validation with rotating window bucket
  - Session expiry handling
  - `FinalizeAttendance` — failed students list (theory/lab/both)
  - Manual attendance entry by instructor (forbidden for non-instructor)

- **meal-service** `reservation_service.go`:
  - `validateReservationDate` — closed days, weekend rejection
  - `validateMealTimeWindow` — UTC+3 hour check
  - QR HMAC signature: `signQRPayload`, `verifyQRSignature`, rotating
    `qrWindow` accepts current+previous bucket
  - Batch reservation conflict resolution
  - Cancel cutoff window
  - Payment client integration (mock client gerekecek)

- **grades-service**:
  - `SubmitScore` finalize trigger when all assessments complete
  - Lock/unlock score (incomplete assessment rejection)
  - Appeal flow (frozen class mean recalc)
  - `MyGradesResponse` cumulative GPA calculation
  - Transcript generation per semester

**Effort tahmini**: testcontainers harness ~4 saat, sonra her servis için
~2-3 saat = toplam **2-3 gün**.

**Başlangıç dosyaları**:
- Mevcut: `backend/services/auth-service/tests/integration_test.go` (line 50-114
  setup pattern'i)
- Mevcut: `backend/services/student-service/tests/integration_test.go`
  (HTTP-based, refactor edilebilir)

#### 1.2 Backend handler tests (8 servis)

Auth-service dışında hiçbir servis HTTP handler katmanını test etmiyor.
Service-layer mock'layarak `httptest.NewRequest` ile route → handler →
response sözleşmesi doğrulanmalı.

**Pattern**: `auth-service/internal/handler/auth_handler_test.go` ve
`verify_test.go` örnek alınabilir.

**Servisler**:
- staff-service: `staff_handler.go`, `teacher_profile_handler.go`
- student-service: handler dir (Bulk import multipart upload, search query)
- course-catalog-service: catalog + semester course handlers
- enrollment-service: student + advisor + admin endpoints
- attendance-service: QR flow + manual entry + admin sessions list
- grades-service: score submit + lock + appeal + transcript
- meal-service: reservation + batch + cancel + QR scan

**Effort**: servis başına ~2-3 saat = **2 gün**.

#### 1.3 Worker/event consumer tests (5 servis)

RabbitMQ event tüketim akışları:
- auth-service: `event_consumer.go` (student.created, staff.created,
  user.updated, user.deactivated)
- student-service: staff.deactivated → orphan students
- catalog-service: future events
- enrollment-service: course events
- attendance-service: semester end events

**Pattern**: Test handler'ı izole edip body unmarshal + service çağrısı
zincirini doğrula. RabbitMQ'yu mock'la veya testcontainers RabbitMQ.

**Effort**: ~4-6 saat.

---

### Tier 2 — Orta değer / orta efor

#### 2.1 Backend repository tests (sqlc-generated)

9 servisin repo katmanı hiç test edilmedi. testcontainers gerek (real PG).

**Önerilen yaklaşım**: Tier 1.1 ile birlikte yap; aynı container'ı paylaş.
Repo testleri DB-only, hızlıca yazılır.

**Effort**: testcontainers kurulu ise servis başına ~1 saat = ~1 gün.

#### 2.2 Shared paketlerin kalan kısmı

- `shared/redis/redis.go` — token blacklist, refresh store, rate limit (Redis testcontainer)
- `shared/redis/ratelimit.go` — sliding window doğruluğu
- `shared/database/database.go` — PG pool init, retry
- `shared/rabbitmq/publisher.go`, `consumer.go`, `dlq.go`, `connection.go`
  — RabbitMQ testcontainer
- `shared/handler/period_handler.go`, `internal_period_handler.go`,
  `time_handler.go`, `simple_period_handler.go` — semester wizard endpoints
- `shared/repository/period_repository.go`,
  `simple_period_repository.go` — DB testleri
- `shared/logger/http_logger.go` — request logging middleware
- `shared/dto/period_dto.go`, `time_dto.go` — validation rules (basic)

**Effort**: **2 gün** (infra + handler testleri ile birlikte).

---

### Tier 3 — Frontend (büyük eksik)

#### 3.1 Sayfa testleri (RTL + MSW)

Frontend'de **hiç sayfa testi yok**. 50+ sayfa var.

**Kurulum gereksinimi**:
```bash
cd frontend && bun add -D msw
```

`src/test/server.ts` oluştur, MSW handlers tanımla.

**Öncelikli sayfalar** (auth flow ve dashboard):
1. `pages/auth/login.tsx` — geçerli/geçersiz creds, force password change
   redirect, account locked toast
2. `pages/auth/change-password.tsx` — policy validation, success
3. `pages/admin/dashboard.tsx` — multi-service data fetching
4. `pages/student/dashboard.tsx`
5. `pages/teacher/attendance/...` — QR oluşturma akışı

**Effort**: MSW kurulumu + sayfa başına ~30dk = **1 gün** ilk 10 sayfa.

#### 3.2 Komponent testleri

Test edilmemiş kompleks komponentler:
- `components/course-hierarchy-view.tsx` (20KB, tree state)
- `components/enrollment/*` — kayıt flow UI
- `components/grades/*` — score grid, finalize modal
- `components/attendance/*` — QR scanner UI, session list
- `components/meal/*` — reservation calendar, batch picker
- `components/admin/*` — semester wizard form (multi-step)
- `components/layout/*` — sidebar, header
- `components/ui/*` — shadcn türevleri (genelde test edilmez)

**Effort**: kompleks olanlar için ~30-60dk = **1-2 gün**.

#### 3.3 Provider/hook testleri

- `components/providers/*` — theme, query client, auth context
- React Query hook'ları (varsa custom hook'lar)

**Effort**: **2-3 saat**.

---

### Tier 4 — Mobile

#### 4.1 Kalan 5 servis

`services/` altında test edilmeyen:
- `attendanceService.ts`
- `catalogService.ts`
- `enrollmentService.ts`
- `gradesService.ts`
- `mealService.ts`

**Pattern**: `mobile/services/authService.test.ts` örnek alınabilir
(api modülü mock'lu).

**Effort**: servis başına ~30dk = **3-4 saat**.

#### 4.2 Screen + komponent testleri

`mobile/app/`, `mobile/screens/`, `mobile/components/` altı:
- `(auth)/login.tsx` — login screen
- `(tabs)/...` — tab navigation
- `(staff)/...` — staff/teacher screens
- `screens/...` — feature screens
- Kritik komponentler: `QRScannerModal.tsx` (handledRef lock),
  Toast (race fix doğrulaması)

**Sorun**: Native modüller (camera, secure-store) mock gerekiyor.
jest-expo preset scope sorunundan dolayı manuel transformIgnorePatterns
ayarı gerek. Detox alternatifi var ama setup ağır.

**Effort**: **1-2 gün** (önce setup düzeltilmeli).

#### 4.3 Hook/context testleri

`AuthContext` (varsa) ve diğer context'ler — RTL Native ile.

**Effort**: **2-3 saat**.

---

### Tier 5 — Cross-cutting / E2E

#### 5.1 SQL migration testleri

Her servisin migration'ları sırayla uygulanabilir mi? Up + Down idempotent mi?

**Test komutu önerisi**:
```bash
# Per service
goose -dir sql/migrations postgres "$DB_URL" up
goose -dir sql/migrations postgres "$DB_URL" down
```

Bunu CI'a (`migrations-check` job'unun yanına) ekle.

**Effort**: **2-3 saat**.

#### 5.2 Service-to-service contract testleri

Pact veya benzer ile event contract'ları doğrula:
- `catalog.semester.created` → enrollment-service
- `student.created` → auth-service
- `grade.student.prerequisite.passed` → enrollment-service

**Etki**: event DTO mismatch'leri (bk. CLAUDE.md memory note) yakalanır.

**Effort**: **1 gün**.

#### 5.3 E2E testleri

- Frontend: Playwright (login → enroll → grade flow)
- Mobile: Detox (auth + QR scan)

**Effort**: **2-3 gün** (kurulum dahil).

#### 5.4 Load tests

k6 veya Vegeta ile login + enrollment endpoint'lerinde performans baseline.

**Effort**: **4-6 saat**.

---

## Başlangıç önerisi (yeni context window için)

### Hızlı kazanım (yarım gün)
1. **Mobile kalan 5 servis** (Tier 4.1) — pattern hazır, copy-paste ile bitebilir
2. **Frontend kritik sayfa testleri** (Tier 3.1) — MSW kur, login + change-password yeter

### Asıl iş (1 hafta)
1. **Testcontainers harness** kur (Tier 1.1 + 2.1 + 2.2 birleştirilmiş)
2. Auth-service tam integration suite — template olarak
3. 8 servise yay
4. Worker testleri (Tier 1.3)

### Sonra
- Frontend sayfa coverage'ı
- E2E (Tier 5.3) — son aşama

---

## Mevcut sistem doğrulama

```bash
# Backend (race detector + tüm servisler)
make test-backend          # ~1dk

# Frontend (Vitest)
make test-frontend         # ~2sn

# Mobile (Jest)
make test-mobile           # ~1sn

# Hepsi birlikte
make test                  # ~1dk

# Coverage
make test-coverage         # backend coverage özet
```

**Bilinen kapsam dışı durumlar**:
- `INTEGRATION_TESTS=true` env olmadan auth-service ve student-service
  integration testleri skip oluyor (bilinçli; Postgres+Redis+RabbitMQ gerekiyor)
- Frontend'de `import.meta.env.VITE_API_BASE_URL` testlerde
  `http://api.test`'e set'li (vitest.config.ts)
- Mobile'da Expo runtime test ortamında yok; service/lib unit testleri için
  babel-jest + Node env yeterli

---

## İlgili dosyalar

- `Makefile` — test hedefleri (line 70+)
- `.github/workflows/ci.yml` — frontend job (line 269), mobile job (line 297)
- `frontend/vitest.config.ts` — Vitest kurulumu
- `mobile/package.json` jest bloğu — test config
- `mobile/jest.setup.js` — SecureStore mock
- `mobile/__mocks__/react-native.js` — Platform mock

## CLAUDE.md memory referansları

- pgtype conversion: her zaman `shared/utils/pgtype_helpers.go` kullan
- Event DTO: `course.semester` event'i `semester_course_id` (course_id değil)
  ve `instructor_fullname` (instructor_name değil) gönderiyor
- ScheduleSessionDTO: `session_type` field'ı zorunlu ("theory" | "lab")
- Frontend: `bun` kullan, npm/npx değil
