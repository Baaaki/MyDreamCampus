import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import { mockFaculties } from '@/mock_data/catalog';
import type { Department, Faculty, SemesterCourse, Semester, SimplePeriod } from '@/lib/types';
import { semesterApi, catalogApiSafe, enrollmentApiSafe, gradesApiSafe, attendanceApiSafe } from '@/lib/api-client';
import { activateSemester } from '@/lib/services/system-service';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
import Toast from '@/components/enrollment/Toast';
import {
  ChevronDown,
  ChevronRight,
  Building2,
  GraduationCap,
  CalendarRange,
  Clock,
  AlertCircle,
  Loader2,
  Play,
  ArrowLeft,
  CheckCircle2,
  Shield,
} from 'lucide-react';

// ─── Constants ───────────────────────────────────────────────────────────────

const timeSlots = [
  { slot: 1, time: '08:30 - 09:15' },
  { slot: 2, time: '09:25 - 10:10' },
  { slot: 3, time: '10:20 - 11:05' },
  { slot: 4, time: '11:15 - 12:00' },
  { slot: 5, time: '12:10 - 12:55' },
  { slot: 6, time: '13:00 - 13:45' },
  { slot: 7, time: '13:55 - 14:40' },
  { slot: 8, time: '14:50 - 15:35' },
  { slot: 9, time: '15:45 - 16:30' },
];

const daysOfWeek = [
  { key: 'monday', label: 'Pzt', fullName: 'Pazartesi' },
  { key: 'tuesday', label: 'Sal', fullName: 'Salı' },
  { key: 'wednesday', label: 'Çar', fullName: 'Çarşamba' },
  { key: 'thursday', label: 'Per', fullName: 'Perşembe' },
  { key: 'friday', label: 'Cum', fullName: 'Cuma' },
];

const dayMap: Record<string, string> = {
  monday: 'Pazartesi', tuesday: 'Salı', wednesday: 'Çarşamba',
  thursday: 'Perşembe', friday: 'Cuma',
};

const courseColors = [
  'bg-blue-100 border-blue-300 text-blue-800',
  'bg-green-100 border-green-300 text-green-800',
  'bg-purple-100 border-purple-300 text-purple-800',
  'bg-yellow-100 border-yellow-300 text-yellow-800',
  'bg-red-100 border-red-300 text-red-800',
  'bg-indigo-100 border-indigo-300 text-indigo-800',
  'bg-teal-100 border-teal-300 text-teal-800',
  'bg-orange-100 border-orange-300 text-orange-800',
  'bg-pink-100 border-pink-300 text-pink-800',
  'bg-cyan-100 border-cyan-300 text-cyan-800',
  'bg-rose-100 border-rose-300 text-rose-800',
  'bg-violet-100 border-violet-300 text-violet-800',
  'bg-emerald-100 border-emerald-300 text-emerald-800',
  'bg-amber-100 border-amber-300 text-amber-800',
  'bg-sky-100 border-sky-300 text-sky-800',
  'bg-lime-100 border-lime-300 text-lime-800',
  'bg-fuchsia-100 border-fuchsia-300 text-fuchsia-800',
  'bg-slate-100 border-slate-300 text-slate-800',
];

const PERIOD_LABELS: Record<string, string> = {
  catalog: 'Ders Açma (Katalog)',
  enrollment: 'Ders Kayıt',
  grading: 'Not Giriş',
  attendance: 'Yoklama',
};

// ─── Types ───────────────────────────────────────────────────────────────────

interface ScheduleEntry {
  course_code: string;
  course_name: string;
  instructor: string;
  classroom: string;
  color: string;
}

interface SemesterCoursesResponse {
  data: SemesterCourse[];
  pagination: { total: number; page: number; limit: number; total_pages: number };
}

interface PeriodInfo {
  key: string;
  label: string;
  start: string | null;
  end: string | null;
}

// ─── API ─────────────────────────────────────────────────────────────────────

const fetchSemesterCourses = async (semester: string, department: string): Promise<SemesterCourse[]> => {
  const response = await semesterApi.get(`${semester}/courses`, {
    searchParams: { department, limit: 100 },
  }).json<SemesterCoursesResponse>();
  return response.data || [];
};

const fetchPlannedSemester = async (): Promise<Semester | null> => {
  const semesters = await catalogApiSafe.get('admin/semesters').json<Semester[]>();
  return semesters.find(s => s.status === 'planned') || null;
};

const SERVICE_APIS = [
  { key: 'catalog', api: catalogApiSafe },
  { key: 'enrollment', api: enrollmentApiSafe },
  { key: 'grading', api: gradesApiSafe },
  { key: 'attendance', api: attendanceApiSafe },
];

const fetchAllPeriods = async (semester: string): Promise<PeriodInfo[]> => {
  const results = await Promise.allSettled(
    SERVICE_APIS.map(async ({ key, api }) => {
      const periods = await api.get('admin/periods', { searchParams: { semester } }).json<SimplePeriod[]>();
      const active = periods.find(p => p.is_active) || periods[0];
      return {
        key,
        label: PERIOD_LABELS[key] || key,
        start: active?.period_start || null,
        end: active?.period_end || null,
      };
    })
  );

  return results.map((r, i) => {
    if (r.status === 'fulfilled') return r.value;
    return { key: SERVICE_APIS[i].key, label: PERIOD_LABELS[SERVICE_APIS[i].key], start: null, end: null };
  });
};

// ─── Component ───────────────────────────────────────────────────────────────

export default function SemesterReviewPage() {
  const navigate = useNavigate();
  const [expandedFaculties, setExpandedFaculties] = useState<string[]>([]);
  const [selectedDepartment, setSelectedDepartment] = useState<{ dept: Department; faculty: Faculty } | null>(null);
  const [activateConfirmOpen, setActivateConfirmOpen] = useState(false);
  const [activating, setActivating] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'error' | 'warning' | 'success' | 'info'; isVisible: boolean }>({
    message: '', type: 'info', isVisible: false,
  });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  // Fetch planned semester
  const { data: semester, isLoading: semesterLoading } = useQuery({
    queryKey: ['planned-semester'],
    queryFn: fetchPlannedSemester,
    staleTime: 30_000,
  });

  // Fetch periods
  const { data: periods = [] } = useQuery({
    queryKey: ['semester-periods', semester?.name],
    queryFn: () => fetchAllPeriods(semester!.name),
    enabled: !!semester?.name,
    staleTime: 60_000,
  });

  // Fetch courses for selected department
  const { data: semesterCourses = [], isLoading: coursesLoading } = useQuery({
    queryKey: ['review-courses', semester?.name, selectedDepartment?.dept.name],
    queryFn: () => fetchSemesterCourses(semester!.name, selectedDepartment!.dept.name),
    enabled: !!semester?.name && !!selectedDepartment,
  });

  // Build schedule grid
  const schedules = useMemo(() => {
    const grid: Record<number, Record<string, Record<number, ScheduleEntry>>> = { 1: {}, 2: {}, 3: {}, 4: {} };
    const colorMap = new Map<string, string>();
    let colorIdx = 0;

    semesterCourses.forEach((course) => {
      if (!colorMap.has(course.course_code)) {
        colorMap.set(course.course_code, courseColors[colorIdx % courseColors.length]);
        colorIdx++;
      }
      const color = colorMap.get(course.course_code)!;
      const cl = course.class_level;
      if (!grid[cl]) grid[cl] = {};

      course.schedule_sessions.forEach((session) => {
        const dayName = dayMap[session.day_of_week];
        if (!dayName) return;
        if (!grid[cl][dayName]) grid[cl][dayName] = {};
        session.slot_numbers.forEach((slot) => {
          grid[cl][dayName][slot] = {
            course_code: course.course_code,
            course_name: course.course_name,
            instructor: course.instructor_fullname,
            classroom: course.classroom_location,
            color,
          };
        });
      });
    });
    return grid;
  }, [semesterCourses]);

  // Activate handler
  const handleActivate = async () => {
    if (!semester) return;
    setActivating(true);
    try {
      await activateSemester(semester.id);
      showToast('Dönem başarıyla aktifleştirildi!', 'success');
      setTimeout(() => navigate('/system/semesters'), 1500);
    } catch (err: any) {
      let message = 'Dönem aktifleştirilemedi';
      try {
        const body = await err.response?.json();
        if (body?.error) message = body.error;
      } catch { /* ignore */ }
      showToast(message, 'error');
    } finally {
      setActivating(false);
      setActivateConfirmOpen(false);
    }
  };

  const toggleFaculty = (id: string) => {
    setExpandedFaculties(prev => prev.includes(id) ? prev.filter(f => f !== id) : [...prev, id]);
  };

  const formatDate = (iso: string | null) => {
    if (!iso) return '—';
    try { return format(new Date(iso), 'dd MMM yyyy HH:mm', { locale: tr }); }
    catch { return '—'; }
  };

  // ─── Loading ───────────────────────────────────────────────────────────────

  if (semesterLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        <span className="ml-3 text-gray-600 dark:text-gray-400">Dönem bilgileri yükleniyor...</span>
      </div>
    );
  }

  if (!semester) {
    return (
      <div className="max-w-2xl mx-auto py-16 text-center">
        <AlertCircle className="h-12 w-12 mx-auto mb-4 text-gray-400" />
        <h2 className="text-xl font-semibold mb-2">Planlanan Dönem Bulunamadı</h2>
        <p className="text-gray-500 dark:text-gray-400 mb-6">Önce yeni bir dönem oluşturmalısınız.</p>
        <Button onClick={() => navigate('/system/semesters/new')}>Yeni Dönem Oluştur</Button>
      </div>
    );
  }

  // ─── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8">
      <div className="max-w-[1600px] mx-auto px-4 space-y-6">

        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="sm" onClick={() => navigate('/semester-courses')}>
              <ArrowLeft className="h-4 w-4 mr-1" />
              Ders Açılışı
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dönem Son Kontrol</h1>
              <p className="text-gray-600 dark:text-gray-400 text-sm">
                Tüm bilgileri gözden geçirin ve dönemi başlatın
              </p>
            </div>
          </div>
          <Button
            size="lg"
            onClick={() => setActivateConfirmOpen(true)}
            className="bg-green-600 hover:bg-green-700 text-white"
          >
            <Play className="h-5 w-5 mr-2" />
            Dönemi Başlat
          </Button>
        </div>

        {/* Semester Info + Periods */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Semester Info */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-lg">
                <CalendarRange className="h-5 w-5 text-indigo-600" />
                Dönem Bilgileri
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <span className="text-sm text-gray-500 dark:text-gray-400">Dönem Adı</span>
                <span className="font-semibold text-lg">{semester.name}</span>
              </div>
              <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <span className="text-sm text-gray-500 dark:text-gray-400">Durum</span>
                <Badge variant="outline" className="border-amber-500 text-amber-600">
                  {semester.status === 'planned' ? 'Planlandı' : semester.status}
                </Badge>
              </div>
              <div className="flex items-center justify-between p-3 bg-red-50 dark:bg-red-950/20 rounded-lg border border-red-200 dark:border-red-800">
                <div className="flex items-center gap-2">
                  <Shield className="h-4 w-4 text-red-500" />
                  <span className="text-sm font-medium text-red-700 dark:text-red-300">Hard Deadline</span>
                </div>
                <span className="font-semibold text-red-700 dark:text-red-300">
                  {formatDate(semester.hard_deadline)}
                </span>
              </div>
            </CardContent>
          </Card>

          {/* Periods */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-lg">
                <Clock className="h-5 w-5 text-indigo-600" />
                Servis Tarih Aralıkları
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {(periods.length > 0 ? periods : Object.entries(PERIOD_LABELS).map(([key, label]) => ({
                  key, label, start: null, end: null,
                }))).map((period) => (
                  <div key={period.key} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                    <div>
                      <span className="text-sm font-medium">{period.label}</span>
                    </div>
                    <div className="text-right text-sm">
                      {period.start ? (
                        <div className="flex items-center gap-2">
                          <span className="text-gray-500">{formatDate(period.start)}</span>
                          <span className="text-gray-400">→</span>
                          <span className="text-gray-500">{formatDate(period.end)}</span>
                          <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />
                        </div>
                      ) : (
                        <span className="text-gray-400">Ayarlanmamış</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Course Schedules - Faculty Accordion */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <GraduationCap className="h-5 w-5 text-indigo-600" />
              Bölüm Ders Programları
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!selectedDepartment ? (
              /* Faculty / Department Selection */
              <div className="space-y-2">
                {mockFaculties.map((faculty) => {
                  const isExpanded = expandedFaculties.includes(faculty.id);
                  return (
                    <div key={faculty.id} className="border dark:border-gray-700 rounded-lg overflow-hidden">
                      <button
                        onClick={() => toggleFaculty(faculty.id)}
                        className="w-full flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors text-left"
                      >
                        <div className="flex items-center gap-3">
                          <Building2 className="h-5 w-5 text-indigo-600" />
                          <span className="font-semibold text-gray-900 dark:text-white">{faculty.name}</span>
                          <Badge variant="outline" className="text-xs">{faculty.departments.length} bölüm</Badge>
                        </div>
                        {isExpanded ? <ChevronDown className="h-5 w-5 text-gray-500" /> : <ChevronRight className="h-5 w-5 text-gray-500" />}
                      </button>
                      {isExpanded && (
                        <div className="border-t dark:border-gray-700 bg-white dark:bg-gray-800">
                          {faculty.departments.map((dept, i) => (
                            <button
                              key={dept.id}
                              onClick={() => setSelectedDepartment({ dept, faculty })}
                              className={`w-full flex items-center justify-between p-3 pl-12 hover:bg-indigo-50 dark:hover:bg-indigo-900/20 transition-colors text-left ${
                                i !== faculty.departments.length - 1 ? 'border-b border-gray-100 dark:border-gray-700' : ''
                              }`}
                            >
                              <div className="flex items-center gap-3">
                                <GraduationCap className="h-4 w-4 text-gray-400" />
                                <span className="text-gray-700 dark:text-gray-300">{dept.name}</span>
                              </div>
                              <ChevronRight className="h-4 w-4 text-gray-400" />
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            ) : (
              /* Weekly Schedule View */
              <div>
                <button
                  onClick={() => setSelectedDepartment(null)}
                  className="flex items-center gap-2 text-indigo-600 hover:text-indigo-800 mb-4 transition-colors"
                >
                  <ArrowLeft className="h-5 w-5" />
                  <span>Tüm Fakülteler</span>
                </button>

                <div className="flex items-center gap-4 mb-6">
                  <div className="w-12 h-12 bg-indigo-100 dark:bg-indigo-900/50 rounded-xl flex items-center justify-center">
                    <GraduationCap className="h-6 w-6 text-indigo-600" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-gray-900 dark:text-white">{selectedDepartment.dept.name}</h3>
                    <p className="text-gray-600 dark:text-gray-400 text-sm">{selectedDepartment.faculty.name}</p>
                  </div>
                </div>

                {coursesLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
                    <span className="ml-2 text-gray-600 dark:text-gray-400">Ders programları yükleniyor...</span>
                  </div>
                ) : (
                  <div className="space-y-8">
                    {[1, 2, 3, 4].map((classLevel) => (
                      <div key={classLevel} className="rounded-lg border dark:border-gray-700 overflow-hidden">
                        <div className="bg-gradient-to-r from-indigo-600 to-indigo-700 px-6 py-3">
                          <h4 className="text-lg font-bold text-white">{classLevel}. Sınıf Haftalık Ders Programı</h4>
                        </div>
                        <div className="overflow-x-auto">
                          <table className="w-full border-collapse">
                            <thead>
                              <tr className="bg-gray-50 dark:bg-gray-700">
                                <th className="border dark:border-gray-600 px-3 py-2 text-sm font-semibold text-gray-700 dark:text-gray-300 w-20">
                                  Saat
                                </th>
                                {daysOfWeek.map((day) => (
                                  <th key={day.key} className="border dark:border-gray-600 px-3 py-2 text-sm font-semibold text-gray-700 dark:text-gray-300 min-w-[180px]">
                                    <span className="hidden sm:inline">{day.fullName}</span>
                                    <span className="sm:hidden">{day.label}</span>
                                  </th>
                                ))}
                              </tr>
                            </thead>
                            <tbody>
                              {timeSlots.map((slot) => (
                                <tr key={slot.slot} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                                  <td className="border dark:border-gray-600 px-2 py-1 text-xs text-center font-medium text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-700">
                                    {slot.slot}. Ders
                                    <br />
                                    <span className="text-[10px]">{slot.time}</span>
                                  </td>
                                  {daysOfWeek.map((day) => {
                                    const dayName = dayMap[day.key];
                                    const entry = schedules[classLevel]?.[dayName]?.[slot.slot];
                                    return (
                                      <td key={day.key} className="border dark:border-gray-600 p-1">
                                        {entry ? (
                                          <div className={`${entry.color} border rounded-md p-2 h-full min-h-[60px] text-xs`}>
                                            <div className="font-bold">{entry.course_code} - {entry.course_name}</div>
                                            <div className="mt-1 text-[10px] opacity-70">{entry.instructor}</div>
                                            <div className="text-[10px] opacity-60">{entry.classroom}</div>
                                          </div>
                                        ) : slot.slot === 5 ? (
                                          <div className="h-[60px] flex items-center justify-center bg-orange-50 dark:bg-orange-950/20 rounded text-[10px] font-medium text-orange-400 dark:text-orange-500">
                                            Öğle Arası
                                          </div>
                                        ) : (
                                          <div className="h-[60px]" />
                                        )}
                                      </td>
                                    );
                                  })}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Activate Warning + Button */}
        <div className="rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-4">
          <div className="flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-amber-600 mt-0.5" />
            <div>
              <h4 className="font-medium text-amber-800 dark:text-amber-300">Aktifleştirme Uyarısı</h4>
              <p className="text-sm text-amber-700 dark:text-amber-400 mt-1">
                Dönemi aktifleştirdiğinizde ders yapısı (ders ekleme, silme, hoca değişikliği vb.) tamamen donar.
                Aktifleştirmeden önce tüm derslerin eklendiğinden emin olun.
              </p>
            </div>
          </div>
        </div>

        <div className="flex justify-between pb-8">
          <Button variant="outline" onClick={() => navigate('/semester-courses')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Ders Açılışına Dön
          </Button>
          <Button
            size="lg"
            onClick={() => setActivateConfirmOpen(true)}
            className="bg-green-600 hover:bg-green-700 text-white"
          >
            <Play className="h-5 w-5 mr-2" />
            Dönemi Başlat
          </Button>
        </div>
      </div>

      {/* Activate Confirmation Dialog */}
      <AlertDialog open={activateConfirmOpen} onOpenChange={setActivateConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Dönemi Aktifleştir</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{semester.name}</strong> dönemini aktifleştirmek istediğinize emin misiniz?
              <span className="block mt-2 text-amber-600 font-medium">
                Aktifleştirmeden sonra dönemlik ders yapısı donar. Ders ekleme/silme/değiştirme yapılamaz.
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={activating}>İptal</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleActivate}
              disabled={activating}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              {activating && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Aktifleştir
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Toast
        message={toast.message}
        type={toast.type}
        isVisible={toast.isVisible}
        onClose={() => setToast(prev => ({ ...prev, isVisible: false }))}
        duration={5000}
      />
    </div>
  );
}
