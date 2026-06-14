# Planned Semester Management — Uygulama Planı

Bu dokümanı oku ve adım adım uygula. Her adımı tamamladıktan sonra bir sonrakine geç.

---

## Adım 1 — Catalog: Ders CRUD'dan Outbox Event Kodlarını Kaldır

`semester_service.go` dosyasını aç (`backend/services/course-catalog-service/internal/service/semester_service.go`).

**1a)** `CreateSemesterCourse()` fonksiyonunda outbox event oluşturan bloğu bul ve SİL. Bu blok `CreateOutboxEvent` çağrısını ve `course.semester.created` payload oluşturma kodunu içerir. Fonksiyonun geri kalanına (semester_courses INSERT, schedule_sessions INSERT) dokunma.

**1b)** `UpdateSemesterCourse()` fonksiyonunda outbox event oluşturan bloğu bul ve SİL. `course.semester.updated` veya benzeri event oluşturan tüm kodu kaldır.

**1c)** `DeleteSemesterCourse()` fonksiyonunda outbox event oluşturan bloğu bul ve SİL. `course.semester.deleted` event oluşturan kodu kaldır.

**1d)** Bu fonksiyonlarda artık kullanılmayan import'ları temizle.

---

## Adım 2 — Shared: Kullanılmayan Event Sabitlerini Kaldır

`backend/shared/events/events.go` dosyasını aç.

Şu sabitleri SİL:
- `EventCourseSemesterUpdated` (`course.semester.updated`)
- `EventCourseSemesterDeleted` (`course.semester.deleted`)
- `EventCourseInstructorChanged` (`course.instructor.changed`)
- `EventCourseScheduleChanged` (`course.schedule.changed`)
- `EventCourseAssessmentSchemaChanged` (`course.assessment.schema.changed`)

Şu sabitleri de SİL (hiç üretilmiyor, dead code):
- `EventCourseCatalogCreated` (`course.catalog.created`)
- `EventCourseCatalogUpdated` (`course.catalog.updated`)

İlgili routing key sabitleri varsa onları da kaldır. Şunları KORU:
- `EventCourseSemesterCreated` (`course.semester.created`)

---

## Adım 3 — Consumer'lardan Kullanılmayan Handler'ları Kaldır

**3a) Enrollment service** — `backend/services/enrollment-service/internal/worker/event_consumer.go`:
- `course.semester.updated` event'ini işleyen handler fonksiyonunu ve switch/case dalını SİL
- `course.semester.deleted` event'ini işleyen handler fonksiyonunu ve switch/case dalını SİL
- RabbitMQ binding'lerden bu routing key'leri kaldır

**3b) Attendance service** — `backend/services/attendance-service/internal/worker/event_consumer.go`:
- `course.semester.updated` handler ve case dalını SİL
- `course.semester.deleted` handler ve case dalını SİL
- Binding'lerden routing key'leri kaldır

**3c) Grades service** — `backend/services/grades-service/internal/worker/event_consumer.go`:
- `course.semester.updated` handler ve case dalını SİL
- `course.semester.deleted` handler ve case dalını SİL
- `course.instructor.changed` handler ve case dalını SİL
- `course.prerequisites.updated` handler ve case dalını SİL
- Binding'lerden routing key'leri kaldır

---

## Adım 4 — Kullanılmayan Repository/SQL Fonksiyonlarını Kaldır

Her serviste (enrollment, attendance, grades) silinen handler'ların çağırdığı repository ve SQL fonksiyonlarını bul. Eğer başka yerden çağrılmıyorsa SİL:

- `UpdateCourseCache` — enrollment, attendance, grades servislerinde ara. Sadece silinen event handler'larından çağrılıyorsa kaldır.
- `DeleteCourseCache` — aynı kontrol, sadece silinen handler'lardan çağrılıyorsa kaldır.
- `UpdateCourseInstructor` — grades service, kaldır.
- `SyncPrerequisiteCourses` — grades service, kaldır.

İlgili `.sql` query dosyalarından da kullanılmayan query'leri kaldır. Sonra etkilenen her serviste `sqlc generate` çalıştır.

---

## Adım 5 — Catalog: Aktivasyonda Toplu Event Oluşturma

**5a)** `backend/services/course-catalog-service/sql/queries/semester.sql` dosyasına şu query'yi EKLE:

```sql
-- name: ListSemesterCoursesForActivation :many
SELECT
    sc.*,
    json_agg(json_build_object(
        'day_of_week', css.day_of_week,
        'slot_number', css.slot_number,
        'session_type', css.session_type
    )) AS schedule_sessions
FROM semester_courses sc
LEFT JOIN course_schedule_sessions css ON css.semester_course_id = sc.id
WHERE sc.semester = $1
GROUP BY sc.id;
```

**5b)** `sqlc generate` çalıştır.

**5c)** `backend/services/course-catalog-service/internal/handler/semester_status_handler.go` dosyasında `ActivateSemester()` fonksiyonunu bul.

Mevcut `ActivateSemester` sadece status güncelliyor. Bunu değiştir: transaction içinde önce tüm dersleri getir, sonra status güncelle, sonra her ders için outbox event oluştur. Payload yapısı olarak Adım 1'de `CreateSemesterCourse()`'dan kaldırdığın payload oluşturma mantığını kullan — aynı alanları (semester_course_id, semester, course_code, course_name, credits, department, faculty, instructor_id, instructor_fullname, classroom_location, max_capacity, assessment_schema, prerequisites, schedule_sessions) içermeli.

```go
// Pseudocode:
courses := queries.ListSemesterCoursesForActivation(ctx, semester.Name)
tx := pool.Begin(ctx)
qtx := queries.WithTx(tx)
qtx.ActivateSemester(ctx, semester.ID)
for _, course := range courses {
    payload := buildCourseSemesterCreatedPayload(course)
    qtx.CreateOutboxEvent(ctx, CreateOutboxEventParams{
        EventType:  "course.semester.created",
        RoutingKey: "course.semester.created",
        Payload:    payload,
    })
}
tx.Commit(ctx)
```

`buildCourseSemesterCreatedPayload` fonksiyonunu yaz. Adım 1'de sildiğin payload yapısını referans al.

---

## Adım 6 — Meal Service: `closed_days` Tablosuna `semester` Kolonu Ekle

**6a)** `backend/services/meal-service/sql/migrations/` klasöründeki son migration numarasını bul. Bir sonraki numarayla yeni migration oluştur:

```sql
-- +goose Up
ALTER TABLE closed_days ADD COLUMN semester VARCHAR(50);

-- +goose Down
ALTER TABLE closed_days DROP COLUMN semester;
```

**6b)** `backend/services/meal-service/sql/queries/closed_days.sql` dosyasına şu query'leri EKLE:

```sql
-- name: DeleteClosedDaysBySemester :exec
DELETE FROM closed_days WHERE semester = $1;
```

**6c)** `BatchCreateClosedDays` fonksiyonunun kullandığı SQL query'yi bul. `semester` alanını INSERT'e ekle.

**6d)** `sqlc generate` çalıştır.

**6e)** `backend/services/meal-service/internal/handler/closed_days_handler.go` dosyasında `RegisterInternalRoutes` fonksiyonuna ekle:

```go
closedDays.DELETE("/by-semester/:semester", h.DeleteClosedDaysBySemester)
closedDays.PUT("/by-semester/:semester", h.UpdateClosedDaysBySemester)
```

**6f)** `DeleteClosedDaysBySemester` handler'ı yaz: semester param al, `DeleteClosedDaysBySemester` query'sini çağır, 204 dön.

**6g)** `UpdateClosedDaysBySemester` handler'ı yaz: semester param al, request body'den `closed_days` array'ini parse et, önce `DeleteClosedDaysBySemester` ile eskileri sil, sonra yenilerini INSERT et (semester alanıyla birlikte), 200 dön.

**6h)** `BatchCreateClosedDays` handler'ını güncelle: request body'de `semester` alanını kabul et ve kaydet.

---

## Adım 7 — Shared: Internal Period Endpoint'leri Ekle

`backend/shared/handler/` altında period handler dosyasını bul (`internal_period_handler.go` veya `simple_period_handler.go` — hangisi internal route'ları register ediyorsa).

**7a)** İlgili servisin SQL query dosyasına (`periods.sql` veya `academic_periods.sql`) şu query'leri EKLE (enrollment, grades, attendance servislerinin her biri için):

```sql
-- name: DeletePeriodBySemester :exec
DELETE FROM academic_periods WHERE semester = $1;

-- name: UpdatePeriodBySemester :one
UPDATE academic_periods
SET period_start = $2, period_end = $3, updated_at = NOW()
WHERE semester = $1 AND course_id IS NULL
RETURNING *;
```

**7b)** Her servis için `sqlc generate` çalıştır.

**7c)** Internal period handler'a route'ları ekle:

```go
rg.DELETE("/periods/by-semester/:semester", h.DeletePeriodBySemester)
rg.PUT("/periods/by-semester/:semester", h.UpdatePeriodBySemester)
```

**7d)** `DeletePeriodBySemester` handler'ı yaz: semester param al, query çağır, 204 dön. Semester active check YAPMA.

**7e)** `UpdatePeriodBySemester` handler'ı yaz: semester param al, body'den `period_start`/`period_end` parse et, query çağır, 200 dön. Semester active check YAPMA.

---

## Adım 8 — Catalog: DELETE Planned Semester Endpoint

**8a)** `backend/services/course-catalog-service/sql/queries/semesters.sql` dosyasına şu query'leri EKLE:

```sql
-- name: GetSemesterByID :one
SELECT * FROM semesters WHERE id = $1;

-- name: DeletePlannedSemester :exec
DELETE FROM semesters WHERE id = $1 AND status = 'planned';

-- name: DeleteSemesterCoursesBySemester :exec
DELETE FROM semester_courses WHERE semester = $1;
```

**8b)** `backend/services/course-catalog-service/sql/queries/periods.sql` dosyasına EKLE:

```sql
-- name: DeletePeriodsBySemester :exec
DELETE FROM academic_periods WHERE semester = $1;
```

**8c)** `sqlc generate` çalıştır.

**8d)** `semester_status_handler.go` dosyasında route registration'a ekle:

```go
semesters.DELETE("/:id", h.DeletePlannedSemester)
```

**8e)** `DeletePlannedSemester` handler'ı yaz. Akış:

1. URL'den `id` param al, UUID parse et
2. `GetSemesterByID(ctx, id)` ile dönemi getir. Yoksa 404.
3. `semester.Status != "planned"` ise 409 Conflict dön: `"Sadece planlanmış dönemler silinebilir"`
4. Transaction başlat:
   - `DeleteSemesterCoursesBySemester(ctx, semester.Name)` — dersler silinir, schedule_sessions CASCADE silinir
   - `DeletePeriodsBySemester(ctx, semester.Name)` — catalog periyodu silinir
   - `DeletePlannedSemester(ctx, semester.ID)` — dönem silinir
5. Transaction commit
6. Diğer servislerin periyotlarını HTTP ile temizle (her biri için `DELETE /api/{service}/internal/periods/by-semester/{name}`). Hata olursa logla ama devam et.
7. Meal servisindeki kapalı günleri temizle: `DELETE /api/meals/internal/closed-days/by-semester/{name}`. Hata olursa logla.
8. Audit log: action `"semester.deleted"`, resource_type `"semester"`
9. 204 No Content dön

---

## Adım 9 — Catalog: PUT (Edit) Planned Semester Endpoint

**9a)** `semesters.sql` dosyasına EKLE:

```sql
-- name: UpdatePlannedSemester :one
UPDATE semesters
SET hard_deadline = $2, updated_at = NOW()
WHERE id = $1 AND status = 'planned'
RETURNING *;
```

**9b)** `periods.sql` dosyasına EKLE:

```sql
-- name: GetPeriodBySemester :one
SELECT * FROM academic_periods WHERE semester = $1 AND course_id IS NULL LIMIT 1;

-- name: UpdatePeriodBySemester :one
UPDATE academic_periods
SET period_start = $2, period_end = $3, updated_at = NOW()
WHERE semester = $1 AND course_id IS NULL
RETURNING *;
```

**9c)** `sqlc generate` çalıştır.

**9d)** `semester_status_handler.go` route registration'a ekle:

```go
semesters.PUT("/:id", h.UpdatePlannedSemester)
```

**9e)** `UpdatePlannedSemester` handler'ı yaz. Request body:

```go
type UpdateSemesterRequest struct {
    HardDeadline time.Time       `json:"hard_deadline" binding:"required"`
    Periods      *ServicePeriods `json:"periods,omitempty"`
    ClosedDays   []ClosedDay     `json:"closed_days,omitempty"`
}
```

Akış:

1. URL'den `id` param al, body parse et
2. `GetSemesterByID` ile dönemi getir. Yoksa 404.
3. Status `planned` değilse 409 dön
4. `hard_deadline` gelecekte olmalı, periyot bitiş tarihleri `hard_deadline`'ı aşmamalı — validasyon
5. `UpdatePlannedSemester(ctx, id, hard_deadline)` çalıştır
6. Periods varsa:
   - Catalog local: `UpdatePeriodBySemester`
   - HTTP: `PUT /api/{service}/internal/periods/by-semester/{name}` (enrollment, grades, attendance)
7. ClosedDays varsa:
   - HTTP: `PUT /api/meals/internal/closed-days/by-semester/{name}` — body'de yeni liste gönder
8. Remote hataları `update_errors` array'inde topla
9. Audit log: action `"semester.updated"`
10. 200 OK + güncellenmiş semester + varsa `update_errors` dön

---

## Adım 10 — Frontend: API + Types

**10a)** `frontend/src/lib/services/system-service.ts` dosyasına EKLE:

```typescript
export async function deleteSemester(id: string): Promise<void> {
  await catalogApiSafe.delete(`admin/semesters/${id}`);
}

export async function updateSemester(
  id: string,
  data: UpdateSemesterRequest
): Promise<Semester> {
  return catalogApiSafe
    .put(`admin/semesters/${id}`, { json: data })
    .json<Semester>();
}
```

**10b)** `frontend/src/lib/types.ts` dosyasında `Semester` tipinin yanına EKLE:

```typescript
interface UpdateSemesterRequest {
  hard_deadline: string;
  periods?: ServicePeriods;
  closed_days?: ClosedDay[];
}
```

---

## Adım 11 — Frontend: Semester List Sayfasına Düzenle/Sil Butonları

`frontend/src/pages/admin/system/semesters/index.tsx` dosyasını aç.

**11a)** Tablo satırlarına (her semester için) yeni bir "İşlemler" kolonu ekle. Sadece `status === 'planned'` olan satırlarda göster:
- **Sil** butonu (kırmızı, Trash ikonu)
- **Düzenle** butonu (Edit ikonu)

**11b)** Sil butonuna tıklayınca confirmation dialog göster:
- Mesaj: "Bu dönem ve tüm içeriği (dersler, periyotlar, kapalı günler) kalıcı olarak silinecek. Emin misiniz?"
- Onaylarsa `deleteSemester(id)` çağır, listeyi yenile

**11c)** Düzenle butonuna tıklayınca edit modal/dialog aç:
- Hard deadline date picker
- Servis periyotları (her servis için başlangıç/bitiş tarihi)
- Kapalı günler listesi (ekle/kaldır)
- Kaydet butonu → `updateSemester(id, data)` çağır, modalı kapat, listeyi yenile

---

## Kapsam Dışı

- Dönem adı (name) değiştirme — tüm servislerde anahtar, çok riskli
- Aktif/tamamlanmış dönemi silme/düzenleme — sadece planned
