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

// catalog names a teacher's session day in English ("monday"), and the
// teacher pages print it as is.
const WEEKDAY_LABELS: Record<string, string> = {
  monday: "Pazartesi",
  tuesday: "Salı",
  wednesday: "Çarşamba",
  thursday: "Perşembe",
  friday: "Cuma",
  saturday: "Cumartesi",
  sunday: "Pazar",
}

export function normalizeTeacherCourse(
  course: TeacherCourseDTO
): TeacherCourse {
  return {
    ...course,
    schedule: (course.schedule ?? []).map((session) => ({
      ...session,
      day: WEEKDAY_LABELS[session.day.toLowerCase()] ?? session.day,
    })),
  }
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
