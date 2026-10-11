import { HTTPError } from "ky"
import {
  attendanceApiSafe,
  catalogApiSafe,
  semesterApi,
  staffApi,
  studentApi,
} from "@/lib/api-client"
import type { AdminSessionsResponse, Semester } from "@/lib/types"
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

// getActiveSemester folds every failure into "no semester"; here only a 404
// means that, anything else must stay unknown rather than become a zero.
async function activeSemesterName(): Promise<string | null> {
  try {
    const semester = await catalogApiSafe
      .get("admin/semesters/active")
      .json<Semester>()
    return semester.name
  } catch (err) {
    if (err instanceof HTTPError && err.response.status === 404) return null
    throw err
  }
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
        const semester = await activeSemesterName()
        // No active semester means nothing is open, which is a real zero.
        if (!semester) return 0
        return pagedTotal(
          semesterApi
            .get(`${encodeURIComponent(semester)}/courses`, {
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
