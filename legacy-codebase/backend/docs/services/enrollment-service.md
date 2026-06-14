# Enrollment Service ⚠️ CRITICAL

## Sorumluluk
Ders programı oluşturma, danışman onayı, kontenjan yönetimi, kademeli kayıt açılışı, prerequisite kontrolü

**Kritik Servis**: En yüksek concurrency ve race condition riski

**Önemli**: Bu servis **program-based enrollment** kullanır. Öğrenciler tek tek ders kayıt yapmaz, tüm dönem ders programını bir seferde gönderir ve danışman onayı bekler.

**Kayıt Akışı**:
1. Öğrenci login olur (JWT token alır)
2. `GET /api/v1/enrollments/available-courses` ile alabileceği dersleri çeker (bölüm ve sınıf seviyesine göre filtrelenmiş)
3. Frontend'de ders seçimi yapar, çakışan dersler varsa gönder butonu disable edilir
4. `POST /api/v1/enrollments` ile tüm ders programını gönderir
5. Backend çakışma ve kontenjan kontrolü yapar (güvenlik için double-check)
6. **Kontenjan arttırılır** (her ders için `current_enrollment++`)
7. Program `pending` status ile kaydedilir
8. Danışman hoca onaylar veya reddeder:
   - **Onay**: Kontenjan değişmez (zaten kaplanmış), Grades/Attendance servisleri kayıt oluşturur
   - **Red**: **Kontenjan azaltılır** (her ders için `current_enrollment--`), program silinir, red kaydı oluşturulur, öğrenci tekrar ders kaydı yapabilir

---

## İletişim

### Inbound (RabbitMQ - Event Consumers)
- `student.created` → Öğrenci bilgisini local DB'ye ekler (advisor_id, class_level, department, email dahil)
- `student.updated` → Öğrenci bilgisini günceller (danışman değişikliği, sınıf seviyesi güncellemesi, email değişikliği)
- `student.deactivated` → Öğrenciyi deaktif yapar (`is_active = false`), aktif enrollment program'ları iptal edilir
- `course.semester.created` → Dönemlik ders bilgisini local DB'ye ekler (via Outbox Pattern)
- `course.semester.updated` → Dönemlik ders bilgisini günceller (via Outbox Pattern)
- `course.semester.deleted` → Dönemlik dersi local DB'den siler (via Outbox Pattern)
- `grade.student.prerequisite.passed` → Önkoşul dersini GEÇEN öğrenciyi `student_passed_prerequisites` tablosuna ekler (prerequisite validation için)

### Outbound (RabbitMQ - Asynchronous)
- `enrollment.program_submitted` → Öğrenci ders programı gönderdi (Notification)
- `enrollment.program_approved` → Danışman onayladı (Grades, Attendance, Notification)
- `enrollment.program_rejected` → Danışman reddetti (Notification)

---

## Database Schema

```sql
-- Enums
CREATE TYPE enrollment_status_enum AS ENUM ('pending', 'approved');
CREATE TYPE day_of_week_enum AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');
-- Note: class_level uses SMALLINT (1-6) consistent with Student Service (source of truth)
CREATE TYPE course_type_enum AS ENUM ('mandatory', 'elective');

-- Note: semester uses VARCHAR(50) format: "2025_spring", "2025_fall", "2026_summer"
-- No enum needed - more flexible for future semesters

-- Local student cache (synced from Student Service via RabbitMQ events)
CREATE TABLE students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,              -- Email adresi (notification event'leri için)
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    department VARCHAR(100),
    class_level SMALLINT CHECK (class_level BETWEEN 1 AND 6),  -- 1-6: Sınıf seviyeleri
    advisor_id UUID,
    status VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,           -- Öğrenci aktif mi? (student.deactivated ile false olur)
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;

-- Local semester course cache (synced from Course Catalog via RabbitMQ events)
CREATE TABLE semester_courses_cache (
    id UUID PRIMARY KEY,                              -- semester_course_id from event (1:1 mapping)
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255),
    faculty VARCHAR(100),
    department VARCHAR(100),
    credits SMALLINT NOT NULL,                        -- ECTS credits (from event)
    course_type course_type_enum NOT NULL,            -- mandatory/elective
    class_level SMALLINT CHECK (class_level BETWEEN 1 AND 6),  -- 1-6: Sınıf seviyeleri                     -- Hangi sınıf seviyesi için açık (1-6: Sınıflar)
    semester VARCHAR(50) NOT NULL,                    -- "2025_spring" format (from event)
    instructor_id UUID,
    instructor_fullname VARCHAR(150),
    classroom_location VARCHAR(100),
    max_capacity SMALLINT NOT NULL,
    current_enrollment SMALLINT DEFAULT 0 CHECK (current_enrollment >= 0),  -- Negatif değer olamaz
    prerequisites JSONB DEFAULT '[]',                 -- [{id, course_code, course_name}] for prerequisite validation
    synced_at TIMESTAMP DEFAULT NOW()
);

-- Note: Enrollment Service is the OWNER of current_enrollment data
-- Course Catalog Service does NOT store or manage current_enrollment
-- Note: semester_courses_cache.id = Event'teki semester_course_id (birebir eşleşme)

-- Course schedule sessions cache (slot-based scheduling)
CREATE TABLE course_sessions_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES semester_courses_cache(id) ON DELETE CASCADE,
    day_of_week day_of_week_enum NOT NULL,
    slot_number INT NOT NULL CHECK (slot_number BETWEEN 1 AND 9),
    synced_at TIMESTAMP DEFAULT NOW(),

    -- Prevent duplicate: Same course, same day, same slot
    UNIQUE(course_id, day_of_week, slot_number)
);

CREATE INDEX idx_sessions_cache_course ON course_sessions_cache(course_id);
CREATE INDEX idx_sessions_cache_day_slot ON course_sessions_cache(day_of_week, slot_number);

-- Note: Backend stores slot numbers only (1-9). Frontend maps to time ranges.

-- Student passed prerequisites (prerequisite validation için)
-- Sadece GEÇİLEN önkoşul dersler kaydedilir
-- Kayıt varsa = geçmiş (bu dersi önkoşul olarak gerektiren dersleri ALABİLİR)
-- Kayıt yoksa = geçmemiş veya hiç almamış (önkoşul gerektiren dersi ALAMAZ)
CREATE TABLE student_passed_prerequisites (
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_code VARCHAR(50) NOT NULL,       -- Stable identifier (e.g., "CS101")
    semester VARCHAR(50) NOT NULL,          -- Hangi dönem geçtiği
    grade_point VARCHAR(10),                -- Geçtiği not (e.g., "2.50")
    synced_at TIMESTAMP DEFAULT NOW(),

    PRIMARY KEY (student_id, course_code)   -- Composite PK = otomatik B-tree index
);

-- Enrollment programs (sadece pending ve approved programlar)
CREATE TABLE enrollment_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id),
    semester VARCHAR(50) NOT NULL,          -- "2025_spring", "2025_fall", "2026_summer"
    status enrollment_status_enum DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(student_id, semester)
);

-- Individual courses in a program
CREATE TABLE enrollment_program_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES enrollment_programs(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES semester_courses_cache(id),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(program_id, course_id)
);

-- Rejection logs (reddedilen programların tarihçesi)
-- Program silindiğinde bile öğrenci red sebebini ve hangi dersleri seçtiğini görebilir
CREATE TABLE enrollment_rejection_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_program_id UUID NOT NULL,      -- Silinen program'ın ID'si (referans/debug için, FK değil)
    student_id UUID NOT NULL REFERENCES students_cache(id),
    advisor_id UUID NOT NULL,               -- Reddeden danışman (FK yok - User Service'te)
    advisor_fullname VARCHAR(150) NOT NULL, -- Danışman adı (snapshot - join gerekmez)
    semester VARCHAR(50) NOT NULL,
    rejection_reason TEXT NOT NULL,         -- Danışmanın girdiği red sebebi
    rejected_courses JSONB NOT NULL,        -- Snapshot: Red anındaki ders bilgileri
    rejected_at TIMESTAMP DEFAULT NOW()
);

-- Note: advisor_id ve advisor_fullname snapshot olarak saklanır
-- Danışman bilgisi değişse bile red anındaki bilgi korunur
-- Foreign key yok çünkü advisor bilgisi User/Staff Service'te

-- rejected_courses JSONB yapısı:
-- {
--   "courses": [
--     {
--       "course_id": "uuid",
--       "course_code": "CS101",
--       "course_name": "Introduction to Computer Science",
--       "credits": 3,
--       "instructor": "Dr. Mehmet Öz"
--     }
--   ],
--   "total_credits": 18,
--   "submitted_at": "2025-11-20T10:00:00Z"
-- }

CREATE INDEX idx_rejection_logs_student ON enrollment_rejection_logs(student_id);
CREATE INDEX idx_rejection_logs_student_semester ON enrollment_rejection_logs(student_id, semester);

CREATE INDEX idx_programs_student ON enrollment_programs(student_id);
CREATE INDEX idx_programs_status ON enrollment_programs(status);
CREATE INDEX idx_programs_semester ON enrollment_programs(semester);
CREATE INDEX idx_students_cache_advisor ON students_cache(advisor_id);
CREATE INDEX idx_semester_courses_cache_department_semester ON semester_courses_cache(department, semester);
CREATE INDEX idx_semester_courses_cache_class_level ON semester_courses_cache(class_level);
-- Note: student_passed_prerequisites uses composite PRIMARY KEY (student_id, course_code)
-- No additional index needed - PK provides optimal lookup performance
```

---

## API Endpoints

### 🔒 GET /api/v1/enrollments/available-courses
Öğrencinin alabileceği dersleri listeler (kendi sınıfı + altındaki tüm sınıflar)

**Role Requirement**: Student

**Query Parameters**:
- `semester` (required): "2025_spring", "2025_fall", "2026_summer" format

**Request Flow**:
1. JWT token'dan `user_id` al
2. `students_cache` tablosundan öğrencinin bilgilerini çek
3. **is_active kontrolü**: `is_active = false` ise → 403 STUDENT_DEACTIVATED hatası döndür
4. `semester_courses_cache` tablosundan filtreleme yap:
   - `department = student.department`
   - `class_level <= student.class_level` ⚠️ **Öğrencinin sınıfına KADAR olan tüm dersler**
   - `semester = query.semester`

**Response** (200):
```json
{
  "student_id": "uuid",
  "department": "Computer Science",
  "class_level": 2,
  "semester": "2025_spring",
  "available_courses": [
    {
      "id": "uuid",
      "course_code": "CS201",
      "course_name": "Data Structures",
      "credits": 4,
      "schedule_sessions": [
        {
          "day_of_week": "monday",
          "slot_numbers": [1, 2, 3]
        }
      ],
      "max_capacity": 150,
      "current_enrollment": 85,
      "available_seats": 65,
      "instructor": "Dr. Mehmet Öz"
    },
    {
      "id": "uuid",
      "course_code": "CS202",
      "course_name": "Computer Organization",
      "credits": 3,
      "schedule_sessions": [
        {
          "day_of_week": "wednesday",
          "slot_numbers": [4, 5]
        }
      ],
      "max_capacity": 100,
      "current_enrollment": 100,
      "available_seats": 0,
      "instructor": "Prof. Ayşe Demir"
    }
  ]
}
```

**Business Logic**:
- **Class level filtering**: `class_level <= student.class_level` (öğrencinin sınıfına KADAR olan tüm dersler)
  - **Örnek 1**: 3. sınıf öğrenci → 1. sınıf + 2. sınıf + 3. sınıf derslerini görür
  - **Örnek 2**: 1. sınıf öğrenci → Sadece 1. sınıf derslerini görür
  - **Örnek 3**: 4. sınıf öğrenci → 1. sınıf + 2. sınıf + 3. sınıf + 4. sınıf derslerini görür
- Öğrencinin bölümüne göre filtreleme
- Kontenjan bilgisi (available_seats = max_capacity - current_enrollment)
- **Schedule sessions**: JOIN `course_sessions_cache`, GROUP BY `course_id` and `day_of_week`
- Kontenjan dolu dersler de gösterilir ama frontend'de disable edilir

**Use Case**: 3. sınıf öğrenci enrollment sayfasına geldiğinde:
- 1. ve 2. sınıf derslerinden kaldığı dersler → Tekrar almalı (prerequisite için)
- 3. sınıf dersleri → Normal şekilde ilk kez alır

---

### 🔒 POST /api/v1/enrollments
Ders programı gönderme (tüm dönem dersleri bir seferde)

**Role Requirement**: Student

**Request**:
```json
{
  "student_id": "uuid",
  "semester": "2025_spring",
  "course_ids": ["course-uuid-1", "course-uuid-2", "course-uuid-3", "course-uuid-4"]
}
```

**Response** (201):
```json
{
  "id": "uuid",
  "student_id": "uuid",
  "semester": "2025_spring",
  "status": "pending",
  "courses": [
    {
      "course_id": "uuid",
      "course_code": "CS101",
      "course_name": "Introduction to Computer Science",
      "credits": 3
    }
  ],
  "created_at": "2025-11-11T10:00:00Z"
}
```

**RabbitMQ Event Published**: `enrollment.program_submitted`

**Business Logic**:
1. **JWT Authorization**: Student ID doğrulama + kayıt dönemi kontrolü (JWT middleware'de yapılır)
2. **is_active kontrolü**: `is_active = false` ise → 403 STUDENT_DEACTIVATED hatası döndür
3. **Duplicate check**: Aynı dönem için daha önce program göndermemiş mi? (❌ 409 ALREADY_SUBMITTED)
4. **Department filtering**: Tüm dersler öğrencinin bölümünden mi?
5. **Class level check**: Tüm dersler öğrencinin sınıf seviyesine uygun mu?
6. **Prerequisite validation**: Her ders için önkoşul kontrolü
   - Ders için prerequisite var mı? (`semester_courses_cache.prerequisites` JSONB array)
   - Varsa: Her prerequisite için `student_passed_prerequisites` tablosunda ara (student_id + course_code)
   - ✅ Tüm önkoşullar GEÇİLMİŞSE → Devam (kayıt yapılabilir)
   - ❌ Herhangi bir önkoşul GEÇİLMEMİŞSE → 400 PREREQUISITES_NOT_MET (önce geçmeli)
   ```sql
   -- Prerequisite pass check query (geçmiş mi?)
   SELECT EXISTS(
       SELECT 1 FROM student_passed_prerequisites
       WHERE student_id = $1 AND course_code = $2
   );
   -- TRUE dönerse → önkoşul geçilmiş, kayıt yapılabilir ✅
   -- FALSE dönerse → önkoşul geçilmemiş veya hiç alınmamış, kayıt YAPILAMAZ ❌
   ```

   **Örnek Senaryo**:
   - CS201 dersi CS101'i önkoşul olarak gerektiriyor
   - Öğrenci CS101'i geçmişse → `student_passed_prerequisites`'te kayıt var → CS201'e kayıt OLABİLİR
   - Öğrenci CS101'i kalmışsa → `student_passed_prerequisites`'te kayıt yok → CS201'e kayıt OLAMAZ
   - Öğrenci CS101'i hiç almamışsa → `student_passed_prerequisites`'te kayıt yok → CS201'e kayıt OLAMAZ
7. **Schedule conflict validation (Backend - SIMPLIFIED)**: Seçilen derslerde slot çakışması var mı?
   ```sql
   SELECT cs1.course_id, cs2.course_id, cs1.day_of_week, cs1.slot_number
   FROM course_sessions_cache cs1
   JOIN course_sessions_cache cs2
     ON cs1.day_of_week = cs2.day_of_week
     AND cs1.slot_number = cs2.slot_number
   WHERE cs1.course_id = ANY($1)  -- Seçilen course_ids
     AND cs2.course_id = ANY($1)
     AND cs1.course_id != cs2.course_id;
   ```
   - ✅ Conflict yoksa devam et
   - ❌ Conflict varsa → 400 SCHEDULE_CONFLICT
8. **Capacity check & increment (PostgreSQL Transaction with Row-Level Locking)**:
   ```sql
   BEGIN;
   
   -- Row-level lock al (ID sırasına göre - deadlock önleme)
   SELECT id, current_enrollment, max_capacity 
   FROM semester_courses_cache 
   WHERE id = ANY($course_ids) 
   ORDER BY id
   FOR UPDATE;
   
   -- Application layer: Her ders için current_enrollment < max_capacity kontrolü
   -- Herhangi biri doluysa → ROLLBACK, 409 COURSE_FULL
   
   -- Tüm dersler müsaitse toplu güncelleme
   UPDATE semester_courses_cache 
   SET current_enrollment = current_enrollment + 1 
   WHERE id = ANY($course_ids);
   
   COMMIT;
   ```
   - ❌ Kontenjan doluysa → ROLLBACK, 409 COURSE_FULL
   - ✅ Kontenjan varsa → `current_enrollment++` (transaction içinde)
9. **Program kaydı oluştur** (`pending` status)
10. **Event yayınla**: `enrollment.program_submitted`

**Concurrency Control**:
- **Isolation Level**: READ COMMITTED (PostgreSQL default)
- **Locking Strategy**: SELECT FOR UPDATE ile row-level lock
- **Deadlock Prevention**: Course ID'ye göre sıralı lock alma (ORDER BY id)
- **Behavior**: Aynı derse kayıt olmak isteyen öğrenciler sırayla işlenir, farklı derslere kayıt olanlar paralel çalışır

**Event Consumers**: Notification Service *(daha sonra eklenecek)*

**Important**:
- **Frontend**: Çakışma kontrolü yaparak kullanıcıya anında feedback verir (gönder butonu disable)
- **Backend**: Güvenlik için çakışma kontrolünü tekrar yapar (malicious request'lere karşı)
- **JWT Middleware**: Öğrencinin kayıt döneminde olup olmadığını kontrol eder (class_level bazlı phased enrollment)

---

### 🔒 GET /api/v1/enrollments/my
Öğrencinin kendi ders programları

**Role Requirement**: Student

**Query Parameters**: `semester` (e.g., "2025_spring"), `status`

**Response** (200):
```json
{
  "student_id": "uuid",
  "programs": [
    {
      "id": "uuid",
      "semester": "2025_spring",
      "status": "approved",
      "courses": [
        {"course_id": "uuid", "course_code": "CS101", "course_name": "Introduction to Computer Science", "credits": 3}
      ],
      "created_at": "2025-11-11T10:00:00Z"
    }
  ]
}
```

**Note**: Reddedilen programlar `enrollment_programs` tablosundan silinir. Red geçmişi için `GET /api/v1/enrollments/my/rejections` endpoint'ini kullanın.

---

### 🔒 GET /api/v1/enrollments/my/rejections/latest
Öğrencinin en son reddedilen ders programı

**Role Requirement**: Student

**Query Parameters**:
- `semester` (required): "2025_spring", "2025_fall", "2026_summer" format

**Response** (200):
```json
{
  "student_id": "uuid",
  "semester": "2025_spring",
  "has_rejection": true,
  "latest_rejection": {
    "id": "uuid",
    "advisor_id": "uuid",
    "advisor_fullname": "Prof. Dr. Ayşe Yılmaz",
    "rejection_reason": "Schedule conflict detected. CS201 and MATH301 are at the same time slot.",
    "rejected_courses": {
      "courses": [
        {
          "course_id": "uuid",
          "course_code": "CS201",
          "course_name": "Data Structures",
          "credits": 4,
          "instructor": "Dr. Mehmet Öz"
        },
        {
          "course_id": "uuid",
          "course_code": "MATH301",
          "course_name": "Linear Algebra",
          "credits": 3,
          "instructor": "Prof. Ayşe Demir"
        }
      ],
      "total_credits": 18,
      "submitted_at": "2025-11-20T10:00:00Z"
    },
    "rejected_at": "2025-11-21T14:30:00Z"
  },
  "total_rejections": 2
}
```

**Response when no rejection exists** (200):
```json
{
  "student_id": "uuid",
  "semester": "2025_spring",
  "has_rejection": false,
  "latest_rejection": null,
  "total_rejections": 0
}
```

**Business Logic**:
```sql
-- En son red kaydını getir
SELECT * FROM enrollment_rejection_logs 
WHERE student_id = $1 AND semester = $2
ORDER BY rejected_at DESC 
LIMIT 1;

-- Toplam red sayısını getir
SELECT COUNT(*) FROM enrollment_rejection_logs 
WHERE student_id = $1 AND semester = $2;
```

**Use Case**: Öğrenci ders programı reddedildiğinde:
- Kim tarafından reddedildiğini görür (`advisor_fullname`)
- Neden reddedildiğini görür (`rejection_reason`)
- Hangi dersleri seçmişti, hatırlar (`rejected_courses`)
- Kaç kez reddedildiğini bilir (`total_rejections`)
- Yeni program oluştururken bu bilgilere bakarak düzeltme yapar

---

### 🔒 GET /api/v1/enrollments/my/rejections
Öğrencinin tüm reddedilen ders programları geçmişi (opsiyonel - detaylı görüntüleme için)

**Role Requirement**: Student

**Query Parameters**:
- `semester` (optional): "2025_spring", "2025_fall", "2026_summer" format

**Response** (200):
```json
{
  "student_id": "uuid",
  "rejections": [
    {
      "id": "uuid",
      "semester": "2025_spring",
      "advisor_id": "uuid",
      "advisor_fullname": "Prof. Dr. Ayşe Yılmaz",
      "rejection_reason": "Schedule conflict detected. CS201 and MATH301 are at the same time slot.",
      "rejected_courses": {
        "courses": [
          {
            "course_id": "uuid",
            "course_code": "CS201",
            "course_name": "Data Structures",
            "credits": 4,
            "instructor": "Dr. Mehmet Öz"
          }
        ],
        "total_credits": 18,
        "submitted_at": "2025-11-20T10:00:00Z"
      },
      "rejected_at": "2025-11-21T14:30:00Z"
    },
    {
      "id": "uuid",
      "semester": "2025_spring",
      "advisor_id": "uuid",
      "advisor_fullname": "Prof. Dr. Ayşe Yılmaz",
      "rejection_reason": "ECTS limit exceeded. Maximum allowed is 30 ECTS.",
      "rejected_courses": {
        "courses": [ ... ],
        "total_credits": 35,
        "submitted_at": "2025-11-19T09:00:00Z"
      },
      "rejected_at": "2025-11-20T11:00:00Z"
    }
  ],
  "total_rejections": 2
}
```

**Business Logic**:
```sql
-- Tüm red kayıtlarını getir (en yeni önce)
SELECT * FROM enrollment_rejection_logs 
WHERE student_id = $1 
AND ($2::varchar IS NULL OR semester = $2)
ORDER BY rejected_at DESC;
```

**Use Case**: Öğrenci tüm red geçmişini görmek istediğinde (örn: "View History" butonu)

---

### 🔒 GET /api/v1/enrollments/pending-approval
Danışman onayı bekleyen ders programları

**Role Requirement**: Teacher (Advisor)

**Response** (200):
```json
{
  "advisor_id": "uuid",
  "pending_programs": [
    {
      "id": "uuid",
      "student": {
        "id": "uuid",
        "student_number": "2021123456",
        "first_name": "Ahmet",
        "last_name": "Yılmaz",
        "department": "Computer Science",
        "class_level": 2
      },
      "semester": "2025_spring",
      "courses": [
        {
          "course_id": "uuid",
          "course_code": "CS101",
          "course_name": "Introduction to Computer Science",
          "credits": 3,
          "current_enrollment": 85,
          "max_capacity": 150
        }
      ],
      "created_at": "2025-11-11T10:00:00Z"
    }
  ]
}
```

**Business Logic**: Danışman sadece **kendi danışmanlık yaptığı öğrencilerin** programlarını görür (`students_cache.advisor_id == JWT user_id`)

---

### 🔒 POST /api/v1/enrollments/:id/approve
Ders programını onaylama

**Role Requirement**: Teacher (Advisor)

**Response** (200):
```json
{
  "id": "uuid",
  "student_id": "uuid",
  "semester": "2025_spring",
  "status": "approved"
}
```

**RabbitMQ Event Published**: `enrollment.program_approved`

**Business Logic**:
1. **Authorization check**: Danışman, bu öğrencinin danışmanı mı?
2. **Status check**: Program `pending` status'ünde mi?
3. **Program status güncelle**: `approved` status'üne çevir
4. **Event yayınla**: `enrollment.program_approved`

**Important**:
- Kontenjan arttırma işlemi YAPILMAZ (zaten submission'da yapıldı)
- Sadece status değişikliği

**Event Consumers**:
- Grades Service: Her ders için öğrenci kaydı oluşturur (not girişi için hazır)
- Attendance Service: Her ders için öğrenci kaydı oluşturur (yoklama alınabilir)
- Notification Service: Öğrenciye bildirim gönderir

---

### 🔒 POST /api/v1/enrollments/:id/reject
Ders programını reddetme

**Role Requirement**: Teacher (Advisor)

**Request**:
```json
{
  "rejection_reason": "Schedule conflict detected. CS201 and MATH301 are at the same time slot."
}
```

**Response** (200):
```json
{
  "message": "Program rejected successfully",
  "rejection_log_id": "uuid",
  "rejection_reason": "Schedule conflict detected. CS201 and MATH301 are at the same time slot."
}
```

**RabbitMQ Event Published**: `enrollment.program_rejected`

**Business Logic**:
1. **Authorization check**: Danışman, bu öğrencinin danışmanı mı?
2. **Status check**: Program `pending` status'ünde mi?
3. **Danışman bilgisini al** (JWT token'dan advisor_id, User Service'ten veya cache'ten advisor_fullname)
4. **Ders bilgilerini snapshot olarak al** (program silinmeden önce):
   ```sql
   SELECT 
       epc.course_id,
       scc.course_code,
       scc.course_name,
       scc.credits,
       scc.instructor_fullname
   FROM enrollment_program_courses epc
   JOIN semester_courses_cache scc ON epc.course_id = scc.id
   WHERE epc.program_id = $1;
   ```
5. **Rejection log kaydı oluştur**:
   ```sql
   INSERT INTO enrollment_rejection_logs (
       original_program_id, student_id, advisor_id, advisor_fullname, semester,
       rejection_reason, rejected_courses, rejected_at
   ) VALUES (
       $program_id, $student_id, $advisor_id, $advisor_fullname, $semester,
       $rejection_reason, $courses_jsonb, NOW()
   );
   ```
   **JSONB yapısı**:
   ```json
   {
     "courses": [
       {
         "course_id": "uuid",
         "course_code": "CS201",
         "course_name": "Data Structures",
         "credits": 4,
         "instructor": "Dr. Mehmet Öz"
       }
     ],
     "total_credits": 18,
     "submitted_at": "2025-11-20T10:00:00Z"
   }
   ```
6. **Kontenjan azaltma (PostgreSQL Transaction with Row-Level Locking)**:
   ```sql
   BEGIN;
   
   -- Program'daki tüm course_id'leri al
   SELECT course_id FROM enrollment_program_courses WHERE program_id = $1;
   
   -- Row-level lock al (ID sırasına göre - deadlock önleme)
   SELECT id, current_enrollment 
   FROM semester_courses_cache 
   WHERE id = ANY($course_ids) 
   ORDER BY id
   FOR UPDATE;
   
   -- Toplu güncelleme (CHECK constraint negatif değeri önler)
   UPDATE semester_courses_cache 
   SET current_enrollment = current_enrollment - 1 
   WHERE id = ANY($course_ids);
   
   COMMIT;
   ```
7. **Event yayınla**: `enrollment.program_rejected` (öğrenciye bildirim için)
8. **Program silme**: `enrollment_programs` ve `enrollment_program_courses` tablosundan tamamen sil (CASCADE delete)

**Event Consumers**: Notification Service *(daha sonra eklenecek)*

**Important**:
- Program reddedilince **tamamen silinir** (enrollment_programs tablosundan)
- Red bilgileri `enrollment_rejection_logs` tablosunda **kalıcı olarak saklanır**
- **Kontenjan geri verilir** (submission'da kaplanmıştı, şimdi serbest bırakılır)
- Öğrenci yeni program oluşturabilir (UNIQUE constraint artık engel değil)
- Öğrenci `GET /api/v1/enrollments/my/rejections` ile red geçmişini görebilir

---

### 🔒 DELETE /api/v1/enrollments/:id
Öğrencinin kendi pending programını iptal etmesi (yeniden ders kaydı yapmak için)

**Role Requirement**: Student

**Response** (200):
```json
{
  "message": "Program cancelled successfully",
  "can_create_new": true
}
```

**Error Responses**:
```json
// 403 - Başkasının programı
{
  "error": "FORBIDDEN",
  "message": "You do not have permission to cancel this program"
}

// 400 - Approved program iptal edilemez
{
  "error": "CANNOT_CANCEL_APPROVED",
  "message": "Approved programs cannot be cancelled"
}

// 404 - Program bulunamadı
{
  "error": "NOT_FOUND",
  "message": "Program not found"
}
```

**RabbitMQ Event Published**: `enrollment.program_cancelled`

**Business Logic**:
1. **JWT Authorization**: Token'dan `student_id` al
2. **Program fetch**: `enrollment_programs` tablosundan program'ı çek
3. **Ownership check**: `program.student_id == JWT.student_id`
   - ❌ Eşleşmezse → 403 FORBIDDEN
4. **Status check**: `program.status == "pending"`
   - ❌ `approved` ise → 400 CANNOT_CANCEL_APPROVED
5. **Kontenjan azaltma (PostgreSQL Transaction with Row-Level Locking)**:
   ```sql
   BEGIN;
   
   -- Program'daki tüm course_id'leri al
   SELECT course_id FROM enrollment_program_courses WHERE program_id = $1;
   
   -- Row-level lock al (ID sırasına göre - deadlock önleme)
   SELECT id, current_enrollment 
   FROM semester_courses_cache 
   WHERE id = ANY($course_ids) 
   ORDER BY id
   FOR UPDATE;
   
   -- Toplu güncelleme (CHECK constraint negatif değeri önler)
   UPDATE semester_courses_cache 
   SET current_enrollment = current_enrollment - 1 
   WHERE id = ANY($course_ids);
   
   COMMIT;
   ```
6. **Program silme**: `enrollment_programs` ve `enrollment_program_courses` tablosundan tamamen sil (CASCADE delete)
7. **Event yayınla**: `enrollment.program_cancelled`
8. **Response**: 200 OK

**Event Consumers**: Notification Service *(daha sonra eklenecek)*

**Important**:
- Sadece `pending` status'ündeki programlar iptal edilebilir
- `approved` programlar iptal edilemez (Grades/Attendance servisleri kayıt oluşturmuş)
- Program silinince öğrenci yeni program oluşturabilir (UNIQUE constraint artık engel değil)
- Kontenjan geri verilir (başka öğrenciler bu kontenjanı kullanabilir)
- **Not**: Öğrenci iptalinde `enrollment_rejection_logs`'a kayıt YAPILMAZ (sadece danışman redlerinde)

---

### 🔒 GET /api/v1/enrollments/course/:courseId/students
Derse kayıtlı öğrenciler

**Role Requirement**: Teacher (course instructor)

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "semester": "2025_spring",
  "current_enrollment": 85,
  "max_capacity": 150,
  "students": [
    {
      "student_id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "department": "Computer Science",
      "class_level": 2,
      "approved_at": "2025-11-11T11:00:00Z"
    }
  ]
}
```

**Business Logic**: Instructor sadece **kendi derslerindeki** öğrencileri görür (sadece `approved` status'ündeki programlar)

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Subscribed Exchanges:
- "student.events" (routing keys: student.created, student.updated, student.deactivated)
- "course.events" (routing keys: course.semester.created, course.semester.updated, course.semester.deleted)
- "grade.events" (routing keys: grade.student.prerequisite.passed)

Publishing Exchange:
- "enrollment.events" (type: topic)

Routing Keys (Publishing):
- enrollment.program_submitted
- enrollment.program_approved
- enrollment.program_rejected
- enrollment.program_cancelled
```

### Event Schemas

#### enrollment.program_submitted
Published when: Öğrenci ders programı gönderdi

```json
{
  "event_type": "enrollment.program_submitted",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "program_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "student_email": "ahmet.yilmaz@university.edu.tr",
    "advisor_id": "uuid",
    "semester": "2025_spring",
    "course_count": 5,
    "created_at": "2025-11-11T10:00:00Z"
  }
}
```

**Note**: `student_email` değeri `students_cache.email` kolonundan alınır.

#### enrollment.program_approved
Published when: Danışman programı onayladı

```json
{
  "event_type": "enrollment.program_approved",
  "timestamp": "2025-11-11T11:00:00Z",
  "data": {
    "program_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "student_email": "ahmet.yilmaz@university.edu.tr",
    "semester": "2025_spring",
    "courses": [
      {"course_id": "uuid", "course_code": "CS101", "course_name": "Introduction to Computer Science", "credits": 3}
    ]
  }
}
```

#### enrollment.program_rejected
Published when: Danışman programı reddetti

```json
{
  "event_type": "enrollment.program_rejected",
  "timestamp": "2025-11-11T11:30:00Z",
  "data": {
    "program_id": "uuid",
    "rejection_log_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "student_email": "ahmet.yilmaz@university.edu.tr",
    "advisor_id": "uuid",
    "advisor_fullname": "Prof. Dr. Ayşe Yılmaz",
    "semester": "2025_spring",
    "rejection_reason": "Schedule conflict detected. CS201 and MATH301 are at the same time slot.",
    "rejected_courses": [
      {"course_code": "CS201", "course_name": "Data Structures", "credits": 4},
      {"course_code": "MATH301", "course_name": "Linear Algebra", "credits": 3}
    ]
  }
}
```

**Note**: `rejection_log_id` ve `advisor_fullname` eklendi. Notification Service bu bilgilerle detaylı bildirim oluşturabilir.

#### enrollment.program_cancelled
Published when: Öğrenci kendi pending programını iptal etti (yeniden ders kaydı yapmak için)

```json
{
  "event_type": "enrollment.program_cancelled",
  "timestamp": "2025-11-11T12:00:00Z",
  "data": {
    "program_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "student_email": "ahmet.yilmaz@university.edu.tr",
    "semester": "2025_spring",
    "course_count": 5,
    "cancelled_by": "student",
    "cancelled_at": "2025-11-11T12:00:00Z"
  }
}
```

**Event Consumers**: Notification Service *(daha sonra eklenecek)*

---

### Notification Service Entegrasyonu

> ⏳ **Not**: Notification Service henüz implement edilmedi. Bu bölüm, Notification Service geliştirildiğinde kullanılacak referans dokümantasyonudur.

Enrollment Service event'leri yayınlar, Notification Service bu event'leri consume ederek öğrencilere bildirim gönderir.

#### enrollment.program_rejected Bildirimi

**Notification Service Sorumluluğu**:

Öğrenci ders programı reddedildiğinde Notification Service şu aksiyonları almalı:

1. **Email Bildirimi**:
   ```
   Subject: Your Course Program Was Rejected - 2025 Spring Semester
   
   Dear Ahmet Yılmaz,
   
   Your course program for the 2025 Spring semester has been rejected by your advisor.
   
   Rejected by: Prof. Dr. Ayşe Yılmaz
   
   Rejection Reason:
   "Schedule conflict detected. CS201 and MATH301 are at the same time slot."
   
   You can log in to DEBIS to view the rejected courses and create a new program.
   
   Your registration period is open until November 25, 2025 at 23:59.
   ```

2. **In-App Notification** (Web/Mobile):
   ```json
   {
     "type": "enrollment_rejected",
     "title": "Course Program Rejected",
     "message": "Your advisor rejected your course program. Click for details.",
     "action_url": "/enrollment",
     "created_at": "2025-11-21T14:30:00Z"
   }
   ```

3. **Push Notification** (Mobile):
   ```
   Your course program was rejected. Open the app to create a new one.
   ```

**Event'ten Alınacak Bilgiler**:
| Event Field | Kullanım |
|-------------|----------|
| `student_email` | Email gönderimi için |
| `student_number` | Öğrenci tanımlama |
| `advisor_fullname` | Email'de "Rejected by" olarak gösterilir |
| `rejection_reason` | Email içeriğinde gösterilir |
| `rejected_courses` | Email'de liste olarak gösterilebilir |
| `semester` | Dönem bilgisi |

**Timing**: Bildirim **hemen** gönderilmeli (async queue ile). Öğrencinin kayıt dönemi sınırlı olduğu için gecikme kabul edilemez.

#### Diğer Enrollment Event'leri için Bildirimler

| Event | Bildirim Türü | Alıcı |
|-------|---------------|-------|
| `enrollment.program_submitted` | In-app + Email | Danışman (onay bekliyor) |
| `enrollment.program_approved` | Email + Push | Öğrenci |
| `enrollment.program_rejected` | Email + Push + In-app | Öğrenci |
| `enrollment.program_cancelled` | In-app | Öğrenci (onay bildirimi) |

#### student.created (Consumed from Student Service)
Consumed when: Yeni öğrenci oluşturulduğunda

**Consumer Action**:
```sql
INSERT INTO students_cache (
    id, student_number, email, first_name, last_name, 
    department, class_level, advisor_id, status, synced_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW());
```

**Event Schema**:
```json
{
  "event_type": "student.created",
  "timestamp": "2025-11-21T10:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "email": "ahmet.yilmaz@university.edu.tr",
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "department": "Bilgisayar Mühendisliği",
    "class_level": 1,
    "advisor_id": "uuid",
    "status": "active"
  }
}
```

#### student.updated (Consumed from Student Service)
Consumed when: Öğrenci bilgileri güncellendiğinde

**Consumer Action**:
```sql
UPDATE students_cache
SET student_number = $2, email = $3, first_name = $4, last_name = $5,
    department = $6, class_level = $7, advisor_id = $8, status = $9, synced_at = NOW()
WHERE id = $1;
```

**Event Schema**: Same as `student.created`

#### course.semester.created (Consumed from Course Catalog Service)
Consumed when: Dönemlik ders oluşturulduğunda (via Outbox Pattern)

**Purpose**: Yeni açılan dersi Enrollment Service cache'ine eklemek

**Consumer Action**:
```sql
-- 1. INSERT semester_courses_cache (all fields from event)
INSERT INTO semester_courses_cache (
    id, course_code, course_name, faculty, department, credits, course_type,
    class_level, semester, instructor_id, instructor_fullname, classroom_location,
    max_capacity, prerequisites, synced_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW());
-- Note: current_enrollment defaults to 0

-- 2. INSERT schedule sessions
INSERT INTO course_sessions_cache (course_id, day_of_week, slot_number, synced_at)
VALUES ($1, 'monday', 1, NOW()), ($1, 'monday', 2, NOW()), ...;
```

**Event Schema**:
```json
{
  "event_id": "uuid",
  "event_type": "course.semester.created",
  "timestamp": "2025-11-21T10:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_spring",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "faculty": "Mühendislik Fakültesi",
    "department": "Bilgisayar Mühendisliği",
    "credits": 6,
    "course_type": "mandatory",
    "class_level": 2,
    "instructor_id": "uuid",
    "instructor_fullname": "Ayşe Demir",
    "classroom_location": "A Blok 301",
    "max_capacity": 150,
    "prerequisites": [
      {"id": "uuid-cs100", "course_code": "CS100", "course_name": "Programming Fundamentals"}
    ],
    "schedule_sessions": [
      {"day_of_week": "monday", "slot_numbers": [1, 2, 3]},
      {"day_of_week": "wednesday", "slot_numbers": [4, 5]}
    ]
  }
}
```

**Important**: 
- `semester_courses_cache.id` = Event'teki `semester_course_id` (birebir eşleşme)
- Eski dönem verileri otomatik silinmez, manuel temizlik gerekir (bkz. Admin Operations)

---

#### course.semester.updated (Consumed from Course Catalog Service)
Consumed when: Dönemlik ders güncellendiğinde (via Outbox Pattern)

**Purpose**: Güncellenen ders bilgilerini Enrollment Service cache'ine yansıtmak

**Consumer Action**:
```sql
-- 1. UPDATE semester_courses_cache (preserve current_enrollment!)
UPDATE semester_courses_cache
SET course_code = $2, course_name = $3, faculty = $4, department = $5, credits = $6,
    course_type = $7, class_level = $8, semester = $9, instructor_id = $10, instructor_fullname = $11,
    classroom_location = $12, max_capacity = $13, prerequisites = $14, synced_at = NOW()
WHERE id = $1;
-- Note: current_enrollment is NOT updated (Enrollment Service owns this data)

-- 2. DELETE old schedule sessions
DELETE FROM course_sessions_cache WHERE course_id = $1;

-- 3. INSERT new schedule sessions
INSERT INTO course_sessions_cache (course_id, day_of_week, slot_number, synced_at)
VALUES ($1, 'tuesday', 3, NOW()), ($1, 'tuesday', 4, NOW()), ...;
```

**Event Schema**: Same as `course.semester.created` (full state gönderilir)

**Important Notes**:
- **current_enrollment preserved**: Güncelleme sırasında `current_enrollment` değeri korunur
- **Schedule sessions replaced**: Eski session'lar silinip yenileri eklenir (clean slate)

---

#### course.semester.deleted (Consumed from Course Catalog Service)
Consumed when: Dönemlik ders silindiğinde (via Outbox Pattern)

**Purpose**: Silinen dersi Enrollment Service cache'inden kaldırmak

**Consumer Action**:
```sql
-- DELETE course cache (course_sessions_cache CASCADE ile otomatik silinir)
DELETE FROM semester_courses_cache WHERE id = $1;
```

**Event Schema**:
```json
{
  "event_id": "uuid",
  "event_type": "course.semester.deleted",
  "timestamp": "2025-11-21T12:00:00Z",
  "data": {
    "semester_course_id": "uuid",
    "semester": "2025_spring",
    "course_code": "CS101",
    "course_name": "Data Structures",
    "department": "Bilgisayar Mühendisliği"
  }
}
```

**Note**: `course_sessions_cache` has `ON DELETE CASCADE` - automatically deleted when course is removed.

---

#### grade.student.prerequisite.passed (Consumed from Grades Service)
Consumed when: Öğrenci bir önkoşul dersini GEÇTİĞİNDE

**Purpose**: Önkoşul dersini geçen öğrenciyi `student_passed_prerequisites` tablosuna eklemek

**Important**:
- Bu event SADECE önkoşul olan dersler için yayınlanır (Grades Service filtreler)
- SADECE GEÇEN öğrenciler için yayınlanır (grade_point >= 1.00)
- Kalan öğrenciler için event YAYINLANMAZ

**Consumer Action**:
```sql
-- Önkoşul dersini geçen öğrenciyi kaydet
INSERT INTO student_passed_prerequisites (student_id, course_code, semester, grade_point, synced_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (student_id, course_code) DO UPDATE
SET semester = $3, grade_point = $4, synced_at = NOW();
-- ON CONFLICT UPDATE: Öğrenci aynı dersi tekrar alıp geçerse bilgiler güncellenir
```

**Event Schema**:
```json
{
  "event_type": "grade.student.prerequisite.passed",
  "timestamp": "2025-01-15T10:00:00Z",
  "data": {
    "student_id": "uuid",
    "course_code": "CS101",
    "course_id": "uuid",
    "semester": "2024_fall",
    "grade_point": "2.50"
  }
}
```

**Prerequisite Check Logic**:
```sql
-- Kayıt varsa GEÇMİŞ (alabilir), yoksa GEÇMEMİŞ veya HİÇ ALMAMIŞ (alamaz)
SELECT EXISTS(
    SELECT 1 FROM student_passed_prerequisites
    WHERE student_id = $1 AND course_code = $2
);
-- TRUE → Önkoşul geçilmiş, kayıt YAPILABİLİR ✅
-- FALSE → Önkoşul geçilmemiş veya hiç alınmamış, kayıt YAPILAMAZ ❌
```

**Flow Example**:
1. CS201 dersi CS101'i önkoşul olarak gerektiriyor
2. Öğrenci CS101'i geçti → `grade.student.prerequisite.passed` event yayınlandı → `student_passed_prerequisites`'e eklendi
3. Öğrenci CS201'e kayıt olmak istiyor → `student_passed_prerequisites`'te CS101 kaydı var mı? → VAR ✅
4. Kayıt yapılabilir

**Scenario: Hiç Almamış Öğrenci**:
1. 2. sınıf öğrenci CS201'e kayıt olmak istiyor (CS101 önkoşul)
2. Öğrenci CS101'i hiç almamış → `student_passed_prerequisites`'te kayıt YOK
3. Kayıt YAPILAMAZ ❌ (önce CS101'i alıp geçmeli)

---

## Admin Operations

### Manuel Dönem Temizliği

Eski dönem verileri **otomatik silinmez**. Yeni dönem başlamadan önce admin/developer tarafından manuel temizlik yapılmalıdır.

**Temizlik Sırası** (foreign key constraints nedeniyle):
```sql
-- 1. Önce enrollment program courses sil (foreign key to semester_courses_cache)
DELETE FROM enrollment_program_courses 
WHERE course_id IN (
    SELECT id FROM semester_courses_cache WHERE semester = '2024_fall'
);

-- 2. Sonra enrollment programs sil
DELETE FROM enrollment_programs WHERE semester = '2024_fall';

-- 3. Rejection logs sil (opsiyonel - audit için saklanabilir)
DELETE FROM enrollment_rejection_logs WHERE semester = '2024_fall';

-- 4. En son semester courses cache sil (course_sessions_cache CASCADE ile silinir)
DELETE FROM semester_courses_cache WHERE semester = '2024_fall';
```

**Important**:
- `student_passed_prerequisites` tablosu SİLİNMEZ (kalıcı veri - prerequisite validation için gerekli)
- `enrollment_rejection_logs` tablosu opsiyonel olarak saklanabilir (audit trail)

**Önerilen Zamanlama**: Yeni dönem dersleri açılmadan 1 hafta önce

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_INPUT | Validation hatası (boş course_ids) |
| 400 | PREREQUISITES_NOT_MET | Önkoşul dersleri geçilmemiş (student_passed_prerequisites'te kayıt yok) |
| 400 | WRONG_DEPARTMENT | Ders öğrencinin bölümünden değil |
| 400 | WRONG_CLASS_LEVEL | Ders öğrencinin sınıf seviyesine uygun değil |
| 400 | SCHEDULE_CONFLICT | Seçilen dersler arasında saat çakışması var |
| 400 | CANNOT_CANCEL_APPROVED | Onaylanmış program iptal edilemez |
| 403 | STUDENT_DEACTIVATED | Öğrenci deaktif edilmiş (is_active = false) |
| 403 | ENROLLMENT_PERIOD_CLOSED | Öğrenci kayıt dönemi kapalı |
| 403 | APPROVAL_PERIOD_CLOSED | Danışman onay dönemi kapalı |
| 403 | FORBIDDEN | Yetkisiz erişim (başkasının programı veya danışman yetkisi yok) |
| 404 | NOT_FOUND | Program bulunamadı |
| 409 | ALREADY_SUBMITTED | Aynı dönem için program zaten gönderilmiş |
| 409 | COURSE_FULL | Bir veya daha fazla ders kontenjan dolu |
| 500 | INTERNAL_ERROR | Server hatası |

---

## JWT Middleware Authentication & Phased Enrollment Control

### Kayıt Dönemi Kontrolü (JWT Middleware)

Enrollment Service'in tüm endpoint'lerinde **JWT middleware** çalışır. Öğrenci ve danışman için **farklı dönemler** uygulanır.

#### Öğrenci Kayıt Dönemi (Kısa - Sınıf bazlı)

Öğrenciler **sadece kendi kayıt dönemlerinde** enrollment API'larına erişebilir:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Öğrenci Kayıt Dönemleri                               │
├─────────────────────────────────────────────────────────────────────────┤
│  Class 4: 2025-11-20 09:00 ──────────── 2025-11-22 23:59 (3 gün)       │
│  Class 3: 2025-11-21 09:00 ──────────── 2025-11-23 23:59 (3 gün)       │
│  Class 2: 2025-11-22 09:00 ──────────── 2025-11-24 23:59 (3 gün)       │
│  Class 1: 2025-11-23 09:00 ──────────── 2025-11-25 23:59 (3 gün)       │
├─────────────────────────────────────────────────────────────────────────┤
│  ❌ Dönem dışında: 403 ENROLLMENT_PERIOD_CLOSED                         │
│  ✅ Dönem içinde: API erişimi açık                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

**Öğrenci erişebileceği endpoint'ler (dönem içinde)**:
- `GET /api/v1/enrollments/available-courses`
- `POST /api/v1/enrollments`
- `GET /api/v1/enrollments/my`
- `GET /api/v1/enrollments/my/rejections/latest`
- `GET /api/v1/enrollments/my/rejections`
- `DELETE /api/v1/enrollments/:id`

#### Danışman Onay Dönemi (Uzun)

Danışmanlar öğrenci kayıt döneminden **daha uzun süre** onay/red yapabilir:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Danışman Onay Dönemi                                  │
├─────────────────────────────────────────────────────────────────────────┤
│  Tüm danışmanlar: 2025-11-20 09:00 ──────────── 2025-11-30 23:59       │
│                                                                          │
│  Öğrenci kayıtları: 20-25 Kasım (5 gün)                                 │
│  Danışman onayları: 20-30 Kasım (10 gün) ← Ekstra 5 gün                 │
├─────────────────────────────────────────────────────────────────────────┤
│  ❌ Dönem dışında: 403 APPROVAL_PERIOD_CLOSED                           │
│  ✅ Dönem içinde: API erişimi açık                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

**Danışman erişebileceği endpoint'ler (dönem içinde)**:
- `GET /api/v1/enrollments/pending-approval`
- `POST /api/v1/enrollments/:id/approve`
- `POST /api/v1/enrollments/:id/reject`
- `GET /api/v1/enrollments/course/:courseId/students`

#### Middleware Flow

```go
1. JWT token parse et
2. user_id ve role çıkar

3. Eğer role == "student" ise:
   a. students_cache'ten class_level bilgisini çek
   b. Sistem konfigürasyonundan öğrenci enrollment phase'i al (class_level bazlı)
   c. Eğer NOW < start_time VEYA NOW > end_time ise:
      → 403 ENROLLMENT_PERIOD_CLOSED
   d. Devam et

4. Eğer role == "teacher" ise:
   a. Sistem konfigürasyonundan danışman approval phase'i al
   b. Eğer NOW < start_time VEYA NOW > end_time ise:
      → 403 APPROVAL_PERIOD_CLOSED
   c. Devam et

5. Authorization geçti, request handler'a devam et
```

**Konfigürasyon Yönetimi**:
- Enrollment phase tarihleri **environment variable** veya **Redis** ile yönetilir
- Database tablosu kullanılmaz (enrollment_phases tablosu kaldırıldı)
- Admin panelinden güncelleme yapılabilir (Redis'e yazılır)

**Örnek Redis Key-Value**:
```
# Öğrenci kayıt dönemleri (class_level bazlı)
enrollment:phase:student:class_4:start = "2025-11-20T09:00:00Z"
enrollment:phase:student:class_4:end = "2025-11-22T23:59:59Z"
enrollment:phase:student:class_3:start = "2025-11-21T09:00:00Z"
enrollment:phase:student:class_3:end = "2025-11-23T23:59:59Z"
enrollment:phase:student:class_2:start = "2025-11-22T09:00:00Z"
enrollment:phase:student:class_2:end = "2025-11-24T23:59:59Z"
enrollment:phase:student:class_1:start = "2025-11-23T09:00:00Z"
enrollment:phase:student:class_1:end = "2025-11-25T23:59:59Z"

# Danışman onay dönemi (tüm danışmanlar için aynı)
enrollment:phase:advisor:start = "2025-11-20T09:00:00Z"
enrollment:phase:advisor:end = "2025-11-30T23:59:59Z"
```

**Avantajlar**:
- Middleware seviyesinde erken kontrol (handler'a gelmeden önce)
- Her endpoint için tekrar kontrol gerekmez
- Phased enrollment kolayca yönetilebilir (Redis update)
- Database query azalır
- Danışmanlar öğrenci kayıt dönemi bittikten sonra da onaylayabilir

---

## Concurrency Control Strategy

### Problem
60.000+ öğrenci aynı anda ders kaydı yaparken:
- Aynı derse kayıt olmak isteyenler sıralı işlenmeli (race condition önleme)
- Farklı derslere kayıt olanlar paralel çalışmalı (performans)

### Çözüm: READ COMMITTED + SELECT FOR UPDATE

**Isolation Level**: PostgreSQL default (READ COMMITTED)

**Locking Strategy**: Row-level lock with SELECT FOR UPDATE

**Deadlock Prevention**: Course ID'ye göre sıralı lock alma

```sql
-- Doğru: ID sırasına göre lock (deadlock önler)
SELECT ... FROM semester_courses_cache 
WHERE id = ANY($course_ids) 
ORDER BY id 
FOR UPDATE;

-- Yanlış: Sırasız lock (deadlock riski)
SELECT ... FROM semester_courses_cache 
WHERE id = ANY($course_ids) 
FOR UPDATE;
```

### Çalışma Örneği

**Senaryo**: CS101 kontenjanı 150, şu an 149 kayıtlı. Öğrenci A ve B aynı anda CS101'e kayıt olmak istiyor.

```
Zaman  │  Öğrenci A                      │  Öğrenci B
───────┼─────────────────────────────────┼─────────────────────────────
T0     │  POST /enrollments              │  POST /enrollments
T1     │  BEGIN                          │  BEGIN
       │  SELECT ... FOR UPDATE          │  SELECT ... FOR UPDATE
       │  ✅ Lock alındı (149)           │  ⏳ Bekliyor (satır kilitli)
T2     │  149 < 150? ✅                  │  ⏳ Bekliyor...
T3     │  UPDATE → 150                   │  ⏳ Bekliyor...
       │  COMMIT                         │  
       │  🔓 Lock açıldı                 │  ✅ Lock alındı (150)
T4     │                                 │  150 < 150? ❌
       │                                 │  ROLLBACK
       │                                 │  409 COURSE_FULL
───────┼─────────────────────────────────┼─────────────────────────────
Sonuç  │  ✅ Kayıt başarılı              │  ❌ Kontenjan dolu
```

**Kritik Nokta**: Öğrenci B beklerken eski veriyi (149) cache'lemez. Lock açıldığında **güncel değeri (150)** okur.

### Performans Karakteristikleri

| Metrik | Değer |
|--------|-------|
| Lock Scope | Sadece seçilen dersler (row-level) |
| Paralel İşlem | Farklı derslere kayıt olanlar paralel |
| Bekleme | Sadece aynı derse kayıt olanlar bekler |
| Deadlock Riski | ORDER BY id ile elimine edildi |

---

## Known Risks & Limitations

### ⚠️ Course Code Değişikliği Riski

**Risk**: `student_passed_prerequisites` tablosu `course_code` (VARCHAR) kullanıyor, `course_id` (UUID) değil.

**Senaryo**: Öğrenci CS101'i geçti ve `student_passed_prerequisites` tablosuna `course_code = "CS101"` olarak kaydedildi. Sonraki dönem Course Catalog Service'te CS101'in kodu CS100 olarak değiştirildi. Bu durumda:
- CS201 dersi artık `prerequisites: [{course_code: "CS100"}]` olarak tanımlı
- Öğrencinin `student_passed_prerequisites` kaydı `course_code = "CS101"`
- Prerequisite validation: Sistem CS100'ü arar, CS101 kaydı eşleşmez → Kayıt YAPILAMAZ (yanlış negatif)

**Etki**: Düşük - Course code değişikliği nadir bir operasyon. Ancak olduğunda manuel müdahale gerekebilir.

**Kabul Edilen Çözüm**: Course Catalog Service'te course_code değişikliği yapıldığında, ilgili tüm bağımlı sistemlere (Enrollment, Grades) migration event'i gönderilmeli. Bu, v7.x scope'u dışında bırakıldı.

---

## Related Services

- **Student Service**: Öğrenci bilgileri (event consumer: `student.created`, `student.updated`)
- **Course Catalog Service**: Dönemlik ders bilgileri (event consumer: `course.semester.created`, `course.semester.updated`, `course.semester.deleted`)
- **Grades Service**:
  - Inbound: Event consumer (`enrollment.program_approved`)
  - Outbound: Event producer (`grade.student.prerequisite.passed`)
- **Attendance Service**: Event consumer (`enrollment.program_approved`)
- **Notification Service**: Event consumer (tüm enrollment events) - ⏳ *Daha sonra eklenecek*

---

**Version**: 7.0.0 (Prerequisite validation refactor)
**Last Updated**: 2025-12-12

**Changes in v7.0.0**:
- ✅ **Prerequisite Logic Değişikliği**: `student_failed_prerequisites` tablosu `student_passed_prerequisites` olarak değiştirildi
- ✅ **Yeni Validation Logic**: Artık "öğrenci kalmış mı" yerine "öğrenci geçmiş mi" kontrolü yapılıyor
- ✅ **Event Değişikliği**: `grade.student.prerequisite.failed` event'i artık dinlenmiyor
- ✅ **Event Güncellemesi**: `grade.student.prerequisite.passed` event'i önkoşulu GEÇEN öğrencileri `student_passed_prerequisites` tablosuna ekliyor
- ✅ **Schema Güncellemesi**: `student_passed_prerequisites` tablosuna `grade_point` kolonu eklendi
- ✅ **Doğru Validation**: Hiç almamış öğrenciler de artık engellenebiliyor (kayıt yoksa = geçmemiş veya almamış)

**Changes in v6.3.0**:
- ✅ Added `enrollment_rejection_logs` table for rejection history
- ✅ Added `advisor_fullname` column to rejection logs (snapshot - no join needed)
- ✅ Added `GET /api/v1/enrollments/my/rejections/latest` endpoint (en son red)
- ✅ Added `GET /api/v1/enrollments/my/rejections` endpoint (tüm geçmiş)
- ✅ Updated reject flow: Now creates snapshot in rejection logs before deleting program
- ✅ Added `rejection_log_id` and `advisor_fullname` to `enrollment.program_rejected` event
- ✅ Added `original_program_id` to rejection logs for audit/debug purposes
- ✅ Added JSONB structure documentation for `rejected_courses`
- ✅ Added indexes for rejection logs queries
- ✅ Added "Notification Service Entegrasyonu" section with email/push notification details
- ✅ Updated Frontend Flow: Only latest rejection shown by default, "View History" for full history
- ✅ Added `rejection-history-modal.tsx` component for full rejection history
- ✅ Added SQL query examples for rejection logs
- ✅ Updated Admin Operations: rejection logs cleanup is now optional (audit trail)
- ✅ Standardized all API responses to English
- ✅ Standardized field naming: `total_rejections` (was `total_count` in some places)
- 🎯 Pattern: Audit trail with immutable snapshots, latest-first display
- 📊 Benefit: Students can see who rejected their program, why, get notified immediately, and view full history

**Changes in v6.2.0**:
- ✅ Added `email` column to `students_cache` table
- ✅ Renamed `courses_cache` → `semester_courses_cache`
- ✅ Renamed `name` → `course_name` in `semester_courses_cache`
- ✅ Added CHECK constraint: `current_enrollment >= 0`
- ✅ Removed `rejected` from `enrollment_status_enum`
- ✅ Removed automatic semester cleanup logic
- ✅ Added "Concurrency Control Strategy" section with SELECT FOR UPDATE pattern
- 🎯 Pattern: Schema-event consistency, explicit concurrency control

**Changes in v6.1.0**:
- ✅ Fixed prerequisite validation: Now uses `student_failed_prerequisites` table
- ✅ Removed `previous_grades` feature from `available-courses` endpoint
- ✅ Added "Known Risks & Limitations" section documenting course_code change risk
- 🎯 Pattern: Documentation-schema consistency

**Changes in v6.0.0**:
- ✅ Replaced `student_grades` table with `student_failed_prerequisites`
- ✅ Replaced `grade.course_completed` event with `grade.student.prerequisite.failed`
- ✅ Simplified prerequisite check: "failed tabloda var mı?" (blacklist approach)
- 🎯 Pattern: Source-side filtering (Grades Service filters before publishing)

---

## Frontend Flow

Bu bölüm, Enrollment Service API'lerini kullanan frontend uygulamasının nasıl tasarlanacağını açıklar.

### Öğrenci Enrollment Flow

#### State Machine

```
┌─────────────┐                              ┌─────────────┐
│             │   POST /enrollments          │             │
│  NO_PROGRAM ├─────────────────────────────►│   PENDING   │
│             │                              │             │
└──────┬──────┘                              └──────┬──────┘
       │                                            │
       │  İlk kez veya                              ├───────────────────────┐
       │  "Create New Program"                      │                       │
       │  butonuna basıldı                          │                       │
       │                                            │                       │
       │                                            │                       │
┌──────┴──────┐    DELETE /enrollments/:id          │                       │
│             │    (Öğrenci iptal etti)             │                       │
│  REJECTED   │◄────────────────────────────────────┘                       │
│  (View)     │                                                             │
│             │    POST /enrollments/:id/reject                             │
│             │◄────────────────────────────────────────────────────────────┘
└─────────────┘
       │
       │  "Create New Program" butonu
       │
       ▼
┌─────────────┐
│   COURSE    │
│  SELECTION  │ → POST /enrollments → PENDING → ...
└─────────────┘


                                     ┌─────────────┐
      POST /enrollments/:id/         │             │
      approve                        │  APPROVED   │
      (Danışman onayladı)            │  (Final)    │
                                     └─────────────┘
                                            ▲
              PENDING ──────────────────────┘
```

#### Sayfa Durumları (5 Farklı View)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        /enrollment sayfası                               │
│                                                                          │
│   1. GET /api/v1/enrollments/my?semester=2025_spring                    │
│   2. GET /api/v1/enrollments/my/rejections/latest?semester=2025_spring  │
│                                                                          │
│                        ┌─────────────────┐                              │
│                        │ Response'ları   │                              │
│                        │ değerlendir     │                              │
│                        └────────┬────────┘                              │
│                                 │                                        │
│         ┌───────────┬───────────┼───────────┬───────────┐               │
│         ▼           ▼           ▼           ▼           ▼               │
│   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│   │ APPROVED │ │ PENDING  │ │ REJECTED │ │  FRESH   │ │  FRESH   │     │
│   │ program  │ │ program  │ │ (has_    │ │ (no      │ │ (first   │     │
│   │ var      │ │ var      │ │ rejection│ │ program, │ │ time)    │     │
│   │          │ │          │ │ = true)  │ │ has_rej  │ │          │     │
│   │          │ │          │ │          │ │ = false) │ │          │     │
│   └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘     │
│        │            │            │            │            │            │
│        ▼            ▼            ▼            ▼            ▼            │
│   ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│   │ Approved │ │ Pending  │ │ Rejected │ │ Course   │ │ Course   │     │
│   │ Program  │ │ Program  │ │ Program  │ │Selection │ │Selection │     │
│   │ View     │ │ View     │ │ View     │ │ View     │ │ View     │     │
│   │          │ │          │ │          │ │          │ │          │     │
│   │ (read-   │ │ (cancel  │ │ (+ Create│ │ (normal) │ │ (normal) │     │
│   │  only)   │ │  button) │ │  New btn)│ │          │ │          │     │
│   └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Karar Mantığı (Frontend Logic)

```typescript
// 1. API çağrıları
const programResponse = await getMyEnrollments({ semester: '2025_spring' })
const rejectionResponse = await getLatestRejection({ semester: '2025_spring' })

// 2. Durumları çıkar
const approvedProgram = programResponse.programs?.find(p => p.status === 'approved')
const pendingProgram = programResponse.programs?.find(p => p.status === 'pending')
const hasRejection = rejectionResponse.has_rejection
const latestRejection = rejectionResponse.latest_rejection

// 3. Hangi view gösterilecek?
if (approvedProgram) {
  return <ApprovedProgramView program={approvedProgram} />
}

if (pendingProgram) {
  return <PendingProgramView program={pendingProgram} />
}

if (hasRejection) {
  return <RejectedProgramView rejection={latestRejection} />
}

// İlk kez veya red geçmişi yok
return <CourseSelectionView />
```

#### View Detayları

##### 1. ApprovedProgramView (Onaylanmış Program)

```
┌─────────────────────────────────────────────────────────────────┐
│  ✅ YOUR PROGRAM HAS BEEN APPROVED                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Semester: 2025 Spring                                          │
│  Status: Approved                                                │
│  Approved at: November 22, 2025, 10:30                          │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  📚 Your Courses (5 courses • 18 ECTS)                          │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ CS201 - Data Structures                     4 ECTS      │    │
│  │ 📍 Room A-301 • 👤 Dr. Mehmet Öz                        │    │
│  │ 📅 Monday 09:00-12:00, Wednesday 14:00-15:00            │    │
│  ├─────────────────────────────────────────────────────────┤    │
│  │ CS202 - Computer Organization               3 ECTS      │    │
│  │ 📍 Room B-201 • 👤 Prof. Ayşe Demir                     │    │
│  │ 📅 Tuesday 10:00-13:00                                  │    │
│  ├─────────────────────────────────────────────────────────┤    │
│  │ ...                                                     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ℹ️ Your program is final and cannot be modified.               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Özellikler:**
- Read-only görünüm
- Hiçbir aksiyon butonu yok
- Ders detayları: saat, yer, hoca

---

##### 2. PendingProgramView (Onay Bekleyen Program)

```
┌─────────────────────────────────────────────────────────────────┐
│  ⏳ YOUR PROGRAM IS WAITING FOR APPROVAL                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Semester: 2025 Spring                                          │
│  Status: Pending                                                 │
│  Submitted at: November 21, 2025, 09:00                         │
│  Advisor: Prof. Dr. Ayşe Yılmaz                                 │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  📚 Selected Courses (5 courses • 18 ECTS)                      │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ CS201 - Data Structures                     4 ECTS      │    │
│  │ CS202 - Computer Organization               3 ECTS      │    │
│  │ MATH301 - Linear Algebra                    3 ECTS      │    │
│  │ CS203 - Algorithms                          4 ECTS      │    │
│  │ ENG201 - Technical English                  4 ECTS      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ⚠️ Want to make changes? You can cancel and create a new one.  │
│                                                                  │
│                                        [Cancel Program]          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Özellikler:**
- Seçilen dersler görünür
- "Cancel Program" butonu (DELETE /enrollments/:id çağırır)
- İptal sonrası → CourseSelectionView'a yönlendir

---

##### 3. RejectedProgramView (Reddedilen Program) ⭐ YENİ

```
┌─────────────────────────────────────────────────────────────────┐
│  ❌ YOUR PROGRAM WAS REJECTED                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Rejected at: November 21, 2025, 14:30                          │
│  Rejected by: Prof. Dr. Ayşe Yılmaz                             │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  📋 Rejection Reason:                                            │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ "Schedule conflict detected. CS201 and MATH301 are at   │    │
│  │  the same time slot. Please select different courses."  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  📚 Rejected Courses (18 ECTS):                                 │
│                                                                  │
│  • CS201 - Data Structures (4 ECTS) - Dr. Mehmet Öz            │
│  • CS202 - Computer Organization (3 ECTS) - Prof. Ayşe Demir   │
│  • MATH301 - Linear Algebra (3 ECTS) - Dr. Ali Kaya            │
│  • CS203 - Algorithms (4 ECTS) - Dr. Mehmet Öz                 │
│  • ENG201 - Technical English (4 ECTS) - Dr. Jane Smith        │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  ℹ️ You have been rejected 2 times this semester.               │
│                                              [View History]      │
│                                                                  │
│  ═══════════════════════════════════════════════════════════   │
│                                                                  │
│           [🔄 Create New Program]                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Özellikler:**
- Red sebebi açıkça gösterilir
- Reddeden danışman adı gösterilir
- Reddedilen dersler listelenir
- "View History" butonu (birden fazla red varsa)
- **"Create New Program"** butonu → CourseSelectionView'a geçiş

**"Create New Program" Butonu Davranışı:**
```typescript
const handleCreateNewProgram = () => {
  setCurrentView('course_selection')  // State değişikliği
}
```

---

##### 4. CourseSelectionView (Ders Seçim Formu)

```
┌─────────────────────────────────────────────────────────────────┐
│  📝 CREATE YOUR COURSE PROGRAM                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Semester: 2025 Spring                                          │
│  Department: Computer Science                                    │
│  Class Level: 3rd Year                                          │
│  Registration Deadline: November 25, 2025, 23:59                │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  📚 Available Courses                      Selected: 3 (12 ECTS)│
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ ☑️ CS201 - Data Structures              4 ECTS  [85/150]│    │
│  │ ☑️ CS202 - Computer Organization        3 ECTS  [90/100]│    │
│  │ ☐ MATH301 - Linear Algebra              3 ECTS  [60/80] │    │
│  │ ☑️ CS203 - Algorithms                   4 ECTS  [75/120]│    │
│  │ ☐ CS304 - Software Engineering          4 ECTS [100/100]│ 🔴│
│  │ ☐ ENG201 - Technical English            4 ECTS  [45/50] │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  🔴 = Full (cannot select)                                      │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│                                        [Submit Program]          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Özellikler:**
- Alınabilir dersler listelenir
- Kontenjan bilgisi gösterilir
- Dolu dersler seçilemez
- "Submit Program" butonu (POST /enrollments çağırır)
- Başarılı submit sonrası → PendingProgramView'a yönlendir

---

#### Component Yapısı

```
app/
└── (dashboard)/
    └── enrollment/
        └── page.tsx                    # Ana sayfa (router)

components/
└── enrollment/
    ├── enrollment-page.tsx             # Main container (state management)
    ├── approved-program-view.tsx       # Onaylanmış program (read-only)
    ├── pending-program-view.tsx        # Onay bekleyen program
    ├── rejected-program-view.tsx       # Reddedilen program + Create New butonu
    ├── course-selection-view.tsx       # Ders seçim formu
    ├── rejection-history-modal.tsx     # Tüm red geçmişi (lazy load)
    └── cancel-confirmation-modal.tsx   # İptal onay modalı
```

---

#### State Management

```typescript
// enrollment-page.tsx
type ViewState = 
  | 'loading'
  | 'approved'
  | 'pending'
  | 'rejected'
  | 'course_selection'

const EnrollmentPage = () => {
  const [viewState, setViewState] = useState<ViewState>('loading')
  const [program, setProgram] = useState(null)
  const [latestRejection, setLatestRejection] = useState(null)
  
  useEffect(() => {
    loadData()
  }, [])
  
  const loadData = async () => {
    const [programRes, rejectionRes] = await Promise.all([
      getMyEnrollments({ semester: currentSemester }),
      getLatestRejection({ semester: currentSemester })
    ])
    
    const approved = programRes.programs?.find(p => p.status === 'approved')
    const pending = programRes.programs?.find(p => p.status === 'pending')
    
    if (approved) {
      setProgram(approved)
      setViewState('approved')
    } else if (pending) {
      setProgram(pending)
      setViewState('pending')
    } else if (rejectionRes.has_rejection) {
      setLatestRejection(rejectionRes.latest_rejection)
      setViewState('rejected')
    } else {
      setViewState('course_selection')
    }
  }
  
  // RejectedProgramView'dan çağrılır
  const handleCreateNewProgram = () => {
    setViewState('course_selection')
  }
  
  // CourseSelectionView'dan submit sonrası çağrılır
  const handleProgramSubmitted = (newProgram) => {
    setProgram(newProgram)
    setViewState('pending')
  }
  
  // PendingProgramView'dan cancel sonrası çağrılır
  const handleProgramCancelled = () => {
    setProgram(null)
    // Tekrar loadData çağır - rejection varsa rejected view, yoksa selection view
    loadData()
  }
  
  switch (viewState) {
    case 'loading':
      return <LoadingSpinner />
    case 'approved':
      return <ApprovedProgramView program={program} />
    case 'pending':
      return <PendingProgramView program={program} onCancel={handleProgramCancelled} />
    case 'rejected':
      return <RejectedProgramView 
               rejection={latestRejection} 
               onCreateNew={handleCreateNewProgram} 
             />
    case 'course_selection':
      return <CourseSelectionView onSubmit={handleProgramSubmitted} />
  }
}
```

---

#### API Kullanımı

```typescript
// 1. Sayfa yüklendiğinde - paralel çağrı
const [programRes, rejectionRes] = await Promise.all([
  getMyEnrollments({ semester: '2025_spring' }),
  getLatestRejection({ semester: '2025_spring' })
])

// 2. Ders seçimi için (CourseSelectionView'da)
const courses = await getAvailableCourses({ semester: '2025_spring' })

// 3. Program gönderme (CourseSelectionView'da)
const newProgram = await createEnrollment({
  semester: '2025_spring',
  course_ids: selectedCourseIds
})

// 4. Program iptal etme (PendingProgramView'da)
await cancelEnrollment(programId)

// 5. Tüm red geçmişi (RejectionHistoryModal'da - lazy load)
const allRejections = await getAllRejections({ semester: '2025_spring' })
```

#### API Endpoints Özeti (Öğrenci)

| Endpoint | Kullanım | Ne Zaman Çağrılır |
|----------|----------|-------------------|
| `GET /enrollments/my` | Mevcut program kontrolü | Sayfa yüklendiğinde |
| `GET /enrollments/my/rejections/latest` | En son red bilgisi | Sayfa yüklendiğinde |
| `GET /enrollments/my/rejections` | Tüm red geçmişi | "View History" tıklandığında |
| `GET /enrollments/available-courses` | Alınabilir dersler | CourseSelectionView açıldığında |
| `POST /enrollments` | Program gönderme | Form submit |
| `DELETE /enrollments/:id` | Program iptal | Cancel butonu |

---

### Danışman Approval Flow

#### Sayfa Yapısı (`/advisor/enrollments`)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     /advisor/enrollments sayfası                         │
│                                                                          │
│   useEffect → GET /api/v1/enrollments/pending-approval                  │
│                                                                          │
│   ┌───────────────────────────────────────────────────────────────────┐ │
│   │                  Onay Bekleyen Programlar                          │ │
│   │                                                                    │ │
│   │  ┌─────────────────────────────────────────────────────────────┐  │ │
│   │  │ 👤 Ahmet Yılmaz (2021123456)                                │  │ │
│   │  │    Computer Science • 3. Sınıf • 5 ders, 18 AKTS            │  │ │
│   │  │                                              [İncele]        │  │ │
│   │  └─────────────────────────────────────────────────────────────┘  │ │
│   │                                                                    │ │
│   │  ┌─────────────────────────────────────────────────────────────┐  │ │
│   │  │ 👤 Ayşe Demir (2021123457)                                  │  │ │
│   │  │    Computer Science • 2. Sınıf • 4 ders, 15 AKTS            │  │ │
│   │  │                                              [İncele]        │  │ │
│   │  └─────────────────────────────────────────────────────────────┘  │ │
│   │                                                                    │ │
│   └───────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Detail Modal (İnceleme + Onay/Red)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ders Programı İnceleme                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Öğrenci: Ahmet Yılmaz              Öğrenci No: 2021123456      │
│  Bölüm: Computer Science            Sınıf: 3. Sınıf             │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  Seçilen Dersler (5 ders • 18 AKTS)                             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ CS301 - Algorithm Analysis           4 AKTS    [85/150] │    │
│  │ CS302 - Database Systems             3 AKTS    [90/100] │    │
│  │ CS303 - Operating Systems            4 AKTS    [75/120] │    │
│  │ MATH301 - Linear Algebra             3 AKTS    [60/80]  │    │
│  │ CS304 - Software Engineering         4 AKTS    [95/100] │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ⚠️ Öğrenci 18 AKTS seçmiş (normal dönem yükü: 30 AKTS)         │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                          [✗ Reddet]    [✓ Onayla]               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Rejection Modal

```
┌─────────────────────────────────────────────────────────────────┐
│                    Ders Programını Reddet                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Ahmet Yılmaz adlı öğrencinin ders programını reddetmek         │
│  üzeresiniz. Lütfen red sebebini belirtin.                      │
│                                                                  │
│  Red Sebebi *                                                    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Ders saatleri çakışıyor. CS301 ile MATH301 aynı        │    │
│  │ saatte.                                                 │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ⚠️ Red işlemi sonrasında:                                       │
│  • Öğrencinin seçtiği derslerin kontenjanları serbest bırakılacak│
│  • Red geçmişi kaydedilecek (öğrenci görebilecek)               │
│  • Öğrenci yeni ders kaydı yapabilecek                          │
│  • Öğrenciye bildirim gönderilecek                              │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                            [Vazgeç]    [Reddet]                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### Component Yapısı

```
app/
└── (dashboard)/
    └── advisor/
        └── enrollments/
            └── page.tsx                    # Pending approvals listesi

components/
└── advisor/
    ├── pending-enrollments-list.tsx        # Liste view
    ├── enrollment-detail-modal.tsx         # Detay inceleme modalı
    ├── approval-confirmation.tsx           # Onay confirmation (optional)
    └── rejection-modal.tsx                 # Red modal (sebep girişi)
```

#### API Kullanımı

```typescript
// 1. Pending programları getir
const response = await getPendingApprovals()
const pendingPrograms = response.pending_programs

// 2. Programı onayla
await approveEnrollment(programId)

// 3. Programı reddet (log kaydı otomatik oluşur)
await rejectEnrollment(programId, {
  rejection_reason: 'Ders saatleri çakışıyor...'
})
```

---

### Güvenlik Kontrolleri (Defense in Depth)

| Katman | Kontrol | Açıklama |
|--------|---------|----------|
| **Frontend** | Route guard | Pending program varsa ders seçim sayfasına erişim engelle |
| **Frontend** | UI disable | Kontenjan dolu derslerde seçim engelle |
| **Backend** | JWT validation | Token geçerliliği ve role kontrolü |
| **Backend** | Ownership check | Öğrenci sadece kendi programını iptal edebilir |
| **Backend** | Status check | Sadece pending programlar iptal edilebilir |
| **Backend** | DB constraint | UNIQUE(student_id, semester) duplicate prevention |
| **Backend** | CHECK constraint | current_enrollment >= 0 (negatif kontenjan önleme) |
| **Backend** | Row-level lock | SELECT FOR UPDATE ile race condition önleme |

**Altın Kural**: Frontend güvenlik için asla tek başına yeterli değil. Her zaman backend validation şart.