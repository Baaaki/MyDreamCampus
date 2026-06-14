package events

// ============================================================================
// EVENT NAMES (Used in outbox publisher & consumer handlers)
// ============================================================================

// Auth Service Events
const (
	EventTypeUserRegistered               = "user.registered"
	EventTypeUserPasswordResetRequested   = "user.password_reset_requested"
)

// Staff Service Events
const (
	EventStaffCreated = "staff.created"
	EventStaffUpdated = "staff.updated"
	EventStaffDeactivated = "staff.deactivated"
)

// Student Service Events
const (
	EventStudentCreated = "student.created"
	EventStudentUpdated = "student.updated"
	EventStudentDeactivated = "student.deactivated"
)

// Course Catalog Service Events
const (
	// Semester Course Events
	EventCourseSemesterCreated = "course.semester.created"
)

// Grades Service Events
const (
	EventGradeStudentPrerequisitePassed = "grade.student.prerequisite.passed"
)

// Enrollment Service Events
const (
	EventEnrollmentProgramSubmitted = "enrollment.program.submitted"
	EventEnrollmentProgramApproved  = "enrollment.program.approved"
	EventEnrollmentProgramRejected  = "enrollment.program.rejected"
	EventEnrollmentProgramCancelled = "enrollment.program.cancelled"
)

// Attendance Service Events
const (
	EventAttendanceSemesterFailed = "attendance.semester.failed"
)

// ============================================================================
// RABBITMQ QUEUE NAMES (Used in consumer setup)
// ============================================================================

const (
	// Auth Service Queues (consumes from other services)
	QueueAuthStaffEvents   = "auth_events_queue"
	QueueAuthStudentEvents = "auth_events_queue"

	// Student Service Queues (consumes from staff service)
	QueueStudentStaffEvents = "student.staff_events"

	// Course Catalog Service Queues (future use)
	QueueCatalogStaffEvents   = "catalog.staff_events"
	QueueCatalogStudentEvents = "catalog.student_events"

	// Enrollment Service Queues (future use)
	QueueEnrollmentCourseEvents  = "enrollment.course_events"
	QueueEnrollmentStudentEvents = "enrollment.student_events"
)

// ============================================================================
// RABBITMQ ROUTING KEYS (Used for topic exchange routing - future use)
// ============================================================================

const (
	// Auth events routing keys
	RoutingKeyUserRegistered               = "user.registered"
	RoutingKeyUserPasswordResetRequested   = "user.password_reset_requested"

	// Staff events routing keys
	RoutingKeyStaffCreated = "staff.created"
	RoutingKeyStaffUpdated = "staff.updated"
	RoutingKeyStaffDeactivated = "staff.deactivated"
	RoutingKeyStaffAll     = "staff.*"

	// Student events routing keys
	RoutingKeyStudentCreated = "student.created"
	RoutingKeyStudentUpdated = "student.updated"
	RoutingKeyStudentDeactivated = "student.deactivated"
	RoutingKeyStudentAll     = "student.*"

	// Course events routing keys
	RoutingKeyCourseCreated = "course.*.created"
	RoutingKeyCourseAll     = "course.#"

	// Enrollment events routing keys
	RoutingKeyEnrollmentProgramSubmitted = "enrollment.program.submitted"
	RoutingKeyEnrollmentProgramApproved  = "enrollment.program.approved"
	RoutingKeyEnrollmentProgramRejected  = "enrollment.program.rejected"
	RoutingKeyEnrollmentProgramCancelled = "enrollment.program.cancelled"
	RoutingKeyEnrollmentAll              = "enrollment.#"
)
