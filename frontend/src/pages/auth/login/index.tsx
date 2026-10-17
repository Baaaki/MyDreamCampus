import { useRef, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { useQueryClient } from "@tanstack/react-query"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { AuthResponse, DemoAccount } from "@/lib/types"
import { useDemoAccounts } from "@/lib/services/demo-service"
import { clearUserSession } from "@/lib/session"
import { cn } from "@/lib/utils"
import { LoginVideoBackground } from "@/components/auth/login-video-background"
import {
  AlertTriangle,
  GraduationCap,
  Presentation,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react"

// Button order and text on the login page; a role the backend does not list
// gets no button.
const demoRoleButtons: { role: string; label: string; icon: LucideIcon }[] = [
  { role: "student", label: "Öğrenci Olarak Giriş", icon: GraduationCap },
  { role: "teacher", label: "Öğretmen Olarak Giriş", icon: Presentation },
  { role: "admin", label: "Admin Olarak Giriş", icon: ShieldCheck },
]

export default function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const submitRef = useRef<HTMLButtonElement>(null)

  const { data: demoAccounts } = useDemoAccounts()
  const demoButtons = demoRoleButtons.flatMap((button) => {
    const account = demoAccounts?.find((a) => a.role === button.role)
    return account ? [{ ...button, account }] : []
  })

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

      // Whatever another account left in this tab's cache must not be
      // served to this one.
      clearUserSession(queryClient)
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

  // Only fills the form: the visitor still signs in with "Giriş Yap", which
  // takes focus so Enter does it too.
  const fillDemoAccount = (account: DemoAccount) => {
    setError("")
    setEmail(account.email)
    setPassword(account.password)
    submitRef.current?.focus()
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center px-4 py-12">
      <LoginVideoBackground />
      <div className="relative z-10 w-full max-w-md space-y-8 rounded-2xl border border-white/20 bg-background/55 p-8 shadow-2xl backdrop-blur-xl dark:border-white/10 dark:bg-background/45">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-foreground">
            MyDreamCampus
          </h2>
          <p className="mt-2 text-center text-sm text-muted-foreground">
            Hesabınıza giriş yapın
          </p>
        </div>

        {demoButtons.length > 0 && (
          <section aria-label="Demo hesapları" className="space-y-3">
            <div
              role="note"
              className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50/85 p-3 text-sm font-medium text-amber-900 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-200"
            >
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <p>
                Bu bir demo. Yaptığınız tüm değişiklikler 1 saat sonra geri
                döndürülecektir.
              </p>
            </div>
            <div className="grid gap-2">
              {demoButtons.map(({ role, label, icon: Icon, account }) => {
                const selected = email === account.email
                return (
                  <button
                    key={role}
                    type="button"
                    aria-pressed={selected}
                    onClick={() => fillDemoAccount(account)}
                    disabled={loading}
                    className={cn(
                      "flex w-full items-center justify-center gap-2 rounded-md border px-4 py-2 text-sm font-semibold tracking-wide uppercase transition-colors focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:ring-offset-background focus:outline-none disabled:cursor-not-allowed disabled:opacity-50",
                      selected
                        ? "border-indigo-600 bg-indigo-50/90 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-950/60 dark:text-indigo-300"
                        : "border-border bg-background/60 text-foreground/80 hover:border-indigo-400 hover:bg-background/80"
                    )}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    <span>{label}</span>
                  </button>
                )
              })}
            </div>
            <p className="text-center text-xs text-muted-foreground">
              Bir hesap seçin, bilgileri doldurulunca Giriş Yap&apos;a basın.
            </p>
          </section>
        )}

        <form className="space-y-6" onSubmit={handleLogin}>
          {error && (
            <div className="rounded-md bg-red-50/90 p-4 dark:bg-red-950/50">
              <p className="text-sm text-red-800 dark:text-red-300">{error}</p>
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
                className="relative block w-full appearance-none rounded-none rounded-t-md border border-input bg-background/70 px-3 py-2 text-foreground placeholder:text-muted-foreground focus:z-10 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none sm:text-sm"
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
                className="relative block w-full appearance-none rounded-none rounded-b-md border border-input bg-background/70 px-3 py-2 text-foreground placeholder:text-muted-foreground focus:z-10 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none sm:text-sm"
                placeholder="Şifre"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={loading}
              />
            </div>
          </div>

          <div>
            <button
              ref={submitRef}
              type="submit"
              disabled={loading}
              className="group relative flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:ring-offset-background focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Giriş yapılıyor..." : "Giriş Yap"}
            </button>
          </div>
          <p className="text-center text-sm">
            <Link
              to="/auth/forgot-password"
              className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              Şifremi unuttum
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}
