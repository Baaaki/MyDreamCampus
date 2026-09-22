import { afterEach, describe, expect, it, vi } from "vitest"
import type { AdminStaffRecord } from "@/lib/types"

const record: AdminStaffRecord = {
  id: "0190a1b2-0000-7000-8000-000000000001",
  email: "ali.vural@uni.edu.tr",
  title: "",
  first_name: "Ali",
  last_name: "Vural",
  faculty: "Mühendislik Fakültesi",
  department: "",
  phone: "+90 232 301 1001",
  profile_image_url: "",
  position: "Fakülte Sekreteri",
  job_description: "İdari işler",
  responsibilities: ["Yazışmalar"],
  working_hours: "08:30 - 17:30",
  office_location: "Dekanlık, 101",
  start_date: "2015-03-01",
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}

// Request bodies are read at call time: ky consumes the request after fetch
// resolves, so reading it from the recorded call afterwards fails.
const sentBodies: string[] = []

function stubFetch(body: unknown) {
  sentBodies.length = 0
  const fetchSpy = vi.fn(
    async (input: RequestInfo | URL, _init?: RequestInit) => {
      if (input instanceof Request) sentBodies.push(await input.clone().text())
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    }
  )
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

async function loadService() {
  vi.resetModules()
  return await import("./admin-staff-service")
}

describe("adminStaffService", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("lists a faculty through /api/admin-staff?faculty=", async () => {
    const fetchSpy = stubFetch({ data: [record] })
    const { adminStaffService } = await loadService()

    const rows = await adminStaffService.list("Mühendislik Fakültesi")

    const url = new URL((fetchSpy.mock.calls[0]![0] as Request).url)
    expect(url.pathname).toBe("/api/admin-staff")
    expect(url.searchParams.get("faculty")).toBe("Mühendislik Fakültesi")
    expect(rows).toEqual([record])
  })

  it("maps a record to the editor's camelCase profile", async () => {
    stubFetch(record)
    const { adminStaffService } = await loadService()

    const profile = await adminStaffService.get(record.id)

    expect(profile).toMatchObject({
      id: record.id,
      firstName: "Ali",
      lastName: "Vural",
      jobDescription: "İdari işler",
      officeLocation: "Dekanlık, 101",
      startDate: "2015-03-01",
      profileImage: undefined,
      department: undefined,
    })
  })

  it("updates with a PUT to the record's id and drops empty responsibility rows", async () => {
    const fetchSpy = stubFetch(record)
    const { adminStaffService, toAdminStaffProfile } = await loadService()
    const edited = {
      ...toAdminStaffProfile(record),
      responsibilities: ["Yazışmalar", "  ", ""],
    }

    await adminStaffService.update(record.id, edited)

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.method).toBe("PUT")
    expect(new URL(req.url).pathname).toBe(`/api/admin-staff/${record.id}`)
    const sent = JSON.parse(sentBodies[0]!)
    expect(sent).toMatchObject({
      first_name: "Ali",
      office_location: "Dekanlık, 101",
      profile_image_url: "",
      responsibilities: ["Yazışmalar"],
    })
  })
})
