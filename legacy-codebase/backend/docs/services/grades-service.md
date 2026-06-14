# Grades Service v7.2.0

## Sorumluluk
Hoca not girişi, öğrenci not görüntüleme, transcript oluşturma, GPA hesaplama

**Read-Heavy Service**: Redis caching zorunlu

**Source of Truth**: Bu servis not verileri için canonical data source'dur.

---

## Mimari Genel Bakış

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         ACTIVE PHASE (Dönem İçi)                                │
│                                                                                  │
│  Hoca notları girer: midterm, quiz, ödev, vb.                                   │
│                                                                                  │
│  ┌─────────────────────────┐      ┌──────────────────────────────┐              │
│  │ student_course_         │ 1:N  │ student_assessment_scores    │              │
│  │ registrations           │─────►│ midterm: 75, quiz: 80, ...   │              │
│  └─────────────────────────┘      └──────────────────────────────┘              │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Son öğrencinin FINAL notu girildi
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        OTOMATİK HESAPLAMA                                        │
│                                                                                  │
│  1. Tüm öğrencilerin weighted average hesapla                                   │
│  2. Sınıf ortalamasını hesapla                                                  │
│                                                                                  │
│         Sınıf Ortalaması >= 60          │        Sınıf Ortalaması < 60          │
│                  │                      │                  │                     │
│                  ▼                      │                  ▼                     │
│         ABSOLUTE (Bağıl)                │         RELATIVE (Z-Score)            │
│         Sabit puan aralıkları           │         Çan eğrisi                    │
│         90+ → 4.00                      │         z >= 2.0 → 4.00               │
│         85+ → 3.50                      │         z >= 1.5 → 3.50               │
│         ...                             │         ...                            │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                       FINALIZED (Otomatik)                                       │
│                                                                                  │
│  → student_completed_courses'a INSERT                                           │
│  → Önkoşul dersleri GEÇENLER için event yayınla                                │
│  → Operasyonel tabloları temizle                                                │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## İletişim

### Inbound (RabbitMQ)
| Event | Açıklama |
|-------|----------|
| `student.created` | Öğrenci cache'e eklenir |
| `student.updated` | Öğrenci güncellenir |
| `student.deactivated` | `is_active = false` |
| `course.semester.created` | Ders cache'e eklenir (assessment_schema dahil) |
| `course.semester.updated` | Ders güncellenir |
| `course.semester.deleted` | DELETE from `courses_cache` (CASCADE to registrations) |
| `course.instructor.changed` | Dersin hocası değiştiğinde `instructor_id`, `instructor_fullname` güncellenir |
| `course.prerequisites.updated` | Önkoşul listesi güncellenir (TRUNCATE + INSERT) |

### Consumed Event Details

#### course.semester.deleted
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

**Consumer Action**:
```sql
-- CASCADE ile student_course_registrations ve student_assessment_scores otomatik silinir
DELETE FROM courses_cache WHERE id = $1;
```

**Note**: `student_course_registrations` tablosunda `REFERENCES courses_cache(id) ON DELETE CASCADE` constraint var. Ders silindiğinde ilgili kayıtlar otomatik temizlenir.

#### course.prerequisites.updated
```json
{
  "event_id": "uuid",
  "event_type": "course.prerequisites.updated",
  "timestamp": "2025-11-23T10:00:00Z",
  "data": {
    "prerequisite_courses": [
      {"course_code": "CS100", "course_id": "uuid-cs100"},
      {"course_code": "CS101", "course_id": "uuid-cs101"},
      {"course_code": "MATH101", "course_id": "uuid-math101"}
    ],
    "updated_at": "2025-11-23T10:00:00Z"
  }
}
```

**Consumer Action**:
```sql
-- Full sync: TRUNCATE + INSERT (idempotent)
BEGIN;

TRUNCATE TABLE prerequisite_courses_cache;

INSERT INTO prerequisite_courses_cache (course_code, course_id, synced_at)
SELECT
    (prereq->>'course_code')::varchar,
    (prereq->>'course_id')::uuid,
    NOW()
FROM jsonb_array_elements($1::jsonb) AS prereq;

COMMIT;
```

**Note**: Bu tablo `isPrerequisiteCourse()` fonksiyonunda kullanılır. Finalize sırasında geçen öğrenciler için `grade.student.prerequisite.passed` event'i yayınlanır.

#### enrollment.program_approved
Danışman onayı sonrası öğrenci-ders kaydı oluşturulur.

**Consumer Action**:
```sql
INSERT INTO student_course_registrations (student_id, course_id, semester)
SELECT $student_id, course_id, $semester
FROM unnest($course_ids::uuid[]) AS course_id
ON CONFLICT (student_id, course_id) DO NOTHING;
```

#### attendance.semester.failed
Devamsızlıktan kalan öğrenci işaretlenir.

**Consumer Action**:
```sql
UPDATE student_course_registrations
SET is_attendance_failed = true
WHERE student_id = $1 AND course_id = $2;
```

### Outbound (RabbitMQ)
| Event | Tetiklenme |
|-------|------------|
| `grade.submitted` | Her not girişinde |
| `grade.finalized` | Otomatik hesaplama sonrası |
| `grade.student.prerequisite.passed` | Önkoşul dersini GEÇEN öğrenciler için (Enrollment Service'e bildirim) |

---

## Database Schema

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ENUM Tanımları
CREATE TYPE grading_type_enum AS ENUM ('absolute', 'relative');
CREATE TYPE grade_point_enum AS ENUM (
    '4.00', '3.75', '3.50', '3.25', '3.00',
    '2.75', '2.50', '2.25', '2.00', '1.75',
    '1.50', '1.25', '1.00', '0.50', '0.00'
);

-- ==========================================
-- A. CACHE TABLOLARI
-- ==========================================

CREATE TABLE students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    department VARCHAR(100),
    class_level SMALLINT,
    is_active BOOLEAN DEFAULT TRUE,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_cache_number ON students_cache(student_number);
CREATE INDEX idx_students_cache_active ON students_cache(is_active) WHERE is_active = true;

CREATE TABLE courses_cache (
    id UUID PRIMARY KEY,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,
    department VARCHAR(100),
    instructor_id UUID NOT NULL,
    instructor_fullname VARCHAR(150),
    assessment_schema JSONB NOT NULL,
    -- Örnek: [
    --   {"slug": "midterm", "name": "Vize", "weight": 40},
    --   {"slug": "final", "name": "Final", "weight": 60}
    -- ]
    synced_at TIMESTAMP DEFAULT NOW()
);
-- Note: Hard delete kullanılıyor (is_active yok). Silinen derse hiçbir öğrenci kayıtlı olamaz
-- çünkü öğrenciler sadece enrollment.program_approved event'i ile gelir ve
-- silinmiş bir ders için approved enrollment olamaz.

CREATE INDEX idx_courses_cache_semester ON courses_cache(semester);
CREATE INDEX idx_courses_cache_instructor ON courses_cache(instructor_id);

CREATE TABLE prerequisite_courses_cache (
    course_code VARCHAR(50) NOT NULL,
    course_id UUID NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (course_code, course_id)
);

-- ==========================================
-- B. OPERASYONEL TABLOLAR (Dönem İçi)
-- ==========================================

CREATE TABLE student_course_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    is_attendance_failed BOOLEAN DEFAULT FALSE,  -- Devamsızlıktan kalma durumu (attendance.semester.failed event'i ile set edilir)
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_registrations_student ON student_course_registrations(student_id);
CREATE INDEX idx_registrations_course ON student_course_registrations(course_id);

CREATE TABLE student_assessment_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_id UUID NOT NULL REFERENCES student_course_registrations(id) ON DELETE CASCADE,
    slug VARCHAR(50) NOT NULL,
    score DECIMAL(5,2) CHECK (score >= 0 AND score <= 100),
    is_absent BOOLEAN DEFAULT FALSE,
    graded_by UUID NOT NULL,
    graded_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(registration_id, slug)
);

CREATE INDEX idx_scores_registration ON student_assessment_scores(registration_id);

-- ==========================================
-- C. TAMAMLANMIŞ DERSLER (Kalıcı)
-- ==========================================

CREATE TABLE student_completed_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Öğrenci Snapshot
    student_id UUID NOT NULL,
    student_number VARCHAR(50) NOT NULL,
    student_first_name VARCHAR(100) NOT NULL,
    student_last_name VARCHAR(100) NOT NULL,
    student_department VARCHAR(100),

    -- Ders Snapshot
    course_id UUID NOT NULL,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,

    -- Hoca Snapshot
    instructor_id UUID NOT NULL,
    instructor_name VARCHAR(150) NOT NULL,

    -- Notlar
    assessment_scores JSONB NOT NULL,

    -- Hesaplanan Sonuç
    weighted_average DECIMAL(5,2) NOT NULL,
    grade_point grade_point_enum NOT NULL,

    -- Notlandırma Bilgisi
    grading_type grading_type_enum NOT NULL,  -- 'absolute' veya 'relative'
    grading_config JSONB,
    -- relative için: {"class_mean": 55.2, "class_stddev": 12.3, "student_z_score": 0.85}

    class_statistics JSONB,
    -- {"total_students": 85, "mean": 55.2, "stddev": 12.3, "min": 20, "max": 92}

    -- Devamsızlık Bilgisi
    is_attendance_failed BOOLEAN DEFAULT FALSE,  -- Devamsızlıktan kalma durumu

    finalized_at TIMESTAMP NOT NULL,
    finalized_by UUID NOT NULL,

    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_completed_student ON student_completed_courses(student_id);
CREATE INDEX idx_completed_semester ON student_completed_courses(semester);
CREATE INDEX idx_completed_course_code ON student_completed_courses(course_code);
CREATE INDEX idx_completed_student_course_code ON student_completed_courses(student_id, course_code);

-- ==========================================
-- D. OUTBOX
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

## Notlandırma Sistemi

### Otomatik Grading Type Belirleme

```
Sınıf Ortalaması >= 60  →  ABSOLUTE (sabit aralıklar)
Sınıf Ortalaması < 60   →  RELATIVE (z-score / çan eğrisi)
```

### Grade Point Enum

```
4.00 → AA        2.00 → CC        0.50 → FD
3.75             1.75             0.00 → FF
3.50 → BA        1.50 → DC
3.25             1.25
3.00 → BB        1.00 → DD
2.75
2.50 → CB
2.25
```

### Geçme Kriteri
```go
func isPassing(gp grade_point_enum) bool {
    return gp >= "1.00"  // DD ve üstü
}
```

---

## Absolute Notlandırma (Ortalama >= 60)

```go
func calculateAbsoluteGradePoint(average float64) grade_point_enum {
    switch {
    case average >= 90.0:  return "4.00"  // AA
    case average >= 87.5:  return "3.75"
    case average >= 85.0:  return "3.50"  // BA
    case average >= 82.5:  return "3.25"
    case average >= 80.0:  return "3.00"  // BB
    case average >= 77.5:  return "2.75"
    case average >= 75.0:  return "2.50"  // CB
    case average >= 72.5:  return "2.25"
    case average >= 70.0:  return "2.00"  // CC
    case average >= 67.5:  return "1.75"
    case average >= 65.0:  return "1.50"  // DC
    case average >= 62.5:  return "1.25"
    case average >= 60.0:  return "1.00"  // DD
    case average >= 50.0:  return "0.50"  // FD
    default:               return "0.00"  // FF
    }
}
```

---

## Relative Notlandırma - Z-Score (Ortalama < 60)

```go
func calculateZScoreGradePoint(average, mean, stddev float64) grade_point_enum {
    // Standart sapma 0 ise herkes aynı notu almış
    if stddev == 0 {
        return "2.00"  // CC
    }
    
    zScore := (average - mean) / stddev
    
    switch {
    case zScore >= 2.00:   return "4.00"  // AA
    case zScore >= 1.75:   return "3.75"
    case zScore >= 1.50:   return "3.50"  // BA
    case zScore >= 1.25:   return "3.25"
    case zScore >= 1.00:   return "3.00"  // BB
    case zScore >= 0.75:   return "2.75"
    case zScore >= 0.50:   return "2.50"  // CB
    case zScore >= 0.25:   return "2.25"
    case zScore >= 0.00:   return "2.00"  // CC
    case zScore >= -0.25:  return "1.75"
    case zScore >= -0.50:  return "1.50"  // DC
    case zScore >= -0.75:  return "1.25"
    case zScore >= -1.00:  return "1.00"  // DD
    case zScore >= -1.50:  return "0.50"  // FD
    default:               return "0.00"  // FF
    }
}
```

---

## Core Logic

### Devamsızlık Kontrolü (Attendance Failure Handling)

**Önemli**: Devamsızlık bilgisi not girişinden önce hesaplanır. `attendance.semester.failed` eventi geldiğinde:

```go
// attendance.semester.failed event handler
func handleAttendanceSemesterFailed(event AttendanceSemesterFailedEvent) error {
    // student_course_registrations tablosunda is_attendance_failed = true yap
    return db.Exec(`
        UPDATE student_course_registrations
        SET is_attendance_failed = true
        WHERE student_id = $1 AND course_id = $2
    `, event.StudentID, event.CourseID)
}
```

**Business Rules**:
1. ❌ Devamsızlıktan kalan öğrenciye **manuel not girişi engellenir** (ATTENDANCE_FAILED hatası)
2. 🔢 Devamsızlıktan kalanların skoru otomatik olarak **0** girilir
3. 📊 Devamsızlıktan kalanlar **çan eğrisi hesabına dahil edilmez**
4. 📝 Final kayıtta `is_attendance_failed = true` ve `grade_point = '0.00'` (FF) olarak işaretlenir

### Not Girişi ve Otomatik Hesaplama

```go
func SubmitScore(courseID, registrationID uuid.UUID, slug string, score float64, isAbsent bool) error {
    // 1. Yetki kontrolü
    course := getCourse(courseID)
    if course.InstructorID != currentUserID {
        return ErrNotCourseInstructor
    }

    // 2. Slug geçerli mi?
    if !isValidSlug(course.AssessmentSchema, slug) {
        return ErrInvalidSlug
    }

    // 3. Devamsızlık kontrolü - devamsızlıktan kalanın notunu elle girme engelle
    registration := getRegistration(registrationID)
    if registration.IsAttendanceFailed {
        return ErrAttendanceFailed  // 403: Bu öğrenci devamsızlıktan kalmış, not girilemez
    }

    // 4. Notu kaydet
    saveScore(registrationID, slug, score, isAbsent)

    // 5. grade.submitted event
    publishGradeSubmitted(registrationID, slug, score)

    // 6. Final notu mu?
    if slug != "final" {
        return nil
    }

    // 7. Tüm final notları girildi mi?
    if !allFinalScoresComplete(courseID) {
        return nil
    }

    // 8. OTOMATİK HESAPLAMA
    return autoFinalize(courseID)
}

func allFinalScoresComplete(courseID uuid.UUID) bool {
    totalStudents := countRegistrations(courseID)
    finalGradedCount := countScoresBySlug(courseID, "final")
    return finalGradedCount >= totalStudents
}
```

### Otomatik Finalize

```go
func autoFinalize(courseID uuid.UUID) error {
    course := getCourse(courseID)

    // 1. Tüm öğrencileri al ve devamsızlık durumuna göre ayır
    allRegistrations := getRegistrations(courseID)

    var regularStudents []Student      // Devamsızlıktan KALMAYANLAR
    var attendanceFailedStudents []Student  // Devamsızlıktan KALANLAR

    for _, reg := range allRegistrations {
        if reg.IsAttendanceFailed {
            // Devamsızlıktan kalan: Otomatik skor 0, grade_point FF
            attendanceFailedStudents = append(attendanceFailedStudents, Student{
                ID:                 reg.StudentID,
                Number:             reg.StudentNumber,
                FirstName:          reg.StudentFirstName,
                LastName:           reg.StudentLastName,
                Department:         reg.StudentDepartment,
                Average:            0.0,  // Otomatik 0
                GradePoint:         "0.00",  // FF
                IsAttendanceFailed: true,
                Scores:             buildZeroScores(course.AssessmentSchema), // Tüm skorlar 0
            })
        } else {
            // Normal öğrenci: weighted average hesapla
            s := calculateStudentWeightedAverage(reg)
            regularStudents = append(regularStudents, s)
        }
    }

    // 2. Sınıf istatistikleri - SADECE devamsızlıktan KALMAYANLAR dahil
    stats := calculateClassStatistics(regularStudents)  // ⚠️ Çan hesabına devamsızlar dahil değil!

    // 3. Grading type belirle (OTOMATİK) - sadece regular students baz alınır
    var gradingType grading_type_enum
    if len(regularStudents) == 0 || stats.Mean >= 60 {
        gradingType = "absolute"
    } else {
        gradingType = "relative"
    }

    // 4. Her NORMAL öğrenci için grade_point hesapla
    for i := range regularStudents {
        s := &regularStudents[i]
        if gradingType == "absolute" {
            s.GradePoint = calculateAbsoluteGradePoint(s.Average)
        } else {
            s.ZScore = (s.Average - stats.Mean) / stats.StdDev
            s.GradePoint = calculateZScoreGradePoint(s.Average, stats.Mean, stats.StdDev)
        }
    }

    // 5. Tüm öğrencileri birleştir
    allStudents := append(regularStudents, attendanceFailedStudents...)

    // 6. Transaction
    tx := db.Begin()

    for _, s := range allStudents {
        // Tekrar alma: eski kaydı sil
        tx.Exec(`DELETE FROM student_completed_courses
                 WHERE student_id = ? AND course_code = ?`, s.ID, course.Code)

        // Yeni kayıt
        completed := StudentCompletedCourse{
            StudentID:           s.ID,
            StudentNumber:       s.Number,
            StudentFirstName:    s.FirstName,
            StudentLastName:     s.LastName,
            StudentDepartment:   s.Department,
            CourseID:            courseID,
            CourseCode:          course.Code,
            CourseName:          course.Name,
            Credits:             course.Credits,
            Semester:            course.Semester,
            InstructorID:        course.InstructorID,
            InstructorName:      course.InstructorName,
            AssessmentScores:    s.Scores,
            WeightedAverage:     s.Average,
            GradePoint:          s.GradePoint,
            GradingType:         gradingType,
            GradingConfig:       buildGradingConfig(gradingType, stats, s),
            ClassStatistics:     buildClassStats(stats),
            IsAttendanceFailed:  s.IsAttendanceFailed,  // ⚠️ Devamsızlık durumu
            FinalizedAt:         time.Now(),
            FinalizedBy:         course.InstructorID,  // Sistem otomatik, hoca adına
        }
        tx.Create(&completed)

        // Önkoşul dersi ise ve öğrenci GEÇTİYSE → Enrollment Service'e bildirim
        // Not: Sadece GEÇEN öğrenciler için event yayınlanır
        // Enrollment Service bu bilgiyi student_passed_prerequisites tablosunda saklar
        if isPrerequisiteCourse(course.Code) && isPassing(s.GradePoint) {
            tx.Create(&Outbox{
                EventType:  "grade.student.prerequisite.passed",
                RoutingKey: "grade.student.prerequisite.passed",
                Payload: PrerequisitePassedEvent{
                    StudentID:  s.ID,
                    CourseID:   courseID,
                    CourseCode: course.Code,
                    Semester:   course.Semester,
                    GradePoint: s.GradePoint,
                },
            })
        }
    }

    // 7. Temizlik
    tx.Exec(`DELETE FROM student_assessment_scores
             WHERE registration_id IN
             (SELECT id FROM student_course_registrations WHERE course_id = ?)`, courseID)
    tx.Exec(`DELETE FROM student_course_registrations WHERE course_id = ?`, courseID)

    // 8. grade.finalized event
    tx.Create(&Outbox{
        EventType:  "grade.finalized",
        RoutingKey: "grade.finalized",
        Payload: GradeFinalizedEvent{
            CourseID:               courseID,
            CourseCode:             course.Code,
            Semester:               course.Semester,
            GradingType:            gradingType,
            TotalStudents:          len(allStudents),
            PassingCount:           countPassing(allStudents),
            FailingCount:           countFailing(allStudents),
            AttendanceFailedCount:  len(attendanceFailedStudents),  // ⚠️ Devamsızlıktan kalan sayısı
            ClassMean:              stats.Mean,
        },
    })

    tx.Commit()
    invalidateCache(courseID, allStudents)

    return nil
}
```

---

## API Endpoints

### 🔒 GET /api/v1/grades/course/:courseId/status
Dersin not durumu

**Role**: Teacher (course instructor only)

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "course_name": "Introduction to CS",
  "semester": "2025_spring",
  "total_students": 85,
  "assessments": [
    {
      "slug": "midterm",
      "name": "Vize",
      "weight": 40,
      "graded_count": 85,
      "pending_count": 0,
      "is_complete": true
    },
    {
      "slug": "final",
      "name": "Final",
      "weight": 60,
      "graded_count": 80,
      "pending_count": 5,
      "is_complete": false
    }
  ],
  "is_finalized": false,
  "pending_message": "final için 5 öğrencinin notu girilmemiş"
}
```

Finalize olduktan sonra:
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "is_finalized": true,
  "finalized_at": "2025-01-15T10:00:00Z",
  "grading_type": "relative",
  "class_statistics": {
    "mean": 55.2,
    "stddev": 12.3,
    "passing_count": 70,
    "failing_count": 15
  }
}
```

---

### 🔒 GET /api/v1/grades/course/:courseId/students
Ders öğrenci listesi ve notları

**Role**: Teacher (course instructor only)

**Response** (200):
```json
{
  "course_id": "uuid",
  "course_code": "CS101",
  "students": [
    {
      "registration_id": "uuid",
      "student_id": "uuid",
      "student_number": "2021123456",
      "first_name": "Ahmet",
      "last_name": "Yılmaz",
      "scores": {
        "midterm": {"score": 75.5, "is_absent": false},
        "final": {"score": 82.0, "is_absent": false}
      },
      "current_average": 79.4
    },
    {
      "registration_id": "uuid",
      "student_id": "uuid", 
      "student_number": "2021123457",
      "first_name": "Ayşe",
      "last_name": "Demir",
      "scores": {
        "midterm": {"score": 45.0, "is_absent": false},
        "final": null
      },
      "current_average": null
    }
  ]
}
```

---

### 🔒 POST /api/v1/grades/course/:courseId/scores
Tekil not girişi

**Request**:
```json
{
  "registration_id": "uuid",
  "slug": "final",
  "score": 75.5
}
```

Gelmedi için:
```json
{
  "registration_id": "uuid",
  "slug": "final",
  "is_absent": true
}
```

**Response** (201):

Normal not girişi:
```json
{
  "id": "uuid",
  "student_number": "2021123456",
  "slug": "final",
  "score": 75.5,
  "graded_at": "2025-01-15T10:00:00Z"
}
```

Son final notu girildi ve otomatik hesaplama yapıldı:
```json
{
  "id": "uuid",
  "student_number": "2021123456",
  "slug": "final",
  "score": 75.5,
  "graded_at": "2025-01-15T10:00:00Z",
  "auto_finalized": true,
  "finalize_result": {
    "grading_type": "absolute",
    "class_mean": 68.5,
    "total_students": 85,
    "passing_count": 78,
    "failing_count": 7
  }
}
```

---

### 🔒 POST /api/v1/grades/course/:courseId/scores/bulk
Toplu not girişi

**Request**:
```json
{
  "slug": "final",
  "scores": [
    {"registration_id": "uuid-1", "score": 75.5},
    {"registration_id": "uuid-2", "score": 82.0},
    {"registration_id": "uuid-3", "is_absent": true}
  ]
}
```

**Response** (201):
```json
{
  "slug": "final",
  "success_count": 3,
  "auto_finalized": true,
  "finalize_result": {
    "grading_type": "relative",
    "class_mean": 52.3,
    "total_students": 85,
    "passing_count": 65,
    "failing_count": 20
  }
}
```

---

### 🔒 PUT /api/v1/grades/course/:courseId/scores/:scoreId
Not güncelleme (finalize öncesi)

**Request**:
```json
{
  "score": 78.0
}
```

**Not**: Finalize olduktan sonra not değiştirilemez.

---

### 🔓 GET /api/v1/grades/student/my
Öğrencinin kendi notları

**Role**: Student

**Business Logic**:
1. Get student_id from JWT payload
2. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Query active courses and completed courses (with Redis caching)

**Response** (200):
```json
{
  "student_id": "uuid",
  "student_number": "2021123456",
  "active_courses": [
    {
      "course_code": "CS102",
      "course_name": "Data Structures",
      "semester": "2025_spring",
      "credits": 4,
      "scores": {
        "midterm": {"score": 75.5, "name": "Vize", "weight": 40},
        "final": null
      }
    }
  ],
  "completed_courses": [
    {
      "course_code": "CS101",
      "course_name": "Introduction to CS",
      "semester": "2024_fall",
      "credits": 3,
      "weighted_average": 79.4,
      "grade_point": "2.50"
    }
  ],
  "cumulative_gpa": 2.50,
  "total_credits": 3
}
```

---

### 🔓 GET /api/v1/grades/transcript/:studentId
Resmi transcript

**Role**: Student (kendi), Admin

**Business Logic**:
1. If Student role: Get student_id from JWT payload, verify :studentId matches
2. **is_active kontrolü** (Student role): students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Query transcript data (with Redis caching)

**Response** (200):
```json
{
  "student": {
    "student_number": "2021123456",
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "department": "Computer Science",
    "enrollment_year": 2021
  },
  "semesters": [
    {
      "semester": "2024_fall",
      "semester_display": "2024-2025 Güz",
      "courses": [
        {
          "course_code": "CS101",
          "course_name": "Introduction to CS",
          "credits": 3,
          "grade_point": "2.50"
        },
        {
          "course_code": "MATH201",
          "course_name": "Linear Algebra",
          "credits": 4,
          "grade_point": "0.00"
        }
      ],
      "semester_credits": 7,
      "semester_gpa": 1.07
    }
  ],
  "summary": {
    "total_credits": 7,
    "cumulative_gpa": 1.07
  },
  "generated_at": "2025-11-10T10:00:00Z"
}
```

---

## GPA Hesaplama

```sql
SELECT
    ROUND(
        SUM(grade_point::text::decimal * credits) / NULLIF(SUM(credits), 0),
        2
    ) as gpa,
    SUM(credits) as total_credits
FROM student_completed_courses
WHERE student_id = $1;
```

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Subscribed Exchanges:
- "student.events" (routing keys: student.created, student.updated, student.deactivated)
- "course.events" (routing keys: course.semester.created, course.semester.updated, course.semester.deleted, course.instructor.changed, course.prerequisites.updated)
- "enrollment.events" (routing keys: enrollment.program_approved)
- "attendance.events" (routing keys: attendance.semester.failed)

Publishing Exchange:
- "grade.events" (type: topic)

Routing Keys (Publishing):
- grade.submitted
- grade.finalized
- grade.student.prerequisite.passed
```

---

## Event Schemas

### grade.submitted
```json
{
  "event_type": "grade.submitted",
  "data": {
    "student_id": "uuid",
    "course_code": "CS101",
    "slug": "final",
    "score": 75.5
  }
}
```

### grade.finalized
```json
{
  "event_type": "grade.finalized",
  "data": {
    "course_id": "uuid",
    "course_code": "CS101",
    "semester": "2025_spring",
    "grading_type": "relative",
    "total_students": 85,
    "passing_count": 65,
    "failing_count": 20,
    "attendance_failed_count": 3,
    "class_mean": 52.3
  }
}
```

### grade.student.prerequisite.passed
Önkoşul dersini GEÇEN öğrenci için yayınlanır. Enrollment Service bu event'i consume ederek `student_passed_prerequisites` tablosuna kayıt ekler.

**Önemli**: Sadece GEÇEN öğrenciler için event yayınlanır. Kalan öğrenciler için event yayınlanmaz.

```json
{
  "event_type": "grade.student.prerequisite.passed",
  "timestamp": "2025-01-15T10:00:00Z",
  "data": {
    "student_id": "uuid",
    "course_id": "uuid",
    "course_code": "CS101",
    "semester": "2024_fall",
    "grade_point": "2.50"
  }
}
```

**Event Consumers**:
- **Enrollment Service**: `student_passed_prerequisites` tablosuna INSERT yapar. Öğrenci bu dersi geçtiği için, bu dersi önkoşul olarak gerektiren derslere kayıt yapabilir.

---

## Error Codes

| HTTP | Code | Açıklama |
|------|------|----------|
| 400 | INVALID_SCORE | Not 0-100 aralığında değil |
| 400 | INVALID_SLUG | Assessment schema'da tanımlı değil |
| 400 | ALREADY_FINALIZED | Ders zaten finalize edilmiş, not değiştirilemez |
| 403 | NOT_COURSE_INSTRUCTOR | Bu dersin hocası değilsiniz |
| 403 | STUDENT_DEACTIVATED | Öğrenci deaktif edilmiş (is_active = false) |
| 403 | ATTENDANCE_FAILED | Öğrenci devamsızlıktan kalmış, manuel not girilemez |
| 404 | COURSE_NOT_FOUND | Ders bulunamadı |
| 404 | REGISTRATION_NOT_FOUND | Kayıt bulunamadı |
| 409 | SCORE_EXISTS | Bu sınav için not zaten girilmiş |

---

## Redis Cache

```
grades:student:{id}:active      → 5 min
grades:student:{id}:completed   → 1 hour  
grades:student:{id}:gpa         → 1 hour
grades:course:{id}:students     → 5 min
grades:course:{id}:status       → 5 min
```

---

**Version**: 7.3.0
**Last Updated**: 2025-12-12

**Önemli Değişiklikler (v7.3.0)**:
- ✅ **Prerequisite Logic Değişikliği**: Artık sadece GEÇEN öğrenciler için `grade.student.prerequisite.passed` event'i yayınlanır
- ❌ **Kaldırıldı**: `grade.student.prerequisite.failed` event'i (artık kullanılmıyor)
- ✅ **Event Schema Güncellemesi**: `grade.student.prerequisite.passed` event'ine `semester` ve `grade_point` eklendi
- ✅ **Enrollment Service Entegrasyonu**: Geçen öğrenciler `student_passed_prerequisites` tablosuna kaydedilir (failed tablosu yerine)

**Önceki Değişiklikler (v7.2.0)**:
- ✅ **Devamsızlık Entegrasyonu**: `attendance.semester.failed` event'i tüketiliyor
- ✅ **Devamsızlık Not Girişi Engeli**: Devamsızlıktan kalan öğrenciye manuel not girilemez (403 ATTENDANCE_FAILED)
- ✅ **Otomatik Skor**: Devamsızlıktan kalanların skoru otomatik 0, grade_point FF
- ✅ **Çan Hesabı Hariç Tutma**: Devamsızlıktan kalanlar çan eğrisi hesabına dahil edilmez
- ✅ **Schema Güncellemesi**: `student_course_registrations` ve `student_completed_courses` tablolarına `is_attendance_failed` boolean eklendi
- ✅ **Event Güncellemesi**: `grade.finalized` event'ine `attendance_failed_count` eklendi

**Önceki Değişiklikler (v7.1.0)**:
- ✅ **Otomatik Hesaplama**: Son final notu girilince otomatik finalize
- ✅ **Otomatik Grading Type**: Sınıf ortalaması >= 60 → Absolute, < 60 → Z-Score
- ✅ **Threshold Yok**: Çan eğrisinde minimum eşik yok
- ❌ **Kaldırıldı**: Preview endpoint (gereksiz)
- ❌ **Kaldırıldı**: Manuel finalize endpoint (otomatik)
- ❌ **Kaldırıldı**: Grading type seçimi (otomatik)