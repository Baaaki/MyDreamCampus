"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { authApi } from "@/lib/api-client";
import type { Session } from "@/lib/types";
import { format } from "date-fns";
import { tr } from "date-fns/locale";

export default function SessionsPage() {
  const router = useRouter();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchSessions();
  }, []);

  const fetchSessions = async () => {
    try {
      const data = await authApi.get("sessions").json<Session[]>();
      setSessions(data);
    } catch (err: any) {
      setError(err.message || "Oturumlar yüklenemedi");
    } finally {
      setLoading(false);
    }
  };

  const handleRevokeSession = async (sessionId: string) => {
    if (!confirm("Bu oturumu sonlandırmak istediğinize emin misiniz?")) {
      return;
    }

    try {
      await authApi.delete(`sessions/${sessionId}`);

      // Check if current session was revoked
      const revokedSession = sessions.find(s => s.id === sessionId);
      if (revokedSession?.is_current) {
        // Current session revoked, logout
        localStorage.removeItem("access_token");
        localStorage.removeItem("user");
        router.push("/auth/login");
      } else {
        // Refresh sessions list
        fetchSessions();
      }
    } catch (err: any) {
      alert(err.message || "Oturum sonlandırılamadı");
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h1 className="text-2xl font-bold text-gray-900 mb-6">Aktif Oturumlar</h1>

          {error && (
            <div className="rounded-md bg-red-50 p-4 mb-4">
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
                  className={`border rounded-lg p-4 ${
                    session.is_current ? "border-indigo-500 bg-indigo-50" : "border-gray-200"
                  }`}
                >
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h3 className="font-semibold text-gray-900">
                          {session.device_info || "Bilinmeyen Cihaz"}
                        </h3>
                        {session.is_current && (
                          <span className="px-2 py-1 text-xs font-medium bg-indigo-100 text-indigo-800 rounded">
                            Mevcut Oturum
                          </span>
                        )}
                      </div>
                      <div className="text-sm text-gray-600 space-y-1">
                        <p>IP Adresi: {session.ip_address}</p>
                        <p>
                          Oluşturulma:{" "}
                          {format(new Date(session.created_at), "dd MMMM yyyy HH:mm", {
                            locale: tr,
                          })}
                        </p>
                        <p>
                          Son Kullanma:{" "}
                          {format(new Date(session.expires_at), "dd MMMM yyyy HH:mm", {
                            locale: tr,
                          })}
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRevokeSession(session.id)}
                      className="ml-4 px-4 py-2 text-sm font-medium text-red-700 bg-red-100 hover:bg-red-200 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
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
              onClick={() => router.back()}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md"
            >
              Geri Dön
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
