import { useQuery } from "@tanstack/react-query"
import { useNavigate } from "react-router"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { Session } from "@/lib/types"
import { format } from "date-fns"
import { tr } from "date-fns/locale"

export default function SessionsPage() {
  const navigate = useNavigate()
  const {
    data: sessions = [],
    isLoading: loading,
    error: queryError,
    refetch: refetchSessions,
  } = useQuery({
    queryKey: ["auth", "sessions"],
    queryFn: () => authApi.get("sessions").json<Session[]>(),
  })
  const error = queryError
    ? apiErrorMessage(queryError, "Oturumlar yüklenemedi")
    : ""

  const handleRevokeSession = async (sessionId: string) => {
    if (!confirm("Bu oturumu sonlandırmak istediğinize emin misiniz?")) {
      return
    }

    try {
      await authApi.delete(`sessions/${sessionId}`)

      // Check if current session was revoked
      const revokedSession = sessions.find((s) => s.id === sessionId)
      if (revokedSession?.is_current) {
        // Current session revoked, logout
        localStorage.removeItem("user")
        navigate("/auth/login")
      } else {
        // Refresh sessions list
        refetchSessions()
      }
    } catch (err) {
      alert(apiErrorMessage(err, "Oturum sonlandırılamadı"))
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="mx-auto max-w-4xl px-4">
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h1 className="mb-6 text-2xl font-bold text-gray-900">
            Aktif Oturumlar
          </h1>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {sessions.length === 0 ? (
            <p className="text-gray-600">Aktif oturum bulunamadı.</p>
          ) : (
            <div className="space-y-4">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className={`rounded-lg border p-4 ${
                    session.is_current
                      ? "border-indigo-500 bg-indigo-50"
                      : "border-gray-200"
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="mb-2 flex items-center gap-2">
                        <h3 className="font-semibold text-gray-900">
                          {session.device_info || "Bilinmeyen Cihaz"}
                        </h3>
                        {session.is_current && (
                          <span className="rounded bg-indigo-100 px-2 py-1 text-xs font-medium text-indigo-800">
                            Mevcut Oturum
                          </span>
                        )}
                      </div>
                      <div className="space-y-1 text-sm text-gray-600">
                        <p>IP Adresi: {session.ip_address}</p>
                        <p>
                          Oluşturulma:{" "}
                          {format(
                            new Date(session.created_at),
                            "dd MMMM yyyy HH:mm",
                            {
                              locale: tr,
                            }
                          )}
                        </p>
                        <p>
                          Son Kullanma:{" "}
                          {format(
                            new Date(session.expires_at),
                            "dd MMMM yyyy HH:mm",
                            {
                              locale: tr,
                            }
                          )}
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRevokeSession(session.id)}
                      className="ml-4 rounded-md bg-red-100 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-200 focus:ring-2 focus:ring-red-500 focus:ring-offset-2 focus:outline-none"
                    >
                      Sonlandır
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="mt-6">
            <button
              onClick={() => navigate(-1)}
              className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            >
              Geri Dön
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
