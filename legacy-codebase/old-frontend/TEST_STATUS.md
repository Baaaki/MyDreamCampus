# Test Status & AI Handoff Briefing

> Bu dokümanın tek amacı: yeni bir AI context window'unda asistanın **hiç tahmin yürütmeden** sıradaki işi alıp uygulayabilmesi.
> Her bölüm self-contained — kod örnekleri kopyala-yapıştır seviyesinde.
>
> Son audit: 2026-04-28
> İlgili: [TEST_PLAN.md](TEST_PLAN.md) (eski detaylı teknik plan, kısmen güncel değil — bu doküman onu öncelikler açısından geçer).
>
> **2026-04-28 güncellemesi**: 7 yapılması gereken iş tamamlandı (Section 9). Backend test sayısı 71 → 89 dosya, ~612 → 846 atomic test. CI'a `backend-integration` ve `backend-coverage` job'ları eklendi.

---

## 1. TL;DR

**Sayılar**: 89 backend + 8 frontend + 9 mobil test dosyası, ~927 atomic check. `make test` yeşil.
**Tamamlandı (2026-04-28)**: CI integration env, coverage gate, 8 producer payload literal extract'i, 7 servisin handler validation testleri, 5 servisin worker (event consumer) JSON contract testleri, 5 servisin service-layer pure helper testleri, mobil QRScannerModal race fix regresyonu.
**Hâlâ açık**: enrollment-service capacity transaction için **testcontainers** tabanlı gerçek integration testi (Section 3.3, 1-2 gün). Mevcut `tests/integration_test.go` dosyaları HTTP-based smoke; yeni CI job'ı bunları çalıştırıyor ama "race condition'ı yakalayan capacity transaction testi" hâlâ yazılmadı.
**Frontend**: 25+ sayfanın 23'ü hâlâ test edilmemiş (Section 3.5).

---

## 2. Mevcut Durum

### 2.1 Sayılar

```
Backend:     89 test dosyası | shared 224 + services 622 = 846 atomic check
Frontend:     8 test dosyası | 46 test
Mobile:       9 test dosyası | 81 test

Test/Production LOC oranı:
  Backend:   yaklasik 25-28% (yeni helperlar + worker + handler testleri ile)
  Frontend:   ~2%   ← hala cok zayif (pages still untouched)
  Mobile:     ~9%   ← qr-payload helperlari + scan gate eklendi
```

### 2.2 Statement Coverage (gerçek `make test-coverage` çıktısı)

| Modül | Coverage | Yorum |
|---|---|---|
| `shared/clock`, `shared/rules` | 100% | Tam |
| `shared/middleware`, `errors`, `client`, `utils`, `semester` | 85-93% | Çok iyi |
| `shared/audit`, `config`, `logger` | 57-67% | Yeterli |
| `shared/handler` | **5%** | period/time admin endpoint'leri test dışı |
| `shared/database`, `rabbitmq`, `redis`, `repository` | **0%** | testcontainer gerek |
| **grades-service** | **8.1%** | En yüksek |
| payment-service | 5.1% | mock + clock injection |
| course-catalog-service | 4.3% | DTO + validation + group_sessions + event contract |
| meal-service | 2.9% | reservation helpers + DTO |
| student-service | 1.8% | DTO + CSV import helpers |
| auth-service | 1.5% | service mock'lu + handler + integration (CI'da kapalı!) |
| attendance-service | 0.8% | qr_service + DTO + event contract |
| **enrollment-service** | **0%** | Hiç service test'i yok |
| **staff-service** | **0%** | Hiç service test'i yok |

### 2.3 Test Piramidi

```
       __         E2E:         0
      /  \        Integration: var ama CI'da KAPALI
     /    \       Handler:     sadece auth-service
    /      \      Service:     %0-8
   /_      _\     Repository:  %0
   /--------\    DTO/Util:     %85+
```

Test yatırımının %70'i en az risk barındıran utility katmana konsantre. Asıl bug yüzeyi kontrolsüz.

### 2.4 Bilinen Contract Drift'leri (henüz çözülmemiş)

#### A. `grade.student.prerequisite.passed` — producer/consumer asimetrisi
- **Producer**: `backend/services/grades-service/internal/service/grade_service.go:511-528` — `data` altında WRAPPED yayınlıyor
- **Consumer (enrollment)**: `backend/services/enrollment-service/internal/dto/event_dto.go:GradeStudentPrerequisitePassedEvent` — top-level FLAT bekliyor
- **Production'da çalışıyor çünkü**: enrollment worker'ı manuel mapping yapıyor (`event_consumer.go`)
- **Risk**: Worker basitleştirilirse sessizce sıfır değer alır
- **Mevcut koruma**:
  - Producer-side: `backend/services/grades-service/internal/dto/event_dto_test.go:TestGradeStudentPrerequisitePassedEvent_PublishedShape`
  - Consumer-side (NEW 2026-04-28): `backend/services/enrollment-service/internal/worker/event_parser_test.go:TestGradeStudentPrerequisitePassedEvent_FlatEnvelopeParses` — flat şekli pinliyor

#### B. `student.created` — DTO'larda farklı yorum
- **Producer (student-service)**: data altında `id`
- **auth, attendance consumer'ları**: `data.id` ✅
- **enrollment consumer DTO**: top-level `student_id` (yanlış görünüyor) ama worker düzeltiyor

#### C. Test edilmemiş `map[string]any` payload literal'leri — TÜMÜ ÇÖZÜLDÜ ✅ (2026-04-28)

| Dosya | Yeni helper | Test |
|---|---|---|
| student-service `CreateStudent` | `buildStudentCreatedPayload` | `event_payloads_test.go:TestBuildStudentCreatedPayload_ContractKeys` |
| student-service `UpdateStudent` | `buildStudentUpdatedPayload` | `event_payloads_test.go:TestBuildStudentUpdatedPayload_ContractKeys` |
| student-service `DeleteStudent` | `buildStudentDeactivatedPayload` | `event_payloads_test.go:TestBuildStudentDeactivatedPayload_ContractKeys` |
| enrollment-service `ApproveEnrollmentProgram` | `buildEnrollmentApprovedPayload` | `event_payloads_test.go:TestBuildEnrollmentApprovedPayload_WrappedShape` |
| enrollment-service `RejectEnrollmentProgram` | `buildEnrollmentRejectedPayload` | `event_payloads_test.go:TestBuildEnrollmentRejectedPayload_FlatShape` |
| enrollment-service `CreateEnrollmentProgram` (cancel) | `buildEnrollmentCancelledPayload` | `event_payloads_test.go:TestBuildEnrollmentCancelledPayload_FlatShape` |
| enrollment-service `CreateEnrollmentProgram` (submit) | `buildEnrollmentSubmittedPayload` | `event_payloads_test.go:TestBuildEnrollmentSubmittedPayload_FlatShape` |
| enrollment-service `CancelMyEnrollment` | (paylaşılan `buildEnrollmentCancelledPayload`) | aynı test, manual variant |
| course-catalog `course.semester.created` | `buildCourseSemesterCreatedPayload` (2026-04-27) | `handler/event_contract_test.go` |
| staff-service `CreateStaff` (BONUS) | `buildStaffCreatedPayload` | `event_payloads_test.go:TestBuildStaffCreatedPayload_ContractKeys` |
| staff-service `UpdateStaff` (BONUS) | `buildStaffUpdatedPayload` | `event_payloads_test.go:TestBuildStaffUpdatedPayload_ContractKeys` |
| staff-service `DeleteStaff` (BONUS) | `buildStaffDeactivatedPayload` | `event_payloads_test.go:TestBuildStaffDeactivatedPayload_ContractKeys` |

### 2.5 Test Edilmemiş Risk Yüzeyleri (öncelik sırası ile, 2026-04-28 güncel)

1. `enrollment-service` — `CreateEnrollmentProgram` capacity transaction RACE — **HÂLÂ AÇIK** (validators_test.go pure-validation kısmını kapatıyor; transaction katmanı testcontainers ile yapılmalı, Section 3.3)
2. `enrollment-service` — `ApproveEnrollmentProgram` (advisor doğrulama + outbox event) — payload contract KAPALI ✅, advisor mismatch path açık
3. `attendance-service` — `FinalizeAttendance` failure-merge mantığı KAPALI ✅ (`attendance_finalize_helpers_test.go`); orchestration (DB + event publish) hâlâ açık
4. `attendance-service` — `ScanQR` (QR validation + idempotency) — qr_service zaten test ediliyor; idempotency Redis'e bağlı, açık
5. `meal-service` — `CreateBatchReservation` transaction conflict — açık (validateMealTime* helperları pin'lendi ✅)
6. `grades-service` — `AutoFinalize` orchestration açık (helper fonksiyonlar zaten pin'liydi)
7. `staff-service` — service-layer payload helperları + handler validation KAPALI ✅; orchestration hâlâ açık
8. **Frontend** — 25+ sayfa, hâlâ sadece login + change-password testli (Section 3.5 hâlâ açık)
9. **Mobile** — `parseQRPayload` + `ScanGate` KAPALI ✅; tam screen render testleri RNTL kurulmadığı için hâlâ açık
10. Worker/event consumer'lar — 5 servisin parsing/dispatch tarafı KAPALI ✅; gerçek mesaj akışını test eden uçtan-uca worker testleri integration'a girer

---

## 3. Yapılacaklar (sırayla, her biri self-contained)

### 3.1 CI'da integration test env'ini açmak

**Önemi**: `INTEGRATION_TESTS=true` env hiçbir yerde set'lenmiyor. Mevcut [auth-service/tests/integration_test.go](backend/services/auth-service/tests/integration_test.go) ve [student-service/tests/integration_test.go](backend/services/student-service/tests/integration_test.go) **pratikte ölü kod**.

**Yapılacak**: `.github/workflows/ci.yml` içindeki `backend-test` job'ı (line 127-152) aşağıdaki ile değiştir:

```yaml
  backend-test:
    needs: changes
    if: needs.changes.outputs.backend == 'true'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: backend

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: postgres
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U postgres"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10

      redis:
        image: redis:7
        ports: ["6379:6379"]
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10

      rabbitmq:
        image: rabbitmq:3-management
        env:
          RABBITMQ_DEFAULT_USER: rabbitmq
          RABBITMQ_DEFAULT_PASS: rabbitmq
        ports: ["5672:5672"]
        options: >-
          --health-cmd "rabbitmq-diagnostics -q ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 10

    env:
      INTEGRATION_TESTS: "true"
      TEST_DB_URL: "postgresql://postgres:postgres@localhost:5432/mydreamcampus_test?sslmode=disable"
      REDIS_ADDR: "localhost:6379"
      RABBITMQ_URL: "amqp://rabbitmq:rabbitmq@localhost:5672/"

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: backend/**/go.sum

      - name: Create test databases
        run: |
          for db in mydreamcampus_auth_test mydreamcampus_student_test mydreamcampus_test; do
            PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE $db;" || true
          done

      - name: Apply migrations (auth-service)
        run: |
          go install github.com/pressly/goose/v3/cmd/goose@latest
          (cd services/auth-service && goose -dir sql/migrations postgres "postgresql://postgres:postgres@localhost:5432/mydreamcampus_auth_test?sslmode=disable" up)

      - name: Apply migrations (student-service)
        run: |
          (cd services/student-service && goose -dir sql/migrations postgres "postgresql://postgres:postgres@localhost:5432/mydreamcampus_student_test?sslmode=disable" up)

      - name: Test shared
        run: cd shared && go test -race -count=1 ./...

      - name: Test all services
        run: |
          for dir in services/*/; do
            svc=$(basename "$dir")
            echo "::group::test ${svc}"
            (cd "$dir" && go test -race -count=1 ./...)
            echo "::endgroup::"
          done
```

**Doğrulama**: PR aç, CI'da `backend-test` job'ı içinde `--- PASS: TestIntegration` satırlarını görmek lazım.

**Maliyet**: 1-2 saat (iterasyon dahil).

---

### 3.2 Coverage threshold gate

**Önemi**: Coverage düşse CI hiç uyarmıyor. %1'e düşse fark etmez.

**Yapılacak**: `.github/workflows/ci.yml`'a `backend-test` job'ından sonra yeni job ekle:

```yaml
  backend-coverage:
    needs: changes
    if: needs.changes.outputs.backend == 'true'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: backend
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: backend/**/go.sum

      - name: Run coverage
        run: |
          set -e
          (cd shared && go test -race -count=1 -coverprofile=coverage.out ./...)
          for dir in services/*/; do
            (cd "$dir" && go test -race -count=1 -coverprofile=coverage.out ./...)
          done

      - name: Check shared/ coverage
        run: |
          cd shared
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          threshold=40  # şu an 46.1%, regresyon olmasın
          echo "shared coverage: $coverage%, threshold: $threshold%"
          awk -v c="$coverage" -v t="$threshold" 'BEGIN { if (c+0 < t+0) exit 1 }'
```

**Threshold önerisi (initial)**:
- `shared/`: 40% (şu an 46.1%)
- Backend overall: gate yok başlangıçta — sadece shared'da sızdırmazlık
- 3 ay sonra: `grades-service` 5%, sonra her servise %5 minimum
- 6 ay sonra: tüm servisler %15+

**Maliyet**: 1 saat.

---

### 3.3 enrollment-service `CreateEnrollmentProgram` integration testi

**Önemi**: Capacity decrement transaction projedeki en pahalı bug noktası. Aynı anda iki öğrenci son boş kontenjanı isterse — bu test olmadan duplicate'i yakalayamayız.

**Yapılacak**: testcontainers-go ile harness kur.

**Adım 1**: Bağımlılık ekle.

```bash
cd backend/services/enrollment-service
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go mod tidy
```

**Adım 2**: `backend/services/enrollment-service/tests/integration_test.go` oluştur.

```go
package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment-service/config"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/db"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/dto"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/repository"
	"github.com/baaaki/mydreamcampus/enrollment-service/internal/service"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type EnrollmentIntegrationSuite struct {
	suite.Suite
	pool      *pgxpool.Pool
	container *postgres.PostgresContainer
	enrollSvc *service.EnrollmentService
}

func (s *EnrollmentIntegrationSuite) SetupSuite() {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		s.T().Skip("INTEGRATION_TESTS=true ile çalıştırılmalı")
	}

	ctx := context.Background()

	c, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16"),
		postgres.WithDatabase("test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	s.Require().NoError(err)
	s.container = c

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Migrations
	sqlDB, err := sql.Open("pgx", dsn)
	s.Require().NoError(err)
	defer sqlDB.Close()
	require.NoError(s.T(), goose.SetDialect("postgres"))
	require.NoError(s.T(), goose.Up(sqlDB, "../sql/migrations"))

	pool, err := pgxpool.New(ctx, dsn)
	s.Require().NoError(err)
	s.pool = pool

	require.NoError(s.T(), logger.Init("test"))

	cfg := &config.Config{} // capacity check için cfg.MaxCoursesPerEnrollment dolu olmalı
	cfg.Enrollment.MaxCoursesPerEnrollment = 8

	enrollRepo := repository.NewEnrollmentRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	cacheRepo := repository.NewCacheRepository(pool)
	s.enrollSvc = service.NewEnrollmentService(enrollRepo, studentRepo, cacheRepo, cfg)
}

func (s *EnrollmentIntegrationSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(context.Background())
	}
}

func (s *EnrollmentIntegrationSuite) SetupTest() {
	// Her test öncesi tabloları temizle
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, `
		TRUNCATE enrollment_programs, semester_courses_cache, students_cache RESTART IDENTITY CASCADE;
	`)
	s.Require().NoError(err)
}

func TestEnrollmentIntegrationSuite(t *testing.T) {
	suite.Run(t, new(EnrollmentIntegrationSuite))
}
```

**Adım 3**: Capacity transaction testini ekle (aynı dosyaya):

```go
func (s *EnrollmentIntegrationSuite) TestCreateEnrollmentProgram_DecrementsCapacityAtomically() {
	ctx := context.Background()
	courseID := uuid.New()

	// Seed: kapasitesi 1 olan bir kurs
	_, err := s.pool.Exec(ctx, `
		INSERT INTO semester_courses_cache
		  (id, course_code, course_name, semester, max_capacity, current_enrolled, class_level, department, instructor_fullname)
		VALUES
		  ($1, 'CS101', 'Intro', '2025-2026-Fall', 1, 0, 1, 'CS', 'Jane')
	`, courseID)
	s.Require().NoError(err)

	// İki öğrenci
	studentA := uuid.New()
	studentB := uuid.New()
	for _, id := range []uuid.UUID{studentA, studentB} {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO students_cache (id, student_number, department, class_level, is_active)
			VALUES ($1, $2, 'CS', 1, true)
		`, id, "20210"+id.String()[:3])
		s.Require().NoError(err)
	}

	// Aynı anda iki istek (errgroup veya basit sync.WaitGroup ile)
	type result struct {
		studentID uuid.UUID
		err       error
	}
	results := make(chan result, 2)

	for _, sid := range []uuid.UUID{studentA, studentB} {
		go func(id uuid.UUID) {
			_, err := s.enrollSvc.CreateEnrollmentProgram(ctx, dto.CreateEnrollmentRequest{
				StudentID: id,
				Semester:  "2025-2026-Fall",
				CourseIDs: []uuid.UUID{courseID},
			})
			results <- result{id, err}
		}(sid)
	}

	r1, r2 := <-results, <-results

	// Tam olarak biri başarılı olmalı
	successCount := 0
	if r1.err == nil { successCount++ }
	if r2.err == nil { successCount++ }
	s.Equal(1, successCount, "tam olarak 1 öğrenci enrollment yapabilmeli, oldu: %d", successCount)

	// DB'de current_enrolled = 1 olmalı (>1 değil)
	var enrolled int
	err = s.pool.QueryRow(ctx,
		`SELECT current_enrolled FROM semester_courses_cache WHERE id = $1`, courseID,
	).Scan(&enrolled)
	s.Require().NoError(err)
	s.Equal(1, enrolled, "race condition: current_enrolled=%d, beklenen 1", enrolled)
}
```

**Adım 4**: Çalıştır.

```bash
cd backend/services/enrollment-service
INTEGRATION_TESTS=true go test -race -count=1 ./tests/...
```

**Maliyet**: 1-2 gün (harness + bu test). Sonraki testler bu harness üstüne yarım saatte yazılır.

---

### 3.4 Producer payload literal'lerini extract + test

**Pattern**: Method içindeki `map[string]any` literal'ini package-private helper fonksiyonuna çıkar, helper'ı test et.

**Örnek (student.created için)**:

#### Önce — `backend/services/student-service/internal/service/student_service.go:105`

```go
func (s *StudentService) CreateStudent(ctx context.Context, req dto.CreateStudentRequest) (...) {
    // ...
    eventPayload := map[string]any{
        "id":              nil,
        "student_number":  req.StudentNumber,
        "first_name":      req.FirstName,
        // ...
        "status":          "active",
    }
    // ...
}
```

#### Sonra — yeni helper file: `backend/services/student-service/internal/service/event_payloads.go`

```go
package service

import "github.com/baaaki/mydreamcampus/student-service/internal/dto"

// buildStudentCreatedPayload assembles the flat data for the student.created
// outbox event. The outbox worker wraps this under {event_id, event_type,
// timestamp, data} before publishing.
//
// Keys here are part of the wire contract. Renaming them silently breaks
// every consumer (auth, attendance, enrollment, grades) — see
// event_payloads_test.go.
func buildStudentCreatedPayload(req dto.CreateStudentRequest) map[string]any {
    return map[string]any{
        "id":              nil, // CreateStudentWithEvent overwrites after insert
        "student_number":  req.StudentNumber,
        "first_name":      req.FirstName,
        "last_name":       req.LastName,
        "email":           req.Email,
        "faculty":         req.Faculty,
        "department":      req.Department,
        "enrollment_year": req.EnrollmentYear,
        "class_level":     req.ClassLevel,
        "status":          "active",
    }
}
```

#### Caller'da değişiklik — `student_service.go:105`

```go
eventPayload := buildStudentCreatedPayload(req)
```

#### Test — yeni dosya: `backend/services/student-service/internal/service/event_payloads_test.go`

```go
package service

import (
    "encoding/json"
    "testing"

    "github.com/baaaki/mydreamcampus/student-service/internal/dto"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestBuildStudentCreatedPayload_ContractKeys(t *testing.T) {
    payload := buildStudentCreatedPayload(dto.CreateStudentRequest{
        StudentNumber:  "20210001",
        FirstName:      "Ahmet",
        LastName:       "Yilmaz",
        Email:          "a@univ.edu",
        Faculty:        "Engineering",
        Department:     "CS",
        EnrollmentYear: 2021,
        ClassLevel:     2,
    })

    // Her downstream consumer'ın okuduğu anahtarlar — silinemez.
    required := []string{
        "id", "student_number", "first_name", "last_name", "email",
        "faculty", "department", "enrollment_year", "class_level", "status",
    }
    for _, k := range required {
        assert.Contains(t, payload, k, "wire contract: anahtar %q kalkamaz", k)
    }

    // Field değerleri input'tan birebir gelmeli
    assert.Equal(t, "20210001", payload["student_number"])
    assert.Equal(t, "Ahmet", payload["first_name"])
    assert.Equal(t, "active", payload["status"])
    assert.Nil(t, payload["id"], "id repository'de doldurulur")
}

func TestBuildStudentCreatedPayload_JSONRoundTrip(t *testing.T) {
    // Outbox worker bunu wrap eder ve marshal eder. Burada smoke check.
    payload := buildStudentCreatedPayload(dto.CreateStudentRequest{
        StudentNumber: "X", FirstName: "X", LastName: "X", Email: "X",
        Faculty: "X", Department: "X", EnrollmentYear: 2021, ClassLevel: 1,
    })
    raw, err := json.Marshal(payload)
    require.NoError(t, err)

    var decoded map[string]any
    require.NoError(t, json.Unmarshal(raw, &decoded))
    assert.Equal(t, "X", decoded["student_number"])
}
```

**Aynı pattern'i 8 literal için tekrarla** (Section 2.4.C tablosu).

**Maliyet**: Literal başına ~30dk = 8 × 30dk = **4 saat**.

**Pattern referansı**: [course-catalog-service/internal/handler/event_contract_test.go](backend/services/course-catalog-service/internal/handler/event_contract_test.go) (catalog tarafında zaten yapılmış).

---

### 3.5 Frontend MSW kurulumu + öncelikli page testleri

**Önemi**: 25+ page var, sadece 2'si test edilmiş. %2 LOC oranı kabul edilemez.

#### Adım 1: MSW kur

```bash
cd frontend
bun add -D msw@latest
```

#### Adım 2: `frontend/src/test/server.ts` oluştur

```ts
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";

// Default handler'lar — testler bunları override eder.
const baseURL = "http://api.test";

export const defaultHandlers = [
  http.post(`${baseURL}/api/auth/refresh`, () =>
    HttpResponse.json({ access_token: "new-token" }),
  ),
];

export const server = setupServer(...defaultHandlers);
```

#### Adım 3: `frontend/src/test/setup.ts`'a MSW lifecycle ekle

```ts
import "@testing-library/jest-dom/vitest";
import { afterAll, afterEach, beforeAll } from "vitest";
import { server } from "./server";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

#### Adım 4: Öncelikli page'lerin sırası (impact'e göre)

| Sıra | Page | Path | Neden öncelikli |
|---|---|---|---|
| 1 | Admin Dashboard | `frontend/src/pages/admin/dashboard/index.tsx` | Multi-service fetch, en görünür sayfa |
| 2 | Student Dashboard | `frontend/src/pages/student/dashboard/index.tsx` | Hot path |
| 3 | Teacher Attendance | `frontend/src/pages/teacher/attendance/index.tsx` | QR oluşturma |
| 4 | Student Enrollment | `frontend/src/pages/student/enrollment/index.tsx` | Capacity check UI |
| 5 | Teacher Grades | `frontend/src/pages/teacher/grades/index.tsx` | Score grid |

#### Adım 5: Admin Dashboard test örneği

`frontend/src/pages/admin/dashboard/index.test.tsx` oluştur:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { MemoryRouter } from "react-router-dom";
import AdminDashboard from "./index";
import { server } from "@/test/server";

const baseURL = "http://api.test";

function renderPage() {
  return render(
    <MemoryRouter>
      <AdminDashboard />
    </MemoryRouter>,
  );
}

describe("admin dashboard", () => {
  it("fetches and renders student/staff/course counts", async () => {
    server.use(
      http.get(`${baseURL}/api/admin/students/count`, () =>
        HttpResponse.json({ total: 1234 }),
      ),
      http.get(`${baseURL}/api/admin/staff/count`, () =>
        HttpResponse.json({ total: 56 }),
      ),
      http.get(`${baseURL}/api/admin/catalog/count`, () =>
        HttpResponse.json({ total: 89 }),
      ),
    );

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("1234")).toBeInTheDocument();
      expect(screen.getByText("56")).toBeInTheDocument();
      expect(screen.getByText("89")).toBeInTheDocument();
    });
  });

  it("shows error state if any endpoint fails", async () => {
    server.use(
      http.get(`${baseURL}/api/admin/students/count`, () =>
        HttpResponse.error(),
      ),
    );

    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/yüklenirken hata/i)).toBeInTheDocument();
    });
  });
});
```

**NOT**: Kart text'lerinin Türkçe/İngilizce olduğunu component'i okuyup ayarla.

**Maliyet**: MSW kurulumu 1 saat, ilk 5 page 30dk-1 saat/page = **1 gün**.

**Pattern referansı**: [frontend/src/pages/auth/login/index.test.tsx](frontend/src/pages/auth/login/index.test.tsx) zaten var.

---

### 3.6 Mobile screen testleri

**Önemi**: `mobile/services/` ve `mobile/lib/` test edilmiş ama **screen render hiç test edilmemiş**. Özellikle [mobile/components/QRScannerModal.tsx](mobile/components/QRScannerModal.tsx) — `handledRef` lock race condition fix'i regresyon koruması olmadan duruyor.

#### Adım 1: Native modül mock'larını genişlet

`mobile/__mocks__/expo-camera.js` oluştur:

```js
module.exports = {
  CameraView: ({ children }) => children,
  useCameraPermissions: () => [
    { granted: true, canAskAgain: true },
    jest.fn().mockResolvedValue({ granted: true }),
  ],
};
```

`mobile/jest.setup.js` dosyasının sonuna ekle:

```js
jest.mock("expo-camera");
jest.mock("expo-haptics", () => ({
  notificationAsync: jest.fn(),
  NotificationFeedbackType: { Success: "success", Error: "error" },
}));
```

#### Adım 2: Öncelikli screen'ler

| Sıra | Screen | Path | Neden |
|---|---|---|---|
| 1 | QRScannerModal | `mobile/components/QRScannerModal.tsx` | Race fix regression — kritik |
| 2 | Auth Login | `mobile/app/(auth)/login.tsx` | Hot path |
| 3 | Student Tab | `mobile/app/(tabs)/index.tsx` | Default landing |
| 4 | Staff Attendance | `mobile/app/(staff)/attendance.tsx` (kontrol et) | QR generation |

#### Adım 3: QRScannerModal test örneği

`mobile/components/QRScannerModal.test.tsx` oluştur:

```tsx
import React from "react";
import { render, fireEvent, waitFor } from "@testing-library/react-native";
import { QRScannerModal } from "./QRScannerModal";

describe("QRScannerModal", () => {
  it("only fires onScan once even when scanner emits multiple times", async () => {
    // Race fix: aynı QR ardı ardına okutulursa handledRef lock'u devreye giriyor.
    const onScan = jest.fn();
    const { getByTestId } = render(
      <QRScannerModal visible={true} onClose={jest.fn()} onScan={onScan} />,
    );

    // Camera mock'ı dışarıdan event simüle eder — gerçek implementasyona göre
    // bu kısım mock'un nasıl scan event üreteceğine bağlı; QRScannerModal'in
    // testid kullandığı Camera prop'unu manuel tetikle.
    const scanner = getByTestId("qr-scanner");
    fireEvent(scanner, "barCodeScanned", { data: "qr-payload-123" });
    fireEvent(scanner, "barCodeScanned", { data: "qr-payload-123" });
    fireEvent(scanner, "barCodeScanned", { data: "qr-payload-123" });

    await waitFor(() => {
      expect(onScan).toHaveBeenCalledTimes(1);
      expect(onScan).toHaveBeenCalledWith("qr-payload-123");
    });
  });
});
```

**NOT**: `testID="qr-scanner"` `QRScannerModal` içinde yoksa, ekle (production değişikliği değil — test marker).

**Maliyet**: Mock setup 30dk + screen başına 30-45dk = **3-4 saat**.

**Pattern referansı**: [mobile/services/api.test.ts](mobile/services/api.test.ts) (mock pattern).

---

### 3.7 Worker / event consumer testleri

**Önemi**: 5 servisin RabbitMQ tüketim akışı test edilmemiş. CLAUDE.md memory'sindeki bug class'ı tam olarak burada doğdu.

**Yaklaşım**: `handleMessage` fonksiyonunu izole et — mock repo + mock event ile test et.

**Her bir servis için pattern**:
1. Worker'ın `handleXxx` fonksiyonunu çağır (body []byte input ile)
2. Mock repo'yu beklenen call ile beklemeye al
3. JSON literal contract'a göre verify et

**Sıra**:
1. attendance-service worker — [event_consumer.go](backend/services/attendance-service/internal/worker/event_consumer.go)
2. enrollment-service worker — [event_consumer.go](backend/services/enrollment-service/internal/worker/event_consumer.go) (manual mapping logic'i çok)
3. grades-service worker
4. auth-service worker
5. student-service worker

**Maliyet**: Servis başına ~3 saat = **2 gün**.

**Pattern referansı**: [auth-service/internal/handler/auth_handler_test.go](backend/services/auth-service/internal/handler/auth_handler_test.go) (mock pattern olarak).

---

### 3.8 Repository testleri

**Önemi**: 9 servisin sqlc-generated repo katmanı test edilmemiş. Custom query'lerin (özellikle transaction'lı olanlar) davranışı kontrolsüz.

**Yaklaşım**: Section 3.3'teki testcontainer harness'i paylaş.

**Sıra**: Capacity-critical repo'lardan başla:
1. `enrollment-service/internal/repository/enrollment_repository.go` — `UpdateProgramStatus`, `IncrementCourseCapacity` transaction'ları
2. `attendance-service/internal/repository/session_repository.go` — finalize state transitions
3. `meal-service/internal/repository/reservation_repository.go` — batch insert + unique constraint

**Maliyet**: testcontainer harness varsa servis başına ~1 saat = **1-2 gün**.

---

### 3.9 Coverage milestone'lar

```
Şu an:        Backend ortalaması ~3%
3 ay sonra:   Tüm servisler %15+ (Section 3.3-3.7 yapılmış)
6 ay sonra:   Tüm servisler %35+ (Section 3.8 + handler testleri)
```

---

## 4. Anti-pattern Uyarıları

### 4.1 Chesterton's Fence — test silmeden önce sorgula

Önceki AI bir aralık "%10 test gereksiz" deyip silmek istedi. Tek tek sorgulayınca:

| Test | İlk yargı | Gerçek sebep |
|---|---|---|
| `groupScheduleSessions...preserves duplicate slot numbers (no dedup)` | "İstemediğimiz davranışı pinliyor" | Upstream data integrity bug'ını gizlememek için savunma |
| `validateScheduleSessionTypes...accepts no sessions when both hours are zero` | "Bug yakalamayı engelliyor" | Layer separation — başka katmanın işi |
| `parseInterfaceToFloat64 default branch (bool → 0)` | "Error fırlatması daha iyi" | Hot path "asla panic etme" sözleşmesi |
| `ValidateDayOfWeek 7 ayrı sub-test` | "Verbose" | **Doğru** — sadece stilistik |

**Kural**: Bir testi silmeden önce somut olarak "bu testin yokluğunda hangi davranış sessizce gerileyebilir?" cevabını verebilmen lazım. Cevap yoksa sil. Cevap varsa, comment ile "neden var" notu ekle.

### 4.2 LSP diagnostic noise — gerçek değil

IDE LSP'si `github.com/stretchr/testify is not in your go.mod file (go mod tidy)` gibi uyarılar üretiyor. Bunlar **stale cross-module noise**. Gerçek `go test` her şey yeşil. Bu uyarılara güvenme — `go test` veya `make test` kullan.

### 4.3 Test çıktısı RTK proxy ile filtreleniyor

`make test` doğrudan çalıştırıldığında çıktı kısıtlı. Tam görmek için:

```bash
rtk proxy make test
```

Sadece UI sorunu, test'lerin kendisi etkilenmiyor.

### 4.4 Frontend `bun` zorunlu

CLAUDE.md memory: `bun tsc`, `bun run test`, `bun add -D <pkg>`. `npm`/`npx` **kullanma**.

### 4.5 Docker komutlarını çalıştırma

CLAUDE.md: Docker komutlarını **kullanıcıya göster**, kendin çalıştırma. `sudo` gerektirir.

### 4.6 enrollment-service ve staff-service'te pure helper aranmaya değmez

Önceki AI tarama yaptı — bu iki serviste **standalone (`func ...`, method değil) pure helper YOK**. Hepsi method (`func (s *Service)`). Service-layer testi için doğrudan integration (Section 3.3) veya mock (auth-service pattern) gerekir.

### 4.7 enrollment-service event_dto.go'daki struct'lar dış sözleşme değil

`backend/services/enrollment-service/internal/dto/event_dto.go` içindeki `StudentCreatedEvent`, `CourseSemesterCreatedEvent` vs. struct'lar **JSON unmarshal target değil**. Worker (`event_consumer.go`) manuel mapping yapıyor; struct'lar internal flow için. Bu struct'ların JSON tag'lerini test etmek **yanlış güvenlik hissi** verir. Worker'daki internal `wrappedEvent`/`studentEventData` tipleri **gerçek** sözleşme.

---

## 5. Pattern Referansları (kopyalanacak örnekler)

### Backend test tipleri

| Test tipi | Referans dosya |
|---|---|
| Pure function helper | [grades-service/.../grade_helpers_test.go](backend/services/grades-service/internal/service/grade_helpers_test.go) |
| Clock-dependent helper | [meal-service/.../reservation_service_helpers_test.go](backend/services/meal-service/internal/service/reservation_service_helpers_test.go) |
| Producer payload contract | [course-catalog-service/.../event_contract_test.go](backend/services/course-catalog-service/internal/handler/event_contract_test.go) |
| Consumer JSON contract | [attendance-service/.../event_contract_test.go](backend/services/attendance-service/internal/dto/event_contract_test.go) |
| Service-level w/ mocks | [auth-service/.../auth_service_test.go](backend/services/auth-service/internal/service/auth_service_test.go) |
| HTTP handler test | [auth-service/.../auth_handler_test.go](backend/services/auth-service/internal/handler/auth_handler_test.go) |
| Integration suite | [auth-service/tests/integration_test.go](backend/services/auth-service/tests/integration_test.go) |
| Outbound event JSON | [grades-service/.../event_dto_test.go](backend/services/grades-service/internal/dto/event_dto_test.go) |
| Table-driven validation | [course-catalog-service/.../validation_test.go](backend/services/course-catalog-service/internal/service/validation_test.go) |
| CSV/security parsing | [student-service/.../import_helpers_test.go](backend/services/student-service/internal/service/import_helpers_test.go) |

### Frontend test tipleri

| Test tipi | Referans dosya |
|---|---|
| Page (login flow) | [frontend/src/pages/auth/login/index.test.tsx](frontend/src/pages/auth/login/index.test.tsx) |
| Page (form validation) | [frontend/src/pages/auth/change-password/index.test.tsx](frontend/src/pages/auth/change-password/index.test.tsx) |
| api-client (CSRF + 401 retry) | [frontend/src/lib/api-client.test.ts](frontend/src/lib/api-client.test.ts) |
| Component (with state) | [frontend/src/components/auth-guard.test.tsx](frontend/src/components/auth-guard.test.tsx) |
| Cross-platform mirror | [frontend/src/lib/password-policy.test.ts](frontend/src/lib/password-policy.test.ts) |

### Mobile test tipleri

| Test tipi | Referans dosya |
|---|---|
| Service (axios mock) | [mobile/services/api.test.ts](mobile/services/api.test.ts) |
| Auth service flow | [mobile/services/authService.test.ts](mobile/services/authService.test.ts) |
| Cross-platform mirror | [mobile/lib/password-policy.test.ts](mobile/lib/password-policy.test.ts) |

---

## 6. Yeni AI İçin İş Akışı

```
1. Bu dokümanı oku.
2. Çalıştır:
   make test                    → her şey yeşil olmalı
   rtk proxy make test-coverage → güncel coverage rakamlarını al
3. Bu dokümandaki Section 2.2 ile karşılaştır. Farklılık varsa Section 2'yi güncelle.
4. Section 3'ten bir adım seç. Sıralama:
   - İlk seçim:    3.1 (CI integration env) — 1-2 saat, ROI çok yüksek
   - İkinci seçim: 3.4 (payload literal extract) — atomik, paralel yapılabilir
   - Üçüncü seçim: 3.3 (enrollment integration) — 1-2 gün, en yüksek değer
5. Tek seferde tek iş. Her commit sonrası `make test`.
6. İş bittiğinde Section 2.4 (drift listesi) ve 2.5 (risk yüzeyleri) güncelle.
   Yeni eklenen testleri Section 5'e referans olarak ekleme şart değil — onlar
   var olan pattern'leri tekrar etmemeli, eğer yeni bir pattern getirdiyse ekle.
```

---

## 7. CLAUDE.md Memory Referansları

- **pgtype dönüşümü**: her zaman `shared/utils/pgtype_helpers.go` kullan
- **Event DTO**: `course.semester` event'i `semester_course_id` ve `instructor_fullname` field'larını kullanır (önceden `course_id` ve `instructor_name`'di — rename geçmişi)
- **ScheduleSessionDTO**: `session_type` ("theory"|"lab") zorunlu
- **Frontend**: `bun` kullan, `npm`/`npx` değil
- **CV/portfolio projesi** — enterprise security/resilience seviyesi gerekmiyor
- **notification servisi** yapılmadı, **payment** mock-only
- **Konuşma Türkçe** kod İngilizce
- **Senior dev verdict**: "Test altyapısı kıdemli mühendis kalitesinde, dağılımı junior tarafından yapılmış gibi. İskelet sağlam, yük taşıyan duvarlar boş."

---

## 8. Bu Doküman Güncel Tutulmalı

İş tamamlandıkça **şu bölümleri güncelle**:
- **Section 1 (TL;DR)** — sayılar, en öncelikli iş
- **Section 2.1 (Sayılar)** — test/LOC, dosya sayısı
- **Section 2.2 (Coverage tablosu)** — yeni `make test-coverage` çıktısından
- **Section 2.4 (Contract drift'leri)** — çözülenler için ✅ ekle
- **Section 2.5 (Risk yüzeyleri)** — yapılanları çıkar
- **Section 3 (Yapılacaklar)** — tamamlananları "yapıldı" olarak işaretle veya kaldır

Pattern referansları (Section 5) genelde sabit kalır — yeni pattern eklenirse oraya ekle.

---

**Doküman versiyon**: 3 (rev 1: ilk handoff; rev 2: self-contained kod örnekleriyle yeniden yazıldı; rev 3 (2026-04-28): bad-sides cleanup tamamlandı)

---

## 9. 2026-04-28 Cleanup Özeti

Bu turda **TEST_STATUS Section 3'teki yapılması gereken işlerin büyük çoğunluğu** ve önceki audit'in işaret ettiği "altyapının kötü yanları" kapatıldı. Her birine ait dosya ve test sayıları:

### 9.1 CI altyapısı
- **`.github/workflows/ci.yml`**: 2 yeni job
  - `backend-integration` — postgres/redis/rabbitmq service container'larıyla `INTEGRATION_TESTS=true` çalıştırıyor; tüm servis binary'leri build edip arka planda başlatıp `tests/integration_test.go` HTTP smoke testlerini koşturuyor
  - `backend-coverage` — `shared/` için %40 coverage floor; her servisin coverage rakamını CI çıktısına yazıyor
- `ci-passed` gate her iki yeni job'ı bekliyor

### 9.2 Producer payload contract'ları (8 literal extract)
- `student-service/internal/service/event_payloads.go` + test → 9 test
- `enrollment-service/internal/service/event_payloads.go` + test → 7 test
- `staff-service/internal/service/event_payloads.go` + test → 5 test (bonus)
- Hepsinin call site'ları helper'a yönlendirildi

### 9.3 Service-layer pure helper testleri
| Servis | Yeni dosya | Test |
|---|---|---|
| enrollment | `validators.go` + test | 13 |
| meal | `reservation_validators_test.go` | 16 |
| attendance | `attendance_finalize_helpers.go` + test | 6 |
| grades | `grade_helpers_test.go` (parseInterfaceToInt64 eklendi) | 7 |

### 9.4 Handler validation harness — 7 servis
Her biri ~100 satır, `gin` binding rules üzerinden DTO contract'ı pinliyor:
- staff_handler_test.go (6 test)
- student_handler_test.go (5 test)
- enrollment_handler_test.go (6 test)
- attendance_handler_test.go (9 test)
- grade_handler_test.go (6 test)
- reservation_handler_test.go (6 test)
- catalog_handler_test.go (8 test)

### 9.5 Worker (event consumer) JSON contract testleri — 5 servis
- attendance: `event_parser.go` (generic `unwrapEventData[T]` helper) + 5 test
- auth: `event_router_test.go` + 4 test
- student: `event_parser_test.go` + 4 test
- enrollment: `event_parser_test.go` + 3 test (CourseSemester FLAT + GradePrereq FLAT)
- grades: `event_parser_test.go` + 4 test (CourseSemester FLAT + EnrollmentApproved WRAPPED)

### 9.6 Mobile race fix regresyonu
- `mobile/lib/qr-payload.ts` — `parseQRPayload` + `createScanGate` extract'i
- `mobile/lib/qr-payload.test.ts` — 14 test (parse: 9, gate: 5)
- `QRScannerModal.tsx` extract'leri kullanacak şekilde refactor edildi

### 9.7 Sayısal etki

```
Önce:                Sonra:
Backend  71 dosya    Backend  89 dosya  (+18)
~612 atomic check   846 atomic check    (+234)
Mobile    8 dosya    Mobile    9 dosya  (+1, +23 test)
```

### 9.8 Hâlâ açık olanlar
1. **Section 3.3** — testcontainers ile gerçek capacity transaction integration testi (1-2 gün; en yüksek değer)
2. **Section 3.5** — Frontend MSW kurulumu + 5 öncelikli sayfa
3. **Section 3.8** — Repository testleri (testcontainers harness paylaşılırsa servis başına ~1 saat)
4. **Mobile screen render testleri** — RNTL/jest-expo preset kurulumu gerekiyor, screen render seviyesinde tam test için
