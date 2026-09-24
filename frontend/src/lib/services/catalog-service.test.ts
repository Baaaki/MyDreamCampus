import { afterEach, describe, expect, it, vi } from "vitest"
import type { Faculty } from "@/lib/types"

const sampleFaculty: Faculty = {
  id: "fac-muhendislik",
  name: "Mühendislik Fakültesi",
  code: "MUH",
  departments: [
    {
      id: "dept-bilgisayar-muhendisligi",
      name: "Bilgisayar Mühendisliği",
      facultyId: "fac-muhendislik",
      code: "CENG",
    },
  ],
}

function stubFetch(body: unknown) {
  const fetchSpy = vi.fn(
    async (input: RequestInfo | URL, _init?: RequestInit) => {
      void input
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
  return await import("./catalog-service")
}

describe("catalogService.getFaculties", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("fetches faculties from /api/catalog/faculties", async () => {
    const fetchSpy = stubFetch({ data: [sampleFaculty] })
    const { catalogService } = await loadService()

    const faculties = await catalogService.getFaculties()

    const url = new URL((fetchSpy.mock.calls[0]![0] as Request).url)
    expect(url.pathname).toBe("/api/catalog/faculties")
    expect(faculties).toEqual([sampleFaculty])
  })
})
