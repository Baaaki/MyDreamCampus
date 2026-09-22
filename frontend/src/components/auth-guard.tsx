import { Navigate, Outlet } from "react-router"

interface AuthGuardProps {
  allowedRoles: string[]
}

export function AuthGuard({ allowedRoles }: AuthGuardProps) {
  const userStr = localStorage.getItem("user")

  // User info in localStorage is for UI routing only.
  // Actual auth is enforced server-side via httpOnly cookie.
  if (!userStr) {
    return <Navigate to="/auth/login" replace />
  }

  let redirectPath: string | null = null

  try {
    const user = JSON.parse(userStr)
    if (!allowedRoles.includes(user.role)) {
      if (user.role === "admin") {
        redirectPath = "/dashboard"
      } else if (user.role === "teacher") {
        redirectPath = "/teacher/attendance"
      } else if (user.role === "student") {
        redirectPath = "/student/dashboard"
      } else {
        redirectPath = "/auth/login"
      }
    }
  } catch {
    redirectPath = "/auth/login"
  }

  if (redirectPath) {
    return <Navigate to={redirectPath} replace />
  }

  return <Outlet />
}
