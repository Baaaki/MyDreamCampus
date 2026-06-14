
import { useState, useEffect } from 'react';
import { enrollmentApi } from '@/lib/api-client';
import type { MyRejectionsResponse, RejectionDetail } from '@/lib/types';

export default function RejectionsPage() {
  const [rejections, setRejections] = useState<RejectionDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchRejections();
  }, []);

  const fetchRejections = async () => {
    try {
      setLoading(true);
      const response = await enrollmentApi
        .get('my-rejections')
        .json<MyRejectionsResponse>();

      setRejections(response.rejections || []);
    } catch (err: any) {
      setError(err.message || 'Reddedilmeler yüklenemedi');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('tr-TR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  if (loading) {
    return (
      <div className="container mx-auto p-4">
        <h1 className="text-3xl font-bold mb-6 text-gray-800 dark:text-white">Reddedilen Ders Kayıtları</h1>
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto p-4">
        <h1 className="text-3xl font-bold mb-6 text-gray-800 dark:text-white">Reddedilen Ders Kayıtları</h1>
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-4">
      <h1 className="text-3xl font-bold mb-6 text-gray-800 dark:text-white">Reddedilen Ders Kayıtları</h1>

      {rejections.length === 0 ? (
        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-8 text-center">
          <svg className="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p className="text-gray-600 dark:text-gray-400 text-lg">Reddedilen ders kaydınız bulunmamaktadır.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {rejections.map((rejection) => (
            <div
              key={rejection.id}
              className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm overflow-hidden"
            >
              {/* Header */}
              <div className="bg-red-50 dark:bg-red-900/30 px-6 py-4 border-b border-red-100 dark:border-red-800">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span className="text-red-800 dark:text-red-200 font-medium">
                      Reddedildi
                    </span>
                  </div>
                  <span className="text-sm text-red-600 dark:text-red-400">
                    {formatDate(rejection.rejected_at)}
                  </span>
                </div>
              </div>

              {/* Content */}
              <div className="p-6">
                {/* Rejection Reason */}
                <div className="mb-6">
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Ret Sebebi</h3>
                  <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4">
                    <p className="text-gray-800 dark:text-gray-200">{rejection.rejection_reason}</p>
                  </div>
                </div>

                {/* Advisor Info */}
                <div className="mb-6">
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">Reddeden Danışman</h3>
                  <p className="text-gray-800 dark:text-gray-200">{rejection.advisor_fullname}</p>
                </div>

                {/* Rejected Courses */}
                {rejection.rejected_courses && rejection.rejected_courses.courses && (
                  <div>
                    <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">
                      Reddedilen Dersler ({rejection.rejected_courses.courses.length} ders, {rejection.rejected_courses.total_credits} kredi)
                    </h3>
                    <div className="bg-gray-50 dark:bg-gray-900 rounded-lg overflow-hidden">
                      <table className="w-full">
                        <thead className="bg-gray-100 dark:bg-gray-800">
                          <tr>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Ders Kodu</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Ders Adı</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Kredi</th>
                            <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Öğretim Görevlisi</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                          {rejection.rejected_courses.courses.map((course) => (
                            <tr key={course.course_id}>
                              <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-100">{course.course_code}</td>
                              <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{course.course_name}</td>
                              <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{course.credits}</td>
                              <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{course.instructor}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
