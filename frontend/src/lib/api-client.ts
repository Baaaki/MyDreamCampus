import ky from "ky"

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ""

/** Read a cookie value by name. Returns null if not found. */
function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp("(^| )" + name + "=([^;]+)"))
  return match ? decodeURIComponent(match[2]) : null
}

/** Attach the CSRF token header (read from the csrf_token cookie) to the request. */
function attachCSRFToken(request: Request): void {
  const csrfToken = getCookie("csrf_token")
  if (csrfToken) {
    request.headers.set("X-CSRF-Token", csrfToken)
  }
}

const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"])

/**
 * A random UUID v4. crypto.randomUUID exists only in secure contexts, and a
 * LAN deployment serves the SPA over plain http:// — getRandomValues works
 * in both.
 */
function randomUUID(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40 // version 4
  bytes[8] = (bytes[8] & 0x3f) | 0x80 // RFC 4122 variant
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

/**
 * Tags a mutation with an Idempotency-Key once. ky's retries and the 401
 * replay below clone the request, so every attempt carries the same key and
 * the backend answers a repeat with the first result instead of doing the
 * work twice (a second reservation, a second grade write).
 */
function attachIdempotencyKey(request: Request): void {
  if (
    MUTATING_METHODS.has(request.method) &&
    !request.headers.has("Idempotency-Key")
  ) {
    request.headers.set("Idempotency-Key", randomUUID())
  }
}

/**
 * - refreshed: new cookies are set.
 * - ended: the backend refused the refresh token (401/403); the session is over.
 * - unavailable: network error or 5xx. Says nothing about the session, so it
 *   must not log the user out.
 */
type RefreshOutcome = "refreshed" | "ended" | "unavailable"

// Single-flight refresh: if multiple in-flight requests hit 401 concurrently,
// they should all wait on one /auth/refresh call rather than racing.
let refreshInFlight: Promise<RefreshOutcome> | null = null

async function refreshAccessToken(): Promise<RefreshOutcome> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = (async (): Promise<RefreshOutcome> => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/auth/refresh`, {
        method: "POST",
        credentials: "include",
      })
      if (res.ok) return "refreshed"
      return res.status === 401 || res.status === 403 ? "ended" : "unavailable"
    } catch {
      return "unavailable"
    } finally {
      // Reset on next tick so concurrent callers in the same microtask batch share this result.
      setTimeout(() => {
        refreshInFlight = null
      }, 0)
    }
  })()
  return refreshInFlight
}

function redirectToLogin(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem("user")
    window.location.href = "/auth/login"
  }
}

const CHANGE_PASSWORD_PATH = "/auth/change-password"

/**
 * Whether the backend refused the request because the user still holds the
 * first-login password. Every service answers that with 403 +
 * FORCE_PASSWORD_CHANGE until the password is changed.
 */
async function isForcedPasswordChange(response: Response): Promise<boolean> {
  if (response.status !== 403) return false
  try {
    const body: unknown = await response.clone().json()
    return (
      typeof body === "object" &&
      body !== null &&
      "code" in body &&
      body.code === "FORCE_PASSWORD_CHANGE"
    )
  } catch {
    return false
  }
}

// Create ky instance with default configuration
const apiClient = ky.create({
  prefix: API_BASE_URL,
  timeout: 30000,
  credentials: "include",
  // POST is not retried: a timed-out POST may well have succeeded, and only
  // the routes behind the Idempotency middleware could absorb a repeat.
  // 500 is not retried either — it is an answer, not a lost request.
  retry: {
    limit: 2,
    methods: ["get", "put", "delete"],
    statusCodes: [408, 502, 503, 504],
  },
  hooks: {
    beforeRequest: [
      async ({ request }) => {
        // Attach CSRF token for state-changing requests
        attachCSRFToken(request)
        attachIdempotencyKey(request)

        // Remove trailing slash from URL (Gin doesn't handle /api/students/ the same as /api/students)
        const url = new URL(request.url)

        if (url.pathname.endsWith("/") && url.pathname !== "/") {
          url.pathname = url.pathname.slice(0, -1)

          // Clone the request to preserve the body (body streams can only be read once)
          const clonedRequest = request.clone()
          const body = await clonedRequest.text()

          return new Request(url.toString(), {
            method: request.method,
            headers: request.headers,
            body: body || undefined,
            credentials: "include",
          })
        }
      },
    ],
    afterResponse: [
      async ({ request, response }) => {
        if (await isForcedPasswordChange(response)) {
          if (
            typeof window !== "undefined" &&
            window.location.pathname !== CHANGE_PASSWORD_PATH
          ) {
            window.location.href = CHANGE_PASSWORD_PATH
          }
          return response
        }

        if (response.status !== 401) return response

        // Don't try to refresh on the auth endpoints themselves —
        // 401 there means the credentials/refresh token are bad.
        const url = new URL(request.url)
        if (
          url.pathname.includes("/auth/login") ||
          url.pathname.includes("/auth/refresh")
        ) {
          return response
        }

        // Avoid infinite retry loops: if we already retried this request, give up.
        if (request.headers.get("X-Refresh-Retry") === "1") {
          redirectToLogin()
          return response
        }

        const outcome = await refreshAccessToken()
        // The refresh endpoint is unreachable, not refusing: keep the session
        // and let the caller see the original error.
        if (outcome === "unavailable") return response

        // Replay the original request with a marker so afterResponse won't
        // loop. Replayed even when this tab's refresh lost: another tab
        // refreshing at the same moment may already hold the new cookies,
        // and the browser sends them with the replay.
        const retryRequest = request.clone()
        retryRequest.headers.set("X-Refresh-Retry", "1")
        attachCSRFToken(retryRequest)
        const retried = await fetch(retryRequest)
        if (outcome === "ended" && retried.status === 401) {
          redirectToLogin()
        }
        return retried
      },
    ],
  },
})

// API clients — one per module; prefixes must match the monolith's route mounts.
export const authApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/auth`,
})
export const staffApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/staff`,
})
export const adminStaffApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/admin-staff`,
})
export const studentApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/students`,
})
export const catalogApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/catalog`,
})
export const semesterApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/semesters`,
})
export const enrollmentApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/enrollment`,
})
export const attendanceApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/attendance`,
})
export const gradesApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/grades`,
})
export const mealApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/meals`,
})
export const paymentApi = apiClient.extend({
  prefix: `${API_BASE_URL}/api/payments`,
})

// API clients without 401 auto-redirect — for admin pages that call
// multiple services in parallel (e.g. system page). The global 401 hook
// would redirect before Promise.allSettled can catch individual failures.
const noRedirectClient = ky.create({
  prefix: API_BASE_URL,
  timeout: 30000,
  credentials: "include",
  retry: { limit: 0 },
  hooks: {
    beforeRequest: [
      ({ request }) => {
        attachCSRFToken(request)
        attachIdempotencyKey(request)
      },
    ],
  },
})

export const gradesApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/grades`,
})
export const enrollmentApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/enrollment`,
})
export const mealApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/meals`,
})
export const catalogApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/catalog`,
})
export const authApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/auth`,
})
export const attendanceApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/attendance`,
})
export const studentApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/students`,
})
export const staffApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/staff`,
})
export const paymentApiSafe = noRedirectClient.extend({
  prefix: `${API_BASE_URL}/api/payments`,
})

// Export the raw ky client for direct use if needed
export { apiClient }
