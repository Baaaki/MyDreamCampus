# Course Catalog Service

## Sorumluluk
Ders kataloğu yönetimi ve dönemlik ders açılışı

**Source of Truth**: Bu servis ders bilgileri için canonical data source'dur.

**İki Ana Fonksiyon**:
1. **Master Ders Kataloğu**: Üniversitede açılabilecek tüm derslerin static listesi
2. **Dönemlik Ders Yönetimi**: Her öğretim dönemi için açılan derslerin yönetimi (hoca, saat, sınıf bilgisi)

---

## İletişim

### Inbound (REST - Staff Service Integration)
Course Service, Staff Service'ten REST API ile hoca bilgilerini çeker:

**GET /api/v1/staff/instructors?department={department}&status=active**
- Bölümdeki aktif öğretim görevlisi listesi (role = "teacher", status = "active")
- **Response**:
  ```json
  {
    "data": [
      {
        "id": "uuid",
        "first_name": "Ayşe",
        "last_name": "Demir",
        "department": "Bilgisayar Mühendisliği"
      }
    ]
  }
  ```
- **Kullanım**:
  - Ders açılırken instructor_id validation (hoca bu bölümde mi?)
  - Instructor fullname oluşturma (first_name + " " + last_name)
  - Frontend'de hoca dropdown için

### Outbound (RabbitMQ)
- `course.semester.created` - Dönemlik ders açıldığında (Enrollment Service, Grades Service consume eder)
- `course.semester.updated` - Dönemlik ders güncellendiğinde (Enrollment Service, Grades Service consume eder)
- `course.semester.deleted` - Dönemlik ders silindiğinde (Enrollment Service consume eder)
- `course.instructor.changed` - Hoca değişikliği olduğunda (Grades Service consume eder - NADİR)
- `course.prerequisites.updated` - Prerequisite ders listesi değiştiğinde (Grades Service consume eder)

---

## Database Schema

### course_catalog (Master Ders Kataloğu)

**UPDATED**: ENUM types + SMALLINT optimization + class_level for prerequisites

```sql
-- Enum definition for course catalog status
CREATE TYPE course_catalog_status_enum AS ENUM (
    'active',           -- Aktif ders (öğrencilere açılabilir)
    'draft',            -- Taslak (henüz onaylanmamış)
    'pending_approval', -- Onay bekliyor
    'under_revision',   -- Revizyon aşamasında
    'archived',         -- Arşivlenmiş (artık açılmaz, eski kayıtlar için)
    'suspended'         -- Askıya alınmış (geçici olarak kapatılmış)
);

-- Enum definition for course type
CREATE TYPE course_type_enum AS ENUM (
    'mandatory',        -- Zorunlu ders
    'elective'          -- Seçmeli ders
);

-- Note: class_level uses SMALLINT (1-6) consistent with Student Service (source of truth)

CREATE TABLE course_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    faculty VARCHAR(100) NOT NULL,              -- Fakülte
    department VARCHAR(100) NOT NULL,           -- Bölüm
    class_level SMALLINT NOT NULL CHECK (class_level BETWEEN 1 AND 6),  -- Dersin ait olduğu sınıf seviyesi (1-6)
    credits SMALLINT NOT NULL CHECK (credits > 0 AND credits <= 30),  -- Toplam kredi (max 30)
    theoretical_hours SMALLINT NOT NULL DEFAULT 0 CHECK (theoretical_hours >= 0 AND theoretical_hours <= 20),
    practical_hours SMALLINT NOT NULL DEFAULT 0 CHECK (practical_hours >= 0 AND practical_hours <= 20),
    course_type course_type_enum NOT NULL DEFAULT 'mandatory',

    -- ⚠️ DENORMALIZED STRUCTURE (Read-Heavy Optimization)
    -- Normalde sadece UUID array saklardık: ["uuid-1", "uuid-2"]
    -- Ancak bu sistemde Read:Write oranı ~1000:1 (öğrenciler sürekli katalog görüntüler)
    -- JSONB array'den UUID çıkarıp JOIN yapmak her okumada maliyet:
    --   - jsonb_array_elements_text() → JSONB parse
    --   - Her element için ayrı lookup
    --   - Index kullanımı zorlaşır
    -- Bu nedenle id, course_code ve name bilgisini de saklıyoruz (denormalized)
    -- Trade-off: Ders adı değişirse batch update gerekir (yılda 1-2 kez, kabul edilebilir)
    prerequisites JSONB DEFAULT '[]',           -- [{id, course_code, course_name}, ...] (denormalized for read performance)

    description TEXT,                           -- Ders tanımı
    learning_outcomes TEXT,                     -- Öğrenim çıktıları
    syllabus TEXT,                              -- Ders içeriği / müfredat
    status course_catalog_status_enum DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_catalog_code ON course_catalog(course_code);
CREATE INDEX idx_catalog_department ON course_catalog(department);
CREATE INDEX idx_catalog_prerequisites_gin ON course_catalog USING GIN(prerequisites);
CREATE INDEX idx_catalog_status ON course_catalog(status);
CREATE INDEX idx_catalog_course_type ON course_catalog(course_type);
CREATE INDEX idx_catalog_class_level ON course_catalog(class_level);
```

**Prerequisites Example Data** (Denormalized):
```sql
-- Önkoşulu olmayan ders (1. sınıf)
INSERT INTO course_catalog (course_code, name, class_level, prerequisites, ...) VALUES
('CS100', 'Programming Fundamentals', 1, '[]', ...);

-- Tek önkoşullu ders (2. sınıf, prerequisite 1. sınıftan)
INSERT INTO course_catalog (course_code, name, class_level, prerequisites, ...) VALUES
('CS101', 'Data Structures', 2,
 '[{"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}]', ...);

-- Birden fazla önkoşullu ders (3. sınıf, prerequisite'ler 1. ve 2. sınıftan)
INSERT INTO course_catalog (course_code, name, class_level, prerequisites, ...) VALUES
('CS301', 'Database Systems', 3,
 '[
   {"id": "uuid-cs101", "course_code": "CS101", "course_name": "Data Structures"},
   {"id": "uuid-cs201", "course_code": "CS201", "course_name": "Algorithms"},
   {"id": "uuid-math101", "course_code": "MATH101", "course_name": "Discrete Mathematics"}
 ]', ...);
```

**Prerequisite Class Level Rule**:
> Prerequisite dersin `class_level` değeri, eklenen dersin `class_level` değerinden **küçük** olmalıdır.

Bu kural üniversite müfredatının doğal hiyerarşisini yansıtır:
- 1. sınıf dersleri → 2. sınıf derslerine prerequisite olabilir
- 2. sınıf dersleri → 3. sınıf derslerine prerequisite olabilir
- Aynı sınıf seviyesindeki dersler birbirinin prerequisite'i **olamaz**
- Üst sınıf dersleri alt sınıf derslerine prerequisite **olamaz**

Bu doğal kısıt sayesinde circular dependency (A→B→C→A döngüsü) matematiksel olarak imkansız hale gelir.

**Why Denormalized?**
| Approach | Query | Performance |
|----------|-------|-------------|
| **Normalized** (UUID only) | JSONB parse + JOIN per UUID | ~5ms (3 prereq) |
| **Denormalized** (full info) | Single SELECT, no JOIN | ~0.5ms |

**Trade-off**: Ders adı değişirse → Batch update gerekir (nadir, yılda 1-2 kez)

### Time Slot System (Frontend Reference)

**Slot Definition**: Backend stores **slot numbers only (1-9)**, frontend displays time ranges.

| Slot | Time Range    |
|------|---------------|
| 1    | 08:30-09:15   |
| 2    | 09:25-10:10   |
| 3    | 10:20-11:05   |
| 4    | 11:15-12:00   |
| 5    | 12:10-12:55   |
| 6    | 13:00-13:45   |
| 7    | 13:55-14:40   |
| 8    | 14:50-15:35   |
| 9    | 15:45-16:30   |

**Notes**:
- Each slot: **45 minutes lesson + 10 minutes break**
- Total daily slots: **9** (08:30-16:30)
- Backend **does not store** time ranges, only slot numbers (immutable)
- Frontend maintains TIME_SLOTS mapping constant

**Frontend Implementation**:
```typescript
// constants/timeSlots.ts
export const TIME_SLOTS = {
  1: "08:30-09:15",
  2: "09:25-10:10",
  3: "10:20-11:05",
  4: "11:15-12:00",
  5: "12:10-12:55",
  6: "13:00-13:45",
  7: "13:55-14:40",
  8: "14:50-15:35",
  9: "15:45-16:30"
} as const;
```

### semester_courses (Dönemlik Açılan Dersler - Single Table)

**UPDATED**: Assessment schema eklendi

```sql
-- Enum definition (consistent with Enrollment Service)
CREATE TYPE day_of_week_enum AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

-- Single table for all semesters (no partitioning)
CREATE TABLE semester_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester VARCHAR(50) NOT NULL,            -- "2025_spring", "2025_fall", "2026_spring"
    course_code VARCHAR(50) NOT NULL REFERENCES course_catalog(course_code),
    credits SMALLINT NOT NULL CHECK (credits > 0 AND credits <= 30),  -- Cached from course_catalog (immutable snapshot for ECTS calculation)
    class_level SMALLINT NOT NULL CHECK (class_level BETWEEN 1 AND 6),  -- Hangi sınıf seviyesi için açık (1-6)
    instructor_id UUID NOT NULL,              -- Staff Service UUID (system reference)
    instructor_fullname VARCHAR(150) NOT NULL, -- Cached: first_name + last_name from Staff Service (immutable snapshot)
    classroom_location VARCHAR(100) NOT NULL, -- "A Blok 301"
    max_capacity SMALLINT NOT NULL CHECK (max_capacity > 0 AND max_capacity <= 1000),  -- Max 1000 kişilik amfi

    -- ✅ NEW: Sınav Yapısı Konfigürasyonu (Assessment Schema)
    -- Admin tarafından dönemlik ders açılırken belirlenir
    -- Grades Service bu bilgiyi kullanarak not girişi formunu oluşturur
    -- Örnek: [{"slug": "midterm", "name": "Vize", "weight": 40}, {"slug": "final", "name": "Final", "weight": 60}]
    assessment_schema JSONB NOT NULL DEFAULT '[{"slug": "midterm", "name": "Vize", "weight": 40}, {"slug": "final", "name": "Final", "weight": 60}]',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Prevent duplicate: Same course, same semester
    UNIQUE(semester, course_code),

    -- Assessment schema validation
    CONSTRAINT chk_assessment_schema_valid CHECK (
        jsonb_typeof(assessment_schema) = 'array'
        AND jsonb_array_length(assessment_schema) > 0
    )
);

CREATE INDEX idx_semester_courses_semester ON semester_courses(semester);
CREATE INDEX idx_semester_courses_course_code ON semester_courses(course_code);
CREATE INDEX idx_semester_courses_instructor ON semester_courses(instructor_id);
CREATE INDEX idx_semester_courses_semester_code ON semester_courses(semester, course_code);

-- Note: current_enrollment is NOT stored here (Enrollment Service responsibility)
-- Note: instructor_fullname is immutable snapshot (set at course opening, never updated)
```

**Semester Format**: `{year}_{semester}` (spring, fall, summer)

**Examples**:
- `2025_spring`
- `2025_fall`
- `2026_spring`

**Scalability Notes**:
- ~10K courses per semester
- 20 years = ~400K rows (easily handled by PostgreSQL with proper indexing)
- No partitioning needed (KISS principle)

### Assessment Schema Yapısı

#### Format
```json
[
  {"slug": "midterm", "name": "Vize", "weight": 40},
  {"slug": "final", "name": "Final", "weight": 60}
]
```

#### Validation Kuralları (Go Tarafında)
1. `slug`: Unique within array, lowercase, no spaces (regex: `^[a-z][a-z0-9_]*$`)
2. `name`: Display name (UTF-8, max 100 char)
3. `weight`: 0-100 arası integer, toplam 100 olmalı
4. Array minimum 1 eleman içermeli

#### Örnek Şemalar

**Standart (Vize + Final)**:
```json
[
  {"slug": "midterm", "name": "Vize", "weight": 40},
  {"slug": "final", "name": "Final", "weight": 60}
]
```

**Proje Ağırlıklı**:
```json
[
  {"slug": "project_1", "name": "Proje 1", "weight": 20},
  {"slug": "project_2", "name": "Proje 2", "weight": 20},
  {"slug": "midterm", "name": "Vize", "weight": 25},
  {"slug": "final", "name": "Final", "weight": 35}
]
```

**Quiz Bazlı**:
```json
[
  {"slug": "quiz_1", "name": "Quiz 1", "weight": 10},
  {"slug": "quiz_2", "name": "Quiz 2", "weight": 10},
  {"slug": "quiz_3", "name": "Quiz 3", "weight": 10},
  {"slug": "midterm", "name": "Vize", "weight": 30},
  {"slug": "final", "name": "Final", "weight": 40}
]
```

### course_schedule_sessions (Course-Day-Slot Mapping - Junction Table)

**Purpose**: Maps each course to specific day-slot combinations

```sql
CREATE TABLE course_schedule_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester_course_id UUID NOT NULL REFERENCES semester_courses(id) ON DELETE CASCADE,
    day_of_week day_of_week_enum NOT NULL,
    slot_number SMALLINT NOT NULL CHECK (slot_number BETWEEN 1 AND 9),
    created_at TIMESTAMP DEFAULT NOW(),

    -- Prevent duplicate: Same course, same day, same slot
    UNIQUE(semester_course_id, day_of_week, slot_number)
);

CREATE INDEX idx_schedule_sessions_course ON course_schedule_sessions(semester_course_id);
CREATE INDEX idx_schedule_sessions_day_slot ON course_schedule_sessions(day_of_week, slot_number);
```

**Example Data**:
```sql
-- CS101: Monday slots 1,2,3 + Wednesday slots 4,5
INSERT INTO course_schedule_sessions (semester_course_id, day_of_week, slot_number) VALUES
('uuid-cs101', 'monday', 1),
('uuid-cs101', 'monday', 2),
('uuid-cs101', 'monday', 3),
('uuid-cs101', 'wednesday', 4),
('uuid-cs101', 'wednesday', 5);
```

**Schema Notes**:
- `slot_number`: INT with CHECK constraint (1-9), no foreign key
- Frontend maps slot numbers to time ranges (TIME_SLOTS constant)
- Backend only validates range (1-9), doesn't store time information

**Why Many-to-Many?**:
- One course → Multiple day-slot combinations (flexible scheduling)
- Supports: Theoretical (3 slots Monday) + Lab (2 slots Wednesday)
- Easy conflict detection: Same instructor, same day-slot check

---

### outbox (Transactional Outbox Pattern)

**Purpose**: Event publish güvenliği - DB transaction ile atomik yazım

```sql
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,           -- 'course.semester.created', 'course.semester.updated', 'course.semester.deleted'
    routing_key VARCHAR(100) NOT NULL,          -- RabbitMQ routing key
    payload JSONB NOT NULL,                     -- Event data (JSON)
    status outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT                          -- Son hata mesajı (debug için)
);

CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_events_retry ON outbox_events(status, retry_count) WHERE status = 'failed';
```

**Outbox Pattern Flow**:
```
1. Business Logic + Outbox INSERT (same transaction)
   ↓
2. Transaction COMMIT
   ↓
3. Background Worker (polling every 100ms):
   - SELECT * FROM outbox_events WHERE status = 'pending' ORDER BY created_at LIMIT 100
   - RabbitMQ publish
   - UPDATE outbox_events SET status = 'processed', processed_at = NOW()
   ↓
4. Failed? → retry_count++, status = 'failed' if max_retries exceeded
```

**Why Outbox Pattern?**
- **Atomicity**: DB write + event publish aynı transaction'da garanti
- **Reliability**: RabbitMQ down olsa bile event kaybolmaz
- **Retry**: Failed events otomatik retry edilir
- **Idempotency**: Consumer tarafında `event_id` ile duplicate check

---

## API Endpoints

### 🔒 POST /api/v1/catalog/courses
Kataloga yeni ders ekleme (nadiren kullanılır)

**Role Requirement**: Admin

**Request**:
```json
{
  "course_code": "CS101",
  "name": "Data Structures",
  "faculty": "Mühendislik Fakültesi",
  "department": "Bilgisayar Mühendisliği",
  "class_level": 2,
  "credits": 6,
  "theoretical_hours": 3,
  "practical_hours": 3,
  "course_type": "mandatory",
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
  ],
  "description": "Advanced study of data structures...",
  "learning_outcomes": "1. Implement common data structures\n2. Analyze algorithm complexity...",
  "syllabus": "Week 1: Arrays and Linked Lists\nWeek 2: Stacks and Queues...",
  "status": "draft"
}
```

**Note**: `status` is optional (defaults to `active` if not provided). Valid values: `active`, `draft`, `pending_approval`, `under_revision`, `archived`, `suspended`

**Response** (201):
```json
{
  "id": "uuid",
  "course_code": "CS101",
  "name": "Data Structures",
  "faculty": "Mühendislik Fakültesi",
  "department": "Bilgisayar Mühendisliği",
  "class_level": 2,
  "credits": 6,
  "theoretical_hours": 3,
  "practical_hours": 3,
  "course_type": "mandatory",
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
  ],
  "description": "Advanced study of data structures...",
  "learning_outcomes": "1. Implement common data structures\n2. Analyze algorithm complexity...",
  "syllabus": "Week 1: Arrays and Linked Lists\nWeek 2: Stacks and Queues...",
  "status": "draft",
  "created_at": "2025-11-12T10:00:00Z"
}
```

**Business Logic**:
1. **Course code uniqueness kontrolü**:
   ```sql
   SELECT id FROM course_catalog WHERE course_code = $1
   ```
   - ❌ Varsa: `409 COURSE_CODE_EXISTS`
2. **Prerequisites validation**:
   - Her prerequisite için: `SELECT id, course_code, name, class_level FROM course_catalog WHERE id = $1`
   - ❌ Bulunamazsa: `400 INVALID_PREREQUISITE`
   - Request'teki `course_code` ve `name` ile DB'deki değerler eşleşmeli
   - **Class level kontrolü**: Prerequisite'in `class_level` değeri, eklenen dersin `class_level` değerinden küçük olmalı
   - ❌ Prerequisite class_level >= course class_level ise: `400 INVALID_PREREQUISITE_LEVEL`
3. **Hours validation**: `theoretical_hours` ve `practical_hours` toplamı mantıklı olmalı
4. **Status validation**: Geçerli ENUM değeri mi? (`active`, `draft`, `pending_approval`, `under_revision`, `archived`, `suspended`)
5. **INSERT** course_catalog tablosuna (prerequisites JSONB olarak saklanır)
6. ❌ Event yayınlamaz (master katalog değişikliği rare event)

---

### 🔓 GET /api/v1/catalog/courses
Katalog listesi (tüm üniversite dersleri)

**Role Requirement**: Authenticated

**Query Parameters**: `page`, `limit`, `faculty`, `department`, `course_type`, `status`, `class_level`, `search`

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "course_code": "CS101",
      "name": "Data Structures",
      "faculty": "Mühendislik Fakültesi",
      "department": "Bilgisayar Mühendisliği",
      "class_level": 2,
      "credits": 6,
      "theoretical_hours": 3,
      "practical_hours": 3,
      "course_type": "mandatory",
      "prerequisites": [
        {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
      ],
      "status": "active"
    }
  ],
  "pagination": {"page": 1, "limit": 20, "total": 450, "total_pages": 23}
}
```

**Business Logic**:
```sql
SELECT id, course_code, name, faculty, department, class_level, credits,
       theoretical_hours, practical_hours, course_type, prerequisites, status
FROM course_catalog
WHERE ($1::text IS NULL OR faculty = $1)
  AND ($2::text IS NULL OR department = $2)
  AND ($3::course_type_enum IS NULL OR course_type = $3)
  AND ($4::course_catalog_status_enum IS NULL OR status = $4)
  AND ($5::SMALLINT IS NULL OR class_level = $5)
  AND ($6::text IS NULL OR name ILIKE '%' || $6 || '%' OR course_code ILIKE '%' || $6 || '%')
ORDER BY course_code
LIMIT $7 OFFSET $8
```

---

### 🔓 GET /api/v1/catalog/courses/:course_code
Katalog ders detayı

**Role Requirement**: Authenticated

**Response** (200):
```json
{
  "id": "uuid",
  "course_code": "CS101",
  "name": "Data Structures",
  "faculty": "Mühendislik Fakültesi",
  "department": "Bilgisayar Mühendisliği",
  "class_level": 2,
  "credits": 6,
  "theoretical_hours": 3,
  "practical_hours": 3,
  "course_type": "mandatory",
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
  ],
  "description": "Advanced study of data structures...",
  "learning_outcomes": "1. Implement common data structures...",
  "syllabus": "Week 1: Arrays and Linked Lists...",
  "status": "active",
  "created_at": "2025-09-01T10:00:00Z",
  "updated_at": "2025-09-01T10:00:00Z"
}
```

**Business Logic**:
```sql
SELECT id, course_code, name, faculty, department, class_level, credits,
       theoretical_hours, practical_hours, course_type, prerequisites,
       description, learning_outcomes, syllabus, status, created_at, updated_at
FROM course_catalog
WHERE course_code = $1
```
- ❌ Bulunamazsa: `404 COURSE_NOT_FOUND`
- `prerequisites` JSONB doğrudan döner (denormalized `[{id, course_code, course_name}]`, JOIN gerekmez)

---

### 🔒 PUT /api/v1/catalog/courses/:course_code
Katalog ders güncelleme

**Role Requirement**: Admin

**Request** (All fields optional):
```json
{
  "name": "Advanced Data Structures",
  "faculty": "Mühendislik Fakültesi",
  "department": "Bilgisayar Mühendisliği",
  "class_level": 2,
  "credits": 7,
  "theoretical_hours": 4,
  "practical_hours": 3,
  "course_type": "mandatory",
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"},
    {"id": "uuid-cs050", "course_code": "CS050", "course_name": "Introduction to Computing"}
  ],
  "description": "Updated comprehensive study...",
  "learning_outcomes": "Updated learning outcomes...",
  "syllabus": "Updated syllabus...",
  "status": "active"
}
```

**Response** (200):
```json
{
  "id": "uuid",
  "course_code": "CS101",
  "name": "Advanced Data Structures",
  "faculty": "Mühendislik Fakültesi",
  "department": "Bilgisayar Mühendisliği",
  "class_level": 2,
  "credits": 7,
  "theoretical_hours": 4,
  "practical_hours": 3,
  "course_type": "mandatory",
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"},
    {"id": "uuid-cs050", "course_code": "CS050", "course_name": "Introduction to Computing"}
  ],
  "description": "Updated comprehensive study...",
  "learning_outcomes": "Updated learning outcomes...",
  "syllabus": "Updated syllabus...",
  "status": "active",
  "updated_at": "2025-11-12T11:00:00Z"
}
```

**Business Logic**:
1. **Existence check**:
   ```sql
   SELECT id, class_level FROM course_catalog WHERE course_code = $1
   ```
   - ❌ Yoksa: `404 COURSE_NOT_FOUND`
2. **Prerequisites validation** (eğer gönderildiyse):
   - Her prerequisite için: `SELECT id, course_code, name, class_level FROM course_catalog WHERE id = $1`
   - ❌ Bulunamazsa: `400 INVALID_PREREQUISITE`
   - Request'teki `course_code` ve `name` ile DB'deki değerler eşleşmeli
   - **Class level kontrolü**: Prerequisite'in `class_level` değeri, güncellenen dersin `class_level` değerinden küçük olmalı
   - ❌ Prerequisite class_level >= course class_level ise: `400 INVALID_PREREQUISITE_LEVEL`
3. **Status validation** (eğer gönderildiyse): Geçerli ENUM değeri mi?
4. **UPDATE** course_catalog:
   ```sql
   UPDATE course_catalog
   SET name = COALESCE($1, name),
       faculty = COALESCE($2, faculty),
       department = COALESCE($3, department),
       class_level = COALESCE($4, class_level),
       credits = COALESCE($5, credits),
       theoretical_hours = COALESCE($6, theoretical_hours),
       practical_hours = COALESCE($7, practical_hours),
       course_type = COALESCE($8, course_type),
       prerequisites = COALESCE($9, prerequisites),
       description = COALESCE($10, description),
       learning_outcomes = COALESCE($11, learning_outcomes),
       syllabus = COALESCE($12, syllabus),
       status = COALESCE($13, status),
       updated_at = NOW()
   WHERE course_code = $14
   RETURNING *
   ```
5. ❌ Event yayınlamaz (master katalog değişikliği rare event)

---

### 🔒 POST /api/v1/semesters/:semester_id/courses
Dönemlik ders açılışı (manuel)

**Role Requirement**: Admin

**Request** (UPDATED - assessment_schema eklendi):
```json
{
  "course_code": "CS101",
  "class_level": 2,
  "instructor_id": "uuid",
  "instructor_fullname": "Ayşe Demir",
  "classroom_location": "A Blok 301",
  "max_capacity": 150,
  "assessment_schema": [
    {"slug": "midterm", "name": "Vize", "weight": 40},
    {"slug": "final", "name": "Final", "weight": 60}
  ],
  "schedule_sessions": [
    {
      "day_of_week": "monday",
      "slot_numbers": [1, 2, 3]
    },
    {
      "day_of_week": "wednesday",
      "slot_numbers": [4, 5]
    }
  ]
}
```

**Response** (201 - UPDATED):
```json
{
  "id": "uuid",
  "semester": "2025_spring",
  "course_code": "CS101",
  "course_name": "Data Structures",
  "credits": 6,
  "class_level": 2,
  "instructor_id": "uuid",
  "instructor_fullname": "Ayşe Demir",
  "classroom_location": "A Blok 301",
  "max_capacity": 150,
  "assessment_schema": [
    {"slug": "midterm", "name": "Vize", "weight": 40},
    {"slug": "final", "name": "Final", "weight": 60}
  ],
  "schedule_sessions": [
    {
      "day_of_week": "monday",
      "slot_numbers": [1, 2, 3]
    },
    {
      "day_of_week": "wednesday",
      "slot_numbers": [4, 5]
    }
  ],
  "created_at": "2025-11-12T10:00:00Z"
}
```

**Note**:
- Backend returns **slot numbers only**. Frontend maps to time ranges using TIME_SLOTS constant.
- `current_enrollment` is NOT included (managed by Enrollment Service)
- `course_name` is fetched from course_catalog via JOIN
- `assessment_schema` is optional - defaults to standard Vize(40%) + Final(60%) if not provided

**RabbitMQ Event Published**: `course.semester.created` (via Outbox Pattern)

**Business Logic**:
1. `course_code` geçerliliği kontrolü (course_catalog'da var mı?)
2. **Status validation**: Ders durumu `active` mi?
   ```sql
   SELECT status, credits, department, class_level FROM course_catalog WHERE course_code = $1
   ```
   - ❌ Status != 'active' ise: `400 COURSE_NOT_ACTIVE` (sadece active dersler dönemlik açılabilir)
3. **Class level consistency**: Request'teki `class_level` ile catalog'daki `class_level` eşleşmeli
   - ❌ Eşleşmiyorsa: `400 CLASS_LEVEL_MISMATCH`
4. Course_catalog'dan ders bilgileri çekilerek `department` ve `credits` belirlenir
5. **Duplicate check**: Aynı semester + course_code zaten açılmış mı?
   ```sql
   SELECT id FROM semester_courses WHERE semester = $1 AND course_code = $2
   ```
   - ❌ Varsa: `409 COURSE_ALREADY_OPENED`
6. **Instructor validation**: Request'teki `instructor_id` geçerli mi?
   - Staff Service'ten doğrulama: `GET /api/v1/staff/{instructor_id}`
   - ❌ Bulunamazsa: `404 INSTRUCTOR_NOT_FOUND`
   - ❌ Aktif değilse: `404 INSTRUCTOR_NOT_ACTIVE`
7. **Slot validation**: Slot numbers valid mi? (1-9 arası)
   - ✅ All slots BETWEEN 1 AND 9 → Continue
   - ❌ Invalid slot → 400 INVALID_SLOT_NUMBER
8. **Assessment schema validation** (✅ NEW):
   - Eğer `assessment_schema` gönderilmemişse → Default kullan
   - Eğer gönderildiyse:
     - Array boş olmamalı (minimum 1 eleman)
     - Her item'da `slug`, `name`, `weight` zorunlu
     - `slug`: Unique within array, lowercase, regex `^[a-z][a-z0-9_]*$`
     - `name`: UTF-8, max 100 karakter
     - `weight`: 0-100 arası integer
     - Toplam weight = 100 olmalı
   - ❌ Validation fail → `400 INVALID_ASSESSMENT_SCHEMA`
9. **Instructor conflict check**: Aynı hoca, aynı semester'da aynı gün-slot'ta başka ders var mı?
   ```sql
   SELECT sc.course_code
   FROM course_schedule_sessions css1
   JOIN course_schedule_sessions css2
     ON css1.day_of_week = css2.day_of_week
     AND css1.slot_number = css2.slot_number
   JOIN semester_courses sc
     ON css2.semester_course_id = sc.id
   WHERE css1.day_of_week = ANY($1)      -- Request'teki günler
     AND css1.slot_number = ANY($2)      -- Request'teki slotlar
     AND sc.semester = $3                 -- Aynı semester
     AND sc.instructor_id = $4            -- Aynı hoca
   ```
   - ✅ Conflict yoksa devam et
   - ❌ Conflict varsa: `409 INSTRUCTOR_SCHEDULE_CONFLICT`
10. **Single Transaction** (Outbox Pattern):
   ```sql
   BEGIN;

   -- 10a. semester_courses tablosuna kaydet (assessment_schema dahil)
   INSERT INTO semester_courses (id, semester, course_code, credits, class_level, instructor_id, instructor_fullname, classroom_location, max_capacity, assessment_schema)
   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
   RETURNING id;

   -- 10b. course_schedule_sessions tablosuna bulk insert
   INSERT INTO course_schedule_sessions (semester_course_id, day_of_week, slot_number)
   VALUES ($semester_course_id, 'monday', 1), ($semester_course_id, 'monday', 2), ...;

   -- 10c. outbox tablosuna event ekle (assessment_schema dahil)
   INSERT INTO outbox_events (event_type, routing_key, payload)
   VALUES ('course.semester.created', 'course.semester.created', '{"event_id": "...", "data": {..., "assessment_schema": [...]}}');

   COMMIT;
   ```
11. **Background Worker**: Outbox'tan eventi RabbitMQ'ya publish eder (100ms polling)

**Event Consumers**: Enrollment Service, Grades Service (dönemlik ders bilgilerini consume eder)

**Error Cases**:
- `400 COURSE_NOT_ACTIVE`: Sadece active durumdaki dersler dönemlik açılabilir
- `400 CLASS_LEVEL_MISMATCH`: Request class_level ile catalog class_level uyuşmuyor
- `400 INVALID_SLOT_NUMBER`: Slot number 1-9 arasında değil
- `400 INVALID_ASSESSMENT_SCHEMA`: Assessment schema validation hatası
- `409 INSTRUCTOR_SCHEDULE_CONFLICT`: Hoca aynı gün-slot'ta başka derste

---

### 🔒 POST /api/v1/semesters/:semester_id/copy-from-previous
Otomatik dönem kopyalama (önceki yılın aynı döneminden kopyalar)

**Role Requirement**: Admin

**Request**:
```json
{
  "source_semester": "2024_spring",
  "target_semester": "2025_spring",
  "copy_mandatory_only": true
}
```

**Response** (200):
```json
{
  "target_semester": "2025_spring",
  "courses_copied": 85,
  "mandatory_courses": 85,
  "elective_courses": 0,
  "copied_at": "2025-11-12T10:00:00Z"
}
```

**RabbitMQ Event Published**: Her kopyalanan ders için `course.semester.created` event yayınlanır (via Outbox Pattern)

**Business Logic**:
1. Source semester data kontrolü (örn: `semester = '2024_spring'` olan kayıtlar var mı?)
   ```sql
   SELECT COUNT(*) FROM semester_courses WHERE semester = $1
   ```
   - ❌ 0 row ise: `404 SOURCE_SEMESTER_NOT_FOUND`
2. Target semester'da zaten ders var mı kontrolü
   ```sql
   SELECT COUNT(*) FROM semester_courses WHERE semester = $1
   ```
   - ❌ > 0 ise: `400 SEMESTER_ALREADY_EXISTS`
3. Source semester'daki zorunlu VE active dersleri getir (assessment_schema dahil)
   ```sql
   SELECT sc.course_code, sc.credits, sc.class_level, sc.instructor_id, sc.classroom_location, sc.max_capacity, sc.assessment_schema, cc.department
   FROM semester_courses sc
   JOIN course_catalog cc ON sc.course_code = cc.course_code
   WHERE sc.semester = $1
     AND cc.course_type = 'mandatory'
     AND cc.status = 'active'
   ```
4. **Bölüm bazlı hoca listelerini toplu çek** (her ders için ayrı call yerine):
   - Unique department'ları belirle
   - Her department için: `GET /api/v1/staff/instructors?department={dept}&status=active`
   - Instructor map oluştur: `map[instructor_id]{first_name, last_name}`
5. **Her ders için döngü** (Single Transaction with Outbox):
   - Instructor map'ten hoca bilgisini al
     - ❌ Hoca artık aktif değilse: Skip course (log warning)
     - ✅ Aktifse: `instructor_fullname = first_name + " " + last_name`
   - **INSERT** new semester_courses row (with credits, class_level, assessment_schema + fresh instructor_fullname)
   - **COPY** course_schedule_sessions (yeni semester_course_id ile)
   - **INSERT** outbox event: `course.semester.created` (assessment_schema dahil, aynı transaction içinde)
6. **Background Worker**: Outbox'tan eventleri RabbitMQ'ya publish eder
7. Return summary (courses_copied, skipped_count)

**Use Case**: Yeni dönem açılırken önceki yılın aynı döneminden zorunlu dersleri kopyalar (2025 Spring açılıyor → 2024 Spring'den kopyala). Assessment schema da kopyalanır.

---

### 🔓 GET /api/v1/semesters/:semester_id/courses
Dönemlik ders listesi

**Role Requirement**: Authenticated

**Query Parameters**: `page`, `limit`, `faculty`, `department`, `instructor_id`, `course_type`, `class_level`

**Response** (200 - UPDATED):
```json
{
  "data": [
    {
      "id": "uuid",
      "semester": "2025_spring",
      "course_code": "CS101",
      "course_name": "Data Structures",
      "credits": 6,
      "class_level": 2,
      "instructor_id": "uuid",
      "instructor_fullname": "Ayşe Demir",
      "classroom_location": "A Blok 301",
      "max_capacity": 150,
      "assessment_schema": [
        {"slug": "midterm", "name": "Vize", "weight": 40},
        {"slug": "final", "name": "Final", "weight": 60}
      ],
      "schedule_sessions": [
        {
          "day_of_week": "monday",
          "slot_numbers": [1, 2, 3]
        }
      ]
    }
  ],
  "pagination": {"page": 1, "limit": 20, "total": 120, "total_pages": 6}
}
```

**Business Logic**:
```sql
SELECT sc.id, sc.semester, sc.course_code, cc.name as course_name, sc.credits, sc.class_level,
       sc.instructor_id, sc.instructor_fullname, sc.classroom_location, sc.max_capacity, sc.assessment_schema
FROM semester_courses sc
JOIN course_catalog cc ON sc.course_code = cc.course_code
WHERE sc.semester = $1
  AND ($2::text IS NULL OR cc.faculty = $2)
  AND ($3::text IS NULL OR cc.department = $3)
  AND ($4::uuid IS NULL OR sc.instructor_id = $4)
  AND ($5::course_type_enum IS NULL OR cc.course_type = $5)
  AND ($6::SMALLINT IS NULL OR sc.class_level = $6)
ORDER BY sc.course_code
LIMIT $7 OFFSET $8
```
- `credits`: Directly from semester_courses table (cached snapshot for ECTS calculation)
- `instructor_fullname`: Directly from semester_courses table (cached snapshot)
- `assessment_schema`: Directly from semester_courses table
- Schedule sessions: Separate query per course, GROUP BY day_of_week
- `current_enrollment` and `available_seats` NOT included (Enrollment Service responsibility)

---

### 🔓 GET /api/v1/semesters/:semester_id/courses/:course_id
Dönemlik ders detayı

**Role Requirement**: Authenticated

**Response** (200 - UPDATED):
```json
{
  "id": "uuid",
  "semester": "2025_spring",
  "course_code": "CS101",
  "course_name": "Data Structures",
  "credits": 6,
  "class_level": 2,
  "instructor_id": "uuid",
  "instructor_fullname": "Ayşe Demir",
  "classroom_location": "A Blok 301",
  "max_capacity": 150,
  "assessment_schema": [
    {"slug": "midterm", "name": "Vize", "weight": 40},
    {"slug": "final", "name": "Final", "weight": 60}
  ],
  "schedule_sessions": [
    {
      "day_of_week": "monday",
      "slot_numbers": [1, 2, 3]
    },
    {
      "day_of_week": "wednesday",
      "slot_numbers": [4, 5]
    }
  ],
  "prerequisites": [
    {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
  ],
  "created_at": "2025-09-01T10:00:00Z",
  "updated_at": "2025-09-01T10:00:00Z"
}
```

**Business Logic**:
```sql
SELECT sc.id, sc.semester, sc.course_code, cc.name as course_name, sc.credits, sc.class_level,
       sc.instructor_id, sc.instructor_fullname, sc.classroom_location, sc.max_capacity,
       sc.assessment_schema, cc.prerequisites, sc.created_at, sc.updated_at
FROM semester_courses sc
JOIN course_catalog cc ON sc.course_code = cc.course_code
WHERE sc.id = $1 AND sc.semester = $2
```
- ❌ Bulunamazsa: `404 SEMESTER_COURSE_NOT_FOUND`
- `credits`: Directly from semester_courses table (cached snapshot for ECTS calculation)
- `instructor_fullname`: Directly from semester_courses table (cached snapshot)
- `assessment_schema`: Directly from semester_courses table
- `prerequisites`: JOIN with course_catalog (JSONB array `[{id, course_code, course_name}]`)
- Schedule sessions: Separate query with GROUP BY day_of_week
- `current_enrollment` NOT included (Enrollment Service responsibility)

---

### 🔒 PUT /api/v1/semesters/:semester_id/courses/:course_id
Dönemlik ders güncelleme (tüm alanlar)

**Role Requirement**: Admin

**Request** (All fields optional - UPDATED):
```json
{
  "instructor_id": "uuid-new-instructor",
  "schedule_sessions": [
    {
      "day_of_week": "tuesday",
      "slot_numbers": [3, 4, 5]
    }
  ],
  "classroom_location": "B Blok 205",
  "max_capacity": 200,
  "assessment_schema": [
    {"slug": "quiz_1", "name": "Quiz 1", "weight": 10},
    {"slug": "midterm", "name": "Vize", "weight": 30},
    {"slug": "final", "name": "Final", "weight": 60}
  ]
}
```

**Response** (200 - UPDATED):
```json
{
  "id": "uuid",
  "semester": "2025_spring",
  "course_code": "CS101",
  "course_name": "Data Structures",
  "credits": 6,
  "class_level": 2,
  "instructor_id": "uuid",
  "instructor_fullname": "Mehmet Yılmaz",
  "classroom_location": "B Blok 205",
  "max_capacity": 200,
  "assessment_schema": [
    {"slug": "quiz_1", "name": "Quiz 1", "weight": 10},
    {"slug": "midterm", "name": "Vize", "weight": 30},
    {"slug": "final", "name": "Final", "weight": 60}
  ],
  "schedule_sessions": [
    {
      "day_of_week": "tuesday",
      "slot_numbers": [3, 4, 5]
    }
  ],
  "updated_at": "2025-11-12T11:00:00Z"
}
```

**RabbitMQ Events Published** (via Outbox Pattern):
- `course.semester.updated` (her güncelleme için - assessment_schema dahil)
- `course.instructor.changed` (ek olarak, sadece `instructor_id` değiştiyse - Grades Service için)

**Business Logic**:
1. **Instructor değişikliği** (eğer `instructor_id` gönderildiyse):
   - Dersin bölümünü course_catalog'dan al
   - **REST API Call**: `GET /api/v1/staff/instructors?department={department}&status=active` - Staff Service'ten hoca listesi
     - Request'teki `instructor_id`, dönen listedeki hocalardan birisi olmalı
     - ❌ Listede yoksa: `404 INSTRUCTOR_NOT_IN_DEPARTMENT` veya `404 INSTRUCTOR_NOT_ACTIVE`
   - Matching instructor'ın first_name + last_name'ini al
   - **Instructor fullname oluştur**: `instructor_fullname = first_name + " " + last_name`
   - **Conflict check**: Yeni hoca, mevcut schedule'da başka derste var mı?
     - ❌ Varsa: `409 INSTRUCTOR_SCHEDULE_CONFLICT`
   - **Old instructor bilgisini** DB'den çek (event için)

2. **Schedule değişikliği** (eğer `schedule_sessions` gönderildiyse):
   - **Slot validation**: Slot numbers 1-9 arası mı?
   - **Instructor conflict check**: Mevcut hoca, aynı semester'da yeni schedule'da başka derste var mı?
     ```sql
     SELECT sc.course_code
     FROM course_schedule_sessions css
     JOIN semester_courses sc ON css.semester_course_id = sc.id
     WHERE css.day_of_week = ANY($1)
       AND css.slot_number = ANY($2)
       AND sc.semester = $3              -- Aynı semester
       AND sc.instructor_id = $4         -- Mevcut hoca
       AND sc.id != $5                   -- Kendisi hariç
     ```
     - ❌ Varsa: `409 INSTRUCTOR_SCHEDULE_CONFLICT`

3. **Assessment schema değişikliği** (eğer `assessment_schema` gönderildiyse - ✅ NEW):
   - **Validation** (POST ile aynı kurallar):
     - Array boş olmamalı (minimum 1 eleman)
     - Her item'da `slug`, `name`, `weight` zorunlu
     - `slug`: Unique within array, lowercase, regex `^[a-z][a-z0-9_]*$`
     - `name`: UTF-8, max 100 karakter
     - `weight`: 0-100 arası integer
     - Toplam weight = 100 olmalı
   - ❌ Validation fail → `400 INVALID_ASSESSMENT_SCHEMA`
   - ⚠️ **Not**: Assessment schema değişikliği her zaman izin verilir. Grades Service event'i consume ederken mevcut notlarla uyumsuzluk varsa log'a yazar (eventual consistency).

4. **Single Transaction** (Outbox Pattern):
   ```sql
   BEGIN;

   -- 4a. semester_courses tablosunu güncelle (assessment_schema dahil)
   UPDATE semester_courses
   SET instructor_id = COALESCE($1, instructor_id),
       instructor_fullname = COALESCE($2, instructor_fullname),
       classroom_location = COALESCE($3, classroom_location),
       max_capacity = COALESCE($4, max_capacity),
       assessment_schema = COALESCE($5, assessment_schema),
       updated_at = NOW()
   WHERE id = $6;

   -- 4b. Schedule değiştiyse: Eski sessions'ları sil, yenilerini ekle
   DELETE FROM course_schedule_sessions WHERE semester_course_id = $6;
   INSERT INTO course_schedule_sessions (semester_course_id, day_of_week, slot_number)
   VALUES ($6, 'tuesday', 3), ($6, 'tuesday', 4), ...;

   -- 4c. outbox'a course.semester.updated eventi ekle (assessment_schema dahil)
   INSERT INTO outbox_events (event_type, routing_key, payload)
   VALUES ('course.semester.updated', 'course.semester.updated', '{"event_id": "...", "data": {..., "assessment_schema": [...]}}');

   -- 4d. Instructor değiştiyse: course.instructor.changed eventi de ekle
   INSERT INTO outbox_events (event_type, routing_key, payload)
   VALUES ('course.instructor.changed', 'course.instructor.changed', '{"event_id": "...", "data": {...}}');

   COMMIT;
   ```

5. **Background Worker**: Outbox'tan eventleri RabbitMQ'ya publish eder (100ms polling)

**Notes**:
- `classroom_location` ve/veya `max_capacity` güncelleme
- ℹ️ Enrollment Service'teki `current_enrollment` kontrol edilmez (Course Catalog sorumluluğu değil)

**Frontend Usage Pattern**:
```typescript
// 1. Ders bilgisini çek (mevcut durumu görmek için)
GET /api/v1/semesters/2025_spring/courses/{courseId}

// 2. Bölüm hocalarını çek (dropdown için)
GET /api/v1/staff/instructors?department={course.department}&status=active

// 3. Güncellemeleri gönder (tek request, istediğin alanları gönder)
PUT /api/v1/semesters/2025_spring/courses/{courseId}
{
  "instructor_id": "selected-instructor-id",
  "max_capacity": 200,
  "assessment_schema": [...]
}
```

**Error Cases**:
- `400 INVALID_SLOT_NUMBER`: Slot number 1-9 arasında değil
- `400 INVALID_ASSESSMENT_SCHEMA`: Assessment schema validation hatası
- `404 INSTRUCTOR_NOT_IN_DEPARTMENT`: Seçilen hoca bu bölümde değil
- `404 INSTRUCTOR_NOT_ACTIVE`: Hoca pasif durumda (Staff Service'te is_active = false)
- `409 INSTRUCTOR_SCHEDULE_CONFLICT`: Hoca aynı gün-slot'ta başka derste

---

### 🔒 DELETE /api/v1/semesters/:semester_id/courses/:course_id
Dönemlik ders silme

**Role Requirement**: Admin

**Response** (200):
```json
{
  "message": "Semester course deleted successfully",
  "semester_course_id": "uuid",
  "course_code": "CS101",
  "semester": "2025_spring"
}
```

**RabbitMQ Event Published**: `course.semester.deleted` (via Outbox Pattern)

**Business Logic**:
1. **Existence check**: Dönemlik ders var mı?
   ```sql
   SELECT id, semester, course_code FROM semester_courses WHERE id = $1 AND semester = $2
   ```
   - ❌ Yoksa: `404 SEMESTER_COURSE_NOT_FOUND`

2. **Enrollment check** (Optional - Policy Decision):
   - ⚠️ **Dikkat**: Eğer derse kayıtlı öğrenci varsa silme işlemi tehlikeli
   - İki seçenek:
     - **Strict**: Kayıtlı öğrenci varsa silme engelle (`409 COURSE_HAS_ENROLLMENTS`)
     - **Force**: Silmeye izin ver, Enrollment Service event ile öğrenci kayıtlarını siler
   - ℹ️ Bu implementasyonda **Force** seçeneği kullanılıyor (event-driven cleanup)

3. **Single Transaction** (Outbox Pattern):
   ```sql
   BEGIN;

   -- 3a. Ders bilgilerini al (event payload için)
   SELECT sc.*, cc.name as course_name, cc.faculty, cc.department, cc.course_type, cc.prerequisites
   FROM semester_courses sc
   JOIN course_catalog cc ON sc.course_code = cc.course_code
   WHERE sc.id = $1;

   -- 3b. Schedule sessions'ları al (event payload için)
   SELECT day_of_week, slot_number FROM course_schedule_sessions WHERE semester_course_id = $1;

   -- 3c. outbox'a course.semester.deleted eventi ekle
   INSERT INTO outbox_events (event_type, routing_key, payload)
   VALUES ('course.semester.deleted', 'course.semester.deleted', '{"event_id": "...", "data": {...}}');

   -- 3d. course_schedule_sessions sil (CASCADE ile otomatik silinir)
   -- 3e. semester_courses tablosundan sil
   DELETE FROM semester_courses WHERE id = $1;

   COMMIT;
   ```

4. **Background Worker**: Outbox'tan eventi RabbitMQ'ya publish eder (100ms polling)

**Event Consumers**:
- **Enrollment Service**: `courses_cache` ve `course_sessions_cache` tablosundan siler
- **Notification Service**: İlgili öğrencilere ders iptal bildirimi gönderir (optional)

**Error Cases**:
- `404 SEMESTER_COURSE_NOT_FOUND`: Dönemlik ders bulunamadı
- `409 COURSE_HAS_ENROLLMENTS`: Derse kayıtlı öğrenci var (strict mode'da silme engeli)

> **📌 TODO (Gelecek Geliştirme)**: `semesters` tablosuna `course_freeze_date DATE NOT NULL` alanı eklenecek. Enrollment başlangıç tarihinden 1 gün önce (freeze_date) ve sonrasında dönemlik ders silme işlemi engellenecek.

**Use Cases**:
1. Yanlışlıkla açılan ders iptali
2. Yeterli katılım olmayan dersin kapatılması
3. Hoca ayrılması durumunda ders iptali

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Exchange: "course.events" (type: topic)

Routing Keys:
- course.semester.created      → Dönemlik ders oluşturulduğunda (via Outbox)
- course.semester.updated      → Dönemlik ders güncellendiğinde (via Outbox)
- course.semester.deleted      → Dönemlik ders silindiğinde (via Outbox)
- course.instructor.changed    → Hoca değişikliği (NADİR, update ile birlikte gönderilir)
- course.prerequisites.updated → Prerequisite listesi değiştiğinde (Grades Service için)

Dead Letter Queue:
- DLQ Exchange: "course.events.dlq"
- DLQ Queue: "course.events.dlq.queue"
```

### Event Schemas

#### course.semester.created
Published when: Dönemlik ders oluşturulduğunda (POST endpoint via Outbox)

```json
{
  "event_id": "uuid",
  "event_type": "course.semester.created",
  "timestamp": "2025-11-12T10:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_spring",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "faculty": "Mühendislik Fakültesi",
    "department": "Bilgisayar Mühendisliği",
    "credits": 6,
    "class_level": 2,
    "course_type": "mandatory",
    "instructor_id": "uuid",
    "instructor_fullname": "Ayşe Demir",
    "classroom_location": "A Blok 301",
    "max_capacity": 150,
    "assessment_schema": [
      {"slug": "midterm", "name": "Vize", "weight": 40},
      {"slug": "final", "name": "Final", "weight": 60}
    ],
    "prerequisites": [
      {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
    ],
    "schedule_sessions": [
      {
        "day_of_week": "monday",
        "slot_numbers": [1, 2, 3]
      },
      {
        "day_of_week": "wednesday",
        "slot_numbers": [4, 5]
      }
    ]
  }
}
```

**Notes**:
- `event_id`: Outbox table'dan gelen UUID (duplicate detection için)
- `instructor_fullname`: Request'ten alınan immutable snapshot
- `class_level`: Dersin hedef sınıf seviyesi (Enrollment Service filtreleme için)
- `assessment_schema`: ✅ NEW - Sınav yapısı (Grades Service not girişi için kullanır)
- `schedule_sessions`: Array of day-slot combinations
- `prerequisites`: Denormalized array `[{id, course_code, course_name}]` - Read-heavy optimization
- Enrollment Service: `semester_course_id` ile INSERT yapar
- Grades Service: `assessment_schema` ile `courses_cache` tablosuna INSERT yapar

**Consumer Behavior (Enrollment Service)**:
```sql
-- INSERT new course to cache (assessment_schema ignored - not needed for enrollment)
INSERT INTO courses_cache (id, course_code, name, department, max_capacity, ...)
VALUES ($1, $2, $3, $4, $5, ...);

-- INSERT schedule sessions
INSERT INTO course_sessions_cache (course_id, day_of_week, slot_number)
VALUES ($1, 'monday', 1), ($1, 'monday', 2), ...;
```

**Consumer Behavior (Grades Service)**:
```sql
-- INSERT new course to cache (assessment_schema included)
INSERT INTO courses_cache (id, course_code, name, credits, semester, department,
                           instructor_id, instructor_fullname, assessment_schema, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW());
```

#### course.semester.updated
Published when: Dönemlik ders güncellendiğinde (PUT endpoint via Outbox)

```json
{
  "event_id": "uuid",
  "event_type": "course.semester.updated",
  "timestamp": "2025-11-12T11:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_spring",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "faculty": "Mühendislik Fakültesi",
    "department": "Bilgisayar Mühendisliği",
    "credits": 6,
    "class_level": 2,
    "course_type": "mandatory",
    "instructor_id": "uuid",
    "instructor_fullname": "Mehmet Yılmaz",
    "classroom_location": "B Blok 205",
    "max_capacity": 200,
    "assessment_schema": [
      {"slug": "quiz_1", "name": "Quiz 1", "weight": 10},
      {"slug": "midterm", "name": "Vize", "weight": 30},
      {"slug": "final", "name": "Final", "weight": 60}
    ],
    "prerequisites": [
      {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
    ],
    "schedule_sessions": [
      {
        "day_of_week": "tuesday",
        "slot_numbers": [3, 4, 5]
      }
    ]
  }
}
```

**Notes**:
- Schema `course.semester.created` ile **aynı** (full state gönderilir, `assessment_schema` dahil)
- Enrollment Service: Tüm alanları günceller (idempotent UPSERT), `assessment_schema` ignore
- Grades Service: `assessment_schema` dahil tüm alanları günceller
- `current_enrollment` korunur (Enrollment Service owns this data)

**Consumer Behavior (Enrollment Service)**:
```sql
-- UPDATE course cache (preserve current_enrollment, assessment_schema ignored)
UPDATE courses_cache
SET course_code = $2, name = $3, department = $4, max_capacity = $5, synced_at = NOW()
WHERE id = $1;

-- Replace schedule sessions
DELETE FROM course_sessions_cache WHERE course_id = $1;
INSERT INTO course_sessions_cache (course_id, day_of_week, slot_number)
VALUES ($1, 'tuesday', 3), ($1, 'tuesday', 4), ($1, 'tuesday', 5);
```

**Consumer Behavior (Grades Service)**:
```sql
-- UPDATE course cache (assessment_schema included)
UPDATE courses_cache
SET course_code = $2,
    name = $3,
    credits = $4,
    instructor_id = $5,
    instructor_fullname = $6,
    assessment_schema = $7,
    synced_at = NOW()
WHERE id = $1;

-- ⚠️ WARNING: If assessment_schema changed and scores already exist,
-- log warning but allow update (eventual consistency)
```

#### course.semester.deleted
Published when: Dönemlik ders silindiğinde (DELETE endpoint via Outbox)

```json
{
  "event_id": "uuid",
  "event_type": "course.semester.deleted",
  "timestamp": "2025-11-12T12:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_spring",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "department": "Bilgisayar Mühendisliği"
  }
}
```

**Notes**:
- Minimal payload (sadece silme için gerekli bilgiler)
- Enrollment Service: Cache'ten siler, kayıtlı öğrencileri handle eder
- Grades Service: Cache'ten siler

**Consumer Behavior (Enrollment Service)**:
```sql
-- DELETE course sessions cache (CASCADE ile otomatik silinebilir)
DELETE FROM course_sessions_cache WHERE course_id = $1;

-- DELETE course cache
DELETE FROM courses_cache WHERE id = $1;

-- ⚠️ IMPORTANT: Handle enrolled students
-- Option 1: Enrollment program'larından dersi çıkar
-- Option 2: Program'ı invalid olarak işaretle
-- Option 3: Notification gönder, admin müdahalesi bekle
```

**Consumer Behavior (Grades Service)**:
```sql
-- DELETE course cache
DELETE FROM courses_cache WHERE id = $1;

-- Note: student_course_registrations may have FK constraint
-- Handle appropriately based on policy
```

#### course.instructor.changed
Published when: Hoca değişikliği olduğunda (NADİR)

```json
{
  "event_id": "uuid",
  "event_type": "course.instructor.changed",
  "timestamp": "2025-11-12T12:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_fall",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "old_instructor_id": "uuid-old",
    "old_instructor_fullname": "Ayşe Demir",
    "new_instructor_id": "uuid-new",
    "new_instructor_fullname": "Mehmet Yılmaz"
  }
}
```

---

#### course.prerequisites.updated
Published when: Katalog derslerinde prerequisite değişikliği olduğunda

**Trigger Conditions**:
- Yeni ders oluşturulduğunda (prerequisites array varsa)
- Mevcut ders güncellendiğinde (prerequisites değiştiyse)
- Ders silindiğinde (prerequisite olarak kullanılıyorsa)

**Publisher Logic**:
```go
// Her CRUD işleminden sonra çağrılır (prerequisites değişikliği varsa)
func (s *CourseService) publishPrerequisiteListUpdate(ctx context.Context) error {
    // Tüm unique prerequisite course bilgilerini çek (course_code + course_id)
    query := `
        SELECT DISTINCT
            prereq->>'course_code' as course_code,
            prereq->>'id' as course_id
        FROM course_catalog,
             jsonb_array_elements(prerequisites) as prereq
        WHERE jsonb_array_length(prerequisites) > 0
    `

    // Full list event yayınla (idempotent)
    return s.publisher.Publish(ctx, "course.events", "course.prerequisites.updated", event)
}
```

**Event Schema** (Full List Sync):
```json
{
  "event_id": "uuid",
  "event_type": "course.prerequisites.updated",
  "timestamp": "2025-11-23T10:00:00Z",
  "data": {
    "prerequisite_courses": [
      {"course_code": "CS100", "course_id": "uuid-cs100"},
      {"course_code": "CS101", "course_id": "uuid-cs101"},
      {"course_code": "CS102", "course_id": "uuid-cs102"},
      {"course_code": "MATH101", "course_id": "uuid-math101"},
      {"course_code": "MATH201", "course_id": "uuid-math201"},
      {"course_code": "PHYS101", "course_id": "uuid-phys101"}
    ],
    "updated_at": "2025-11-23T10:00:00Z"
  }
}
```

**Notes**:
- **Full list sync**: Her değişiklikte TÜM prerequisite course bilgileri gönderilir (course_code + course_id)
- **Idempotent**: Consumer TRUNCATE + INSERT yapar (sıra önemsiz)
- **Small payload**: ~100-200 course (tipik üniversite için)
- **Rare event**: Prerequisite değişikliği nadiren olur (yılda birkaç kez)

**Event Consumers**: Grades Service (prerequisite filtering için cache günceller)

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_INPUT | Validation hatası |
| 400 | INVALID_PREREQUISITE | Prerequisite course geçersiz |
| 400 | INVALID_PREREQUISITE_LEVEL | Prerequisite class_level >= course class_level (✅ NEW) |
| 400 | INVALID_SLOT_NUMBER | Slot number 1-9 arasında değil |
| 400 | INVALID_STATUS | Geçersiz catalog status (ENUM dışında) |
| 400 | INVALID_COURSE_TYPE | Geçersiz course type (ENUM dışında) |
| 400 | INVALID_ASSESSMENT_SCHEMA | Assessment schema validation hatası |
| 400 | COURSE_NOT_ACTIVE | Ders active değil (dönemlik açılış için active olmalı) |
| 400 | CLASS_LEVEL_MISMATCH | Request class_level ile catalog class_level uyuşmuyor |
| 400 | SEMESTER_ALREADY_EXISTS | Target semester zaten oluşturulmuş |
| 403 | FORBIDDEN | Sadece admin ders açabilir/güncelleyebilir/silebilir |
| 404 | COURSE_NOT_FOUND | Katalog dersi bulunamadı |
| 404 | SEMESTER_COURSE_NOT_FOUND | Dönemlik ders bulunamadı |
| 404 | INSTRUCTOR_NOT_FOUND | Instructor bulunamadı (Staff Service'te yok) |
| 404 | INSTRUCTOR_NOT_IN_DEPARTMENT | Seçilen hoca bu bölümde değil |
| 404 | INSTRUCTOR_NOT_ACTIVE | Hoca pasif durumda (is_active = false) |
| 404 | SOURCE_SEMESTER_NOT_FOUND | Kopyalanacak dönem bulunamadı |
| 409 | COURSE_CODE_EXISTS | Ders kodu zaten kayıtlı |
| 409 | COURSE_ALREADY_OPENED | Bu ders bu dönemde zaten açılmış |
| 409 | INSTRUCTOR_SCHEDULE_CONFLICT | Hoca aynı gün-slot'ta başka derste |
| 409 | COURSE_HAS_ENROLLMENTS | Derse kayıtlı öğrenci var (strict mode'da silme engeli) |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Related Services

- **Staff Service**: Bölüm bazlı öğretim görevlisi listesi (REST API)
  - Endpoint: `GET /api/v1/staff/instructors?department={dept}&status=active`
  - Returns: `[{id, first_name, last_name, department}]`
  - Kullanım:
    - Ders açılırken instructor validation (hoca bu bölümde mi?)
    - Instructor fullname oluşturma (first_name + " " + last_name)
    - Instructor değişikliğinde validation
    - Frontend'de instructor dropdown
- **Enrollment Service**: Event consumer (`course.semester.created`, `course.semester.updated`, `course.semester.deleted`)
  - Course schedule ve capacity bilgilerini cache'ler
  - Enrollment Service manages `current_enrollment` (NOT Course Catalog)
  - ℹ️ `assessment_schema` ignore edilir (Enrollment Service kullanmaz)
- **Grades Service**: Event consumer (`course.semester.created`, `course.semester.updated`, `course.instructor.changed`, `course.prerequisites.updated`)
  - ✅ `assessment_schema` consume eder (not girişi formu için)
  - Prerequisite listesini cache'ler (event filtering için)

---

**Version**: 5.4.0 (Class level validation for prerequisites)
**Last Updated**: 2025-12-05

**Changes in v5.4.0**:
- ✅ Added `class_level` field to `course_catalog` table
- ✅ Added prerequisite class level validation rule: prerequisite's class_level must be less than course's class_level
- ✅ Removed circular dependency check (unnecessary due to natural class level hierarchy)
- ✅ Added `400 INVALID_PREREQUISITE_LEVEL` error code
- ✅ Updated POST /api/v1/catalog/courses to validate prerequisite class levels
- ✅ Updated PUT /api/v1/catalog/courses/:course_code to validate prerequisite class levels
- ✅ Added `class_level` filter to GET /api/v1/catalog/courses
- ✅ Added index on `class_level` column
- 🎯 Pattern: Natural hierarchy validation replaces complex graph traversal
- 📊 Benefit: Simpler validation logic, mathematically guaranteed no circular dependencies

**Previous Changes (v5.3.0)**:
- ✅ Added `assessment_schema JSONB` to `semester_courses` table
- ✅ Added assessment schema validation rules (slug uniqueness, weight sum = 100, etc.)
- ✅ Updated POST endpoint to accept `assessment_schema` (optional, has default)
- ✅ Updated PUT endpoint to accept `assessment_schema` (optional)
- ✅ Updated GET endpoints to return `assessment_schema`
- ✅ Updated `course.semester.created` event schema (includes `assessment_schema`)
- ✅ Updated `course.semester.updated` event schema (includes `assessment_schema`)
- ✅ Added Grades Service as consumer for `course.semester.created` and `course.semester.updated`
- ✅ Added `400 INVALID_ASSESSMENT_SCHEMA` error code
- ✅ Updated copy-from-previous logic to include `assessment_schema`
- 🎯 Pattern: Assessment schema flows from Course Catalog → Grades Service via RabbitMQ
- 📊 Benefit: Admin can configure exam structure per course, Grades Service knows how many scores to collect

**Previous Changes (v5.2.0)**:
- ✅ Added `class_level` field to `semester_courses` table (1-6: Sınıflar)
- ✅ `class_level` uses SMALLINT (1-6) consistent with Student Service (source of truth)
- ✅ Updated all API endpoints to include `class_level` (POST create, GET list, GET detail, PUT update)
- ✅ Updated event schemas: `course.semester.created` and `course.semester.updated` now include `class_level`
- ✅ Standardized prerequisites format: `{id, course_code, course_name}` (consistent naming)
- ✅ Fixed routing key consistency: All events now use `course.semester.*` format
- 🎯 Pattern: Schema alignment between Course Catalog and Enrollment Service
- 📊 Benefit: Enrollment Service can now filter courses by class_level for students

**Previous Changes (v5.1.0)**:
- ✅ Added `course.prerequisites.updated` event (full list sync)
- ✅ Event published on catalog CRUD when prerequisites change
- ✅ Grades Service consumes this event for filtering `grade.student.prerequisite.passed` events
- 🎯 Pattern: Full list sync (idempotent, no delta tracking needed)
- 📊 Benefit: Enables source-side filtering in Grades Service (50% less RabbitMQ traffic)

**Previous Changes (v5.0.0)**:
- ✅ Real-time CRUD events for Course-Enrollment sync
- ✅ Added `course.semester.created` event (POST endpoint via Outbox)
- ✅ Added `course.semester.updated` event (PUT endpoint via Outbox)
- ✅ Added `course.semester.deleted` event (DELETE endpoint via Outbox)
- ✅ Added `DELETE /api/v1/semesters/:semester_id/courses/:course_id` endpoint
- 🎯 Pattern: All CRUD operations write to outbox in same transaction
- 📊 Benefit: Real-time event-driven sync between Course Catalog and Enrollment Service