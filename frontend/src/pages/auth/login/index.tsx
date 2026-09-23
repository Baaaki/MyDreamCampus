import { useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { AuthResponse } from "@/lib/types"

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const response: AuthResponse = await authApi
        .post("login", {
          json: { email, password },
        })
        .json()

      // Store user info in localStorage for UI display only
      // Access token is stored as httpOnly cookie by the backend
      localStorage.setItem("user", JSON.stringify(response.user))

      // Check if password change is required
      if (response.force_password_change) {
        navigate("/auth/change-password")
        return
      }

      // Check for redirect parameter (only allow safe relative paths)
      const redirectTo = searchParams.get("redirect")
      if (
        redirectTo &&
        redirectTo.startsWith("/") &&
        !redirectTo.startsWith("//") &&
        !redirectTo.includes("://")
      ) {
        navigate(redirectTo)
        return
      }

      // Redirect based on role
      switch (response.user.role) {
        case "admin":
          navigate("/dashboard")
          break
        case "teacher":
          navigate("/teacher/attendance")
          break
        case "student":
          navigate("/student/dashboard")
          break
        default:
          navigate("/dashboard")
      }
    } catch (err) {
      setError(
        apiErrorMessage(
          err,
          "Giriş başarısız. Lütfen bilgilerinizi kontrol edin."
        )
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-md space-y-8 rounded-lg bg-white p-8 shadow-md">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            MyDreamCampus
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Hesabınıza giriş yapın
          </p>
        </div>
        <form className="mt-8 space-y-6" onSubmit={handleLogin}>
          {error && (
            <div className="rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}
          <div className="-space-y-px rounded-md shadow-sm">
            <div>
              <label htmlFor="email" className="sr-only">
                E-posta
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="relative block w-full appearance-none rounded-none rounded-t-md border border-gray-300 px-3 py-2 text-gray-900 placeholder-gray-500 focus:z-10 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none sm:text-sm"
                placeholder="E-posta adresi"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
              />
            </div>
            <div>
              <label htmlFor="password" className="sr-only">
                Şifre
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                className="relative block w-full appearance-none rounded-none rounded-b-md border border-gray-300 px-3 py-2 text-gray-900 placeholder-gray-500 focus:z-10 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none sm:text-sm"
                placeholder="Şifre"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={loading}
              />
            </div>
          </div>

          <div>
            <button
              type="submit"
              disabled={loading}
              className="group relative flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Giriş yapılıyor..." : "Giriş Yap"}
            </button>
          </div>
          <p className="text-center text-sm">
            <Link
              to="/auth/forgot-password"
              className="font-medium text-indigo-600 hover:text-indigo-500"
            >
              Şifremi unuttum
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}
