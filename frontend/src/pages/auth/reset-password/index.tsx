import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import { validatePasswordPolicy } from "@/lib/password-policy"

export default function ResetPasswordPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  // Read once: the token is removed from the URL right after mount.
  const [token] = useState(() => searchParams.get("token") ?? "")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [done, setDone] = useState(false)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  // A token left in the address bar ends up in history, bookmarks and
  // screenshots; it is single-use but valid for an hour until spent.
  useEffect(() => {
    if (searchParams.has("token")) {
      navigate("/auth/reset-password", { replace: true })
    }
  }, [searchParams, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (newPassword !== confirmPassword) {
      setError("Şifreler eşleşmiyor")
      return
    }
    const policyError = validatePasswordPolicy(newPassword)
    if (policyError) {
      setError(policyError)
      return
    }

    setLoading(true)
    try {
      await authApi.post("reset-password", {
        json: { token, new_password: newPassword },
      })
      setDone(true)
    } catch (err) {
      setError(
        await apiErrorMessage(err, "Şifre sıfırlanamadı, lütfen tekrar deneyin")
      )
    } finally {
      setLoading(false)
    }
  }

  const inputClass =
    "mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-md space-y-8 rounded-lg bg-white p-8 shadow-md">
        <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Yeni Şifre Belirle
        </h2>

        {done ? (
          <div className="rounded-md bg-green-50 p-4">
            <p className="text-sm text-green-800">
              Şifreniz güncellendi. Tüm oturumlarınız kapatıldı; yeni şifrenizle
              giriş yapabilirsiniz.
            </p>
          </div>
        ) : !token ? (
          <div className="rounded-md bg-red-50 p-4">
            <p className="text-sm text-red-800">
              Sıfırlama bağlantısı eksik veya geçersiz. Lütfen yeni bir bağlantı
              isteyin.
            </p>
          </div>
        ) : (
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            {error && (
              <div className="rounded-md bg-red-50 p-4">
                <p className="text-sm text-red-800">{error}</p>
              </div>
            )}
            <div className="space-y-4">
              <div>
                <label
                  htmlFor="new-password"
                  className="block text-sm font-medium text-gray-700"
                >
                  Yeni Şifre
                </label>
                <input
                  id="new-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  className={inputClass}
                  placeholder="En az 8 karakter, 1 büyük, 1 küçük harf, 1 rakam"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={loading}
                />
              </div>
              <div>
                <label
                  htmlFor="confirm-password"
                  className="block text-sm font-medium text-gray-700"
                >
                  Yeni Şifre (Tekrar)
                </label>
                <input
                  id="confirm-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  className={inputClass}
                  placeholder="Yeni şifrenizi tekrar girin"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={loading}
                />
              </div>
            </div>
            <button
              type="submit"
              disabled={loading}
              className="group relative flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Kaydediliyor..." : "Şifreyi Güncelle"}
            </button>
          </form>
        )}

        <p className="text-center text-sm">
          {done || !token ? (
            <Link
              to={done ? "/auth/login" : "/auth/forgot-password"}
              className="font-medium text-indigo-600 hover:text-indigo-500"
            >
              {done ? "Giriş yap" : "Yeni bağlantı iste"}
            </Link>
          ) : (
            <Link
              to="/auth/login"
              className="font-medium text-indigo-600 hover:text-indigo-500"
            >
              Girişe dön
            </Link>
          )}
        </p>
      </div>
    </div>
  )
}
