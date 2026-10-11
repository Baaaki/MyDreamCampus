import {
  attendanceApiSafe,
  semesterApi,
  staffApi,
  studentApi,
} from "@/lib/api-client"
import type { AdminSessionsResponse } from "@/lib/types"
import { getActiveSemester } from "@/lib/services/system-service"
import { toLocalISODate } from "@/lib/services/meal-service"

// A figure that could not be read stays null so the card shows a dash
// instead of a zero that looks like a real count.
export interface AdminDashboardStats {
  students: number | null
  staff: number | null
  openedCourses: number | null
  todaySessions: number | null
}

interface PagedTotal {
  pagination: { total: number }
}

async function pagedTotal(
  request: Promise<PagedTotal> | PromiseLike<PagedTotal>
): Promise<number> {
  return (await request).pagination.total
}

function valueOf(result: PromiseSettledResult<number>): number | null {
  return result.status === "fulfilled" ? result.value : null
}

export async function getAdminDashboardStats(
  now: Date = new Date()
): Promise<AdminDashboardStats> {
  const today = toLocalISODate(now)

  const [students, staff, openedCourses, todaySessions] =
    await Promise.allSettled([
      pagedTotal(
        studentApi.get("", { searchParams: { limit: 1 } }).json<PagedTotal>()
      ),
      pagedTotal(
        staffApi.get("", { searchParams: { limit: 1 } }).json<PagedTotal>()
      ),
      (async () => {
        const semester = await getActiveSemester()
        // No active semester means nothing is open, which is a real zero.
        if (!semester) return 0
        return pagedTotal(
          semesterApi
            .get(`${encodeURIComponent(semester.name)}/courses`, {
              searchParams: { limit: 1 },
            })
            .json<PagedTotal>()
        )
      })(),
      attendanceApiSafe
        .get("admin/sessions", {
          searchParams: { start_date: today, end_date: today },
        })
        .json<AdminSessionsResponse>()
        .then((res) => res.total),
    ])

  return {
    students: valueOf(students),
    staff: valueOf(staff),
    openedCourses: valueOf(openedCourses),
    todaySessions: valueOf(todaySessions),
  }
}
