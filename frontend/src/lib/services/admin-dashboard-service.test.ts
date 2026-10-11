import { afterEach, describe, expect, it, vi } from "vitest"

function json(body: unknown, statusCode = 200): Response {
  return new Response(JSON.stringify(body), {
    status: statusCode,
    headers: { "Content-Type": "application/json" },
  })
}

function stubFetch(respond: (url: URL) => Response) {
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) =>
    respond(new URL((input as Request).url))
  )
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

async function loadService() {
  vi.resetModules()
  return await import("./admin-dashboard-service")
}

const paged = (total: number) => ({ data: [], pagination: { total } })

describe("getAdminDashboardStats", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("reads the counts from the backend and asks for today's sessions", async () => {
    const fetchSpy = stubFetch((url) => {
      if (url.pathname === "/api/students") return json(paged(120))
      if (url.pathname === "/api/staff") return json(paged(30))
      if (url.pathname === "/api/catalog/admin/semesters/active")
        return json({ name: "2026-2027-Fall" })
      if (url.pathname === "/api/semesters/2026-2027-Fall/courses")
        return json(paged(45))
      if (url.pathname === "/api/attendance/admin/sessions")
        return json({ sessions: [], total: 7 })
      return json({}, 404)
    })
    const { getAdminDashboardStats } = await loadService()

    const stats = await getAdminDashboardStats(new Date(2026, 9, 12, 0, 30))

    expect(stats).toEqual({
      students: 120,
      staff: 30,
      openedCourses: 45,
      todaySessions: 7,
    })
    const sessionsUrl = fetchSpy.mock.calls
      .map(([input]) => new URL((input as Request).url))
      .find((u) => u.pathname === "/api/attendance/admin/sessions")
    expect(sessionsUrl?.searchParams.get("start_date")).toBe("2026-10-12")
    expect(sessionsUrl?.searchParams.get("end_date")).toBe("2026-10-12")
  })

  it("leaves a figure null when its request fails", async () => {
    stubFetch((url) => {
      if (url.pathname === "/api/students") return json({}, 500)
      if (url.pathname === "/api/staff") return json(paged(30))
      if (url.pathname === "/api/catalog/admin/semesters/active")
        return json({}, 404)
      return json({ sessions: [], total: 0 })
    })
    const { getAdminDashboardStats } = await loadService()

    const stats = await getAdminDashboardStats()

    expect(stats.students).toBeNull()
    expect(stats.staff).toBe(30)
    expect(stats.openedCourses).toBe(0)
  })

  it("keeps the course count unknown when the active semester cannot be read", async () => {
    stubFetch((url) => {
      if (url.pathname === "/api/catalog/admin/semesters/active")
        return json({}, 500)
      return json({ sessions: [], total: 0, ...paged(1) })
    })
    const { getAdminDashboardStats } = await loadService()

    expect((await getAdminDashboardStats()).openedCourses).toBeNull()
  })
})
