import { useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { AuthResponse, DemoAccount } from "@/lib/types"
import { useDemoAccounts } from "@/lib/services/demo-service"
import { Badge } from "@/components/ui/badge"
import { Copy, Check, LogIn } from "lucide-react"

const roleBadgeVariant: Record<string, string> = {
  admin:
    "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  teacher: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  student:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
}

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [copiedEmail, setCopiedEmail] = useState<string | null>(null)

  const { data: demoAccounts } = useDemoAccounts()

  const executeLogin = async (loginEmail: string, loginPassword: string) => {
    setError("")
    setLoading(true)

    try {
      const response: AuthResponse = await authApi
        .post("login", {
          json: { email: loginEmail, password: loginPassword },
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

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    await executeLogin(email, password)
  }

  const handleDemoLogin = (account: DemoAccount) => {
    setEmail(account.email)
    setPassword(account.password)
    void executeLogin(account.email, account.password)
  }

  const handleCopy = async (account: DemoAccount) => {
    try {
      if (navigator.clipboard) {
        await navigator.clipboard.writeText(
          `${account.email} / ${account.password}`
        )
      }
      setCopiedEmail(account.email)
      setTimeout(() => setCopiedEmail(null), 2000)
    } catch {
      // Ignore if clipboard API is not available
    }
  }

  const hasDemoAccounts = Boolean(demoAccounts && demoAccounts.length > 0)

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-12">
      <div className="flex w-full max-w-4xl flex-col items-center justify-center gap-8 lg:flex-row lg:items-start">
        {/* Main Login Form Card */}
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

        {/* Demo Accounts Panel */}
        {hasDemoAccounts && (
          <div
            aria-label="Demo hesapları paneli"
            className="w-full max-w-md space-y-4 rounded-lg border border-gray-200 bg-white p-6 shadow-md"
          >
            <div>
              <h3 className="text-lg font-bold text-gray-900">
                Demo Hesapları
              </h3>
              <p className="mt-1 text-xs text-gray-500">
                Giriş yapmak için bir hesaba tıklayabilir veya bilgileri
                kopyalayabilirsiniz.
              </p>
            </div>

            <div className="space-y-3">
              {demoAccounts?.map((account) => {
                const isCopied = copiedEmail === account.email
                const badgeClass =
                  roleBadgeVariant[account.role] ?? "bg-gray-100 text-gray-800"

                return (
                  <div
                    key={account.email}
                    className="space-y-2 rounded-lg border border-gray-100 bg-gray-50 p-3.5 transition-colors hover:border-gray-200 dark:border-gray-800 dark:bg-gray-900/50"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        {account.label || account.role}
                      </span>
                      <Badge
                        className={`px-2 py-0.5 text-xs font-medium tracking-wider uppercase ${badgeClass}`}
                      >
                        {account.role}
                      </Badge>
                    </div>

                    <div className="space-y-0.5 text-xs text-gray-600 dark:text-gray-400">
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-gray-500">
                          E-posta:
                        </span>
                        <span className="font-mono text-gray-800 dark:text-gray-200">
                          {account.email}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-gray-500">
                          Şifre:
                        </span>
                        <span className="font-mono text-gray-800 dark:text-gray-200">
                          {account.password}
                        </span>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 pt-1">
                      <button
                        type="button"
                        onClick={() => handleCopy(account)}
                        className="inline-flex flex-1 items-center justify-center gap-1 rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-700 shadow-xs hover:bg-gray-50 focus:ring-2 focus:ring-indigo-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700"
                      >
                        {isCopied ? (
                          <>
                            <Check className="h-3.5 w-3.5 text-green-600" />
                            <span>Kopyalandı</span>
                          </>
                        ) : (
                          <>
                            <Copy className="h-3.5 w-3.5" />
                            <span>Kopyala</span>
                          </>
                        )}
                      </button>

                      <button
                        type="button"
                        onClick={() => handleDemoLogin(account)}
                        disabled={loading}
                        className="inline-flex flex-1 items-center justify-center gap-1 rounded-md bg-indigo-600 px-2.5 py-1.5 text-xs font-medium text-white shadow-xs hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        <LogIn className="h-3.5 w-3.5" />
                        <span>Bu hesapla gir</span>
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
