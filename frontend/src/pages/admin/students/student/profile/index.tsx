
import { useEffect, useState } from "react";
import { studentApi } from "@/lib/api-client";
import type { Student } from "@/lib/types";
import { format } from "date-fns";
import { tr } from "date-fns/locale";

export default function StudentProfilePage() {
  const [profile, setProfile] = useState<Student | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchProfile();
  }, []);

  const fetchProfile = async () => {
    try {
      const data = await studentApi.get("me").json<Student>();
      setProfile(data);
    } catch (err: any) {
      setError(err.message || "Profil bilgileri yüklenemedi");
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="rounded-md bg-red-50 p-4">
          <p className="text-sm text-red-800">{error || "Profil bulunamadı"}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-3xl mx-auto px-4">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h1 className="text-2xl font-bold text-gray-900 mb-6">Profil Bilgilerim</h1>

          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700">Öğrenci Numarası</label>
                <p className="mt-1 text-sm text-gray-900">{profile.student_id}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Ad Soyad</label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.first_name} {profile.last_name}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">E-posta</label>
                <p className="mt-1 text-sm text-gray-900">{profile.email}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Telefon</label>
                <p className="mt-1 text-sm text-gray-900">{profile.phone}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Bölüm</label>
                <p className="mt-1 text-sm text-gray-900">{profile.department}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Sınıf</label>
                <p className="mt-1 text-sm text-gray-900">{profile.class_level}. Sınıf</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Kayıt Yılı</label>
                <p className="mt-1 text-sm text-gray-900">{profile.enrollment_year}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">Danışman</label>
                <p className="mt-1 text-sm text-gray-900">
                  {profile.advisor_name || "Atanmamış"}
                </p>
              </div>
            </div>

            <div className="border-t pt-6">
              <div className="flex gap-4">
                <button
                  onClick={() => window.history.back()}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md"
                >
                  Geri Dön
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
