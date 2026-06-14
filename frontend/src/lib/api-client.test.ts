import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Helper that re-imports the module fresh after manipulating cookies/fetch
async function loadClient() {
  vi.resetModules();
  return await import("./api-client");
}

describe("api-client - CSRF token attachment", () => {
  beforeEach(() => {
    document.cookie = "csrf_token=secret-token-abc; path=/";
    // Reset window.fetch
    vi.stubGlobal("fetch", vi.fn(async () => new Response("{}", { status: 200 })));
  });

  afterEach(() => {
    document.cookie = "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
    vi.unstubAllGlobals();
    vi.resetAllMocks();
  });

  it("reads X-CSRF-Token from csrf_token cookie on state-changing requests", async () => {
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);

    const { apiClient } = await loadClient();
    await apiClient.post("api/echo", { json: { x: 1 } });

    const call = fetchSpy.mock.calls[0];
    expect(call).toBeDefined();
    const req = call![0] as Request;
    expect(req.headers.get("X-CSRF-Token")).toBe("secret-token-abc");
  });

  it("omits X-CSRF-Token when csrf_token cookie missing", async () => {
    document.cookie = "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
    const fetchSpy = vi.fn(async () => new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);

    const { apiClient } = await loadClient();
    await apiClient.post("api/echo");

    const req = fetchSpy.mock.calls[0]![0] as Request;
    expect(req.headers.get("X-CSRF-Token")).toBeNull();
  });
});

describe("api-client - 401 refresh logic", () => {
  beforeEach(() => {
    document.cookie = "csrf_token=t; path=/";
    localStorage.setItem("user", JSON.stringify({ role: "student" }));
    // window.location is read-only in jsdom by default; mock it
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...window.location, href: "http://localhost/" },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetAllMocks();
    localStorage.clear();
  });

  it("does NOT trigger refresh on /auth/login 401 — returns the 401 directly", async () => {
    const refreshFetch = vi.fn();
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : input.url;
      if (url.includes("/auth/refresh")) {
        refreshFetch();
        return new Response("{}", { status: 200 });
      }
      return new Response('{"error":"INVALID_CREDENTIALS"}', { status: 401 });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { authApi } = await loadClient();
    const res = await authApi.post("login", { json: { email: "a", password: "b" } }).catch((e) => e.response);

    expect(res.status).toBe(401);
    expect(refreshFetch).not.toHaveBeenCalled();
  });

  it("retries the original request once after a successful refresh", async () => {
    let originalCall = 0;
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (url.includes("/auth/refresh")) {
        return new Response("{}", { status: 200 });
      }
      // First call: 401, second call (the retry): 200
      originalCall++;
      if (originalCall === 1) return new Response("{}", { status: 401 });
      return new Response('{"data":"ok"}', { status: 200 });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { studentApi } = await loadClient();
    const res = await studentApi.get("me");
    expect(res.status).toBe(200);
    expect(originalCall).toBe(2);
  });

  it("refresh failure triggers logout (clears localStorage + redirects)", async () => {
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (url.includes("/auth/refresh")) {
        return new Response("{}", { status: 401 });
      }
      return new Response("{}", { status: 401 });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { studentApi } = await loadClient();
    await studentApi.get("me").catch(() => {});

    expect(localStorage.getItem("user")).toBeNull();
    expect(window.location.href).toContain("/auth/login");
  });

  it("does not loop: a request marked X-Refresh-Retry won't refresh again", async () => {
    let refreshCalls = 0;
    const fetchSpy = vi.fn(async (input: Request | string) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (url.includes("/auth/refresh")) {
        refreshCalls++;
        return new Response("{}", { status: 200 });
      }
      return new Response("{}", { status: 401 });
    });
    vi.stubGlobal("fetch", fetchSpy);

    const { studentApi } = await loadClient();
    await studentApi.get("me").catch(() => {});

    // refresh called once; second 401 (after retry) must not trigger another
    expect(refreshCalls).toBe(1);
  });
});
