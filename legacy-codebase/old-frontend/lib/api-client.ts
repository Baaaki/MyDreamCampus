import ky from 'ky';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost';

// Create ky instance with default configuration
const apiClient = ky.create({
  prefixUrl: API_BASE_URL,
  timeout: 30000,
  retry: {
    limit: 2,
    methods: ['get', 'post', 'put', 'delete'],
    statusCodes: [408, 413, 429, 500, 502, 503, 504],
  },
  hooks: {
    beforeRequest: [
      async (request) => {
        // Remove trailing slash from URL (Gin doesn't handle /api/students/ the same as /api/students)
        const url = new URL(request.url);
        let needsNewRequest = false;

        if (url.pathname.endsWith('/') && url.pathname !== '/') {
          url.pathname = url.pathname.slice(0, -1);
          needsNewRequest = true;
        }

        // Add authorization token if exists
        const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;

        if (needsNewRequest) {
          // Clone the request to preserve the body (body streams can only be read once)
          const clonedRequest = request.clone();
          const body = await clonedRequest.text();

          const newRequest = new Request(url.toString(), {
            method: request.method,
            headers: request.headers,
            body: body || undefined,
          });

          if (token) {
            newRequest.headers.set('Authorization', `Bearer ${token}`);
          }
          return newRequest;
        } else if (token) {
          request.headers.set('Authorization', `Bearer ${token}`);
        }
      },
    ],
    afterResponse: [
      async (_request, _options, response) => {
        // Handle 401 Unauthorized - token expired
        if (response.status === 401) {
          // Clear token and redirect to login
          if (typeof window !== 'undefined') {
            // Clear localStorage
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('user');
            // Clear cookies (for middleware)
            document.cookie = 'access_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
            document.cookie = 'user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
            // Redirect to login
            window.location.href = '/auth/login';
          }
        }
        return response;
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
  retry: { limit: 0 },
  hooks: {
    beforeRequest: [
      async (request) => {
        const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`);
        }
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
