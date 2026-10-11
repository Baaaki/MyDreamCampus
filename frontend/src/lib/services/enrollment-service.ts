import { enrollmentApi } from "@/lib/api-client"
import type { AdvisorPendingProgramsResponse, OpenSemester } from "@/lib/types"

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

export const enrollmentService = {
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
