import { useState } from "react"
import { Link } from "react-router"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import { useIsDemoMode } from "@/lib/services/demo-service"

export default function ForgotPasswordPage() {
  const isDemo = useIsDemoMode()
  const [email, setEmail] = useState("")
  const [sent, setSent] = useState(false)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setLoading(true)
    try {
      await authApi.post("request-password-reset", { json: { email } })
      // Same confirmation whether or not the address has an account — the
      // backend does not tell, and neither may the page.
      setSent(true)
    } catch (err) {
      setError(
        apiErrorMessage(err, "İstek gönderilemedi, lütfen tekrar deneyin")
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
            Şifremi Unuttum
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            E-posta adresinize bir sıfırlama bağlantısı göndereceğiz
          </p>
        </div>
        {isDemo && (
          <div
            role="status"
            className="rounded-md border border-amber-200 bg-amber-50 p-3 text-center text-sm font-medium text-amber-800"
          >
            Demo ortamında e-posta gönderilmez
          </div>
        )}
        {sent ? (
          <div className="rounded-md bg-green-50 p-4">
            <p className="text-sm text-green-800">
              Bu adrese kayıtlı bir hesap varsa şifre sıfırlama bağlantısı
              gönderildi. Bağlantı 1 saat geçerlidir.
            </p>
          </div>
        ) : (
          <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
            {error && (
              <div className="rounded-md bg-red-50 p-4">
                <p className="text-sm text-red-800">{error}</p>
              </div>
            )}
            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700"
              >
                E-posta
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="relative mt-1 block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none sm:text-sm"
                placeholder="E-posta adresi"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="group relative flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? "Gönderiliyor..." : "Bağlantı Gönder"}
            </button>
          </form>
        )}
        <p className="text-center text-sm">
          <Link
            to="/auth/login"
            className="font-medium text-indigo-600 hover:text-indigo-500"
          >
            Girişe dön
          </Link>
        </p>
      </div>
    </div>
  )
}
