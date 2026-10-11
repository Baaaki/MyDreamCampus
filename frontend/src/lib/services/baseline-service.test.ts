import { act, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

function json(body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json", ...headers },
  })
}

function stubFetch(respond: () => Response) {
  const fetchSpy = vi.fn<typeof fetch>(async () => respond())
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

function requested(fetchSpy: ReturnType<typeof stubFetch>) {
  return fetchSpy.mock.calls.map(([input]) => {
    const request = input as Request
    return { method: request.method, path: new URL(request.url).pathname }
  })
}

// jsdom's Request has no base URL, so a relative path fails to parse instead
// of resolving against the page the way a browser does. Production builds
// have an empty VITE_API_BASE_URL, so reproduce the browser's resolution.
function resolveRelativeRequests() {
  const NativeRequest = Request
  vi.stubGlobal(
    "Request",
    class extends NativeRequest {
      constructor(input: RequestInfo | URL, init?: RequestInit) {
        super(
          typeof input === "string"
            ? new URL(input, window.location.href)
            : input,
          init
        )
      }
    }
  )
}

async function loadService() {
  vi.resetModules()
  return await import("./baseline-service")
}

describe("baseline service", () => {
  // A relative path resolves against the page URL; at "/" it would work by
  // accident, so sit on the real route like the browser does.
  beforeEach(() => {
    window.history.pushState({}, "", "/system/baseline")
    vi.stubEnv("VITE_API_BASE_URL", "")
    resolveRelativeRequests()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
    vi.resetAllMocks()
    window.history.pushState({}, "", "/")
  })

  it("reads the status from the absolute ops path", async () => {
    const fetchSpy = stubFetch(() => json({ mode: "normal" }))
    const { getBaselineStatus } = await loadService()

    await getBaselineStatus()

    expect(requested(fetchSpy)).toEqual([
      { method: "GET", path: "/api/catalog/admin/ops/status" },
    ])
  })

  it("queues every command under the catalog ops path", async () => {
    const fetchSpy = stubFetch(() => json({ command_id: "cmd-1" }))
    const { beginEdit, saveBaseline, cancelEdit, restoreNow, restoreVersion } =
      await loadService()

    await expect(beginEdit()).resolves.toBe("cmd-1")
    await saveBaseline()
    await cancelEdit()
    await restoreNow()
    await restoreVersion("v2")

    expect(requested(fetchSpy)).toEqual([
      { method: "POST", path: "/api/catalog/admin/ops/begin-edit" },
      { method: "POST", path: "/api/catalog/admin/ops/save" },
      { method: "POST", path: "/api/catalog/admin/ops/cancel-edit" },
      { method: "POST", path: "/api/catalog/admin/ops/restore-now" },
      { method: "POST", path: "/api/catalog/admin/ops/restore/v2" },
    ])
  })

  it("still tracks the editing header through the extended client", async () => {
    stubFetch(() => json({ mode: "editing" }, { "X-System-Editing": "1" }))
    const { getBaselineStatus, useIsSystemEditing } = await loadService()
    const { result } = renderHook(() => useIsSystemEditing())
    expect(result.current).toBe(false)

    await act(async () => {
      await getBaselineStatus()
    })

    expect(result.current).toBe(true)
  })
})
