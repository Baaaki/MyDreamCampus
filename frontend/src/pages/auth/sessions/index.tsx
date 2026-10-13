import { useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import { useNavigate } from "react-router"
import { LogOut } from "lucide-react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { authApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import type { SessionsResponse, User } from "@/lib/types"
import { format } from "date-fns"
import { tr } from "date-fns/locale"

/**
 * Whether the signed-in account is a shared demo account. A UI hint only:
 * auth refuses logout-all for demo accounts either way, since one visitor
 * would sign out every other visitor on the same account.
 */
function isDemoAccount(): boolean {
  try {
    const stored = localStorage.getItem("user")
    return stored ? (JSON.parse(stored) as User).is_demo === true : false
  } catch {
    return false
  }
}

export default function SessionsPage() {
  const navigate = useNavigate()
  const [isDemo] = useState(isDemoAccount)
  const [confirmLogoutAll, setConfirmLogoutAll] = useState(false)
  const {
    data: sessions = [],
    isLoading: loading,
    error: queryError,
    refetch: refetchSessions,
  } = useQuery({
    queryKey: ["auth", "sessions"],
    queryFn: async () => {
      const res = await authApi.get("sessions").json<SessionsResponse>()
      return res.sessions ?? []
    },
  })
  const error = queryError
    ? apiErrorMessage(queryError, "Oturumlar yüklenemedi")
    : ""

  const logoutAll = useMutation({
    mutationFn: () => authApi.post("logout-all").json(),
    onSuccess: () => {
      localStorage.removeItem("user")
      navigate("/auth/login", { replace: true })
    },
  })

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
        <div className="rounded-lg bg-white p-4 shadow-md sm:p-6">
          <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <h1 className="text-2xl font-bold text-gray-900">
              Aktif Oturumlar
            </h1>
            <div className="sm:text-right">
              <Button
                variant="destructive"
                disabled={isDemo || logoutAll.isPending}
                onClick={() => setConfirmLogoutAll(true)}
              >
                <LogOut className="mr-2 h-4 w-4" />
                Tüm Cihazlardan Çık
              </Button>
              {isDemo && (
                <p className="mt-2 text-xs text-gray-500">
                  Demo hesabı ortak kullanıldığı için bu işlem kapalı.
                </p>
              )}
            </div>
          </div>

          {logoutAll.isError && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-800">
                {apiErrorMessage(
                  logoutAll.error,
                  "Tüm cihazlardan çıkış yapılamadı"
                )}
              </p>
            </div>
          )}

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
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0 flex-1">
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
                      className="self-start rounded-md bg-red-100 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-200 focus:ring-2 focus:ring-red-500 focus:ring-offset-2 focus:outline-none"
                    >
                      Sonlandır
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <AlertDialog
            open={confirmLogoutAll}
            onOpenChange={setConfirmLogoutAll}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  Tüm cihazlardan çıkılsın mı?
                </AlertDialogTitle>
                <AlertDialogDescription>
                  Bu cihaz dahil bütün oturumlarınız sonlandırılır; devam etmek
                  için yeniden giriş yapmanız gerekir.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Vazgeç</AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={() => logoutAll.mutate()}
                >
                  Çıkış Yap
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

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
