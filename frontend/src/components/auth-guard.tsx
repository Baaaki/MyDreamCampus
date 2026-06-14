import { Navigate, Outlet } from 'react-router';

interface AuthGuardProps {
  allowedRoles: string[];
}

export function AuthGuard({ allowedRoles }: AuthGuardProps) {
  const userStr = localStorage.getItem('user');

  // User info in localStorage is for UI routing only.
  // Actual auth is enforced server-side via httpOnly cookie.
  if (!userStr) {
    return <Navigate to="/auth/login" replace />;
  }

  try {
    const user = JSON.parse(userStr);
    if (!allowedRoles.includes(user.role)) {
      if (user.role === 'admin') return <Navigate to="/dashboard" replace />;
      if (user.role === 'teacher') return <Navigate to="/teacher/attendance" replace />;
      if (user.role === 'student') return <Navigate to="/student/dashboard" replace />;
      return <Navigate to="/auth/login" replace />;
    }
  } catch {
    return <Navigate to="/auth/login" replace />;
  }

  return <Outlet />;
}
