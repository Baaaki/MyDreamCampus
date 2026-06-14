import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Helper function to check if JWT token is expired
function isTokenExpired(token: string): boolean {
  try {
    // JWT format: header.payload.signature
    const parts = token.split('.');
    if (parts.length !== 3) return true;

    // Decode payload (base64url)
    const payload = parts[1];
    const decoded = JSON.parse(
      Buffer.from(payload.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString()
    );

    // Check expiry
    if (!decoded.exp) return true;

    // exp is in seconds, Date.now() is in milliseconds
    const now = Math.floor(Date.now() / 1000);
    return decoded.exp < now;
  } catch {
    // If we can't decode, consider it expired
    return true;
  }
}

// Public routes - no authentication required
const publicRoutes = ['/auth/login', '/auth/forgot-password'];

// Auth routes - redirect to dashboard if already logged in
const authRoutes = ['/auth/login'];

// Admin-only routes (requires admin or teacher role)
const adminRoutes = ['/dashboard', '/staff', '/students', '/catalog', '/enrollment', '/semester-courses', '/settings', '/meal'];

// Student-only routes
const studentRoutes = ['/student/dashboard', '/student/enrollment', '/student/attendance', '/student/grades', '/student/cafeteria'];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Get token from cookies
  const token = request.cookies.get('access_token')?.value;
  const userCookie = request.cookies.get('user')?.value;

  const isPublicRoute = publicRoutes.some((route) => pathname.startsWith(route));
  const isAuthRoute = authRoutes.some((route) => pathname.startsWith(route));

  // Check admin routes - but exclude /student/* paths
  const isAdminRoute = adminRoutes.some((route) => pathname.startsWith(route)) && !pathname.startsWith('/student/');
  const isStudentRoute = studentRoutes.some((route) => pathname.startsWith(route));

  // Parse user data
  let user: { role?: string } | null = null;
  if (userCookie) {
    try {
      user = JSON.parse(userCookie);
    } catch {
      // Invalid cookie
    }
  }

  // No token + protected route → redirect to login
  if (!token && !isPublicRoute) {
    const loginUrl = new URL('/auth/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Token expired → clear cookies and redirect to login
  if (token && isTokenExpired(token)) {
    const response = NextResponse.redirect(new URL('/auth/login', request.url));
    // Clear expired cookies
    response.cookies.delete('access_token');
    response.cookies.delete('user');
    return response;
  }

  // Has token + auth route (login) → redirect based on role
  if (token && isAuthRoute) {
    let redirectPath = '/dashboard';

    if (user?.role === 'student') {
      redirectPath = '/student/dashboard';
    }

    return NextResponse.redirect(new URL(redirectPath, request.url));
  }

  // Role-based access control
  if (token && user) {
    // Student trying to access admin routes → redirect to student dashboard
    if (isAdminRoute && user.role === 'student') {
      return NextResponse.redirect(new URL('/student/dashboard', request.url));
    }

    // Admin/teacher trying to access student routes → redirect to admin dashboard
    if (isStudentRoute && (user.role === 'admin' || user.role === 'teacher')) {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
};
