import { useState, useMemo } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { attendanceApiSafe } from '@/lib/api-client';
import type { SessionRecordsResponse, AdminSessionItem } from '@/lib/types';
import { generateMockSessionRecords, mockAdminSessionsResponse, markMockStudentPresent } from '@/mock_data/admin_attendance';
import { mockCourseCatalog } from '@/mock_data/catalog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { ChevronLeft } from 'lucide-react';

export default function AdminAttendanceSessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  
  // Persist useMockData across reloads manually if needed, 
  // or default to false. Since the user arrives via URL mostly, let's keep it stateful.
  const [useMockData, setUseMockData] = useState<boolean>(
    () => sessionId?.startsWith('sess-mock-') || false
  );
  const [mockRefresh, setMockRefresh] = useState(0);

  // Read session item from router state if available
  const sessionItemState = location.state?.session as AdminSessionItem | undefined;
  
  // If directly navigated, fallback to finding it in mock data (if mock is enabled)
  const sessionItem = sessionItemState || (useMockData ? mockAdminSessionsResponse.sessions.find(s => s.session_id === sessionId) : undefined);
  
  // Lookup comprehensive course info from catalog
  const courseInfo = useMemo(() => {
    if (!sessionItem) return undefined;
    return mockCourseCatalog.find(c => c.course_code === sessionItem.course_code);
  }, [sessionItem]);

  const { data: recordsApiData, isLoading: recordsLoadingApi } = useQuery({
    queryKey: ['admin-session-records', sessionId],
    queryFn: () =>
      attendanceApiSafe
        .get(`admin/sessions/${sessionId}/records`)
        .json<SessionRecordsResponse>(),
    enabled: !!sessionId && !useMockData,
  });

  const records = useMockData && sessionId
    ? generateMockSessionRecords(sessionId)
    : recordsApiData;
  const recordsLoading = !useMockData && recordsLoadingApi;

  const presentStudents = records?.records.filter(r => r.is_present) || [];
  const absentStudents = records?.records.filter(r => !r.is_present) || [];

  const [studentToMark, setStudentToMark] = useState<string | null>(null);

  const confirmMarkPresent = () => {
    if (studentToMark) {
      if (useMockData && sessionId) {
        markMockStudentPresent(sessionId, studentToMark);
        setMockRefresh(prev => prev + 1);
      } else {
        alert('Backend endpoint bağlatısı henüz yapılmadı. Lütfen "Test Modu (Mock Veri)"nu aktif edip deneyin.');
      }
      setStudentToMark(null);
    }
  };

  // just to silence linter if mockRefresh is considered unused in strict modes.
  console.debug('mockRefresh', mockRefresh);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="icon" onClick={() => navigate('/attendance')}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              Oturum Detayları
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Bu oturuma katılan öğrencilerin detaylı yoklama kayıtları
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 rounded border px-3 py-1.5 shadow-sm dark:border-gray-800 bg-white dark:bg-gray-900">
          <label htmlFor="mock-toggle" className="text-xs font-semibold text-gray-700 dark:text-gray-300 select-none cursor-pointer">
            Test Modu (Mock Veri)
          </label>
          <input 
            id="mock-toggle" 
            type="checkbox" 
            className="cursor-pointer rounded accent-indigo-600"
            checked={useMockData} 
            onChange={(e) => setUseMockData(e.target.checked)} 
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Ders Bilgileri Sidebar */}
        <div className="lg:col-span-1 space-y-6">
          <div className="rounded-lg border bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Ders Bilgileri</h2>
            {sessionItem ? (
              <div className="space-y-4">
                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Ders Kodu ve Adı</p>
                  <p className="font-medium text-gray-900 dark:text-white mt-0.5">
                    {sessionItem.course_code} - {sessionItem.course_name}
                  </p>
                </div>
                
                {courseInfo && (
                  <>
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Fakülte / Bölüm</p>
                      <p className="font-medium text-gray-900 dark:text-white mt-0.5">
                        {courseInfo.faculty} / {courseInfo.department}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Ders Sorumlusu (Genel)</p>
                      <p className="font-medium text-gray-900 dark:text-white mt-0.5">
                        {courseInfo.coordinator?.name || '-'}
                      </p>
                    </div>
                  </>
                )}

                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Dönem</p>
                  <p className="font-medium text-gray-900 dark:text-white mt-0.5">
                    {sessionItem.semester}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Oturum Tipi / Durumu</p>
                  <div className="mt-1 flex gap-2">
                    <Badge variant={sessionItem.session_type === 'theory' ? 'default' : 'secondary'}>
                      {sessionItem.session_type === 'theory' ? 'Teorik' : 'Uygulama'}
                    </Badge>
                    {sessionItem.is_active ? (
                      <Badge className="bg-green-500 border-none">Aktif</Badge>
                    ) : (
                      <Badge variant="secondary">Kapandı</Badge>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="py-4 text-center text-sm text-gray-500">
                {!useMockData ? "Oturum bilgisine ulaşılamadı. Test Modunu aktifleştirmeyi deneyin." : "Böyle bir oturum bulunamadı."}
              </div>
            )}
          </div>
        </div>

        {/* Ana Katılım Listesi Alanı */}
        <div className="lg:col-span-2 rounded-lg border bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Katılım Kayıtları</h2>
          {recordsLoading ? (
            <div className="py-12 text-center text-sm text-gray-500">
              Kayıtlar yükleniyor...
            </div>
          ) : records ? (
            <div className="space-y-6">
              <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">Tarih</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-white mt-1 truncate">
                    {sessionItem?.session_date || '-'}
                  </p>
                </div>
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">Hafta</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-white mt-1">
                    {records.week_number}. Hafta
                  </p>
                </div>
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">Toplam Kayıtlı</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-white mt-1">
                    {records.total_count} Öğrenci
                  </p>
                </div>
                <div className="rounded-lg border bg-emerald-50 p-4 dark:border-emerald-900/20 dark:bg-emerald-950/20">
                  <p className="text-sm text-emerald-600 dark:text-emerald-400 font-medium">Katılım</p>
                  <p className="text-xl font-bold text-emerald-700 dark:text-emerald-300 mt-1">
                    {records.present_count} / {records.total_count}
                  </p>
                </div>
              </div>

              <div className="space-y-8">
                {/* Var / Katılanlar Tablosu */}
                <div>
                  <h3 className="text-md font-semibold text-emerald-700 dark:text-emerald-400 mb-3 flex items-center gap-2">
                    Katılan Öğrenciler (Var)
                    <Badge variant="outline" className="bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-400">
                      {presentStudents.length}
                    </Badge>
                  </h3>
                  <div className="rounded-md border dark:border-gray-800">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Öğrenci No</TableHead>
                          <TableHead>Ad Soyad</TableHead>
                          <TableHead>Durum</TableHead>
                          <TableHead>Yöntem</TableHead>
                          <TableHead>Yoklama Saati</TableHead>
                          <TableHead>Not / Açıklama</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {presentStudents.length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={6} className="h-24 text-center text-gray-500 text-sm">
                              Katılan öğrenci bulunamadı.
                            </TableCell>
                          </TableRow>
                        ) : (
                          presentStudents.map((r) => (
                            <TableRow key={r.id}>
                              <TableCell className="font-medium">{r.student_number}</TableCell>
                              <TableCell>{r.student_name}</TableCell>
                              <TableCell>
                                <Badge className="bg-emerald-500 hover:bg-emerald-600 border-none">Katıldı (Var)</Badge>
                              </TableCell>
                              <TableCell>
                                {r.marked_via === 'qr' ? 'QR Okutma' : r.marked_via === 'manual' ? 'Manuel (Elle)' : '-'}
                              </TableCell>
                              <TableCell>
                                {r.marked_at
                                  ? new Date(r.marked_at).toLocaleTimeString('tr-TR', {
                                      hour: '2-digit',
                                      minute: '2-digit',
                                    })
                                  : '-'}
                              </TableCell>
                              <TableCell className="text-sm text-gray-500 truncate max-w-xs" title={r.note}>
                                {r.note || '-'}
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>

                {/* Yok / Katılmayanlar Tablosu */}
                <div>
                  <h3 className="text-md font-semibold text-red-700 dark:text-red-400 mb-3 flex items-center gap-2">
                    Katılmayan Öğrenciler (Yok)
                    <Badge variant="outline" className="bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-400">
                      {absentStudents.length}
                    </Badge>
                  </h3>
                  <div className="rounded-md border dark:border-gray-800">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Öğrenci No</TableHead>
                          <TableHead>Ad Soyad</TableHead>
                          <TableHead>Durum</TableHead>
                          <TableHead className="text-right">İşlem</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {absentStudents.length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={4} className="h-24 text-center text-gray-500 text-sm">
                              Tüm öğrenciler katıldı.
                            </TableCell>
                          </TableRow>
                        ) : (
                          absentStudents.map((r) => (
                            <TableRow key={r.id}>
                              <TableCell className="font-medium">{r.student_number}</TableCell>
                              <TableCell>{r.student_name}</TableCell>
                              <TableCell>
                                <Badge variant="destructive">Katılmadı (Yok)</Badge>
                              </TableCell>
                              <TableCell className="text-right">
                                <Button 
                                  variant="outline" 
                                  size="sm" 
                                  onClick={() => setStudentToMark(r.student_id)}
                                >
                                  Yoklamaya Ekle
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="py-12 text-center text-sm text-gray-500">
              Kayıt bilgisi bulunamadı.
            </div>
          )}
        </div>
      </div>

      <AlertDialog open={!!studentToMark} onOpenChange={(open) => !open && setStudentToMark(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Öğrenciyi Yoklamaya Ekle</AlertDialogTitle>
            <AlertDialogDescription>
              Bu öğrenciyi yoklamaya eklemek istiyor musunuz? Öğrenci onayınız ardından <strong>Var</strong> olarak (manuel) işaretlenecektir.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction onClick={confirmMarkPresent}>Evet, Ekle</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
