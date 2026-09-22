import { useEffect, useState } from "react"
import { studentApi } from "@/lib/api-client"
import type { Student } from "@/lib/types"

export default function StudentProfilePage() {
  const [profile, setProfile] = useState<Student | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  useEffect(() => {
    fetchProfile()
  }, [])

  const fetchProfile = async () => {
    try {
      // Backend has no /me route; own record is fetched by the auth user id,
      // same pattern as the student dashboard.
      const userStr = localStorage.getItem("user")
      if (!userStr) throw new Error("Oturum bilgisi bulunamadı")
      const user = JSON.parse(userStr)
      const data = await studentApi.get(`${user.id}`).json<Student>()
      setProfile(data)
    } catch (err: any) {
      setError(err.message || "Profil bilgileri yüklenemedi")
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    )
  }

  if (error || !profile) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="rounded-md bg-red-50 p-4">
          <p className="text-sm text-red-800">{error || "Profil bulunamadı"}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="mx-auto max-w-3xl px-4">
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h1 className="mb-6 text-2xl font-bold text-gray-900">
            Profil Bilgilerim
          </h1>

          <div className="space-y-6">
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Öğrenci Numarası
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.student_number}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Ad Soyad
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.first_name} {profile.last_name}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  E-posta
                </label>
                <p className="mt-1 text-sm text-gray-900">{profile.email}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Fakülte
                </label>
                <p className="mt-1 text-sm text-gray-900">{profile.faculty}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Bölüm
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.department}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Sınıf
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.class_level}. Sınıf
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Kayıt Yılı
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.enrollment_year}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Danışman
                </label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.advisor_name || "Atanmamış"}
                </p>
              </div>
            </div>

            <div className="border-t pt-6">
              <div className="flex gap-4">
                <button
                  onClick={() => window.history.back()}
                  className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
                >
                  Geri Dön
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
