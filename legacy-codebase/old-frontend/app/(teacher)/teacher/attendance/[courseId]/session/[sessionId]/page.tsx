'use client';

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { QRCodeSVG } from 'qrcode.react';
import {
  ArrowLeft,
  QrCode,
  UserPlus,
  Users,
  Clock,
  CheckCircle2,
  XCircle,
  RefreshCw,
  StopCircle,
  Loader2,
  Search,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { attendanceApi } from '@/lib/api-client';
import type {
  SessionDetailsResponse,
  QRCodeResponse,
  SessionRecordsResponse,
  SessionStudentsResponse,
  AttendanceRecordItem,
  EnrolledStudentItem,
  ManualAttendanceResponse,
} from '@/lib/types';

export default function AttendanceSessionPage() {
  const params = useParams();
  const router = useRouter();
  const courseId = params.courseId as string;
  const sessionId = params.sessionId as string;

  const [sessionInfo, setSessionInfo] = useState<SessionDetailsResponse | null>(null);
  const [qrPayload, setQrPayload] = useState<QRCodeResponse['qr_payload'] | null>(null);
  const [attendanceRecords, setAttendanceRecords] = useState<AttendanceRecordItem[]>([]);
  const [enrolledStudents, setEnrolledStudents] = useState<EnrolledStudentItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Manual attendance states
  const [addingStudent, setAddingStudent] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Close session states
  const [closeDialogOpen, setCloseDialogOpen] = useState(false);
  const [closing, setClosing] = useState(false);

  // Time remaining
  const [timeRemaining, setTimeRemaining] = useState<string>('');

  // Fetch QR code
  const fetchQRCode = useCallback(async () => {
    try {
      const response = await attendanceApi.get(`sessions/${sessionId}/qr`).json<QRCodeResponse>();
      setQrPayload(response.qr_payload);
    } catch (err: any) {
      console.error('Failed to fetch QR code:', err);
    }
  }, [sessionId]);

  // Fetch attendance records
  const fetchRecords = useCallback(async () => {
    try {
      const response = await attendanceApi.get(`sessions/${sessionId}/records`).json<SessionRecordsResponse>();
      setAttendanceRecords(response.records || []);
    } catch (err: any) {
      console.error('Failed to fetch attendance records:', err);
    }
  }, [sessionId]);

  // Fetch enrolled students for manual attendance
  const fetchStudents = useCallback(async () => {
    try {
      const response = await attendanceApi.get(`sessions/${sessionId}/students`).json<SessionStudentsResponse>();
      setEnrolledStudents(response.students || []);
    } catch (err: any) {
      console.error('Failed to fetch students:', err);
    }
  }, [sessionId]);

  // Initial load
  useEffect(() => {
    const fetchSessionInfo = async () => {
      try {
        const response = await attendanceApi.get(`sessions/${sessionId}`).json<SessionDetailsResponse>();
        setSessionInfo(response);

        await Promise.all([fetchQRCode(), fetchRecords(), fetchStudents()]);
        setLoading(false);
      } catch (err: any) {
        console.error('Failed to fetch session:', err);
        setError(err.message || 'Oturum bilgileri yüklenemedi.');
        setLoading(false);
      }
    };

    fetchSessionInfo();
  }, [sessionId, fetchQRCode, fetchRecords, fetchStudents]);

  // QR code auto-refresh
  useEffect(() => {
    if (!sessionInfo || !sessionInfo.is_active) return;

    const interval = setInterval(() => {
      fetchQRCode();
    }, (sessionInfo.qr_rotation_interval || 30) * 1000);

    return () => clearInterval(interval);
  }, [sessionInfo, fetchQRCode]);

  // Attendance records auto-refresh (every 5 seconds)
  useEffect(() => {
    if (!sessionInfo || !sessionInfo.is_active) return;

    const interval = setInterval(() => {
      fetchRecords();
    }, 5000);

    return () => clearInterval(interval);
  }, [sessionInfo, fetchRecords]);

  // Time remaining countdown
  useEffect(() => {
    if (!sessionInfo) return;

    const updateTimeRemaining = () => {
      const now = new Date();
      const expires = new Date(sessionInfo.expires_at);
      const diff = expires.getTime() - now.getTime();

      if (diff <= 0) {
        setTimeRemaining('Süre doldu');
        return;
      }

      const minutes = Math.floor(diff / 60000);
      const seconds = Math.floor((diff % 60000) / 1000);
      setTimeRemaining(`${minutes}:${seconds.toString().padStart(2, '0')}`);
    };

    updateTimeRemaining();
    const interval = setInterval(updateTimeRemaining, 1000);

    return () => clearInterval(interval);
  }, [sessionInfo]);

  // Add manual attendance
  const handleAddManualAttendance = async (student: EnrolledStudentItem) => {
    setAddingStudent(student.student_id);
    try {
      const response = await attendanceApi.post(`sessions/${sessionId}/manual`, {
        json: {
          student_id: student.student_id,
          is_present: true,
          note: 'Manuel olarak eklendi',
        },
      }).json<ManualAttendanceResponse>();

      // Add to local list
      setAttendanceRecords(prev => [
        {
          id: response.id,
          student_id: response.student_id,
          student_number: response.student_number,
          student_name: response.student_name,
          is_present: response.is_present,
          marked_via: response.marked_via,
          marked_at: response.marked_at,
        },
        ...prev,
      ]);

      // Mark student as marked in local list
      setEnrolledStudents(prev =>
        prev.map(s =>
          s.student_id === student.student_id ? { ...s, is_marked: true } : s
        )
      );
    } catch (err: any) {
      console.error('Failed to add attendance:', err);
    } finally {
      setAddingStudent(null);
    }
  };

  // Close session
  const handleCloseSession = async () => {
    setClosing(true);
    try {
      await attendanceApi.post(`sessions/${sessionId}/close`).json();
      setCloseDialogOpen(false);
      router.push(`/teacher/attendance/${courseId}`);
    } catch (err: any) {
      console.error('Failed to close session:', err);
    } finally {
      setClosing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <Link
          href={`/teacher/attendance/${courseId}`}
          className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4" />
          Geri Dön
        </Link>
        <div className="rounded-xl border border-red-200 bg-red-50 p-8 text-center dark:border-red-800 dark:bg-red-900/20">
          <XCircle className="mx-auto h-12 w-12 text-red-500" />
          <p className="mt-4 text-red-700 dark:text-red-400">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <Link
          href={`/teacher/attendance/${courseId}`}
          className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4" />
          Ders Sayfasına Dön
        </Link>
        <Button variant="destructive" onClick={() => setCloseDialogOpen(true)}>
          <StopCircle className="mr-2 h-4 w-4" />
          Yoklamayı Bitir
        </Button>
      </div>

      {/* Session Info */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <span className="rounded-md bg-blue-100 px-3 py-1.5 text-sm font-semibold text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
                {sessionInfo?.course_code}
              </span>
              <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                {sessionInfo?.course_name}
              </h1>
              <span className="rounded-full bg-green-100 px-3 py-1 text-sm font-medium text-green-700 dark:bg-green-900/50 dark:text-green-300">
                Hafta {sessionInfo?.week_number}
              </span>
              {sessionInfo?.session_type && (
                <span className={`rounded-full px-3 py-1 text-sm font-medium ${
                  sessionInfo.session_type === 'theory'
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                    : 'bg-purple-100 text-purple-700 dark:bg-purple-900/50 dark:text-purple-300'
                }`}>
                  {sessionInfo.session_type === 'theory' ? 'Teorik' : 'Uygulama'}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
              <Users className="h-4 w-4" />
              <span>{attendanceRecords.filter(r => r.is_present).length} / {sessionInfo?.enrolled_student_count}</span>
            </div>
            <div className="flex items-center gap-2 text-orange-600 dark:text-orange-400">
              <Clock className="h-4 w-4" />
              <span className="font-mono">{timeRemaining}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Content Area */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Left: QR Code */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
          <div className="text-center">
            <div className="mb-4 flex items-center justify-center gap-2">
              <QrCode className="h-5 w-5 text-blue-600" />
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Yoklama QR Kodu
              </h2>
            </div>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
              Öğrenciler bu QR kodu telefonlarıyla tarayarak yoklamaya katılabilir.
            </p>

            {qrPayload ? (
              <div className="flex flex-col items-center">
                <div className="rounded-2xl bg-white p-4 shadow-lg">
                  <QRCodeSVG
                    value={JSON.stringify(qrPayload)}
                    size={256}
                    level="H"
                    includeMargin
                  />
                </div>
                <div className="mt-4 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <RefreshCw className="h-4 w-4" />
                  <span>
                    Her {sessionInfo?.qr_rotation_interval || 30} saniyede otomatik yenilenir
                  </span>
                </div>
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={fetchQRCode}
                >
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Şimdi Yenile
                </Button>
              </div>
            ) : (
              <div className="flex h-64 items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
              </div>
            )}
          </div>
        </div>

        {/* Right: Enrolled Students List */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Users className="h-5 w-5 text-blue-600" />
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Kayıtlı Öğrenciler
              </h2>
            </div>
            <span className="text-sm text-gray-500 dark:text-gray-400">
              {enrolledStudents.filter(s => s.is_marked).length} / {enrolledStudents.length}
            </span>
          </div>

          {/* Search Box */}
          <div className="mb-4">
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-gray-500 dark:text-gray-400" />
              <Input
                placeholder="Öğrenci ara..."
                className="pl-9"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>

          {/* Student List */}
          {enrolledStudents.length > 0 ? (
            <div className="max-h-96 space-y-2 overflow-y-auto">
              {enrolledStudents
                .filter((student) => {
                  if (!searchQuery) return true;
                  const query = searchQuery.toLowerCase();
                  return (
                    student.first_name.toLowerCase().includes(query) ||
                    student.last_name.toLowerCase().includes(query) ||
                    student.student_number.toLowerCase().includes(query)
                  );
                })
                .map((student) => (
                <div
                  key={student.student_id}
                  className={`flex items-center justify-between rounded-lg border p-3 ${
                    student.is_marked
                      ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20'
                      : 'border-gray-200 dark:border-gray-700'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    {student.is_marked ? (
                      <CheckCircle2 className="h-5 w-5 text-green-500" />
                    ) : (
                      <div className="h-5 w-5 rounded-full border-2 border-gray-300 dark:border-gray-600" />
                    )}
                    <div>
                      <p className="font-medium text-gray-900 dark:text-white">
                        {student.first_name} {student.last_name}
                      </p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {student.student_number}
                      </p>
                    </div>
                  </div>
                  {student.is_marked ? (
                    <span className="text-xs text-green-600 dark:text-green-400">Katıldı</span>
                  ) : (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleAddManualAttendance(student)}
                      disabled={addingStudent === student.student_id}
                    >
                      {addingStudent === student.student_id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <>
                          <UserPlus className="mr-1 h-4 w-4" />
                          Ekle
                        </>
                      )}
                    </Button>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="flex h-40 flex-col items-center justify-center text-center">
              <Users className="h-10 w-10 text-gray-300 dark:text-gray-600" />
              <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                Bu derse kayıtlı öğrenci bulunamadı.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Close Session Dialog */}
      <Dialog open={closeDialogOpen} onOpenChange={setCloseDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yoklamayı Bitir</DialogTitle>
            <DialogDescription>
              Yoklamayı bitirdiğinizde, katılmayan öğrenciler otomatik olarak devamsız olarak işaretlenecektir.
              Bu işlem geri alınamaz.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600 dark:text-gray-400">Katılan öğrenci:</span>
                <span className="font-medium text-green-600">{attendanceRecords.filter(r => r.is_present).length}</span>
              </div>
              <div className="mt-2 flex justify-between text-sm">
                <span className="text-gray-600 dark:text-gray-400">Devamsız sayılacak:</span>
                <span className="font-medium text-red-600">
                  {(sessionInfo?.enrolled_student_count || 0) - attendanceRecords.filter(r => r.is_present).length}
                </span>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCloseDialogOpen(false)}>
              İptal
            </Button>
            <Button variant="destructive" onClick={handleCloseSession} disabled={closing}>
              {closing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Kapatılıyor...
                </>
              ) : (
                <>
                  <StopCircle className="mr-2 h-4 w-4" />
                  Yoklamayı Bitir
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
