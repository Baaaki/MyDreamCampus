package service

import (
	"fmt"

	"github.com/baaaki/mydreamcampus/attendance-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
)

// FailureInfo aggregates a single student's theory + lab failure state during
// FinalizeAttendance. Exported only so the helper tests can build instances —
// production code never touches it outside of mergeFailureRows / deriveFailedType.
type FailureInfo struct {
	StudentNumber string
	StudentName   string
	StudentEmail  string
	TheoryPresent int
	TheoryAbsent  int
	TheoryFailed  bool
	LabPresent    int
	LabAbsent     int
	LabFailed     bool
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

// mergeFailureRows combines failing-student rows from the theory and lab
// queries into a single per-student record. A student can appear in either
// list (or both) — the merge marks each failure type independently so the
// publisher can build a complete event without re-querying.
//
// Returning a map (not a slice) is intentional: downstream iteration order
// is not part of the contract, and the map structure makes the lab-side
// "is this student already in the failure set?" lookup an O(1) op.
func mergeFailureRows(
	theory []db.GetFailingStudentsByCourseByTypeRow,
	lab []db.GetFailingStudentsByCourseByTypeRow,
) map[uuid.UUID]*FailureInfo {
	out := make(map[uuid.UUID]*FailureInfo, len(theory)+len(lab))

	for _, row := range theory {
		sid := utils.PgUUIDToUUID(row.StudentID)
		out[sid] = &FailureInfo{
			StudentNumber: row.StudentNumber,
			StudentName:   fmt.Sprintf("%s %s", utils.PgTextToString(row.FirstName), utils.PgTextToString(row.LastName)),
			StudentEmail:  utils.PgTextToString(row.Email),
			TheoryPresent: int(row.PresentCount),
			TheoryAbsent:  int(row.AbsentCount),
			TheoryFailed:  true,
		}
	}

	for _, row := range lab {
		sid := utils.PgUUIDToUUID(row.StudentID)
		if existing, ok := out[sid]; ok {
			existing.LabPresent = int(row.PresentCount)
			existing.LabAbsent = int(row.AbsentCount)
			existing.LabFailed = true
			continue
		}
		out[sid] = &FailureInfo{
			StudentNumber: row.StudentNumber,
			StudentName:   fmt.Sprintf("%s %s", utils.PgTextToString(row.FirstName), utils.PgTextToString(row.LastName)),
			StudentEmail:  utils.PgTextToString(row.Email),
			LabPresent:    int(row.PresentCount),
			LabAbsent:     int(row.AbsentCount),
			LabFailed:     true,
		}
	}

	return out
}
