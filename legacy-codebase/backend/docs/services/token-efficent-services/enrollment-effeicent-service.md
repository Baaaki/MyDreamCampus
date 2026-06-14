# Enrollment Service

## Overview
- **Type**: Critical (highest concurrency/race condition risk)
- **Pattern**: Program-based enrollment (full semester submission, advisor approval required)
- **Capacity**: 60,000+ concurrent students

## Core Flow
1. Student submits full course program → `current_enrollment++` for each course
2. Status: `pending` → awaits advisor decision
3. Advisor approves → status: `approved`, Grades/Attendance services create records
4. Advisor rejects → `current_enrollment--`, program deleted, rejection logged, student can resubmit

---

## Events

### Inbound (Consumed)
| Event | Source | Action |
|-------|--------|--------|
| `student.created` | Student Service | INSERT students_cache |
| `student.updated` | Student Service | UPDATE students_cache |
| `course.semester.created` | Course Catalog | INSERT semester_courses_cache + course_sessions_cache |
| `course.semester.updated` | Course Catalog | UPDATE semester_courses_cache, REPLACE course_sessions_cache |
| `course.semester.deleted` | Course Catalog | DELETE semester_courses_cache (CASCADE) |
| `grade.student.prerequisite.passed` | Grades Service | INSERT student_passed_prerequisites |

### Outbound (Published)
| Event | Trigger | Consumers |
|-------|---------|-----------|
| `enrollment.program_submitted` | POST /enrollments | Notification |
| `enrollment.program_approved` | POST /:id/approve | Grades, Attendance, Notification |
| `enrollment.program_rejected` | POST /:id/reject | Notification |
| `enrollment.program_cancelled` | DELETE /:id | Notification |

---

## Database Schema

### Enums
```sql
CREATE TYPE enrollment_status_enum AS ENUM ('pending', 'approved');
CREATE TYPE day_of_week_enum AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');
-- Note: class_level uses SMALLINT (1-6) consistent with Student Service (source of truth)
CREATE TYPE course_type_enum AS ENUM ('mandatory', 'elective');
```

### Tables

#### students_cache
Synced from Student Service. Fields: `id` (PK), `student_number`, `email`, `first_name`, `last_name`, `department`, `class_level`, `advisor_id`, `status`, `synced_at`

#### semester_courses_cache
Synced from Course Catalog. Enrollment Service OWNS `current_enrollment`.
Fields: `id` (PK, = event's semester_course_id), `course_code`, `course_name`, `faculty`, `department`, `credits`, `course_type`, `class_level`, `semester` (VARCHAR "2025_spring"), `instructor_id`, `instructor_fullname`, `classroom_location`, `max_capacity`, `current_enrollment` (DEFAULT 0, CHECK >= 0), `prerequisites` (JSONB), `synced_at`

#### course_sessions_cache
Slot-based scheduling. Fields: `id` (PK), `course_id` (FK CASCADE), `day_of_week`, `slot_number` (1-9), `synced_at`. UNIQUE(course_id, day_of_week, slot_number)

#### student_passed_prerequisites
Only PASSED prerequisite courses stored. Existence = passed, absence = not passed.
Fields: `student_id` (FK CASCADE), `course_code` (VARCHAR). PRIMARY KEY(student_id, course_code)

#### enrollment_programs
Active programs only (pending/approved). Fields: `id` (PK), `student_id` (FK), `semester`, `status` (DEFAULT 'pending'), `created_at`. UNIQUE(student_id, semester)

#### enrollment_program_courses
Fields: `id` (PK), `program_id` (FK CASCADE), `course_id` (FK), `created_at`. UNIQUE(program_id, course_id)

#### enrollment_rejection_logs
Immutable audit trail. Fields: `id` (PK), `original_program_id` (no FK), `student_id` (FK), `advisor_id`, `advisor_fullname` (snapshot), `semester`, `rejection_reason`, `rejected_courses` (JSONB), `rejected_at`

**rejected_courses JSONB structure:**
```json
{"courses": [{"course_id": "uuid", "course_code": "CS101", "course_name": "...", "credits": 3, "instructor": "..."}], "total_credits": 18, "submitted_at": "ISO8601"}
```

### Indexes
```sql
idx_sessions_cache_course(course_id)
idx_sessions_cache_day_slot(day_of_week, slot_number)
idx_rejection_logs_student(student_id)
idx_rejection_logs_student_semester(student_id, semester)
idx_programs_student(student_id)
idx_programs_status(status)
idx_programs_semester(semester)
idx_students_cache_advisor(advisor_id)
idx_semester_courses_cache_department_semester(department, semester)
idx_semester_courses_cache_class_level(class_level)
```

---

## API Endpoints

### Common Patterns

**Concurrency Control** (used in POST, DELETE, reject):
```sql
BEGIN;
SELECT id, current_enrollment, max_capacity FROM semester_courses_cache 
WHERE id = ANY($course_ids) ORDER BY id FOR UPDATE;
-- App validates, then:
UPDATE semester_courses_cache SET current_enrollment = current_enrollment +/- 1 WHERE id = ANY($course_ids);
COMMIT;
```
- Isolation: READ COMMITTED
- Locking: Row-level (SELECT FOR UPDATE)
- Deadlock prevention: ORDER BY id

---

### Student Endpoints

#### GET /available-courses
**Auth**: Student | **Params**: semester (required)

**Logic**: Filter semester_courses_cache by `department = student.department` AND `class_level <= student.class_level` AND `semester = param`

**Response**: `{student_id, department, class_level, semester, available_courses: [{id, course_code, course_name, credits, schedule_sessions: [{day_of_week, slot_numbers}], max_capacity, current_enrollment, available_seats, instructor}]}`

---

#### POST /enrollments
**Auth**: Student

**Request**: `{student_id, semester, course_ids[]}`

**Validations** (in order):
1. JWT auth + enrollment period check (middleware)
2. Duplicate: UNIQUE(student_id, semester) → 409 ALREADY_SUBMITTED
3. Department: all courses from student's department
4. Class level: all courses ≤ student's class_level
5. Prerequisites: for each course, check student_passed_prerequisites table → 400 PREREQUISITES_NOT_MET
6. Schedule conflict: self-join course_sessions_cache for same day+slot → 400 SCHEDULE_CONFLICT
7. Capacity: SELECT FOR UPDATE, check current < max → 409 COURSE_FULL
8. Increment: current_enrollment++ for all courses
9. Create program (status: pending)
10. Publish: enrollment.program_submitted

**Response** (201): `{id, student_id, semester, status, courses: [{course_id, course_code, course_name, credits}], created_at}`

---

#### GET /my
**Auth**: Student | **Params**: semester, status (optional)

**Response**: `{student_id, programs: [{id, semester, status, courses[], created_at}]}`

**Note**: Rejected programs not included (deleted). Use /my/rejections for history.

---

#### GET /my/rejections/latest
**Auth**: Student | **Params**: semester (required)

**Response**: `{student_id, semester, has_rejection, latest_rejection: {id, advisor_id, advisor_fullname, rejection_reason, rejected_courses (JSONB), rejected_at} | null, total_rejections}`

---

#### GET /my/rejections
**Auth**: Student | **Params**: semester (optional)

**Response**: `{student_id, rejections: [{id, semester, advisor_id, advisor_fullname, rejection_reason, rejected_courses, rejected_at}], total_rejections}`

---

#### DELETE /:id
**Auth**: Student (owner only)

**Validations**:
1. Ownership: program.student_id == JWT.student_id → 403 FORBIDDEN
2. Status: must be pending → 400 CANNOT_CANCEL_APPROVED

**Logic**: Decrement current_enrollment (FOR UPDATE), delete program (CASCADE), publish enrollment.program_cancelled

**Response**: `{message: "Program cancelled successfully", can_create_new: true}`

**Note**: No rejection log created (only for advisor rejections)

---

### Advisor Endpoints

#### GET /pending-approval
**Auth**: Teacher (Advisor)

**Logic**: Filter programs where students_cache.advisor_id = JWT.user_id AND status = 'pending'

**Response**: `{advisor_id, pending_programs: [{id, student: {id, student_number, first_name, last_name, department, class_level}, semester, courses: [{course_id, course_code, course_name, credits, current_enrollment, max_capacity}], created_at}]}`

---

#### POST /:id/approve
**Auth**: Teacher (Advisor of student)

**Validations**: Authorization (advisor of student), status = pending

**Logic**: Update status to 'approved', publish enrollment.program_approved

**Response**: `{id, student_id, semester, status: "approved"}`

**Note**: No capacity change (already incremented at submission)

**Consumers**: Grades Service (create student records), Attendance Service (create student records), Notification Service

---

#### POST /:id/reject
**Auth**: Teacher (Advisor of student)

**Request**: `{rejection_reason}`

**Logic**:
1. Snapshot courses from enrollment_program_courses JOIN semester_courses_cache
2. INSERT enrollment_rejection_logs (with advisor_fullname, rejected_courses JSONB)
3. Decrement current_enrollment (FOR UPDATE)
4. Publish enrollment.program_rejected
5. DELETE program (CASCADE)

**Response**: `{message: "Program rejected successfully", rejection_log_id, rejection_reason}`

---

#### GET /course/:courseId/students
**Auth**: Teacher (course instructor)

**Logic**: Only approved programs, only instructor's courses

**Response**: `{course_id, course_code, semester, current_enrollment, max_capacity, students: [{student_id, student_number, first_name, last_name, department, class_level, approved_at}]}`

---

## Event Schemas

### Common Fields
All outbound events include: `event_type`, `timestamp`, `data.student_id`, `data.student_number`, `data.student_email` (from students_cache.email), `data.semester`

### enrollment.program_submitted
Additional: `program_id`, `advisor_id`, `course_count`, `created_at`

### enrollment.program_approved
Additional: `program_id`, `courses: [{course_id, course_code, course_name, credits}]`

### enrollment.program_rejected
Additional: `program_id`, `rejection_log_id`, `advisor_id`, `advisor_fullname`, `rejection_reason`, `rejected_courses: [{course_code, course_name, credits}]`

### enrollment.program_cancelled
Additional: `program_id`, `course_count`, `cancelled_by: "student"`, `cancelled_at`

### Consumed Event Schemas

#### student.created / student.updated
```json
{"student_id", "student_number", "email", "first_name", "last_name", "department", "class_level", "advisor_id", "status"}
```

#### course.semester.created / updated
```json
{"semester_course_id", "semester", "course_code", "course_name", "faculty", "department", "credits", "course_type", "class_level", "instructor_id", "instructor_fullname", "classroom_location", "max_capacity", "prerequisites": [{id, course_code, course_name}], "schedule_sessions": [{day_of_week, slot_numbers[]}]}
```
Note: current_enrollment NOT in event (Enrollment Service owns it)

#### course.semester.deleted
```json
{"semester_course_id", "semester", "course_code", "course_name", "department"}
```

#### grade.student.prerequisite.passed
```json
{"student_id", "course_id", "course_code", "semester", "grade_point"}
```
Note: Only published for prerequisite courses where student passed (grade_point >= 1.00)

---

## Phased Enrollment (JWT Middleware)

### Student Registration Periods (class-based, staggered)
| Class | Start | End | Duration |
|-------|-------|-----|----------|
| 4 | 2025-11-20 09:00 | 2025-11-22 23:59 | 3 days |
| 3 | 2025-11-21 09:00 | 2025-11-23 23:59 | 3 days |
| 2 | 2025-11-22 09:00 | 2025-11-24 23:59 | 3 days |
| 1 | 2025-11-23 09:00 | 2025-11-25 23:59 | 3 days |

Outside period → 403 ENROLLMENT_PERIOD_CLOSED

### Advisor Approval Period
All advisors: 2025-11-20 09:00 → 2025-11-30 23:59 (10 days, extends 5 days beyond student registration)

Outside period → 403 APPROVAL_PERIOD_CLOSED

### Configuration
Redis keys: `enrollment:phase:student:class_{N}:start/end`, `enrollment:phase:advisor:start/end`

### Middleware Flow
1. Parse JWT → extract user_id, role
2. If student: get class_level from students_cache, check against phase config
3. If teacher: check against advisor phase config
4. Fail → 403, Pass → continue to handler

---

## Notification Integration

### enrollment.program_rejected notifications
| Channel | Content |
|---------|---------|
| Email | Subject: "Your Course Program Was Rejected - {semester}", Body: advisor name, reason, link to DEBIS, deadline |
| In-app | type: enrollment_rejected, action_url: /enrollment |
| Push | "Your course program was rejected. Open app to create new one." |

**Timing**: Immediate (async queue). Registration period is limited.

### All Events Summary
| Event | Channels | Recipient |
|-------|----------|-----------|
| program_submitted | In-app, Email | Advisor |
| program_approved | Email, Push | Student |
| program_rejected | Email, Push, In-app | Student |
| program_cancelled | In-app | Student |

---

## Error Codes

| Code | Error | Description |
|------|-------|-------------|
| 400 | INVALID_INPUT | Validation error |
| 400 | PREREQUISITES_NOT_MET | Missing prerequisites |
| 400 | WRONG_DEPARTMENT | Course not in student's department |
| 400 | WRONG_CLASS_LEVEL | Course above student's class level |
| 400 | SCHEDULE_CONFLICT | Time slot collision |
| 400 | CANNOT_CANCEL_APPROVED | Approved programs are final |
| 403 | ENROLLMENT_PERIOD_CLOSED | Student registration closed |
| 403 | APPROVAL_PERIOD_CLOSED | Advisor approval closed |
| 403 | FORBIDDEN | Unauthorized access |
| 404 | NOT_FOUND | Resource not found |
| 409 | ALREADY_SUBMITTED | Duplicate program for semester |
| 409 | COURSE_FULL | Capacity exceeded |
| 500 | INTERNAL_ERROR | Server error |

---

## Admin Operations

### Semester Cleanup (Manual)
Execute before new semester, in order (FK constraints):
```sql
DELETE FROM enrollment_program_courses WHERE course_id IN (SELECT id FROM semester_courses_cache WHERE semester = 'OLD');
DELETE FROM enrollment_programs WHERE semester = 'OLD';
DELETE FROM enrollment_rejection_logs WHERE semester = 'OLD'; -- Optional, keep for audit
DELETE FROM semester_courses_cache WHERE semester = 'OLD'; -- CASCADE deletes course_sessions_cache
```

**Never delete**: student_passed_prerequisites (permanent prerequisite history)

---

## Frontend Flow

### State Machine
**States**: APPROVED, PENDING, REJECTED, COURSE_SELECTION

**Transitions**:
- COURSE_SELECTION → PENDING (submit)
- PENDING → APPROVED (advisor approves)
- PENDING → REJECTED (advisor rejects)
- PENDING → COURSE_SELECTION (student cancels)
- REJECTED → COURSE_SELECTION (click "Create New")

**Terminal**: APPROVED

### View Selection (priority order)
1. approved program exists → ApprovedProgramView (read-only)
2. pending program exists → PendingProgramView (cancel button)
3. has_rejection = true → RejectedProgramView (shows reason, courses, "Create New" button)
4. default → CourseSelectionView

### Initial Load (parallel)
```
GET /enrollments/my?semester={s}
GET /enrollments/my/rejections/latest?semester={s}
```

### View Actions
| View | Button | API | Next State |
|------|--------|-----|------------|
| PendingProgramView | Cancel | DELETE /:id | reload → re-evaluate |
| RejectedProgramView | Create New | (state only) | COURSE_SELECTION |
| RejectedProgramView | View History | GET /my/rejections | modal |
| CourseSelectionView | Submit | POST /enrollments | PENDING |

### Components
```
enrollment-page.tsx (container, state management)
approved-program-view.tsx
pending-program-view.tsx
rejected-program-view.tsx
course-selection-view.tsx
rejection-history-modal.tsx
cancel-confirmation-modal.tsx
```

### Advisor Flow
**Page**: /advisor/enrollments

**Load**: GET /pending-approval

**Actions**:
| Action | API | Result |
|--------|-----|--------|
| Approve | POST /:id/approve | Remove from list |
| Reject | POST /:id/reject with {rejection_reason} | Remove from list |

---

## Security (Defense in Depth)

| Layer | Control |
|-------|---------|
| Frontend | Route guard, UI disable for full courses |
| Backend | JWT validation, ownership check, status check |
| Database | UNIQUE constraints, CHECK constraints, FK CASCADE |
| Concurrency | SELECT FOR UPDATE with ORDER BY id |

---

## Known Risks

### Course Code Change
student_passed_prerequisites uses VARCHAR course_code (not UUID). If course code changes in Course Catalog, prerequisite validation may fail (false negative).

**Impact**: Low (code changes rare)
**Mitigation**: Course Catalog should emit migration events (out of scope)

---

## Version
**7.0.0** | 2025-12-12

### Changelog v7.0.0
- ✅ Prerequisite logic: `student_failed_prerequisites` → `student_passed_prerequisites` (whitelist approach)
- ✅ Event: Now consumes `grade.student.prerequisite.passed` instead of failed
- ✅ Validation: "record exists = passed = can enroll" (fixes: never-took vs failed distinction)

### Changelog v6.3.0
- Added enrollment_rejection_logs table with advisor_fullname
- Added GET /my/rejections/latest and GET /my/rejections endpoints
- Added rejection snapshot before program deletion
- Added rejection_log_id and advisor_fullname to program_rejected event
- Added Notification Service integration details
- Standardized API responses to English
- Optimized documentation format for AI consumption