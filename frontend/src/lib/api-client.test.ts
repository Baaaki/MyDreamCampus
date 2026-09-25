import { HTTPError } from "ky"
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { act, renderHook } from "@testing-library/react"

// Helper that re-imports the module fresh after manipulating cookies/fetch
async function loadClient() {
  vi.resetModules()
  return await import("./api-client")
}

describe("api-client - CSRF token attachment", () => {
  beforeEach(() => {
    document.cookie = "csrf_token=secret-token-abc; path=/"
    // Reset window.fetch
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("{}", { status: 200 }))
    )
  })

  afterEach(() => {
    document.cookie =
      "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/"
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("reads X-CSRF-Token from csrf_token cookie on state-changing requests", async () => {
    const fetchSpy = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response("{}", { status: 200 })
    )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await apiClient.post("api/echo", { json: { x: 1 } })

    const call = fetchSpy.mock.calls[0]
    expect(call).toBeDefined()
    const req = call![0] as Request
    expect(req.headers.get("X-CSRF-Token")).toBe("secret-token-abc")
  })

  it("omits X-CSRF-Token when csrf_token cookie missing", async () => {
    document.cookie =
      "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/"
    const fetchSpy = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response("{}", { status: 200 })
    )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await apiClient.post("api/echo")

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.headers.get("X-CSRF-Token")).toBeNull()
  })
})

describe("api-client - 401 refresh logic", () => {
  beforeEach(() => {
    document.cookie = "csrf_token=t; path=/"
    localStorage.setItem("user", JSON.stringify({ role: "student" }))
    // window.location is read-only in jsdom by default; mock it
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...window.location, href: "http://localhost/" },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
    localStorage.clear()
  })

  it("does NOT trigger refresh on /auth/login 401 — returns the 401 directly", async () => {
    const refreshFetch = vi.fn()
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : input.url
      if (url.includes("/auth/refresh")) {
        refreshFetch()
        return new Response("{}", { status: 200 })
      }
      return new Response('{"error":"INVALID_CREDENTIALS"}', { status: 401 })
    })
    vi.stubGlobal("fetch", fetchSpy)

    const { authApi } = await loadClient()
    const res = await authApi
      .post("login", { json: { email: "a", password: "b" } })
      .catch((e) => e.response)

    expect(res.status).toBe(401)
    expect(refreshFetch).not.toHaveBeenCalled()
  })

  it("retries the original request once after a successful refresh", async () => {
    let originalCall = 0
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url
      if (url.includes("/auth/refresh")) {
        return new Response("{}", { status: 200 })
      }
      // First call: 401, second call (the retry): 200
      originalCall++
      if (originalCall === 1) return new Response("{}", { status: 401 })
      return new Response('{"data":"ok"}', { status: 200 })
    })
    vi.stubGlobal("fetch", fetchSpy)

    const { studentApi } = await loadClient()
    const res = await studentApi.get("me")
    expect(res.status).toBe(200)
    expect(originalCall).toBe(2)
  })

  it("refresh failure triggers logout (clears localStorage + redirects)", async () => {
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url
      if (url.includes("/auth/refresh")) {
        return new Response("{}", { status: 401 })
      }
      return new Response("{}", { status: 401 })
    })
    vi.stubGlobal("fetch", fetchSpy)

    const { studentApi } = await loadClient()
    await studentApi.get("me").catch(() => {})

    expect(localStorage.getItem("user")).toBeNull()
    expect(window.location.href).toContain("/auth/login")
  })

  it("replays once after a lost refresh and keeps the session if another tab rotated the cookies", async () => {
    let originalCall = 0
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url
      if (url.includes("/auth/refresh")) {
        return new Response("{}", { status: 401 })
      }
      originalCall++
      if (originalCall === 1) return new Response("{}", { status: 401 })
      return new Response('{"data":"ok"}', { status: 200 })
    })
    vi.stubGlobal("fetch", fetchSpy)

    const { studentApi } = await loadClient()
    const res = await studentApi.get("me")

    expect(res.status).toBe(200)
    expect(originalCall).toBe(2)
    expect(localStorage.getItem("user")).not.toBeNull()
    expect(window.location.href).not.toContain("/auth/login")
  })

  it.each([
    ["a 503", async () => new Response("{}", { status: 503 })],
    [
      "a network error",
      async () => {
        throw new TypeError("Failed to fetch")
      },
    ],
  ])(
    "keeps the session when the refresh gets %s",
    async (_label, refreshReply) => {
      const fetchSpy = vi.fn(async (input: Request | string) => {
        const url = typeof input === "string" ? input : (input as Request).url
        if (url.includes("/auth/refresh")) return refreshReply()
        return new Response("{}", { status: 401 })
      })
      vi.stubGlobal("fetch", fetchSpy)

      const { studentApi } = await loadClient()
      const err = await studentApi.get("me").catch((e: unknown) => e)

      expect(err).toBeInstanceOf(HTTPError)
      expect((err as HTTPError).response.status).toBe(401)
      expect(localStorage.getItem("user")).not.toBeNull()
      expect(window.location.href).not.toContain("/auth/login")
    }
  )

  it("does not loop: a request marked X-Refresh-Retry won't refresh again", async () => {
    let refreshCalls = 0
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url
      if (url.includes("/auth/refresh")) {
        refreshCalls++
        return new Response("{}", { status: 200 })
      }
      return new Response("{}", { status: 401 })
    })
    vi.stubGlobal("fetch", fetchSpy)

    const { studentApi } = await loadClient()
    await studentApi.get("me").catch(() => {})

    // refresh called once; second 401 (after retry) must not trigger another
    expect(refreshCalls).toBe(1)
  })
})

describe("api-client - forced password change", () => {
  beforeEach(() => {
    Object.defineProperty(window, "location", {
      writable: true,
      value: {
        ...window.location,
        pathname: "/dashboard",
        href: "http://localhost/dashboard",
      },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("redirects to the change-password page on 403 FORCE_PASSWORD_CHANGE", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            '{"error":"Devam etmek için şifrenizi değiştirmeniz gerekiyor","code":"FORCE_PASSWORD_CHANGE"}',
            { status: 403, headers: { "Content-Type": "application/json" } }
          )
      )
    )

    const { studentApi } = await loadClient()
    await studentApi.get("me").catch(() => {})

    expect(window.location.href).toBe("/auth/change-password")
  })

  it("leaves other 403 responses alone", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response('{"error":"Yetkiniz yok","code":"FORBIDDEN"}', {
            status: 403,
            headers: { "Content-Type": "application/json" },
          })
      )
    )

    const { studentApi } = await loadClient()
    await studentApi.get("me").catch(() => {})

    expect(window.location.href).toBe("http://localhost/dashboard")
  })
})

describe("api-client - idempotency and retries", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("tags a mutation with a UUID Idempotency-Key", async () => {
    const fetchSpy = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response("{}", { status: 200 })
    )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await apiClient.post("api/meals/reservations", { json: { menu: 1 } })

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.headers.get("Idempotency-Key")).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
    )
  })

  it("does not tag reads", async () => {
    const fetchSpy = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response("{}", { status: 200 })
    )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await apiClient.get("api/meals/menus")

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.headers.get("Idempotency-Key")).toBeNull()
  })

  it("never retries a POST that failed with 503", async () => {
    const fetchSpy = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response("{}", { status: 503 })
    )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await expect(
      apiClient.post("api/meals/reservations", { json: {} })
    ).rejects.toThrow()

    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })

  it("retries a PUT with the same Idempotency-Key", async () => {
    const fetchSpy = vi
      .fn(
        async (_input: RequestInfo | URL, _init?: RequestInit) =>
          new Response("{}", { status: 200 })
      )
      .mockResolvedValueOnce(new Response("{}", { status: 503 }))
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    await apiClient.put("api/students/1", {
      json: {},
      retry: {
        limit: 1,
        methods: ["put"],
        statusCodes: [503],
        backoffLimit: 1,
      },
    })

    expect(fetchSpy).toHaveBeenCalledTimes(2)
    const first = fetchSpy.mock.calls[0]![0] as Request
    const second = fetchSpy.mock.calls[1]![0] as Request
    expect(second.headers.get("Idempotency-Key")).toBe(
      first.headers.get("Idempotency-Key")
    )
  })
})

describe("api-client - system editing notice", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetAllMocks()
  })

  it("clears the notice once responses stop carrying X-System-Editing", async () => {
    const fetchSpy = vi
      .fn(
        async (_input: RequestInfo | URL, _init?: RequestInit) =>
          new Response("{}", { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response("{}", {
          status: 200,
          headers: { "X-System-Editing": "1" },
        })
      )
    vi.stubGlobal("fetch", fetchSpy)

    const { apiClient } = await loadClient()
    const { useIsSystemEditing } = await import("./services/baseline-service")
    const { result } = renderHook(() => useIsSystemEditing())

    await act(async () => {
      await apiClient.get("api/echo")
    })
    expect(result.current).toBe(true)

    await act(async () => {
      await apiClient.get("api/echo")
    })
    expect(result.current).toBe(false)
  })
})
