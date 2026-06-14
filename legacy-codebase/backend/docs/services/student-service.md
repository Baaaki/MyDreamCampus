# Student Service ⚠️ CRITICAL

## Sorumluluk
Öğrenci master datası (ad, soyad, numara, fakülte, bölüm, kayıt yılı, durum, danışman hoca)

**Source of Truth**: Bu servis öğrenci bilgileri için canonical data source'dur.
- **Okula kayıt yapan öğrenci asla silinmez** (soft delete: `is_active = false`)
- **Diğer servisler**: Inactive öğrencileri hard delete yapar (local cache'den kaldırır)

---

## İletişim

### Inbound (REST)
- Diğer servisler öğrenci detayı sorgulayabilir (opsiyonel, genelde event yeterli)
- `GET /api/v1/students/:id` - Tek öğrenci detayı
- `GET /api/v1/students?filters...` - Öğrenci listesi (pagination, filtering)
- `POST /api/v1/students/search` - Advanced search (full-text + filters)
- `POST /api/v1/students/bulk-import` - CSV toplu import

### Outbound (RabbitMQ)
Öğrenci ekleme/güncelleme/deaktivasyon olaylarında event yayınlar:
- `student.created` - Yeni öğrenci ekleme
- `student.updated` - Öğrenci bilgisi güncelleme
- `student.deactivated` - Öğrenci soft delete (is_active = false)

### Event Consumers
- **Enrollment Service**: Öğrenci listesini güncel tutmak için (local cache sync)
- **Meal Service**: Öğrenci listesini güncel tutmak için (local cache sync)
- **Auth Service**: Email değişikliklerini senkronize etmek için

### Inbound (RabbitMQ)
- **Staff Service**: `staff.deactivated` eventi dinler → orphaned öğrencilerin advisor_id = NULL yapılır

**Important**:
- `student.created` / `student.updated` → Enrollment ve Meal servisleri local cache'i günceller
- `student.deactivated` → Enrollment ve Meal servisleri local cache'den **hard delete** yapar (inactive öğrenciler silinir)
- `staff.deactivated` → Student Service dinler, ilgili öğrencilerin advisor_id = NULL yapılır (orphaned)

### Outbound (REST/gRPC - Synchronous)
- **Staff Service**: Advisor validation (`GET /api/v1/staff/:id` - danışman geçerliliği kontrolü)
- **Staff Service**: Bulk import'ta öğretmen listesi alma (`GET /api/v1/staff/instructors?department=X`)

---

## Database Schema

### Students Table

```sql
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    faculty VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    enrollment_year INT NOT NULL,
    class_level SMALLINT NOT NULL DEFAULT 1 CHECK (class_level BETWEEN 1 AND 6), -- 1-6 (for phased enrollment)
    advisor_id UUID,                        -- Danışman hoca UUID (Staff Service'ten, FK yok - Database per Service)
    status VARCHAR(50) DEFAULT 'active',    -- Akademik durum: active, graduated, suspended, withdrawn
    is_active BOOLEAN DEFAULT true,         -- Sistem durumu: true = sistemde aktif, false = soft deleted
    deleted_at TIMESTAMP DEFAULT NULL,      -- Soft delete timestamp (is_active = false olduğunda set edilir)
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
    -- NOT: advisor_id için Foreign Key YOK (Database per Service pattern)
    -- Validation: Staff Service REST call ile yapılır
);

-- Unique constraints only for active students
CREATE UNIQUE INDEX idx_students_number_unique
    ON students(student_number) WHERE is_active = true;

CREATE UNIQUE INDEX idx_students_email_unique
    ON students(email) WHERE is_active = true;

-- Performance indexes (only active students)
CREATE INDEX idx_students_department ON students(department) WHERE is_active = true;
CREATE INDEX idx_students_status ON students(status) WHERE is_active = true;
CREATE INDEX idx_students_class_level ON students(class_level) WHERE is_active = true;
CREATE INDEX idx_students_advisor ON students(advisor_id) WHERE is_active = true;
CREATE INDEX idx_students_is_active ON students(is_active);

-- Full-text search index
CREATE INDEX idx_students_fulltext ON students
    USING gin(to_tsvector('english', first_name || ' ' || last_name || ' ' || student_number));

-- Compound index for common queries
CREATE INDEX idx_students_dept_class ON students(department, class_level) WHERE is_active = true;
```

### Status vs is_active Açıklaması

| Alan | Amacı | Değerler |
|------|-------|----------|
| **status** | Öğrencinin **akademik durumu** | active, graduated, suspended, withdrawn |
| **is_active** | Öğrencinin **sistem durumu** (soft delete) | true, false |

**Örnek Senaryolar**:

| Senaryo | status | is_active | Açıklama |
|---------|--------|-----------|----------|
| Aktif öğrenci | active | true | Normal durum, ders kayıt yapabilir |
| Mezun öğrenci | graduated | true | Sistemde var, transcript görebilir, ders kayıt yapamaz |
| Uzaklaştırılan | suspended | true | Geçici olarak ders/sınav engelli |
| Kendi isteğiyle ayrılan | withdrawn | true | Kayıt sildirmiş, geçmiş kayıtları görüntülenebilir |
| Sistemden silinen | (herhangi) | false | Admin tarafından soft delete edilmiş |

**Kurallar**:
- `is_active = false` → Sistemden silinmiş (unique constraint'lerden muaf, diğer servisler hard delete yapar)
- `status = 'suspended'` → Login engellenecekse Auth Service'e `student.deactivated` event gönderilmeli
- `status = 'graduated'` veya `status = 'withdrawn'` → Enrollment Service ders kaydını engeller

**Auth Service İlişkisi**:
- Auth Service sadece `is_active` kontrol eder
- Suspended öğrencinin login'i engellenecekse → Admin `student.deactivated` event tetikler (geçici)
- Veya Auth Service'e student status bilgisi de gönderilir (event payload'da)

### Outbox Events Table

```sql
-- Transactional Outbox Pattern (ensures atomicity between DB write and event publishing)
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,        -- 'student.created', 'student.updated', 'student.deactivated'
    routing_key VARCHAR(100) NOT NULL,       -- RabbitMQ routing key
    payload JSONB NOT NULL,                  -- Event data (JSON)
    status outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT                       -- Son hata mesajı (debug için)
);

CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_events_retry ON outbox_events(status, retry_count) WHERE status = 'failed';
```

### Processed Events Table (Idempotency)

```sql
-- Event idempotency için (duplicate event processing prevention)
CREATE TABLE processed_events (
    event_id VARCHAR(255) PRIMARY KEY,      -- Event'in unique ID'si
    event_type VARCHAR(100) NOT NULL,       -- staff.deactivated, etc.
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON processed_events(processed_at);
```

**Neden Gerekli**:
- RabbitMQ at-least-once delivery garantisi verir (aynı mesaj birden fazla gelebilir)
- Network timeout, consumer crash gibi durumlarda event tekrar gönderilir
- Bu tablo ile aynı event'in birden fazla işlenmesi önlenir

**Kullanım (staff.deactivated handler)**:
```go
func HandleStaffDeactivated(event Event) error {
    // 1. Event daha önce işlendi mi?
    exists := db.QueryRow("SELECT 1 FROM processed_events WHERE event_id = $1", event.ID)
    if exists {
        log.Info("Event already processed, skipping")
        return nil
    }

    // 2. Transaction başlat
    tx := db.Begin()

    // 3. Business logic - orphaned öğrencilerin advisor_id = NULL
    tx.Exec("UPDATE students SET advisor_id = NULL WHERE advisor_id = $1", event.Data.StaffID)

    // 4. Event'i processed olarak işaretle
    tx.Exec("INSERT INTO processed_events (event_id, event_type) VALUES ($1, $2)",
        event.ID, event.Type)

    // 5. Commit
    return tx.Commit()
}
```

**Cleanup (Cron Job)**:
```sql
-- 30 günden eski processed_events kayıtlarını temizle
DELETE FROM processed_events WHERE processed_at < NOW() - INTERVAL '30 days';
```

### Import Jobs Table (Bulk Import Tracking)

```sql
-- Bulk import job tracking
CREATE TYPE import_job_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(255) NOT NULL,
    total_records INT NOT NULL DEFAULT 0,
    processed_records INT NOT NULL DEFAULT 0,
    successful_records INT NOT NULL DEFAULT 0,
    failed_records INT NOT NULL DEFAULT 0,
    status import_job_status DEFAULT 'pending',
    errors JSONB DEFAULT '[]',              -- Array of error objects
    created_by UUID NOT NULL,               -- Admin user ID
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_jobs_created_by ON import_jobs(created_by);
CREATE INDEX idx_import_jobs_created_at ON import_jobs(created_at DESC);
```

**Job Error Format**:
```json
{
  "errors": [
    {
      "row": 42,
      "student_number": "2021123999",
      "error_code": "STUDENT_NUMBER_EXISTS",
      "message": "Student number already exists"
    },
    {
      "row": 87,
      "student_number": "2021124000",
      "error_code": "INVALID_EMAIL",
      "message": "Invalid email format"
    }
  ]
}
```

**Outbox Pattern Flow**:
```
1. Business Logic + Outbox INSERT (same transaction)
   ↓
2. Background Worker polls pending events (5s interval)
   ↓
3. Publish to RabbitMQ
   ↓
4. Mark as 'processed' or 'failed' (with retry)
```

**Why Outbox Pattern**:
- ✅ **Atomicity**: DB write + event write aynı transaction'da (ACID guarantee)
- ✅ **At-least-once delivery**: Event kaybı imkansız (DB'de persist edilir)
- ✅ **Resilience**: RabbitMQ down olsa bile event DB'de saklanır
- ✅ **Retry mechanism**: Failed events otomatik retry edilir

---

## API Endpoints

### 🔒 POST /api/v1/students

Yeni öğrenci kaydı (tekil ekleme - manuel danışman seçimi)

**Role Requirement**: Admin only

**Request**:
```json
{
  "student_number": "2021123456",
  "first_name": "Ahmet",
  "last_name": "Yılmaz",
  "email": "ahmet.yilmaz@university.edu.tr",
  "faculty": "Engineering",
  "department": "Computer Science",
  "enrollment_year": 2021,
  "class_level": 4,
  "advisor_id": "uuid"  // REQUIRED: Danışman hoca (admin manuel seçer)
}
```

**Response** (201):
```json
{
  "id": "uuid",
  "student_number": "2021123456",
  "first_name": "Ahmet",
  "last_name": "Yılmaz",
  "email": "ahmet.yilmaz@university.edu.tr",
  "faculty": "Engineering",
  "department": "Computer Science",
  "enrollment_year": 2021,
  "class_level": 4,
  "advisor_id": "uuid",
  "status": "active",
  "created_at": "2025-11-11T10:00:00Z"
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_student_created_uuid",
  "event_type": "student.created",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "email": "ahmet.yilmaz@university.edu.tr",
    "faculty": "Engineering",
    "department": "Computer Science",
    "enrollment_year": 2021,
    "class_level": 4,
    "advisor_id": "uuid",
    "status": "active"
  }
}
```

**Business Logic**:
1. Student number uniqueness kontrolü (is_active = true)
2. Email uniqueness kontrolü (is_active = true)
3. `advisor_id` geçerliliği kontrolü (Staff Service validation with cache)
4. DB'ye kaydet (is_active = true)
5. Outbox Pattern ile event insert (same transaction)

**Event Consumers**: Enrollment Service, Meal Service (local cache update)

---

### 🔒 POST /api/v1/students/bulk-import

Toplu öğrenci yüklemesi (CSV) - Otomatik danışman atama

**Role Requirement**: Admin only

**Request** (multipart/form-data):
```
POST /api/v1/students/bulk-import
Content-Type: multipart/form-data

file: students.csv
```

**CSV Format**:
```csv
student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level
2021123456,Ahmet,Yılmaz,ahmet@university.edu.tr,Engineering,Computer Science,2021,4
2021123457,Ayşe,Demir,ayse@university.edu.tr,Engineering,Computer Science,2021,4
```

**Response** (202 Accepted):
```json
{
  "job_id": "uuid",
  "status": "processing",
  "message": "Import job started",
  "total_records": 1500,
  "estimated_completion": "2025-11-11T10:15:00Z"
}
```

**Job Status**: `GET /api/v1/students/bulk-import/:job_id`

**Response** (200):
```json
{
  "job_id": "uuid",
  "status": "completed",
  "total_records": 1500,
  "successful": 1485,
  "failed": 15,
  "errors": [
    {
      "row": 42,
      "student_number": "2021123999",
      "error": "STUDENT_NUMBER_EXISTS"
    }
  ],
  "completed_at": "2025-11-11T10:12:34Z"
}
```

**RabbitMQ Event Published** (per student):
Same as `student.created.v1` (batch events published via outbox)

**Business Logic**:
1. **Authorization**: Admin role kontrolü (JWT token'dan)
2. **File Validation**:
   - Max file size: 10MB
   - File extension: .csv
   - CSV header validation (required columns)
3. **Job Creation**:
   - Import job kaydı oluştur (status: 'pending')
   - Job ID döndür (202 Accepted)
4. **Background Processing** (async):
   - Job status: 'processing', started_at = NOW()
   - Parse CSV (streaming, memory-efficient)
   - Group students by department
   - Fetch advisors from Staff Service (`GET /api/v1/staff/instructors?department=X`)
   - Cache advisor list (Redis, TTL: 5 dakika)
5. **Auto-assign Advisors** (round-robin):
   ```go
   advisorIndex := studentIndex % len(departmentAdvisors)
   student.AdvisorID = departmentAdvisors[advisorIndex].ID
   ```
6. **Row Validation** (per row):
   - Email format validation (regex)
   - Student number format validation
   - Email uniqueness check (batch query)
   - Student number uniqueness check (batch query)
   - class_level range check (1-6)
7. **Batch Insert** (100 records per transaction):
   - INSERT students + outbox events (same transaction)
   - Update job progress: processed_records++
   - On error: log to job.errors, continue with next row
8. **Job Completion**:
   - Job status: 'completed', completed_at = NOW()
   - Calculate: successful_records, failed_records
9. **Event Publishing**:
   - Background outbox worker publishes student.created events

**Error Handling**:
- Row-level errors: Log and continue (partial success allowed)
- System errors (DB down): Job status = 'failed', retry later

**Event Consumers**: Enrollment Service, Meal Service (local cache update)

---

### 🔒 GET /api/v1/students/bulk-import/:job_id

Import job durumu sorgulama

**Role Requirement**: Admin only (sadece kendi oluşturduğu job'ları görebilir)

**Response** (200):
```json
{
  "job_id": "uuid",
  "file_name": "students_2025.csv",
  "status": "completed",
  "total_records": 1500,
  "processed_records": 1500,
  "successful_records": 1485,
  "failed_records": 15,
  "errors": [
    {
      "row": 42,
      "student_number": "2021123999",
      "error_code": "STUDENT_NUMBER_EXISTS",
      "message": "Student number already exists"
    }
  ],
  "created_by": "admin-uuid",
  "started_at": "2025-11-11T10:00:00Z",
  "completed_at": "2025-11-11T10:12:34Z",
  "created_at": "2025-11-11T09:59:55Z"
}
```

**Response - Processing** (200):
```json
{
  "job_id": "uuid",
  "status": "processing",
  "total_records": 1500,
  "processed_records": 750,
  "progress_percentage": 50,
  "started_at": "2025-11-11T10:00:00Z"
}
```

**Business Logic**:
1. Job ID ile import_jobs tablosundan job'ı çek
2. Authorization: `job.created_by == current_user_id` kontrolü
3. Job not found: 404 döndür
4. Job durumuna göre response formatla

---

### 🔒 GET /api/v1/students/bulk-import

Admin'in tüm import job'larını listele

**Role Requirement**: Admin only

**Query Parameters**:
- `page` (default: 1)
- `limit` (default: 20, max: 100)
- `status` (filter: pending, processing, completed, failed)

**Response** (200):
```json
{
  "data": [
    {
      "job_id": "uuid",
      "file_name": "students_2025.csv",
      "status": "completed",
      "total_records": 1500,
      "successful_records": 1485,
      "failed_records": 15,
      "created_at": "2025-11-11T09:59:55Z",
      "completed_at": "2025-11-11T10:12:34Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

**Business Logic**:
- Filter by `created_by = current_user_id`
- Sort by `created_at DESC` (en yeni job'lar önce)

---

### 🔓 POST /api/v1/students/search

Gelişmiş arama (full-text search, multi-field filtering, sorting)

**Role Requirement**:
- **Teacher**: Sadece danışmanlık yaptığı öğrenciler
- **Admin**: Tüm öğrenciler

**Request**:
```json
{
  "query": "ahmet yılmaz",
  "filters": {
    "department": ["Computer Science", "Software Engineering"],
    "class_level": [3, 4],
    "status": ["active"],
    "enrollment_year": [2021, 2022],
    "advisor_id": "uuid"  // Admin only
  },
  "sort": {
    "field": "last_name",
    "order": "asc"
  },
  "pagination": {
    "cursor": "encoded_cursor",
    "limit": 50
  }
}
```

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "department": "Computer Science",
      "class_level": 4,
      "advisor": {
        "id": "uuid",
        "first_name": "Ayşe",
        "last_name": "Demir"
      },
      "status": "active"
    }
  ],
  "pagination": {
    "next_cursor": "encoded_cursor",
    "has_more": true,
    "total_count": 1250
  }
}
```

**Business Logic**:
- Full-text search using PostgreSQL GIN index
- Role-based filtering (teacher: only advisees)
- Cursor-based pagination (performance optimization)

---

### 🔓 GET /api/v1/students/:id

Öğrenci detayı görüntüleme

**Role Requirement**:
- **Student**: Sadece kendi bilgisi (`user_id == student_id`)
- **Teacher**: Sadece danışmanlık yaptığı öğrenciler (`advisor_id == user_id`)
- **Admin**: Tüm öğrenciler

**Response** (200):
```json
{
  "id": "uuid",
  "student_number": "2021123456",
  "first_name": "Ahmet",
  "last_name": "Yılmaz",
  "email": "ahmet.yilmaz@university.edu.tr",
  "faculty": "Engineering",
  "department": "Computer Science",
  "enrollment_year": 2021,
  "class_level": 4,
  "advisor": {
    "id": "uuid",
    "first_name": "Ayşe",
    "last_name": "Demir",
    "email": "ayse.demir@university.edu.tr"
  },
  "status": "active",
  "created_at": "2025-11-11T10:00:00Z",
  "updated_at": "2025-11-11T10:00:00Z"
}
```

**Business Logic**:
- Authorization check based on role
- Cache lookup (Redis: `student:{id}`, TTL: 1 hour)

---

### 🔒 PUT /api/v1/students/:id

Öğrenci bilgisi güncelleme

**Role Requirement**: Admin only

**Request**:
```json
{
  "class_level": 4,
  "advisor_id": "uuid",
  "status": "active"
}
```

**Response** (200):
```json
{
  "id": "uuid",
  "student_number": "2021123456",
  "class_level": 4,
  "advisor_id": "uuid",
  "status": "active",
  "updated_at": "2025-11-11T11:00:00Z"
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_student_updated_uuid",
  "event_type": "student.updated",
  "timestamp": "2025-11-11T11:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "changed_fields": {
      "class_level": 4,
      "advisor_id": "uuid",
      "status": "active"
    }
  }
}
```

**Business Logic**:
1. Danışman değişikliği varsa `advisor_id` geçerliliği kontrolü (Staff Service cache)
2. Email değişikliği varsa uniqueness kontrolü
3. DB update + outbox event insert (same transaction)
4. Cache invalidation (Redis: `student:{id}`)

**Event Consumers**: Enrollment Service, Meal Service, Auth Service (email update)

---

### 🔒 DELETE /api/v1/students/:id

Öğrenci deaktivasyon (soft delete)

**Role Requirement**: Admin only

**Response** (200):
```json
{
  "message": "Student deactivated successfully",
  "id": "uuid",
  "is_active": false
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_student_deactivated_uuid",
  "event_type": "student.deactivated",
  "timestamp": "2025-11-11T12:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "is_active": false,
    "deleted_at": "2025-11-11T12:00:00Z"
  }
}
```

**Business Logic**:
1. Set `is_active = false`, `deleted_at = NOW()`
2. Outbox event insert (same transaction)
3. Cache invalidation
4. **Important**: Student Service soft deletes, but Enrollment/Meal services **hard delete** from local cache

**Event Consumers**: Enrollment Service, Meal Service (hard delete from local cache)

---

### 🔓 GET /api/v1/students

Öğrenci listesi (pagination, filtering)

**Role Requirement**:
- **Teacher**: Sadece danışmanlık yaptığı öğrenciler
- **Admin**: Tüm öğrenciler

**Query Parameters**:
- `page` (default: 1)
- `limit` (default: 20, max: 100)
- `department` (filter by department)
- `class_level` (filter by class)
- `status` (filter by status)
- `advisor_id` (filter by advisor - admin only)
- `sort_by` (student_number, last_name, enrollment_year)
- `sort_order` (asc, desc)

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "department": "Computer Science",
      "class_level": 4,
      "advisor": {
        "id": "uuid",
        "first_name": "Ayşe",
        "last_name": "Demir"
      },
      "status": "active"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

**Business Logic**:
- Role-based filtering (teacher: `advisor_id = user_id`)
- Pagination with N+1 query prevention (eager load advisor)

---

### 🔒 GET /api/v1/students/orphaned

Danışmansız öğrenciler listesi (advisor_id = NULL)

**Role Requirement**: Admin only

**Query Parameters**:
- `page` (default: 1)
- `limit` (default: 20, max: 100)
- `department` (filter by department)
- `class_level` (filter by class)

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "department": "Computer Science",
      "class_level": 4,
      "advisor_id": null,
      "status": "active"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

**Business Logic**:
- Filter by `advisor_id IS NULL` AND `is_active = true`
- Sort by `department ASC, class_level ASC, last_name ASC`

---

### 🔒 PUT /api/v1/students/bulk-advisor-assign

Seçilen öğrencilere toplu danışman atama

**Role Requirement**: Admin only

**Request**:
```json
{
  "student_ids": ["uuid1", "uuid2", "uuid3"],
  "advisor_id": "teacher-uuid"
}
```

**Response** (200):
```json
{
  "message": "Advisor assigned successfully",
  "updated_count": 3,
  "advisor": {
    "id": "teacher-uuid",
    "first_name": "Ayşe",
    "last_name": "Demir"
  },
  "students": [
    {"id": "uuid1", "student_number": "2021123456"},
    {"id": "uuid2", "student_number": "2021123457"},
    {"id": "uuid3", "student_number": "2021123458"}
  ]
}
```

**RabbitMQ Event Published** (per student):
```json
{
  "event_id": "evt_student_updated_uuid",
  "event_type": "student.updated",
  "timestamp": "2025-11-11T11:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "changed_fields": {
      "advisor_id": "teacher-uuid"
    }
  }
}
```

**Business Logic**:
1. Validate `advisor_id` exists and is active teacher (Staff Service)
2. Validate all `student_ids` exist and are active
3. Batch update: `UPDATE students SET advisor_id = :advisor_id WHERE id IN (:student_ids)`
4. Insert outbox events for each student (same transaction)
5. Return updated students list

**Event Consumers**: Enrollment Service, Meal Service (advisor info update in local cache)

---

### 🔓 GET /api/v1/students/my-advisees

Danışman hocanın öğrencileri (teacher için shortcut)

**Role Requirement**: Teacher only

**Response** (200):
```json
{
  "advisor": {
    "id": "uuid",
    "first_name": "Ayşe",
    "last_name": "Demir"
  },
  "students": [
    {
      "id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "class_level": 4,
      "status": "active"
    }
  ],
  "total_count": 15
}
```

**Business Logic**:
- Filter by `advisor_id = current_user_id` AND `is_active = true` AND `status = active`
- Sort by `class_level ASC, last_name ASC`

---

## RabbitMQ Configuration

### Exchange & Routing Keys

```
Exchange: "student.events" (type: topic)

Routing Keys:
- student.created
- student.updated
- student.deactivated

Dead Letter Queue:
- DLQ Exchange: "student.events.dlq"
- DLQ Queue: "student.events.dlq.queue"
```

### Subscribed Events (Inbound)

```
Exchange: "staff.events" (type: topic)

Subscribed Routing Keys:
- staff.deactivated → Orphaned öğrencilerin advisor_id = NULL yapılır
```

### Event Schemas

#### student.created
Published when: New student created (POST /students)

#### student.updated
Published when: Student data updated (PUT /students/:id)

#### student.deactivated
Published when: Student soft deleted (DELETE /students/:id)

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_INPUT | Validation hatası |
| 400 | INVALID_CSV_FORMAT | CSV format hatası |
| 403 | FORBIDDEN | Yetkisiz erişim |
| 404 | STUDENT_NOT_FOUND | Öğrenci bulunamadı |
| 404 | ADVISOR_NOT_FOUND | Danışman hoca bulunamadı |
| 409 | STUDENT_NUMBER_EXISTS | Öğrenci numarası zaten kayıtlı |
| 409 | EMAIL_EXISTS | Email zaten kayıtlı |
| 500 | INTERNAL_ERROR | Server hatası |
| 503 | STAFF_SERVICE_UNAVAILABLE | Staff Service erişilemiyor |

---

## Related Services

- **Staff Service**: Advisor validation + bulk import'ta öğretmen listesi alma
- **Enrollment Service**: Event consumer (student.created, student.updated, student.deactivated)
- **Meal Service**: Event consumer (student.created, student.updated, student.deactivated)
- **Auth Service**: Event consumer (student.updated - email değişikliği için)
- **Notification Service**: Event consumer (student.created, student.updated, student.deactivated)
  > **Not**: Notification Service tüm servislere en son eklenecektir. Şu an event'ler sadece yukarıdaki servisler tarafından consume edilmektedir.

---

**Version**: 5.0.0 (Added import_jobs, processed_events tables, detailed bulk import)
**Last Updated**: 2025-12-12
