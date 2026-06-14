
import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router';
import { Link } from 'react-router';
import { ArrowLeft, Play, Users, Calendar, Clock, MapPin, Loader2, AlertCircle, BookOpen, FlaskConical } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { attendanceApi, semesterApi } from '@/lib/api-client';
import type { TeacherCourse, TeacherCoursesResponse } from '@/lib/types';

const WEEKS = Array.from({ length: 14 }, (_, i) => i + 1);

export default function AttendanceStartPage() {
  const params = useParams();
  const navigate = useNavigate();
  const courseId = params.courseId as string;

  const [course, setCourse] = useState<TeacherCourse | null>(null);
  const [pageLoading, setPageLoading] = useState(true);
  const [pageError, setPageError] = useState('');

  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedWeek, setSelectedWeek] = useState<number | null>(null);
  const [sessionType, setSessionType] = useState<'theory' | 'lab'>('theory');
  const [duration, setDuration] = useState(30);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchCourse = async () => {
      try {
        const response = await semesterApi.get('teacher/courses').json<TeacherCoursesResponse>();
        const foundCourse = response.courses?.find((c) => c.id === courseId);
        if (foundCourse) {
          setCourse(foundCourse);
        } else {
          setPageError('Ders bulunamadı.');
        }
      } catch (err: any) {
        console.error('Failed to fetch course:', err);
        setPageError('Ders bilgileri yüklenirken bir hata oluştu.');
      } finally {
        setPageLoading(false);
      }
    };

    fetchCourse();
  }, [courseId]);

  const handleStartAttendance = async () => {
    if (!selectedWeek) return;

    setLoading(true);
    setError('');

    try {
      const response = await attendanceApi.post('sessions', {
        json: {
          course_id: courseId,
          week_number: selectedWeek,
          duration_minutes: duration,
          session_type: sessionType,
        },
      }).json<{ session_id: string }>();

      setDialogOpen(false);
      navigate(`/teacher/attendance/${courseId}/session/${response.session_id}`);
    } catch (err: any) {
      setError(err.message || 'Yoklama başlatılamadı. Lütfen tekrar deneyin.');
    } finally {
      setLoading(false);
    }
  };

  if (pageLoading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (pageError || !course) {
    return (
      <div className="space-y-6">
        <Link
          to="/teacher/attendance"
          className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4" />
          Geri Dön
        </Link>
        <div className="flex flex-col items-center justify-center gap-4 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-12 dark:border-gray-700 dark:bg-gray-900">
          <AlertCircle className="h-12 w-12 text-red-500" />
          <p className="text-red-600 dark:text-red-400">
            {pageError || 'Ders bulunamadı.'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Back Button */}
      <Link
        to="/teacher/attendance"
        className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
      >
        <ArrowLeft className="h-4 w-4" />
        Ders Listesine Dön
      </Link>

      {/* Course Header */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3">
              <span className="rounded-md bg-blue-100 px-3 py-1.5 text-sm font-semibold text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
                {course.course_code}
              </span>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {course.course_name}
              </h1>
            </div>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {course.faculty} • {course.department}
            </p>
          </div>
        </div>

        {/* Course Info */}
        <div className="mt-4 flex flex-wrap items-center gap-6 border-t border-gray-200 pt-4 text-sm text-gray-600 dark:border-gray-700 dark:text-gray-400">
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <span>{course.max_capacity} Kontenjan</span>
          </div>
          {course.schedule.map((s, idx) => (
            <div key={idx} className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                <span>{s.day}</span>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4" />
                <span>{s.time}</span>
              </div>
              <div className="flex items-center gap-2">
                <MapPin className="h-4 w-4" />
                <span>{s.room}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Start Attendance Section */}
      <div className="rounded-xl border border-gray-200 bg-white p-8 dark:border-gray-800 dark:bg-gray-900">
        <div className="text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/50">
            <Play className="h-8 w-8 text-blue-600 dark:text-blue-400" />
          </div>
          <h2 className="mt-4 text-xl font-semibold text-gray-900 dark:text-white">
            Yoklama Başlat
          </h2>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Bu ders için yeni bir yoklama oturumu başlatmak üzeresiniz.
          </p>
          <button
            className="mt-6 inline-flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            onClick={() => setDialogOpen(true)}
          >
            <Play className="h-4 w-4" />
            Yoklamayı Başlat
          </button>
        </div>
      </div>

      {/* Placeholder for future features */}
      <div className="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-8 text-center dark:border-gray-700 dark:bg-gray-900">
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Yoklama istatistikleri ve geçmiş yoklamalar bu alanda görüntülenecek.
        </p>
      </div>

      {/* Week Selection Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Yoklama Oturumu Başlat</DialogTitle>
            <DialogDescription>
              {course.lab_hours > 0
                ? 'Yoklama alınacak haftayı, ders türünü ve süreyi seçin.'
                : 'Yoklama alınacak haftayı ve süreyi seçin.'}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {/* Week Selection */}
            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                Hafta Seçin
              </label>
              <div className="grid grid-cols-7 gap-2">
                {WEEKS.map((week) => (
                  <button
                    key={week}
                    onClick={() => setSelectedWeek(week)}
                    className={`flex h-10 w-10 items-center justify-center rounded-lg text-sm font-medium transition-colors ${
                      selectedWeek === week
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'
                    }`}
                  >
                    {week}
                  </button>
                ))}
              </div>
            </div>

            {/* Session Type Selection - only show if course has lab hours */}
            {course.lab_hours > 0 && (
              <div>
                <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Ders Türü
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setSessionType('theory')}
                    className={`flex items-center justify-center gap-2 rounded-lg border-2 px-4 py-3 text-sm font-medium transition-all ${
                      sessionType === 'theory'
                        ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-blue-400 dark:bg-blue-900/30 dark:text-blue-300'
                        : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:border-gray-600'
                    }`}
                  >
                    <BookOpen className="h-4 w-4" />
                    Teorik
                  </button>
                  <button
                    type="button"
                    onClick={() => setSessionType('lab')}
                    className={`flex items-center justify-center gap-2 rounded-lg border-2 px-4 py-3 text-sm font-medium transition-all ${
                      sessionType === 'lab'
                        ? 'border-purple-500 bg-purple-50 text-purple-700 dark:border-purple-400 dark:bg-purple-900/30 dark:text-purple-300'
                        : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:border-gray-600'
                    }`}
                  >
                    <FlaskConical className="h-4 w-4" />
                    Uygulama
                  </button>
                </div>
              </div>
            )}

            {/* Duration Selection */}
            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                Yoklama Süresi (dakika)
              </label>
              <select
                value={duration}
                onChange={(e) => setDuration(Number(e.target.value))}
                className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800"
              >
                <option value={15}>15 dakika</option>
                <option value={30}>30 dakika</option>
                <option value={45}>45 dakika</option>
                <option value={60}>60 dakika</option>
                <option value={90}>90 dakika</option>
                <option value={120}>120 dakika</option>
              </select>
            </div>

            {error && (
              <div className="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
                {error}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              İptal
            </Button>
            <Button
              onClick={handleStartAttendance}
              disabled={!selectedWeek || loading}
            >
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Başlatılıyor...
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Başlat
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
