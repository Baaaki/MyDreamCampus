# Attendance Service

## Sorumluluk
QR kod ile yoklama, manuel yoklama girişi, devamsızlık takibi, dönem sonu devamsızlık raporu

**High-Write Service**: Redis buffering + async batch insert zorunlu (60,000 öğrenci × 5 ders = 300,000 yoklama/gün peak load)

---

## Temel Mimari

### Write Path (QR Scan - Hot Path)
```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌────────────────┐
│ Student │────►│   API   │────►│  Redis  │────►│ Background Job │
│  Phone  │     │ Gateway │     │  Buffer │     │ (Batch Writer) │
└─────────┘     └─────────┘     └─────────┘     └───────┬────────┘
                     │                                   │
                     │ Response < 50ms                   ▼
                     ▼                            ┌────────────┐
                  ✅ OK                           │ PostgreSQL │
                                                  └────────────┘
```

### Read Path (Student/Instructor View)
```
┌─────────┐     ┌─────────┐     
│ Request │────►│  Redis  │──── (hit) ────► Response
└─────────┘     │  Cache  │
                └────┬────┘
                     │ (miss)
                     ▼
              ┌────────────┐
              │ PostgreSQL │────► Response + Cache Update
              └────────────┘
```

---

## İletişim

### Inbound (RabbitMQ - Event Consumers)
- `student.created` → Öğrenci bilgisini local cache'e ekler
- `student.updated` → Öğrenci bilgisini günceller
- `student.deactivated` → Öğrenciyi deaktif yapar (`is_active = false`)
- `course.semester.created` → Dönemlik ders bilgisini ekler
- `course.semester.updated` → Ders bilgisini günceller (instructor değişikliği vb.)
- `course.semester.deleted` → İlgili dersi ve bağlı enrollments_cache kayıtlarını siler (attendance_sessions ve attendance_records CASCADE ile silinir)
- `enrollment.program_approved` → Öğrenci-ders kaydı oluşturur (yoklama alınabilmesi için)

### Outbound (RabbitMQ - Asynchronous)
- `attendance.semester.failed` → Dönem sonu: Devamsızlıktan kalan öğrenciler (Grades Service, Notification Service)

---

## Database Schema

```sql
-- ==========================================
-- A. CACHE TABLOLARI (Diğer Servislerden)
-- ==========================================

CREATE TABLE students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    department VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,           -- Öğrenci aktif mi? (student.deactivated ile false olur)
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_cache_number ON students_cache(student_number);
CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;

-- Ders Bilgileri (Course Catalog'dan)
CREATE TABLE courses_cache (
    id UUID PRIMARY KEY,                    -- semester_course_id
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,          -- "2025_spring" format
    department VARCHAR(100),
    instructor_id UUID NOT NULL,
    instructor_fullname VARCHAR(150),
    total_weeks SMALLINT DEFAULT 14,        -- Sabit 14 hafta
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_courses_cache_semester ON courses_cache(semester);
CREATE INDEX idx_courses_cache_instructor ON courses_cache(instructor_id);

-- Öğrenci-Ders Kayıtları (Enrollment'tan)
CREATE TABLE enrollments_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_enrollments_student ON enrollments_cache(student_id);
CREATE INDEX idx_enrollments_course ON enrollments_cache(course_id);
CREATE INDEX idx_enrollments_semester ON enrollments_cache(semester);

-- ==========================================
-- B. OPERASYONEL TABLOLAR
-- ==========================================

-- Yoklama Oturumları (Hoca başlatır)
CREATE TABLE attendance_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    instructor_id UUID NOT NULL,
    semester VARCHAR(50) NOT NULL,
    week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 14),
    session_date DATE NOT NULL,
    
    -- QR Security
    qr_secret VARCHAR(64) NOT NULL,         -- HMAC secret (her session için unique)
    qr_rotation_interval SMALLINT DEFAULT 15,  -- QR yenilenme süresi (saniye)
    
    -- Session durumu
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,          -- Session bitiş zamanı
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Aynı ders için aynı hafta sadece 1 session
    UNIQUE(course_id, week_number)
);

CREATE INDEX idx_sessions_course ON attendance_sessions(course_id);
CREATE INDEX idx_sessions_active ON attendance_sessions(is_active, expires_at) WHERE is_active = TRUE;
CREATE INDEX idx_sessions_semester ON attendance_sessions(semester);

-- Yoklama Kayıtları
CREATE TABLE attendance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES attendance_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL,                -- Denormalized (query performance)
    semester VARCHAR(50) NOT NULL,          -- Denormalized (query performance)
    week_number SMALLINT NOT NULL,          -- Denormalized (öğrenci haftalık görünüm)
    
    -- Yoklama bilgisi
    is_present BOOLEAN NOT NULL DEFAULT TRUE,
    marked_via VARCHAR(20) NOT NULL CHECK (marked_via IN ('qr_scan', 'manual')),
    
    -- QR scan detayları (sadece qr_scan için)
    scanned_at TIMESTAMP,
    qr_timestamp BIGINT,                    -- QR'daki timestamp (replay attack prevention)
    
    -- Manuel giriş detayları (sadece manual için)
    manually_marked_by UUID,                -- Instructor ID
    manually_marked_at TIMESTAMP,
    manual_note TEXT,                       -- "Telefonu bozuk", "Geç geldi" vb.
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Bir öğrenci bir session'da sadece 1 kayıt
    UNIQUE(session_id, student_id)
);

CREATE INDEX idx_records_session ON attendance_records(session_id);
CREATE INDEX idx_records_student ON attendance_records(student_id);
CREATE INDEX idx_records_student_course ON attendance_records(student_id, course_id);
CREATE INDEX idx_records_course_semester ON attendance_records(course_id, semester);
CREATE INDEX idx_records_week ON attendance_records(course_id, week_number);

-- ==========================================
-- C. OUTBOX TABLOSU (Event Publishing)
-- ==========================================

CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';
```

---

## Redis Yapısı

### 1. Aktif Session Cache
```
Key: attendance:session:{session_id}
Type: HASH
TTL: Session süresi + 5 dakika
Fields:
  - course_id: "uuid"
  - instructor_id: "uuid"
  - semester: "2025_spring"
  - week_number: "5"
  - qr_secret: "random_64_char_secret"
  - qr_rotation_interval: "15"
  - expires_at: "1699999999"
  - enrolled_count: "150"
```

### 2. Session Enrolled Students (Bloom Filter Alternative)
```
Key: attendance:session:{session_id}:enrolled
Type: SET
TTL: Session süresi + 5 dakika
Members: [student_id_1, student_id_2, ...]

Purpose: O(1) enrollment check during QR scan
```

### 3. Attendance Buffer (Write Buffer)
```
Key: attendance:buffer:{session_id}
Type: HASH
TTL: 10 dakika
Fields:
  - {student_id}: "{timestamp}|{qr_timestamp}|qr_scan"
  
Example:
  - "uuid-student-1": "1699999999|1699999950|qr_scan"
  - "uuid-student-2": "1699999999|1699999955|qr_scan"
```

### 4. Already Marked Check (Duplicate Prevention)
```
Key: attendance:marked:{session_id}
Type: SET
TTL: Session süresi + 5 dakika
Members: [student_id_1, student_id_2, ...]

Purpose: O(1) duplicate scan prevention
```

### 5. Student Attendance Summary Cache
```
Key: attendance:student:{student_id}:summary:{semester}
Type: HASH
TTL: 1 saat
Fields:
  - courses: "[{course_id, present_count, absent_count, absent_weeks}]" (JSON string)

Purpose: GET /api/v1/attendance/my endpoint cache
Invalidation: On attendance record insert/update for this student
```

---

## QR Kod Güvenlik Mekanizması

### QR Payload Yapısı
```json
{
  "sid": "session_uuid",
  "ts": 1699999999,
  "sig": "hmac_signature"
}
```

### Signature Hesaplama
```
signature = HMAC-SHA256(
    key: session.qr_secret,
    message: session_id + "|" + timestamp_window
)

timestamp_window = floor(current_timestamp / rotation_interval)
```

### Validation Flow
```
1. Parse QR payload
2. Check session exists and is_active
   - Redis'ten dene, yoksa DB'den oku (fallback)
   - ❌ Not found: 404 SESSION_NOT_FOUND
3. Check session not expired
   - ❌ Expired: 400 SESSION_EXPIRED
4. Validate signature:
   - Calculate expected signature for current window
   - Also calculate for previous window (grace period)
   - ❌ Mismatch: 400 INVALID_QR_CODE
5. Check timestamp freshness (replay attack prevention):
   - max_age = rotation_interval * 3 (45 saniye default)
   - ❌ If (current_timestamp - qr_payload.ts) > max_age: 400 QR_EXPIRED
6. Check student enrolled in course (with fallback)
   - Redis available: SISMEMBER attendance:session:{sid}:enrolled {student_id}
   - Redis down: PostgreSQL'den kontrol
   - ❌ Not enrolled: 403 NOT_ENROLLED
7. Check not already marked (duplicate prevention):
   - Redis available: SISMEMBER attendance:marked:{sid} {student_id}
   - Redis down: PostgreSQL'den kontrol
     SELECT EXISTS(SELECT 1 FROM attendance_records 
     WHERE session_id = $1 AND student_id = $2)
   - ❌ Already marked: 409 ALREADY_MARKED
     Response: {"message": "Bu dersin yoklamasında zaten varsınız"}
8. Write to Redis buffer (or direct DB if Redis down)
9. Return success
```

### Rotation Mechanism
```
rotation_interval = 15 seconds

Frontend her 15 saniyede yeni QR ister:
GET /api/v1/attendance/sessions/{id}/qr?t={timestamp}

Backend:
1. Verify instructor owns session
2. Calculate current timestamp_window
3. Generate signature
4. Return QR data
```

---

## Redis Fallback Mekanizması

### Problem
Redis crash veya restart sonrası cache boşalır. Session PostgreSQL'de mevcut olsa bile QR endpoint'leri çalışmaz.

### Çözüm: DB Fallback + Cache Warming

**Session Lookup (her Redis read öncesi):**
```go
func GetSession(sessionID string) (*Session, error) {
    // 1. Redis'ten dene
    session, err := redis.HGetAll("attendance:session:" + sessionID)
    if err == nil && len(session) > 0 {
        return parseSession(session), nil
    }
    
    // 2. Redis miss veya down → DB'den oku
    session, err := db.Query(`
        SELECT id, course_id, instructor_id, semester, week_number, 
               qr_secret, qr_rotation_interval, expires_at
        FROM attendance_sessions 
        WHERE id = $1 AND is_active = TRUE AND expires_at > NOW()
    `, sessionID)
    if err != nil {
        return nil, err
    }
    
    // 3. Cache'i yeniden ısıt
    go warmSessionCache(session)
    
    return session, nil
}

func warmSessionCache(session *Session) {
    pipe := redis.Pipeline()
    
    // Session hash
    pipe.HSet("attendance:session:"+session.ID, sessionToMap(session))
    pipe.Expire("attendance:session:"+session.ID, time.Until(session.ExpiresAt)+5*time.Minute)
    
    // Enrolled students set
    enrolledIDs := db.Query(`
        SELECT student_id FROM enrollments_cache 
        WHERE course_id = $1 AND semester = $2
    `, session.CourseID, session.Semester)
    
    for _, id := range enrolledIDs {
        pipe.SAdd("attendance:session:"+session.ID+":enrolled", id)
    }
    pipe.Expire("attendance:session:"+session.ID+":enrolled", time.Until(session.ExpiresAt)+5*time.Minute)
    
    pipe.Exec()
}
```

**Enrollment Check (with fallback):**
```go
func IsEnrolled(sessionID, studentID, courseID, semester string) (bool, error) {
    // 1. Redis'ten dene
    exists, err := redis.SIsMember("attendance:session:"+sessionID+":enrolled", studentID)
    if err == nil {
        return exists, nil
    }
    
    // 2. Redis down → DB'den kontrol
    var count int
    err = db.QueryRow(`
        SELECT COUNT(*) FROM enrollments_cache 
        WHERE student_id = $1 AND course_id = $2 AND semester = $3
    `, studentID, courseID, semester).Scan(&count)
    
    if err != nil {
        return false, err
    }
    
    return count > 0, nil
}
```

**Marked Check Fallback:**
```go
func IsAlreadyMarked(sessionID, studentID string) bool {
    // 1. Redis'ten dene
    exists, err := redis.SIsMember("attendance:marked:"+sessionID, studentID)
    if err == nil {
        return exists
    }
    
    // 2. Redis down → DB'den kontrol
    count := db.QueryRow(`
        SELECT COUNT(*) FROM attendance_records 
        WHERE session_id = $1 AND student_id = $2
    `, sessionID, studentID)
    
    return count > 0
}
```

**Write Buffer Fallback (Redis down durumunda):**
```go
func WriteAttendance(sessionID, studentID string, data AttendanceData) error {
    // 1. Redis'e yazmayı dene
    err := redis.HSet("attendance:buffer:"+sessionID, studentID, data.Encode())
    if err == nil {
        redis.SAdd("attendance:marked:"+sessionID, studentID)
        return nil
    }
    
    // 2. Redis down → Direkt DB'ye yaz (sync)
    return db.Exec(`
        INSERT INTO attendance_records (session_id, student_id, ...)
        VALUES ($1, $2, ...)
        ON CONFLICT (session_id, student_id) DO NOTHING
    `, sessionID, studentID, ...)
}
```

**ClearSessionRedisKeys (Session kapatıldığında):**
```go
func ClearSessionRedisKeys(sessionID string) {
    keys := []string{
        "attendance:session:" + sessionID,              // Session hash
        "attendance:session:" + sessionID + ":enrolled", // Enrolled set
        "attendance:marked:" + sessionID,               // Marked set
        "attendance:buffer:" + sessionID,               // Buffer hash (normalde flush edilmiş olmalı)
    }
    redis.Del(keys...)
}
```

### Trade-offs

| Durum | Latency | Güvenilirlik |
|-------|---------|--------------|
| Redis up | ~10ms | Buffer + async flush |
| Redis down (read) | ~50-100ms | DB fallback |
| Redis down (write) | ~50-100ms | Direkt DB write (sync) |

### Race Condition Notu (Fallback + Background Worker)

**Senaryo**: Redis kısmen erişilebilir durumda. Öğrenci A scan yapar:
1. Redis write başarısız → Direkt DB'ye yazılır (timestamp: T1)
2. Aynı anda Redis recovery olur, eski buffer'daki veri flush edilir
3. Aynı öğrenci için iki write attempt

**Davranış**: `ON CONFLICT (session_id, student_id) DO NOTHING` sayesinde ilk yazılan kayıt korunur. Timestamp farkı oluşabilir (T1 vs buffer'daki T0) ancak yoklama alındı/alınmadı binary sonucu değişmez.

**Kabul Edilen Trade-off**: Yoklama sisteminde önemli olan öğrencinin var/yok durumudur. Milisaniye seviyesinde timestamp farkı operasyonel açıdan önemsizdir.

### Notlar
- Fallback modunda duplicate check DB'ye gider, latency artar ama sistem çalışmaya devam eder
- Cache warming async yapılır, ilk request yavaş olabilir
- Redis recovery sonrası otomatik olarak cache dolmaya başlar

---

## API Endpoints

### 🔒 POST /api/v1/attendance/sessions
Yoklama oturumu başlatma

**Role Requirement**: Teacher (course instructor)

**Request**:
```json
{
  "course_id": "uuid",
  "week_number": 5,
  "duration_minutes": 15
}
```

**Response** (201):
```json
{
  "session_id": "uuid",
  "course_id": "uuid",
  "course_code": "CS101",
  "course_name": "Introduction to Computer Science",
  "week_number": 5,
  "session_date": "2025-11-15",
  "qr_rotation_interval": 15,
  "started_at": "2025-11-15T10:00:00Z",
  "expires_at": "2025-11-15T10:15:00Z",
  "enrolled_student_count": 150
}
```

**Business Logic**:
1. **Authorization check**: `courses_cache.instructor_id == JWT.user_id`
2. **Duplicate check**: Bu hafta için session var mı?
   - ❌ Varsa: `409 SESSION_ALREADY_EXISTS`
3. **Generate qr_secret**: `crypto.randomBytes(32).toString('hex')`
4. **Create session** in PostgreSQL
5. **Warm Redis cache**:
   - Session hash
   - Enrolled students set (from `enrollments_cache`)
6. **Return** session info

---

### 🔒 GET /api/v1/attendance/sessions/:sessionId/qr
QR kod verisi alma (instructor için, frontend'de QR generate edilir)

**Role Requirement**: Teacher (session owner)

**Query Parameters**:
- `t`: Current timestamp (cache busting)

**Response** (200):
```json
{
  "session_id": "uuid",
  "qr_payload": {
    "sid": "uuid",
    "ts": 1699999999,
    "sig": "a1b2c3d4e5f6..."
  },
  "valid_until": "2025-11-15T10:00:15Z",
  "rotation_interval": 15
}
```

**Business Logic**:
1. **Authorization check**: Session owner mı?
2. **Active check**: Session aktif mi ve süresi dolmamış mı?
3. **Calculate current window**: `floor(now() / rotation_interval)`
4. **Generate signature**: `HMAC-SHA256(qr_secret, session_id + "|" + window)`
5. **Return** QR payload

**Frontend**:
- Bu payload'ı QR kod olarak render eder
- Her `rotation_interval` saniyede yeni QR ister
- Süre dolunca session otomatik kapanır

---

### 🔒 POST /api/v1/attendance/scan
QR kod tarama (öğrenci)

**Role Requirement**: Student

**Authentication**: 
- `student_id` JWT payload'dan alınır (Bearer token)
- Request body'de student bilgisi gönderilmez (güvenlik)

**Request**:
```json
{
  "qr_payload": {
    "sid": "uuid",
    "ts": 1699999999,
    "sig": "a1b2c3d4e5f6..."
  }
}
```

**Response** (200):
```json
{
  "message": "Yoklama başarıyla alındı",
  "course_code": "CS101",
  "course_name": "Introduction to Computer Science",
  "week_number": 5,
  "marked_at": "2025-11-15T10:05:30Z"
}
```

**Business Logic** (Optimized for < 50ms response):
```
1. Parse qr_payload
2. Get student_id from JWT payload
3. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
4. Redis/DB: Get session (with fallback)
   - ❌ Not found: 404 SESSION_NOT_FOUND
   - ❌ Expired: 400 SESSION_EXPIRED

5. Validate signature:
   - current_window = floor(qr_payload.ts / rotation_interval)
   - expected_sig = HMAC(qr_secret, sid + "|" + current_window)
   - Also check previous window (grace period)
   - ❌ Mismatch: 400 INVALID_QR_CODE

6. Check timestamp freshness (replay attack prevention):
   - max_age = rotation_interval * 3 (45 saniye)
   - ❌ If (current_time - qr_payload.ts) > max_age: 400 QR_EXPIRED

7. Enrollment check (with fallback):
   - Redis available: SISMEMBER attendance:session:{sid}:enrolled {student_id}
   - Redis down: DB'den kontrol (IsEnrolled fonksiyonu)
   - ❌ Not enrolled: 403 NOT_ENROLLED

8. Duplicate check (with fallback):
   - Redis available: SISMEMBER attendance:marked:{sid} {student_id}
   - Redis down: DB'den kontrol
   - ❌ Already marked: 409 ALREADY_MARKED
     Response: {"message": "Bu dersin yoklamasında zaten varsınız"}

9. Write attendance (with fallback):
   - Redis up: SADD marked + HSET buffer
   - Redis down: Direkt DB write

10. Return success (DB write happens async if Redis up)
```

**Background Worker** (her 5 saniye):
```sql
-- Redis buffer'dan PostgreSQL'e batch insert
INSERT INTO attendance_records (session_id, student_id, course_id, semester, week_number, is_present, marked_via, scanned_at, qr_timestamp)
SELECT 
    session_id,
    student_id,
    course_id,
    semester,
    week_number,
    TRUE,
    'qr_scan',
    to_timestamp(scanned_at),
    qr_timestamp
FROM unnest($1::attendance_buffer[])
ON CONFLICT (session_id, student_id) DO NOTHING;
```

---

### 🔒 POST /api/v1/attendance/sessions/:sessionId/manual
Manuel yoklama girişi (instructor)

**Role Requirement**: Teacher (session owner)

**Request**:
```json
{
  "student_id": "uuid",
  "is_present": true,
  "note": "Telefonu bozuk, manuel olarak eklendi"
}
```

**Response** (201):
```json
{
  "id": "uuid",
  "session_id": "uuid",
  "student_id": "uuid",
  "student_number": "2021123456",
  "student_name": "Ahmet Yılmaz",
  "is_present": true,
  "marked_via": "manual",
  "note": "Telefonu bozuk, manuel olarak eklendi",
  "marked_at": "2025-11-15T10:10:00Z"
}
```

**Business Logic**:
1. **Authorization check**: Session owner mı?
2. **Session active check**: Session aktif mi?
3. **Enrollment check**: Öğrenci bu derse kayıtlı mı?
4. **Duplicate check**: Zaten yoklama var mı?
   - ✅ Varsa güncelle (override with manual)
5. **Direct PostgreSQL insert** (no Redis buffer for manual)
6. **Update Redis marked set** (consistency):
   ```go
   // Redis'e yazmayı dene, başarısız olursa ignore et
   // (DB zaten source of truth, Redis sadece optimization)
   err := redis.SAdd("attendance:marked:"+sessionID, studentID)
   if err != nil {
       // Log warning but don't fail the request
       log.Warn("Failed to update Redis marked set", "error", err)
   }
   ```
7. **Invalidate student summary cache**:
   ```go
   redis.Del("attendance:student:" + studentID + ":summary:" + semester)
   ```

---

### 🔒 POST /api/v1/attendance/sessions/:sessionId/close
Session'ı kapatma ve devamsızları işaretleme

**Role Requirement**: Teacher (session owner)

**Description**: Session'ı kapatır, buffer'ı flush eder ve yoklamaya katılmamış tüm öğrencileri otomatik "yok" olarak işaretler. **Toplu yok yazma işlemi SADECE bu endpoint üzerinden yapılır.**

**Idempotency Note**: Bu endpoint ve Session Expiry Handler aynı işlemleri yapar. Her ikisi de `is_active = TRUE` kontrolü yapar, dolayısıyla:
- Instructor `/close` çağırdıktan sonra Expiry Handler çalışırsa → `is_active = FALSE` olduğu için skip eder
- Expiry Handler çalıştıktan sonra instructor `/close` çağırırsa → `is_active = FALSE` olduğu için 400 SESSION_NOT_ACTIVE döner

**Response** (200):
```json
{
  "session_id": "uuid",
  "closed_at": "2025-11-15T10:15:00Z",
  "summary": {
    "total_enrolled": 150,
    "present_count": 142,
    "absent_count": 8
  },
  "newly_marked_absent": [
    {"student_id": "uuid", "student_number": "2021123456", "student_name": "Ahmet Yılmaz"},
    {"student_id": "uuid", "student_number": "2021123457", "student_name": "Ayşe Demir"}
  ]
}
```

**Business Logic**:
1. **Authorization check**: Session owner mı?
2. **Active check**: `is_active = TRUE` mi?
   - ❌ Değilse: 400 SESSION_NOT_ACTIVE (zaten kapatılmış)
3. **Flush Redis buffer** to PostgreSQL (immediate)
4. **Get all enrolled students** from `enrollments_cache`
5. **Get marked students** from `attendance_records` WHERE session_id
6. **Calculate absent** = enrolled - marked
7. **Batch insert absent records**:
   ```sql
   INSERT INTO attendance_records (session_id, student_id, course_id, semester, week_number, is_present, marked_via, manually_marked_by, manually_marked_at)
   SELECT 
       $1, -- session_id
       student_id,
       $2, -- course_id
       $3, -- semester
       $4, -- week_number
       FALSE, -- is_present
       'manual',
       $5, -- instructor_id
       NOW()
   FROM unnest($6::uuid[]) AS student_id
   ON CONFLICT (session_id, student_id) DO NOTHING;
   ```
8. **Set session inactive**: `is_active = FALSE`
9. **Clear all Redis keys** for this session (ClearSessionRedisKeys)
10. **Invalidate student attendance caches** for all affected students

---

### 🔒 GET /api/v1/attendance/courses/:courseId/sessions
Dersin tüm yoklama oturumları

**Role Requirement**: Teacher (course instructor)

**Query Parameters**:
- `semester` (required): "2025_spring"

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "course_name": "Introduction to Computer Science",
  "semester": "2025_spring",
  "total_weeks": 14,
  "sessions": [
    {
      "session_id": "uuid",
      "week_number": 1,
      "session_date": "2025-09-15",
      "present_count": 145,
      "absent_count": 5,
      "is_active": false
    },
    {
      "session_id": "uuid",
      "week_number": 2,
      "session_date": "2025-09-22",
      "present_count": 142,
      "absent_count": 8,
      "is_active": false
    },
    {
      "week_number": 3,
      "session_id": null,
      "status": "not_created"
    }
  ],
  "overall_stats": {
    "completed_sessions": 2
  }
}
```

---

### 🔒 GET /api/v1/attendance/courses/:courseId/students
Derse kayıtlı öğrencilerin yoklama durumu

**Role Requirement**: Teacher (course instructor)

**Query Parameters**:
- `semester` (required): "2025_spring"
- `week` (optional): Specific week filter

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "semester": "2025_spring",
  "total_weeks": 14,
  "completed_weeks": 10,
  "students": [
    {
      "student_id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "present_count": 9,
      "absent_count": 1,
      "absent_weeks": [3]
    },
    {
      "student_id": "uuid",
      "student_number": "2021123457",
      "first_name": "Ayşe",
      "last_name": "Demir",
      "present_count": 7,
      "absent_count": 3,
      "absent_weeks": [2, 5, 9]
    }
  ]
}
```

**Frontend Status Calculation** (Devamsızlık Hakkı: 3):
- `absent_count <= 2` → Passing (yeşil)
- `absent_count == 3` → At Risk (sarı, son hak kullanıldı)
- `absent_count >= 4` → Failing (kırmızı, dönem sonu FF alır)

---

### 🔒 GET /api/v1/attendance/my
Öğrencinin kendi yoklama kayıtları

**Role Requirement**: Student

**Query Parameters**:
- `semester` (optional): Filter by semester
- `course_id` (optional): Filter by course

**Business Logic**:
1. Get student_id from JWT payload
2. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Query attendance records (with Redis caching)

**Response** (200):
```json
{
  "student_id": "uuid",
  "student_number": "2021123456",
  "semester": "2025_spring",
  "courses": [
    {
      "course_id": "uuid",
      "course_code": "CS101",
      "course_name": "Introduction to Computer Science",
      "instructor": "Prof. Ayşe Demir",
      "total_weeks": 14,
      "completed_weeks": 10,
      "present_count": 9,
      "absent_count": 1,
      "absent_weeks": [3],
      "weekly_records": [
        {"week": 1, "date": "2025-09-15", "is_present": true, "marked_via": "qr_scan"},
        {"week": 2, "date": "2025-09-22", "is_present": true, "marked_via": "qr_scan"},
        {"week": 3, "date": "2025-09-29", "is_present": false, "marked_via": "manual", "note": null}
      ]
    },
    {
      "course_id": "uuid",
      "course_code": "MATH201",
      "course_name": "Linear Algebra",
      "instructor": "Prof. Mehmet Öz",
      "total_weeks": 14,
      "completed_weeks": 10,
      "present_count": 7,
      "absent_count": 3,
      "absent_weeks": [2, 5, 9],
      "weekly_records": [
        {"week": 1, "date": "2025-09-16", "is_present": true, "marked_via": "qr_scan"},
        {"week": 2, "date": "2025-09-23", "is_present": false, "marked_via": "manual", "note": null}
      ]
    }
  ]
}
```

**Frontend Status Calculation** (Devamsızlık Hakkı: 3):
- `remaining_absences = 3 - absent_count` (frontend hesaplar)
- `absent_count <= 2` → Passing
- `absent_count == 3` → At Risk (0 hak kaldı)
- `absent_count >= 4` → Failing

**Redis Caching**:
- Key: `attendance:student:{student_id}:summary:{semester}`
- TTL: 1 saat
- Invalidation: On attendance record insert/update

---

### 🔒 POST /api/v1/attendance/courses/:courseId/finalize
Dönem sonu devamsızlık finalizasyonu

**Role Requirement**: Teacher (course instructor)

**Description**: Dönem sonunda çağrılır. 4 veya daha fazla devamsızlığı olan öğrenciler için `attendance.semester.failed` event'i yayınlar.

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "semester": "2025_spring",
  "total_students": 150,
  "total_weeks": 14,
  "finalization_summary": {
    "passing_count": 142,
    "failing_count": 8,
    "max_allowed_absences": 3
  },
  "failed_students": [
    {
      "student_id": "uuid",
      "student_number": "2021123456",
      "student_name": "Ahmet Yılmaz",
      "present_count": 10,
      "absent_count": 4
    }
  ],
  "events_published": 8,
  "finalized_at": "2025-01-15T10:00:00Z"
}
```

**RabbitMQ Event Published**: Her devamsız öğrenci için `attendance.semester.failed`

**Business Logic**:
1. **Authorization check**: Course instructor mı?
2. **Completion check**: Tüm haftalar için session var mı?
   - ⚠️ Warning if some weeks missing (continue anyway)
3. **Calculate attendance** for each student
4. **Filter failing students**: absent_count >= 4
5. **Publish events** via Outbox Pattern:
   ```sql
   INSERT INTO outbox_events (event_type, routing_key, payload)
   SELECT 
       'attendance.semester.failed',
       'attendance.semester.failed',
       jsonb_build_object(
           'student_id', student_id,
           'course_id', course_id,
           'course_code', course_code,
           'semester', semester,
           'present_count', present_count,
           'absent_count', absent_count,
           'max_allowed_absences', 3
       )
   FROM failing_students;
   ```
6. **Return summary**

**Event Consumers**: 
- Grades Service (öğrenciye otomatik FF notu verir)
- Notification Service (öğrenciye bildirim)

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Subscribed Exchanges:
- "student.events" (routing keys: student.created, student.updated, student.deactivated)
- "course.events" (routing keys: course.semester.created, course.semester.updated, course.semester.deleted)
- "enrollment.events" (routing keys: enrollment.program_approved)

Publishing Exchange:
- "attendance.events" (type: topic)

Routing Keys (Publishing):
- attendance.semester.failed
```

### Event Schemas

#### Consumed: enrollment.program_approved
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

**Consumer Action**:
```sql
INSERT INTO enrollments_cache (student_id, course_id, semester)
SELECT $1, course_id, $2
FROM unnest($3::uuid[]) AS course_id
ON CONFLICT (student_id, course_id) DO NOTHING;
```

#### Consumed: course.semester.deleted
```json
{
  "event_type": "course.semester.deleted",
  "timestamp": "2025-11-11T11:00:00Z",
  "data": {
    "course_id": "uuid",
    "course_code": "CS101",
    "semester": "2025_spring"
  }
}
```

**Consumer Action**:
```sql
-- CASCADE ile enrollments_cache, attendance_sessions, attendance_records otomatik silinir
DELETE FROM courses_cache WHERE id = $1;
```

**Not**: Bu işlem geri alınamaz ve tüm geçmiş yoklama kayıtlarını siler. Production'da dikkatli kullanılmalı.

---

#### Published: attendance.semester.failed
Published when: Dönem sonu finalizasyonunda devamsızlık sayısı >= 4

```json
{
  "event_id": "uuid",
  "event_type": "attendance.semester.failed",
  "timestamp": "2025-01-15T10:00:00Z",
  "data": {
    "student_id": "uuid",
    "student_number": "2021123456",
    "student_email": "ahmet.yilmaz@university.edu.tr",
    "course_id": "uuid",
    "course_code": "CS101",
    "course_name": "Introduction to Computer Science",
    "semester": "2025_spring",
    "total_weeks": 14,
    "present_count": 10,
    "absent_count": 4,
    "max_allowed_absences": 3
  }
}
```

**Event Consumers**:
- **Grades Service**: Öğrenciye otomatik olarak `grade_point = "0.00"` (FF) ve `is_attendance_failed = true` işaretler
- **Notification Service**: Öğrenciye devamsızlıktan kaldığı bildirimi gönderir

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_QR_CODE | QR kod imzası geçersiz |
| 400 | QR_EXPIRED | QR kod süresi dolmuş (timestamp çok eski veya window dışında) |
| 400 | SESSION_EXPIRED | Yoklama oturumu süresi dolmuş |
| 400 | SESSION_NOT_ACTIVE | Yoklama oturumu aktif değil (zaten kapatılmış) |
| 400 | INVALID_WEEK_NUMBER | Hafta numarası 1-14 aralığında değil |
| 403 | NOT_ENROLLED | Öğrenci bu derse kayıtlı değil |
| 403 | STUDENT_DEACTIVATED | Öğrenci deaktif edilmiş (is_active = false) |
| 403 | FORBIDDEN | Yetkisiz erişim (başkasının dersi/yoklaması) |
| 404 | SESSION_NOT_FOUND | Yoklama oturumu bulunamadı |
| 404 | COURSE_NOT_FOUND | Ders bulunamadı |
| 404 | STUDENT_NOT_FOUND | Öğrenci bulunamadı |
| 409 | SESSION_ALREADY_EXISTS | Bu hafta için zaten yoklama oturumu var |
| 409 | ALREADY_MARKED | Bu dersin yoklamasında zaten varsınız |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Background Workers

### 1. Redis Buffer Flusher
```
Schedule: Her 5 saniye
Purpose: Redis attendance buffer'ını PostgreSQL'e batch insert
```

**Logic**:
```go
func FlushAttendanceBuffer() {
    // 1. Get all active session IDs
    sessionKeys := redis.Keys("attendance:buffer:*")
    
    for _, key := range sessionKeys {
        sessionID := extractSessionID(key)
        
        // 2. Get all buffered records atomically
        records := redis.HGetAll(key)
        if len(records) == 0 {
            continue
        }
        
        // 3. Batch insert to PostgreSQL
        tx := db.Begin()
        processedStudents := []string{}
        
        for studentID, data := range records {
            parsed := parseBufferData(data) // "timestamp|qr_ts|marked_via"
            err := tx.Exec(`
                INSERT INTO attendance_records (...)
                VALUES (...)
                ON CONFLICT (session_id, student_id) DO NOTHING
            `, sessionID, studentID, parsed...)
            
            if err == nil {
                processedStudents = append(processedStudents, studentID)
            }
        }
        tx.Commit()
        
        // 4. Remove ONLY processed records from buffer (field bazında silme)
        // Bu sayede flush sırasında gelen yeni kayıtlar kaybolmaz
        if len(processedStudents) > 0 {
            redis.HDel(key, processedStudents...)
        }
        
        // Not: attendance:marked set'i silinmez, session aktif olduğu sürece kalır
        // Session kapandığında ClearSessionRedisKeys ile temizlenir
    }
}
```

### 2. Session Expiry Handler
```
Schedule: Her 1 dakika
Purpose: Süresi dolan session'ları kapat ve yok yazılanları işaretle
```

**Idempotency Note**: Bu handler ve `/close` endpoint'i aynı işlemleri yapar. `is_active = TRUE` kontrolü sayesinde bir session sadece bir kez kapatılır.

**Logic**:
```go
func HandleExpiredSessions() {
    // 1. Find expired but still active sessions
    expiredSessions := db.Query(`
        SELECT id, course_id, instructor_id, semester, week_number
        FROM attendance_sessions
        WHERE is_active = TRUE AND expires_at < NOW()
    `)
    
    for _, session := range expiredSessions {
        // 2. Flush any remaining buffer
        FlushSessionBuffer(session.ID)
        
        // 3. Mark absent students (same logic as /close endpoint)
        MarkAbsentStudents(session)
        
        // 4. Deactivate session
        db.Exec(`UPDATE attendance_sessions SET is_active = FALSE WHERE id = $1`, session.ID)
        
        // 5. Clear all Redis keys for this session
        ClearSessionRedisKeys(session.ID)
    }
}
```

### 3. Outbox Publisher
```
Schedule: Her 1 saniye
Purpose: Outbox tablosundaki pending eventleri RabbitMQ'ya publish et
```

**Logic**:
```go
func PublishOutboxEvents() {
    events := db.Query(`
        SELECT id, event_type, routing_key, payload
        FROM outbox
        WHERE status = 'pending'
        ORDER BY created_at
        LIMIT 100
    `)
    
    for _, event := range events {
        err := rabbitmq.Publish(event.RoutingKey, event.Payload)
        if err != nil {
            db.Exec(`
                UPDATE outbox SET retry_count = retry_count + 1, error_message = $1
                WHERE id = $2
            `, err.Error(), event.ID)
        } else {
            db.Exec(`
                UPDATE outbox SET status = 'processed', processed_at = NOW()
                WHERE id = $1
            `, event.ID)
        }
    }
}
```

---

## Performance Optimizations

### 1. QR Scan Path (< 50ms target)

| Step | Operation | Expected Time |
|------|-----------|---------------|
| 1 | Parse QR payload | < 1ms |
| 2 | Redis: Get session (with fallback) | < 2ms (Redis) / ~50ms (DB) |
| 3 | HMAC verification | < 1ms |
| 4 | Timestamp freshness check | < 1ms |
| 5 | Redis: SISMEMBER enrolled (with fallback) | < 1ms (Redis) / ~10ms (DB) |
| 6 | Redis: SISMEMBER marked (with fallback) | < 1ms (Redis) / ~10ms (DB) |
| 7 | Redis: SADD + HSET (pipeline) | < 2ms |
| 8 | Response serialization | < 1ms |
| **Total (Redis up)** | | **< 10ms** |
| **Total (Redis down)** | | **< 100ms** |

### 2. Redis Memory Estimation

```
Per active session:
- Session hash: ~500 bytes
- Enrolled set: 150 students × 36 bytes = ~5.4 KB
- Marked set: 150 students × 36 bytes = ~5.4 KB
- Buffer hash: 150 students × 100 bytes = ~15 KB

Total per session: ~26 KB

Peak concurrent sessions: ~500 (across all courses)
Peak Redis memory: 500 × 26 KB = ~13 MB

Student summary cache: 60,000 × 200 bytes = ~12 MB

Total Redis memory: ~25 MB (comfortable)
```

### 3. PostgreSQL Optimization

```sql
-- Partial index for active sessions (most queries)
CREATE INDEX idx_sessions_active_partial 
ON attendance_sessions(course_id, expires_at) 
WHERE is_active = TRUE;

-- Covering index for student attendance lookup
CREATE INDEX idx_records_student_covering 
ON attendance_records(student_id, course_id) 
INCLUDE (week_number, is_present);

-- BRIN index for time-based queries (large table)
CREATE INDEX idx_records_created_brin 
ON attendance_records USING BRIN(created_at);
```

---

## Related Services

- **Student Service**: Event consumer (`student.created`, `student.updated`)
- **Course Catalog Service**: Event consumer (`course.semester.created`, `course.semester.updated`, `course.semester.deleted`)
- **Enrollment Service**: Event consumer (`enrollment.program_approved`)
- **Grades Service**: Event consumer (`attendance.semester.failed`) - Devamsızlıktan kalan öğrenciye FF notu verir
- **Notification Service**: Event consumer (`attendance.semester.failed`) - Öğrenciye bildirim - ⏳ *Daha sonra eklenecek*

---

## Frontend Flow

### Instructor Flow (Yoklama Başlatma)

```
┌─────────────────────────────────────────────────────────────────┐
│                    /instructor/attendance                        │
│                                                                  │
│   1. Ders seçimi (dropdown)                                     │
│   2. Hafta seçimi (1-14)                                        │
│   3. [Yoklama Başlat] butonu                                    │
│                                                                  │
│   POST /api/v1/attendance/sessions                              │
│                    ↓                                             │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │              QR Kod Ekranı (Fullscreen)                 │   │
│   │                                                          │   │
│   │          ┌─────────────────────┐                        │   │
│   │          │                     │                        │   │
│   │          │      [QR CODE]      │  ← Her 15 sn yenilenir │   │
│   │          │                     │                        │   │
│   │          └─────────────────────┘                        │   │
│   │                                                          │   │
│   │   Kalan süre: 12:45                                     │   │
│   │   Katılan: 87 / 150                                     │   │
│   │                                                          │   │
│   │   [Manuel Ekle]  [Kapat]                                │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Student Flow (QR Tarama)

```
┌─────────────────────────────────────────────────────────────────┐
│                    /student/attendance                           │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │              Kamera Görünümü                             │   │
│   │                                                          │   │
│   │          [  Camera Preview  ]                           │   │
│   │                                                          │   │
│   │          QR kodu kameraya gösterin                      │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   POST /api/v1/attendance/scan                                  │
│                    ↓                                             │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    ✅ Başarılı                           │   │
│   │                                                          │   │
│   │   CS101 - Introduction to Computer Science              │   │
│   │   5. Hafta yoklaması alındı                             │   │
│   │                                                          │   │
│   │   Saat: 10:05:30                                        │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                    ⚠️ Zaten Alındı                       │   │
│   │                                                          │   │
│   │   Bu dersin yoklamasında zaten varsınız.                │   │
│   │                                                          │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Student Flow (Yoklama Geçmişi)

```
┌─────────────────────────────────────────────────────────────────┐
│                    /student/attendance/history                   │
│                                                                  │
│   GET /api/v1/attendance/my?semester=2025_spring                │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ CS101 - Introduction to Computer Science                │   │
│   │ Devam: 9/10 ✅ (2 hak kaldı)                            │   │
│   │                                                          │   │
│   │ Hafta  Tarih       Durum                                │   │
│   │ 1      15.09.2025  ✅ Var                               │   │
│   │ 2      22.09.2025  ✅ Var                               │   │
│   │ 3      29.09.2025  ❌ Yok                               │   │
│   │ 4      06.10.2025  ✅ Var                               │   │
│   │ ...                                                      │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ MATH201 - Linear Algebra                                │   │
│   │ Devam: 6/10 ⚠️ (0 hak kaldı - dikkat!)                 │   │
│   │ ...                                                      │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

Frontend Status Logic:
- remaining = 3 - absent_count
- if (absent_count <= 2) → ✅ Yeşil, "(remaining) hak kaldı"
- if (absent_count == 3) → ⚠️ Sarı, "0 hak kaldı - dikkat!"
- if (absent_count >= 4) → ❌ Kırmızı, "Devamsızlıktan kaldınız"
```

---

**Version**: 2.3.0
**Last Updated**: 2025-12-04

**Changes in v2.3.0**:
- ✅ API response'lardan `status` field'ları kaldırıldı (frontend hesaplayacak)
- ✅ Redis key pattern tutarlılığı sağlandı (`attendance:student:{id}:summary:{semester}`)
- ✅ `course.semester.created` event'inden eski dönem silme logic'i kaldırıldı
- ✅ Enrollment check için DB fallback eklendi (IsEnrolled fonksiyonu)
- ✅ Event payload'dan `attendance_rate` field'ı kaldırıldı
- ✅ Week number constraint 1-14 olarak sabitlendi (esneklik kaldırıldı)
- ✅ Buffer flusher race condition düzeltildi (field bazında silme)
- ✅ Session close vs expiry handler idempotency belgelendi
- ✅ Manual attendance Redis consistency implementation detaylandırıldı

**Changes in v2.2.0**:
- ✅ Yüzdelik devamsızlık sistemi kaldırıldı
- ✅ Devamsızlık hakkı sistemi eklendi (max 3 devamsızlık)
- ✅ Status threshold'lar güncellendi: passing (<=2), at_risk (==3), failing (>=4)
- ✅ `remaining_absences` field'ı eklendi (öğrenci kaç devamsızlık hakkı kaldığını görebilir)
- ✅ Event payload'lar güncellendi (`max_allowed_absences: 3`)

**Changes in v2.1.0**:
- ✅ Week number constraint açıklaması eklendi (14 hafta + 2 esneklik)
- ✅ QR timestamp freshness kontrolü eklendi (replay attack prevention)
- ✅ `/bulk-absent` endpoint kaldırıldı, logic `/close`'a taşındı
- ✅ Student ID JWT'den alındığı belgelendi
- ✅ Duplicate check fallback açıklaması eklendi
- ✅ `ClearSessionRedisKeys` fonksiyon tanımı eklendi
- ✅ Race condition (fallback + worker) trade-off belgelendi
- ✅ `course.semester.deleted` event handler açıklaması eklendi
- ✅ Error message'lar Türkçeleştirildi (ALREADY_MARKED)

**Changes in v2.0.0**:
- ✅ Complete architecture redesign for high-write workload
- ✅ Redis buffering for QR scan path (< 50ms response time)
- ✅ HMAC-based QR code security with time-window rotation
- ✅ Background workers for buffer flushing and session expiry
- ✅ Outbox pattern for guaranteed event delivery
- ✅ Simplified schema (removed partitioning - not needed for this data volume)
- ✅ Week-based attendance tracking (14 weeks per semester)
- ✅ Manual attendance entry for edge cases
- ✅ Student attendance history view
- ✅ Performance optimizations and memory estimation
- 🎯 Pattern: Write-heavy optimization with async persistence
- 📊 Benefit: Handles 300,000 concurrent QR scans with sub-50ms response