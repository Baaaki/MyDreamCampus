import { afterEach, describe, expect, it, vi } from "vitest"
import type { ServiceTimeStatus, TimeStatus } from "@/lib/types"

function status(overrides: Partial<TimeStatus> = {}): TimeStatus {
  return {
    service: "catalog",
    active: true,
    current_time: "2027-09-24T10:00:00Z",
    real_time: "2026-09-24T10:00:00Z",
    offset_seconds: 31_536_000,
    ...overrides,
  }
}

function json(body: unknown, statusCode = 200): Response {
  return new Response(JSON.stringify(body), {
    status: statusCode,
    headers: { "Content-Type": "application/json" },
  })
}

// Request bodies are read at call time: ky consumes the request after fetch
// resolves, so reading it from the recorded call afterwards fails.
const sentBodies: string[] = []

function stubFetch(respond: (url: URL) => Response) {
  sentBodies.length = 0
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const request = input as Request
    sentBodies.push(await request.clone().text())
    return respond(new URL(request.url))
  })
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

async function loadService() {
  vi.resetModules()
  return await import("./system-service")
}

describe("time machine service", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("reads every service's own status endpoint, payment included", async () => {
    const fetchSpy = stubFetch(() => json(status()))
    const { getAllTimeStatuses, SERVICE_KEYS } = await loadService()

    const rows = await getAllTimeStatuses()

    const paths = fetchSpy.mock.calls.map(
      ([input]) => new URL((input as Request).url).pathname
    )
    expect(paths).toEqual([
      "/api/grades/admin/time/status",
      "/api/enrollment/admin/time/status",
      "/api/meals/admin/time/status",
      "/api/catalog/admin/time/status",
      "/api/auth/admin/time/status",
      "/api/attendance/admin/time/status",
      "/api/students/admin/time/status",
      "/api/staff/admin/time/status",
      "/api/payments/admin/time/status",
    ])
    expect(rows.map((r) => r.service)).toEqual(SERVICE_KEYS)
    expect(rows.every((r) => r.status !== null && r.error === null)).toBe(true)
  })

  it("keeps the other services when one fails", async () => {
    stubFetch((url) =>
      url.pathname.startsWith("/api/meals")
        ? json({ error: "Servis kullanılamıyor", code: "UNAVAILABLE" }, 503)
        : json(status())
    )
    const { getAllTimeStatuses } = await loadService()

    const rows = await getAllTimeStatuses()

    const meal = rows.find((r) => r.service === "meal")!
    expect(meal.status).toBeNull()
    expect(meal.error).toBe("Servis kullanılamıyor")
    expect(rows.filter((r) => r.status).length).toBe(8)
  })

  it("simulates with one call to catalog", async () => {
    const fetchSpy = stubFetch(() => json(status()))
    const { simulateTimeAll } = await loadService()

    const result = await simulateTimeAll("2027-09-24T10:00:00.000Z")

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const request = fetchSpy.mock.calls[0]![0] as Request
    expect(request.method).toBe("POST")
    expect(new URL(request.url).pathname).toBe(
      "/api/catalog/admin/time/simulate"
    )
    expect(JSON.parse(sentBodies[0]!)).toEqual({
      time: "2027-09-24T10:00:00.000Z",
    })
    expect(result.active).toBe(true)
  })

  it("resets with one call to catalog", async () => {
    const fetchSpy = stubFetch(() => json(status({ active: false })))
    const { resetTimeAll } = await loadService()

    await resetTimeAll()

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const request = fetchSpy.mock.calls[0]![0] as Request
    expect(request.method).toBe("POST")
    expect(new URL(request.url).pathname).toBe("/api/catalog/admin/time/reset")
  })
})

describe("clock offset spread", () => {
  const row = (s: TimeStatus | null): ServiceTimeStatus => ({
    service: s?.service ?? "x",
    label: "x",
    status: s,
    error: s ? null : "Hata",
  })

  it("measures each offset on the service itself", async () => {
    const { clockOffsetMs } = await loadService()
    expect(clockOffsetMs(status())).toBe(31_536_000_000)
  })

  it("is the largest offset difference between answering services", async () => {
    const { clockOffsetSpreadMs } = await loadService()
    const spread = clockOffsetSpreadMs([
      row(status()),
      row(
        status({
          current_time: "2027-09-24T10:00:03Z",
          real_time: "2026-09-24T10:00:00Z",
        })
      ),
      row(null),
    ])
    expect(spread).toBe(3000)
  })

  it("counts a service still on real time as drift", async () => {
    const { clockOffsetSpreadMs } = await loadService()
    const real = status({
      active: false,
      current_time: "2026-09-24T10:00:00Z",
      real_time: "2026-09-24T10:00:00Z",
      offset_seconds: 0,
    })
    expect(clockOffsetSpreadMs([row(status()), row(real)])).toBe(31_536_000_000)
  })

  it("is null with fewer than two answers", async () => {
    const { clockOffsetSpreadMs } = await loadService()
    expect(clockOffsetSpreadMs([row(status()), row(null)])).toBeNull()
  })
})

describe("academic periods management", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("lists grades periods from catalog with type=grading", async () => {
    const fetchSpy = stubFetch(() => json([]))
    const { listGradesPeriods } = await loadService()

    await listGradesPeriods("2025-2026-Fall")

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const request = fetchSpy.mock.calls[0]![0] as Request
    const url = new URL(request.url)
    expect(url.pathname).toBe("/api/catalog/admin/periods")
    expect(url.searchParams.get("type")).toBe("grading")
    expect(url.searchParams.get("semester")).toBe("2025-2026-Fall")
  })

  it("lists simple periods from catalog with requested type", async () => {
    const fetchSpy = stubFetch(() => json([]))
    const { listSimplePeriods } = await loadService()

    await listSimplePeriods("enrollment", "2025-2026-Fall")

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const request = fetchSpy.mock.calls[0]![0] as Request
    const url = new URL(request.url)
    expect(url.pathname).toBe("/api/catalog/admin/periods")
    expect(url.searchParams.get("type")).toBe("enrollment")
    expect(url.searchParams.get("semester")).toBe("2025-2026-Fall")
  })

  it("creates a period in catalog with period_type", async () => {
    const fetchSpy = stubFetch(() => json({ id: "p1" }))
    const { createGradesPeriod, createSimplePeriod } = await loadService()

    await createGradesPeriod({
      semester: "2025-2026-Fall",
      period_start: "2025-10-01T00:00:00Z",
      period_end: "2025-11-01T00:00:00Z",
    })
    await createSimplePeriod("attendance", {
      semester: "2025-2026-Fall",
      period_start: "2025-10-01T00:00:00Z",
      period_end: "2025-11-01T00:00:00Z",
    })

    expect(fetchSpy).toHaveBeenCalledTimes(2)
    const req1 = fetchSpy.mock.calls[0]![0] as Request
    const req2 = fetchSpy.mock.calls[1]![0] as Request
    expect(new URL(req1.url).pathname).toBe("/api/catalog/admin/periods")
    expect(new URL(req2.url).pathname).toBe("/api/catalog/admin/periods")
    expect(JSON.parse(sentBodies[0]!).period_type).toBe("grading")
    expect(JSON.parse(sentBodies[1]!).period_type).toBe("attendance")
  })
})

describe("audit log service", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("queries catalog admin/audit-log with filter query params", async () => {
    const fetchSpy = stubFetch(() => json({ data: [], total: 0 }))
    const { listAuditLog } = await loadService()

    await listAuditLog({
      service: "catalog",
      action: "semester.activated",
      actor_id: "00000000-0000-0000-0000-000000000001",
      limit: 20,
      offset: 40,
    })

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const request = fetchSpy.mock.calls[0]![0] as Request
    const url = new URL(request.url)
    expect(url.pathname).toBe("/api/catalog/admin/audit-log")
    expect(url.searchParams.get("service")).toBe("catalog")
    expect(url.searchParams.get("action")).toBe("semester.activated")
    expect(url.searchParams.get("actor_id")).toBe(
      "00000000-0000-0000-0000-000000000001"
    )
    expect(url.searchParams.get("limit")).toBe("20")
    expect(url.searchParams.get("offset")).toBe("40")
  })
})
