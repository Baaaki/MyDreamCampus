package events

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEventNames_FollowDottedConvention guards against accidental renames
// that would silently break consumers subscribed to existing routing keys.
func TestEventNames_FollowDottedConvention(t *testing.T) {
	all := map[string]string{
		"EventStaffCreated":                   EventStaffCreated,
		"EventStaffUpdated":                   EventStaffUpdated,
		"EventStaffDeactivated":               EventStaffDeactivated,
		"EventStudentCreated":                 EventStudentCreated,
		"EventStudentUpdated":                 EventStudentUpdated,
		"EventStudentDeactivated":             EventStudentDeactivated,
		"EventCourseSemesterCreated":          EventCourseSemesterCreated,
		"EventGradeStudentPrerequisitePassed": EventGradeStudentPrerequisitePassed,
		"EventEnrollmentProgramSubmitted":     EventEnrollmentProgramSubmitted,
		"EventEnrollmentProgramApproved":      EventEnrollmentProgramApproved,
		"EventEnrollmentProgramRejected":      EventEnrollmentProgramRejected,
		"EventEnrollmentProgramCancelled":     EventEnrollmentProgramCancelled,
		"EventAttendanceSemesterFailed":       EventAttendanceSemesterFailed,
	}
	for name, value := range all {
		assert.NotEmpty(t, value, "%s must not be empty", name)
		assert.True(t, strings.Contains(value, "."), "%s=%q must use dotted convention", name, value)
		assert.Equal(t, strings.ToLower(value), value, "event names must be lowercase")
	}
}

func TestRoutingKeys_MatchEventNames(t *testing.T) {
	pairs := map[string]string{
		EventStaffCreated:               RoutingKeyStaffCreated,
		EventStaffUpdated:               RoutingKeyStaffUpdated,
		EventStaffDeactivated:           RoutingKeyStaffDeactivated,
		EventStudentCreated:             RoutingKeyStudentCreated,
		EventStudentUpdated:             RoutingKeyStudentUpdated,
		EventStudentDeactivated:         RoutingKeyStudentDeactivated,
		EventEnrollmentProgramSubmitted: RoutingKeyEnrollmentProgramSubmitted,
		EventEnrollmentProgramApproved:  RoutingKeyEnrollmentProgramApproved,
		EventEnrollmentProgramRejected:  RoutingKeyEnrollmentProgramRejected,
		EventEnrollmentProgramCancelled: RoutingKeyEnrollmentProgramCancelled,
	}
	for event, routing := range pairs {
		assert.Equal(t, event, routing,
			"routing key must match event name for direct binding")
	}
}

func TestWildcardRoutingKeys_AreValid(t *testing.T) {
	wildcards := []string{
		RoutingKeyStaffAll,
		RoutingKeyStudentAll,
		RoutingKeyCourseAll,
		RoutingKeyEnrollmentAll,
	}
	for _, key := range wildcards {
		assert.True(t,
			strings.HasSuffix(key, ".*") || strings.HasSuffix(key, ".#"),
			"wildcard %q must end with .* or .#", key)
	}
}

func TestQueueNames_NotEmpty(t *testing.T) {
	queues := []string{
		QueueAuthStaffEvents,
		QueueAuthStudentEvents,
		QueueStudentStaffEvents,
		QueueCatalogStaffEvents,
		QueueCatalogStudentEvents,
		QueueEnrollmentCourseEvents,
		QueueEnrollmentStudentEvents,
	}
	for _, q := range queues {
		assert.NotEmpty(t, q)
	}
}
