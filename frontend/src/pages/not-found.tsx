import { Link } from "react-router"
import { Button } from "@/components/ui/button"

function getHomePath(): { path: string; label: string } {
  const userStr = localStorage.getItem("user")
  if (!userStr) return { path: "/auth/login", label: "Girişe dön" }

  try {
    const user = JSON.parse(userStr)
    if (user.role === "admin")
      return { path: "/dashboard", label: "Yönetim paneline dön" }
    if (user.role === "teacher")
      return { path: "/teacher/attendance", label: "Öğretmen paneline dön" }
    if (user.role === "student")
      return { path: "/student/dashboard", label: "Öğrenci paneline dön" }
  } catch {
    // fallthrough
  }
  return { path: "/auth/login", label: "Girişe dön" }
}

export default function NotFoundPage() {
  const home = getHomePath()

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <p className="text-6xl font-extrabold text-indigo-600">404</p>
        <div className="space-y-2">
          <h1 className="text-2xl font-bold text-gray-900">Sayfa bulunamadı</h1>
          <p className="text-sm text-gray-600">
            Aradığınız sayfa taşınmış, silinmiş veya hiç var olmamış olabilir.
          </p>
        </div>
        <Button asChild size="lg">
          <Link to={home.path}>{home.label}</Link>
        </Button>
      </div>
    </div>
  )
}
