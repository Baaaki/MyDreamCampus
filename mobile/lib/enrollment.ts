import type { EnrollmentProgramResponse } from '@/types/enrollment.types';

// The program the student is taking now: the approved one of the semester
// attendance runs in, else the latest approved semester. The backend lists
// programs by submission, so a next-semester program comes first as soon as
// its advisor approves it, while this semester's classes are still running.
// Semester names ("2026-2027-Fall") sort chronologically as text.
export function currentProgram(
  programs: EnrollmentProgramResponse[],
  activeSemester?: string
): EnrollmentProgramResponse | null {
  const approved = programs.filter((p) => p.status === 'approved');
  return (
    approved.find((p) => p.semester === activeSemester) ??
    [...approved].sort((a, b) => b.semester.localeCompare(a.semester))[0] ??
    null
  );
}
