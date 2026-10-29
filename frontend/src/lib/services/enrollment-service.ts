import { HTTPError } from "ky"
import { enrollmentApi } from "@/lib/api-client"
import { normalizeAvailableCourse } from "@/lib/normalize-schedule"
import type {
  AdvisorPendingProgramsResponse,
  AvailableCourse,
  AvailableCourseDTO,
  EnrollmentProgramResponse,
  LatestRejectionResponse,
  MyEnrollmentsResponse,
  OpenSemester,
  RejectionDetail,
} from "@/lib/types"

const TERM_LABELS: Record<string, string> = {
  Fall: "Güz",
  Spring: "Bahar",
  Summer: "Yaz",
}

// "2026-2027-Fall" -> "2026-2027 Güz"
export function semesterLabel(semester: string): string {
  const match = /^(\d{4}-\d{4})-(\w+)$/.exec(semester)
  if (!match) return semester
  return `${match[1]} ${TERM_LABELS[match[2]] ?? match[2]}`
}

// The semester whose window is open; failing that, the current one whose
// program the student can still look at.
export function defaultSemester(semesters: OpenSemester[]): string | null {
  return (
    (semesters.find((s) => s.enrollment_open) ?? semesters[0])?.semester ?? null
  )
}

// The approved program of the active semester; failing that, the latest
// one. Semester ids sort chronologically: Fall < Spring < Summer.
export function currentProgram(
  programs: EnrollmentProgramResponse[],
  activeSemester?: string
): EnrollmentProgramResponse | null {
  const approved = programs.filter((p) => p.status === "approved")
  return (
    approved.find((p) => p.semester === activeSemester) ??
    approved.sort((a, b) => b.semester.localeCompare(a.semester))[0] ??
    null
  )
}

export const enrollmentService = {
  /**
   * Courses the logged-in student may take in a semester
   */
  async getAvailableCourses(semester: string): Promise<AvailableCourse[]> {
    const response = await enrollmentApi
      .get("available-courses", { searchParams: { semester } })
      .json<{ available_courses: AvailableCourseDTO[] | null }>()
    return (response.available_courses ?? []).map(normalizeAvailableCourse)
  },

  /**
   * The logged-in student's most recent program in a semester, if any
   */
  async getLatestProgram(
    semester: string
  ): Promise<EnrollmentProgramResponse | null> {
    const response = await enrollmentApi
      .get("my-enrollments", { searchParams: { semester } })
      .json<MyEnrollmentsResponse>()
    const programs = [...(response.programs ?? [])].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )
    return programs[0] ?? null
  },

  /**
   * The advisor's latest rejection of the student's program in a semester
   */
  async getLatestRejection(semester: string): Promise<RejectionDetail | null> {
    const response = await enrollmentApi
      .get("latest-rejection", { searchParams: { semester } })
      .json<LatestRejectionResponse>()
    return response.has_rejection ? response.latest_rejection : null
  },

  /**
   * Send the chosen courses to the advisor as the semester's program
   */
  async submitProgram(
    semester: string,
    courseIds: string[]
  ): Promise<EnrollmentProgramResponse> {
    return enrollmentApi
      .post("programs", { json: { semester, course_ids: courseIds } })
      .json<EnrollmentProgramResponse>()
  },

  /**
   * Withdraw the semester's pending program. One that is already gone is
   * what the student asked for; any other failure leaves it standing.
   */
  async withdrawProgram(semester: string): Promise<void> {
    try {
      await enrollmentApi.delete("programs", { searchParams: { semester } })
    } catch (err) {
      if (err instanceof HTTPError && err.response.status === 404) return
      throw err
    }
  },

  /**
   * The logged-in student's approved programs across semesters
   */
  async getApprovedPrograms(): Promise<EnrollmentProgramResponse[]> {
    const response = await enrollmentApi
      .get("my-enrollments", { searchParams: { status: "approved" } })
      .json<MyEnrollmentsResponse>()
    return response.programs ?? []
  },

  /**
   * Semesters the logged-in student can enroll in or look back at, oldest first
   */
  async getOpenSemesters(): Promise<OpenSemester[]> {
    return enrollmentApi.get("open-semesters").json<OpenSemester[]>()
  },

  /**
   * Get pending enrollment programs for the logged-in advisor
   */
  async getPendingEnrollments(): Promise<AdvisorPendingProgramsResponse> {
    try {
      // Use the correct endpoint path relative to enrollmentApi's base URL
      // enrollmentApi base is /api/enrollment, so this request goes to /api/enrollment/advisor/pending-programs
      const response = await enrollmentApi
        .get("advisor/pending-programs")
        .json<AdvisorPendingProgramsResponse>()
      return response
    } catch (error) {
      console.error("Failed to fetch pending enrollments:", error)
      throw error
    }
  },

  /**
   * Approve an enrollment program
   */
  async approveEnrollment(programId: string): Promise<void> {
    try {
      await enrollmentApi.post(`advisor/programs/${programId}/approve`)
    } catch (error) {
      console.error(`Failed to approve enrollment ${programId}:`, error)
      throw error
    }
  },

  /**
   * Reject an enrollment program
   */
  async rejectEnrollment(
    programId: string,
    rejectionReason: string
  ): Promise<void> {
    try {
      await enrollmentApi.post(`advisor/programs/${programId}/reject`, {
        json: { rejection_reason: rejectionReason },
      })
    } catch (error) {
      console.error(`Failed to reject enrollment ${programId}:`, error)
      throw error
    }
  },
}

export default enrollmentService
