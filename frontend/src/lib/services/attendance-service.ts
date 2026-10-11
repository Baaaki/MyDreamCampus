import { useQuery } from "@tanstack/react-query"
import { attendanceApi } from "@/lib/api-client"
import type {
  AttendanceFinalization,
  AttendanceFinalizationStatus,
  CourseAttendanceDetail,
  MyAttendanceResponse,
  SessionTypeAttendance,
} from "@/lib/types"

// An attendance service older than missed sessions sends none; the page
// lists them unconditionally, so they are filled in where the reply is read.
type MyAttendanceDTO = Omit<MyAttendanceResponse, "courses"> & {
  courses: (Omit<CourseAttendanceDetail, "missed_sessions"> &
    Partial<Pick<CourseAttendanceDetail, "missed_sessions">>)[]
}

/**
 * The logged-in student's attendance per course. Without a semester the
 * backend answers for the active one.
 */
export async function getMyAttendance(
  semester?: string
): Promise<MyAttendanceResponse> {
  const response = await attendanceApi
    .get("my", { searchParams: semester ? { semester } : undefined })
    .json<MyAttendanceDTO>()
  return {
    ...response,
    courses: response.courses.map((course) => ({
      ...course,
      missed_sessions: course.missed_sessions ?? [],
    })),
  }
}

// Sessions out of every 14 a student must attend, as attendance-service
// enforces at finalization (MinTheoryAttendance, MinLabAttendance).
const REQUIRED_PER_14 = { theory: 10, lab: 11 } as const
const STANDARD_SEMESTER_WEEKS = 14
const NEAR_LIMIT = 1

export type SessionType = keyof typeof REQUIRED_PER_14

/**
 * Absences the student can still afford in a session type over the whole
 * semester; negative once the limit is past. min_required in the response is
 * drawn over the sessions held so far, which makes one early absence look
 * like failing, so a warning looks at the semester instead.
 */
export function absencesLeft(
  type: SessionType,
  stats: SessionTypeAttendance,
  totalWeeks: number
): number {
  const weeks = totalWeeks > 0 ? totalWeeks : STANDARD_SEMESTER_WEEKS
  const required = Math.ceil(
    (weeks * REQUIRED_PER_14[type]) / STANDARD_SEMESTER_WEEKS
  )
  return weeks - required - stats.absent_count
}

export interface AbsenceWarning {
  course_id: string
  course_code: string
  session_type: SessionType
  absences_left: number
}

// Session types with at most one absence to spare, tightest first.
export function absenceWarnings(
  courses: CourseAttendanceDetail[]
): AbsenceWarning[] {
  const warnings: AbsenceWarning[] = []
  for (const course of courses) {
    for (const type of ["theory", "lab"] as const) {
      const stats = course[type]
      if (!stats) continue
      const left = absencesLeft(type, stats, course.total_weeks)
      if (left <= NEAR_LIMIT) {
        warnings.push({
          course_id: course.course_id,
          course_code: course.course_code,
          session_type: type,
          absences_left: left,
        })
      }
    }
  }
  return warnings.sort((a, b) => a.absences_left - b.absences_left)
}

export function getAttendanceFinalization(
  courseId: string
): Promise<AttendanceFinalizationStatus> {
  return attendanceApi
    .get(`courses/${courseId}/finalization`)
    .json<AttendanceFinalizationStatus>()
}

/**
 * Finalizes the course's attendance. The backend takes the course's own
 * semester and refuses a second call (409), so this is done once.
 */
export function finalizeAttendance(
  courseId: string
): Promise<AttendanceFinalization> {
  return attendanceApi
    .post(`courses/${courseId}/finalize`)
    .json<AttendanceFinalization>()
}

export function attendanceFinalizationKey(courseId: string) {
  return ["attendance", "finalization", courseId] as const
}

export function useAttendanceFinalization(courseId: string) {
  return useQuery({
    queryKey: attendanceFinalizationKey(courseId),
    queryFn: () => getAttendanceFinalization(courseId),
  })
}
