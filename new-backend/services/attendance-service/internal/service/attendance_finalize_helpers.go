package service

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/attendance/internal/db"
	"github.com/baaaki/mydreamcampus/attendance/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
)

// FailureInfo is one failing student's standing in both session types during
// FinalizeAttendance. Exported only so the helper tests can build instances —
// production code never touches it outside of failingStudents / deriveFailedType.
type FailureInfo struct {
	StudentNumber string
	StudentName   string
	StudentEmail  string
	Theory        TypeAttendance
	Lab           TypeAttendance
}

// TypeAttendance is a student's standing in one session type: the sessions
// held since they were enrolled, how many they attended, and the minimum
// that number of sessions requires.
type TypeAttendance struct {
	Held        int
	Present     int
	MinRequired int
	Failed      bool
}

func (a TypeAttendance) response() *dto.SessionTypeAttendance {
	return &dto.SessionTypeAttendance{
		PresentCount:  a.Present,
		AbsentCount:   a.Held - a.Present,
		TotalSessions: a.Held,
		MinRequired:   a.MinRequired,
		Passed:        !a.Failed,
	}
}

func (a TypeAttendance) eventDetail() *dto.AttendanceFailedTypeDetail {
	return &dto.AttendanceFailedTypeDetail{
		TotalSessions: a.Held,
		PresentCount:  a.Present,
		AbsentCount:   a.Held - a.Present,
		MinRequired:   a.MinRequired,
	}
}

// minTheoryRequired / minLabRequired scale the attendance thresholds to the
// number of sessions actually held. The historical rule was written for a
// 14-week semester (theory 10/14, lab 11/14); keeping the same ratios with a
// ceiling division means a course that only ran e.g. 8 theory sessions
// requires ceil(8*10/14)=6 instead of a flat 10 — which would fail everyone.
func minTheoryRequired(totalSessions int64) int {
	if totalSessions <= 0 {
		return 0
	}
	return int((totalSessions*MinTheoryAttendance + StandardSemesterWeeks - 1) / StandardSemesterWeeks)
}

func minLabRequired(totalSessions int64) int {
	if totalSessions <= 0 {
		return 0
	}
	return int((totalSessions*MinLabAttendance + StandardSemesterWeeks - 1) / StandardSemesterWeeks)
}

// deriveFailedType returns the public-facing failure category for the
// student's combined theory/lab state.
//
// The contract is: "theory" iff only theory failed, "lab" iff only lab failed,
// "both" iff at least one of them failed (the both-pass case never reaches
// here — callers only invoke deriveFailedType for students already in the
// failure set). Reordering these branches changes downstream grade-service
// behaviour silently; this is why it lives in its own helper.
func deriveFailedType(theoryFailed, labFailed bool) string {
	if theoryFailed && !labFailed {
		return "theory"
	}
	if !theoryFailed && labFailed {
		return "lab"
	}
	return "both"
}

// failingStudents holds every enrolled student to the sessions of each type
// run since they were enrolled, with the minimum scaled to that count, and
// returns the ones below it in either type. Holding them to the course's
// total failed a student approved in week 5 for the four weeks they could
// not attend. A student enrolled after a type's last session had nothing to
// attend there and cannot fail it.
//
// Returning a map (not a slice) is intentional: downstream iteration order
// is not part of the contract.
func failingStudents(theory, lab []db.GetCourseAttendanceByStudentAndTypeRow) map[uuid.UUID]*FailureInfo {
	out := make(map[uuid.UUID]*FailureInfo, len(theory))
	info := func(row db.GetCourseAttendanceByStudentAndTypeRow) *FailureInfo {
		sid := utils.PgUUIDToUUID(row.StudentID)
		if existing, ok := out[sid]; ok {
			return existing
		}
		created := &FailureInfo{
			StudentNumber: row.StudentNumber,
			StudentName:   fmt.Sprintf("%s %s", utils.PgTextToString(row.FirstName), utils.PgTextToString(row.LastName)),
			StudentEmail:  utils.PgTextToString(row.Email),
		}
		out[sid] = created
		return created
	}
	for _, row := range theory {
		info(row).Theory = standing(row, minTheoryRequired)
	}
	for _, row := range lab {
		info(row).Lab = standing(row, minLabRequired)
	}
	for sid, student := range out {
		if !student.Theory.Failed && !student.Lab.Failed {
			delete(out, sid)
		}
	}
	return out
}

func standing(row db.GetCourseAttendanceByStudentAndTypeRow, minRequired func(int64) int) TypeAttendance {
	minimum := minRequired(row.HeldCount)
	return TypeAttendance{
		Held:        int(row.HeldCount),
		Present:     int(row.PresentCount),
		MinRequired: minimum,
		Failed:      int(row.PresentCount) < minimum,
	}
}
