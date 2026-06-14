# Event & DTO Tutarsizlik Fix Notlari

> **Tarih**: 2026-02-26
> **Kapsam**: Mikroservisler arasi event isimlendirme, payload yapisi ve DTO tutarsizliklari

---

## Bug 1 + 1b: student.deactivated / staff.deactivated Event Uyumsuzlugu

**Sorun**: Shared constants `"student.deleted"` / `"staff.deleted"` diyordu ama repository'ler `"student.deactivated"` / `"staff.deactivated"` yayinliyordu. Consumer'lar shared constant kullandigi icin event'ler eslesmiyor ve kayboluyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/shared/events/events.go` | `EventStaffDeleted` → `EventStaffDeactivated` (`"staff.deactivated"`), `EventStudentDeleted` → `EventStudentDeactivated` (`"student.deactivated"`), ayni sekilde routing key'ler |
| `backend/services/student-service/internal/repository/student_repository.go` | Hardcoded `"student.deactivated"` string'leri yerine `events.EventStudentDeactivated` / `events.RoutingKeyStudentDeactivated` constant'lari, `events` import'u eklendi |
| `backend/services/staff-service/internal/repository/staff_repository.go` | Hardcoded `"staff.deactivated"` string'leri yerine `events.EventStaffDeactivated` / `events.RoutingKeyStaffDeactivated` constant'lari, `events` import'u eklendi |
| `backend/services/student-service/internal/worker/outbox_worker.go` | `EventStudentDeleted` → `EventStudentDeactivated`, `RoutingKeyStudentDeleted` → `RoutingKeyStudentDeactivated` |
| `backend/services/staff-service/internal/worker/outbox_worker.go` | `EventStaffDeleted` → `EventStaffDeactivated`, `RoutingKeyStaffDeleted` → `RoutingKeyStaffDeactivated` |
| `backend/services/auth-service/internal/worker/event_consumer.go` | Switch case: `EventStudentDeleted, EventStaffDeleted` → `EventStudentDeactivated, EventStaffDeactivated` |
| `backend/services/enrollment-service/internal/worker/event_consumer.go` | Switch case: `EventStudentDeleted` → `EventStudentDeactivated` |
| `backend/services/student-service/internal/worker/event_consumer.go` | Switch case: `EventStaffDeleted` → `EventStaffDeactivated`, queue binding: `RoutingKeyStaffDeleted` → `RoutingKeyStaffDeactivated`, log mesaji guncellendi |

---

## Bug 2: student.updated Event Payload Eksik Veri

**Sorun**: Update event payload'inda sadece `id`, bos `student_number` ve `changed_fields` gonderiliyordu. Consumer'lar (enrollment, attendance, grades, meal) tam veri bekliyordu ve bos string'lerle cache'i eziyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/student-service/internal/service/student_service.go` (satir ~285) | `eventPayload` artik `currentStudent` uzerinden tam veri gonderiyor: `id`, `student_number`, `first_name`, `last_name`, `email`, `faculty`, `department`, `enrollment_year`, `class_level`, `advisor_id`, `status`, `changed_fields`. Ayrica `req.Status` ve `req.AdvisorID` varsa guncel deger override ediliyor. |

---

## Bug 3: Attendance-Service Student Event Deserialization

**Sorun 1**: Student-service outbox worker event'i `{event_id, event_type, timestamp, data: {...}}` seklinde wrap ediyordu. Attendance-service body'yi direkt flat struct'a unmarshal ediyordu → tum field'lar zero value.

**Sorun 2**: Attendance DTO'lari `json:"student_id"` kullaniyor ama publisher `"id"` gonderiyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/attendance-service/internal/dto/event_dto.go` | `StudentCreatedEventData`, `StudentUpdatedEventData`, `StudentDeactivatedEventData` struct'larinda json tag `"student_id"` → `"id"` |
| `backend/services/attendance-service/internal/worker/event_consumer.go` | `handleStudentCreated`, `handleStudentUpdated`, `handleStudentDeactivated` fonksiyonlarina two-step unwrap eklendi: once `BaseEvent` parse → `baseEvent.Data`'yi marshal → sonra ilgili `EventData` struct'ina unmarshal |

---

## Bug 7: Enrollment ScheduleSession'da session_type Eksik

**Sorun**: Catalog service `session_type` gonderiyordu ama enrollment DTO'larinda bu field yoktu, kayboluyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/enrollment-service/internal/dto/event_dto.go` | `CourseSession` struct'ina `SessionType string \`json:"session_type"\`` eklendi |
| `backend/services/enrollment-service/internal/dto/common_dto.go` | `ScheduleSession` struct'ina `SessionType string \`json:"session_type"\`` eklendi |

---

## Bug 8: InstructorName → InstructorFullname

**Sorun**: Enrollment event DTO'larinda Go field adi `InstructorName` idi ama json tag `"instructor_fullname"` idi. Karisikliga yol aciyordu ve event_service.go'da `.InstructorName` olarak erisim saglaniyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/enrollment-service/internal/dto/event_dto.go` | `CourseSemesterCreatedEvent` ve `CourseSemesterUpdatedEvent` struct'larinda `InstructorName` → `InstructorFullname` |
| `backend/services/enrollment-service/internal/service/event_service.go` | Tum `event.InstructorName` referanslari → `event.InstructorFullname` |

---

## Bug 5: Pagination Field Isimleri Standardizasyonu

**Sorun**: staff/student/catalog zaten `{page, limit, total, total_pages}` kullaniyor ama enrollment ve grades servisleri `page_size` ve `total_items` kullaniyor, frontend `per_page` kullaniyor.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/enrollment-service/internal/dto/common_dto.go` | `PaginationRequest.PageSize` → `Limit` (json: `"limit"`), `PaginationResponse.PageSize` → `Limit` (json: `"limit"`), `PaginationResponse.TotalItems` → `Total` (json: `"total"`) |
| `backend/services/enrollment-service/internal/service/enrollment_service_program.go` | `PageSize:` → `Limit:`, `TotalItems:` → `Total:` |
| `backend/services/grades-service/internal/dto/common_dto.go` | `PaginationParams.PageSize` → `Limit` (form: `"limit"`), `Pagination.PageSize` → `Limit` (json: `"limit"`), `Pagination.TotalItems` → `Total` (json: `"total"`) |
| `frontend/lib/types.ts` | `PaginatedResponse` interface: `per_page` → `limit` |
| `frontend/app/(admin)/semester-courses/list/page.tsx` | Inline pagination type: `per_page` → `limit` |
| `frontend/app/(admin)/semester-courses/page.tsx` | Inline pagination type'lar (2 adet): `per_page` → `limit` |

---

## Bug 4: practical_hours → lab_hours Rename

**Sorun**: Backend'de kolon `lab_hours` olarak guncellenmisti ama frontend hala `practical_hours` kullaniyordu.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `frontend/mock_data/teacher.ts` | Tum `practical_hours` → `lab_hours` (interface + 4 mock obje) |
| `frontend/mock_data/catalog.ts` | 85 adet `practical_hours` satiri silindi (`lab_hours` zaten mevcuttu) |
| `frontend/app/(admin)/catalog/page.tsx` | `course.practical_hours` → `course.lab_hours`, `selectedCourse.practical_hours` → `selectedCourse.lab_hours` |

---

## Bug 6: camelCase Route Parametreleri → snake_case

**Sorun**: Attendance ve grades servislerinde route parametreleri camelCase (`:sessionId`, `:courseId`, `:studentId`) idi, diger servislerle tutarsizdi.

**Degisiklikler**:

| Dosya | Degisiklik |
|-------|-----------|
| `backend/services/attendance-service/cmd/main.go` | `:sessionId` → `:session_id` (6 route), `:courseId` → `:course_id` (1 route) |
| `backend/services/attendance-service/internal/handler/attendance_handler.go` | Tum `c.Param("sessionId")` → `c.Param("session_id")`, `c.Param("courseId")` → `c.Param("course_id")` |
| `backend/services/grades-service/cmd/main.go` | `:courseId` → `:course_id` (4 route), `:studentId` → `:student_id` (1 route) |
| `backend/services/grades-service/internal/handler/grade_handler.go` | Tum `c.Param("courseId")` → `c.Param("course_id")`, `c.Param("studentId")` → `c.Param("student_id")` |

---

## Dogrulama

- Tum Go servisleri `go build ./...` ile basariyla derlendi (shared, auth, staff, student, enrollment, attendance, grades)
- Frontend `bun tsc` kontrolunde bu degisikliklerden kaynaklanan yeni hata yok
