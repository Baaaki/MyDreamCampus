import type {
  AvailableCourse,
  AvailableCourseDTO,
  SemesterCourse,
  SemesterCourseDTO,
  TeacherCourse,
  TeacherCourseDTO,
} from "@/lib/types"

// Go marshals an empty slice as JSON null, and a course without a weekly
// schedule (the seed has none) arrives that way. Normalize where the response
// is read so the rest of the UI can iterate the schedule unconditionally.

export function normalizeTeacherCourse(
  course: TeacherCourseDTO
): TeacherCourse {
  return { ...course, schedule: course.schedule ?? [] }
}

export function normalizeSemesterCourse(
  course: SemesterCourseDTO
): SemesterCourse {
  return { ...course, schedule_sessions: course.schedule_sessions ?? [] }
}

export function normalizeAvailableCourse(
  course: AvailableCourseDTO
): AvailableCourse {
  return { ...course, schedule_sessions: course.schedule_sessions ?? [] }
}
