import { attendanceApi } from "@/lib/api-client"
import type { MyAttendanceResponse } from "@/lib/types"

/**
 * The logged-in student's attendance per course. Without a semester the
 * backend answers for the active one.
 */
export async function getMyAttendance(
  semester?: string
): Promise<MyAttendanceResponse> {
  return attendanceApi
    .get("my", { searchParams: semester ? { semester } : undefined })
    .json<MyAttendanceResponse>()
}
