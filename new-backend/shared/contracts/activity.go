package contracts

// SemesterActivity tells enrollment whether a student has already left a
// trace in a semester's courses (an attendance record, a score) that taking
// their approved program back would orphan. Served by attendance and grades
// on GET /internal/students/:id/semester-activity?semester=, read by
// enrollment before it lets a student change an approved program.
type SemesterActivity struct {
	HasActivity bool `json:"has_activity"`
}
