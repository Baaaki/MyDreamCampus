import ky from 'ky';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

/** Read a cookie value by name. Returns null if not found. */
function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'));
  return match ? decodeURIComponent(match[2]) : null;
}

/** Attach the CSRF token header (read from the csrf_token cookie) to the request. */
function attachCSRFToken(request: Request): void {
  const csrfToken = getCookie('csrf_token');
  if (csrfToken) {
    request.headers.set('X-CSRF-Token', csrfToken);
  }
}

// Single-flight refresh: if multiple in-flight requests hit 401 concurrently,
// they should all wait on one /auth/refresh call rather than racing.
let refreshInFlight: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      });
      return res.ok;
    } catch {
      return false;
    } finally {
      // Reset on next tick so concurrent callers in the same microtask batch share this result.
      setTimeout(() => {
        refreshInFlight = null;
      }, 0);
    }
  })();
  return refreshInFlight;
}

// Create ky instance with default configuration
const apiClient = ky.create({
  prefixUrl: API_BASE_URL,
  timeout: 30000,
  credentials: 'include',
  retry: {
    limit: 2,
    methods: ['get', 'post', 'put', 'delete'],
    statusCodes: [408, 413, 500, 502, 503, 504],
  },
  hooks: {
    beforeRequest: [
      async (request) => {
        // Attach CSRF token for state-changing requests
        attachCSRFToken(request);

        // Remove trailing slash from URL (Gin doesn't handle /api/students/ the same as /api/students)
        const url = new URL(request.url);

        if (url.pathname.endsWith('/') && url.pathname !== '/') {
          url.pathname = url.pathname.slice(0, -1);

          // Clone the request to preserve the body (body streams can only be read once)
          const clonedRequest = request.clone();
          const body = await clonedRequest.text();

          return new Request(url.toString(), {
            method: request.method,
            headers: request.headers,
            body: body || undefined,
            credentials: 'include',
          });
        }
      },
    ],
    afterResponse: [
      async (request, _options, response) => {
        if (response.status !== 401) return response;

        // Don't try to refresh on the auth endpoints themselves —
        // 401 there means the credentials/refresh token are bad.
        const url = new URL(request.url);
        if (url.pathname.includes('/auth/login') || url.pathname.includes('/auth/refresh')) {
          return response;
        }

        // Avoid infinite retry loops: if we already retried this request, give up.
        if (request.headers.get('X-Refresh-Retry') === '1') {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('user');
            window.location.href = '/auth/login';
          }
          return response;
        }

        const refreshed = await refreshAccessToken();
        if (!refreshed) {
          if (typeof window !== 'undefined') {
            localStorage.removeItem('user');
            window.location.href = '/auth/login';
          }
          return response;
        }

        // Replay the original request with a marker so afterResponse won't loop.
        const retryRequest = request.clone();
        retryRequest.headers.set('X-Refresh-Retry', '1');
        attachCSRFToken(retryRequest);
        return fetch(retryRequest);
      },
    ],
  },
});

// API clients - URLs must match Traefik routing in dynamic.yml
export const authApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/auth` });
export const staffApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/staff` });
export const adminStaffApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/admin-staff` });
export const studentApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/students` });
export const catalogApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/catalog` });
export const semesterApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/semesters` });
export const enrollmentApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/enrollment` });
export const attendanceApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/attendance` });
export const gradesApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/grades` });
export const mealApi = apiClient.extend({ prefixUrl: `${API_BASE_URL}/api/meals` });

// API clients without 401 auto-redirect — for admin pages that call
// multiple services in parallel (e.g. system page). The global 401 hook
// would redirect before Promise.allSettled can catch individual failures.
const noRedirectClient = ky.create({
  prefixUrl: API_BASE_URL,
  timeout: 30000,
  credentials: 'include',
  retry: { limit: 0 },
  hooks: {
    beforeRequest: [
      (request) => {
        attachCSRFToken(request);
      },
    ],
  },
});

export const gradesApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/grades` });
export const enrollmentApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/enrollment` });
export const mealApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/meals` });
export const catalogApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/catalog` });
export const authApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/auth` });
export const attendanceApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/attendance` });
export const studentApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/students` });
export const staffApiSafe = noRedirectClient.extend({ prefixUrl: `${API_BASE_URL}/api/staff` });

// Export the raw ky client for direct use if needed
export { apiClient };
