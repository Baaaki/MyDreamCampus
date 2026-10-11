// @vitest-environment node
// jsdom's File cannot cross into Node's Request (vitest's bridge reads a
// buffer this jsdom no longer keeps), so the multipart body is checked with
// Node's own FormData and File.
import { afterEach, describe, expect, it, vi } from "vitest"

const JOB_ID = "0190a1b2-0000-7000-8000-0000000000aa"

// ky consumes the request after fetch resolves, so the multipart body is read
// at call time.
const sentBodies: string[] = []

function stubFetch(body: unknown, status = 200) {
  sentBodies.length = 0
  // api-client reads the CSRF cookie on every mutating request.
  vi.stubGlobal("document", { cookie: "" })
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const request = input as Request
    if (request.method === "POST") sentBodies.push(await request.clone().text())
    return new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    })
  })
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

async function loadService() {
  vi.resetModules()
  return await import("./student-import-service")
}

describe("studentImportService", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("uploads the file as multipart to /api/students/bulk-import", async () => {
    const fetchSpy = stubFetch({ job_id: JOB_ID, status: "pending" }, 202)
    const { studentImportService } = await loadService()
    const file = new File(["student_number\n"], "ogrenciler.csv", {
      type: "text/csv",
    })

    const jobId = await studentImportService.start(file)

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.method).toBe("POST")
    expect(new URL(req.url).pathname).toBe("/api/students/bulk-import")
    expect(req.headers.get("Content-Type")).toMatch(/^multipart\/form-data/)
    expect(sentBodies[0]).toContain('name="file"; filename="ogrenciler.csv"')
    expect(jobId).toBe(JOB_ID)
  })

  it("reads a job from /api/students/bulk-import/:job_id", async () => {
    const fetchSpy = stubFetch({ job_id: JOB_ID, status: "completed" })
    const { studentImportService } = await loadService()

    const job = await studentImportService.get(JOB_ID)

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(new URL(req.url).pathname).toBe(
      `/api/students/bulk-import/${JOB_ID}`
    )
    expect(job.status).toBe("completed")
  })

  it("treats only completed and failed as finished", async () => {
    const { isImportFinished } = await loadService()

    expect(isImportFinished("completed")).toBe(true)
    expect(isImportFinished("failed")).toBe(true)
    expect(isImportFinished("pending")).toBe(false)
    expect(isImportFinished("processing")).toBe(false)
    expect(isImportFinished(undefined)).toBe(false)
  })
})
