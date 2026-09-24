package events

import "strings"

// ============================================================================
// EVENT NAMES (Used in outbox publisher & consumer handlers)
// ============================================================================

// Auth Service Events
const (
	EventTypeUserRegistered             = "user.registered"
	EventTypeUserPasswordResetRequested = "user.password_reset_requested"
)

// Staff Service Events
const (
	EventStaffCreated     = "staff.created"
	EventStaffUpdated     = "staff.updated"
	EventStaffDeactivated = "staff.deactivated"
)

// Student Service Events
const (
	EventStudentCreated     = "student.created"
	EventStudentUpdated     = "student.updated"
	EventStudentDeactivated = "student.deactivated"
)

// Course Catalog Service Events
const (
	// Semester Course Events
	EventCourseSemesterCreated = "course.semester.created"
)

// Academic period projection events. Catalog owns the period definitions and
// publishes one event per consuming service; the routing key doubles as the
// event type:
//
//	course_catalog.period.<consumer>.<action>
//
// Each consumer binds a single <consumer> value, so the broker does the
// filtering and a consumer only ever switches on <action>.
const (
	periodEventPrefix = "course_catalog.period."

	PeriodActionCreated = "created"
	PeriodActionUpdated = "updated"
	PeriodActionDeleted = "deleted"
)

// PeriodEventType builds the routing key / event type for one consumer.
// consumer is a period type ("enrollment", "grading", "attendance").
func PeriodEventType(consumer, action string) string {
	return periodEventPrefix + consumer + "." + action
}

// PeriodEventRoutingPattern is the topic pattern a consumer binds to receive
// every action for its own period type.
func PeriodEventRoutingPattern(consumer string) string {
	return periodEventPrefix + consumer + ".*"
}

// PeriodEventAction returns the trailing action of a period event type, or an
// empty string when the type is not a period event.
func PeriodEventAction(eventType string) string {
	if !strings.HasPrefix(eventType, periodEventPrefix) {
		return ""
	}
	rest := eventType[len(periodEventPrefix):]
	idx := strings.LastIndex(rest, ".")
	if idx < 0 {
		return ""
	}
	return rest[idx+1:]
}

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

// Payment Service Events
const (
	EventPaymentCompleted = "payment.completed"
	EventPaymentFailed    = "payment.failed"
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
	RoutingKeyUserRegistered             = "user.registered"
	RoutingKeyUserPasswordResetRequested = "user.password_reset_requested"

	// Staff events routing keys
	RoutingKeyStaffCreated     = "staff.created"
	RoutingKeyStaffUpdated     = "staff.updated"
	RoutingKeyStaffDeactivated = "staff.deactivated"
	RoutingKeyStaffAll         = "staff.*"

	// Student events routing keys
	RoutingKeyStudentCreated     = "student.created"
	RoutingKeyStudentUpdated     = "student.updated"
	RoutingKeyStudentDeactivated = "student.deactivated"
	RoutingKeyStudentAll         = "student.*"

	// Course events routing keys
	RoutingKeyCourseCreated = "course.*.created"
	RoutingKeyCourseAll     = "course.#"

	// Enrollment events routing keys
	RoutingKeyEnrollmentProgramSubmitted = "enrollment.program.submitted"
	RoutingKeyEnrollmentProgramApproved  = "enrollment.program.approved"
	RoutingKeyEnrollmentProgramRejected  = "enrollment.program.rejected"
	RoutingKeyEnrollmentProgramCancelled = "enrollment.program.cancelled"
	RoutingKeyEnrollmentAll              = "enrollment.#"

	// Payment events routing keys
	RoutingKeyPaymentCompleted = "payment.completed"
	RoutingKeyPaymentFailed    = "payment.failed"
)
