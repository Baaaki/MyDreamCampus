import { useState } from "react";
import { Link } from "react-router";
import { authApi } from "@/lib/api-client";
import { apiErrorMessage } from "@/lib/api-error";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await authApi.post("request-password-reset", { json: { email } });
      // Same confirmation whether or not the address has an account — the
      // backend does not tell, and neither may the page.
      setSent(true);
    } catch (err) {
      setError(await apiErrorMessage(err, "İstek gönderilemedi, lütfen tekrar deneyin"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 p-8 bg-white rounded-lg shadow-md">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">Şifremi Unuttum</h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            E-posta adresinize bir sıfırlama bağlantısı göndereceğiz
          </p>
        </div>
        {sent ? (
          <div className="rounded-md bg-green-50 p-4">
            <p className="text-sm text-green-800">
              Bu adrese kayıtlı bir hesap varsa şifre sıfırlama bağlantısı gönderildi. Bağlantı 1 saat
              geçerlidir.
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
              <label htmlFor="email" className="block text-sm font-medium text-gray-700">
                E-posta
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                placeholder="E-posta adresi"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={loading}
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "Gönderiliyor..." : "Bağlantı Gönder"}
            </button>
          </form>
        )}
        <p className="text-center text-sm">
          <Link to="/auth/login" className="font-medium text-indigo-600 hover:text-indigo-500">
            Girişe dön
          </Link>
        </p>
      </div>
    </div>
  );
}
