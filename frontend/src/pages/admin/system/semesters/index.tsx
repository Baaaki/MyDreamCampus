import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import {
  ShieldCheck, RefreshCw, Loader2, AlertTriangle, CalendarRange, Wand2,
  GraduationCap, BookOpen, ClipboardCheck, CalendarOff, Clock, Eye,
  Trash2, Pencil
} from 'lucide-react';


import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table';
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import Toast from '@/components/enrollment/Toast';

import type { Semester, SemesterStatus, AcademicPeriod, SimplePeriod, ClosedDay } from '@/lib/types';
import {
  listSemesters,
  listGradesPeriods,
  listSimplePeriods,
  listClosedDays,
  deleteSemester,
} from '@/lib/services/system-service';

const STATUS_BADGE: Record<SemesterStatus, { label: string; className: string }> = {
  planned: { label: 'Planlandı', className: 'border-blue-500 text-blue-600 dark:text-blue-400' },
  active: { label: 'Aktif', className: 'border-green-500 text-green-600 dark:text-green-400' },
  completed: { label: 'Tamamlandı', className: 'border-gray-400 text-gray-500 dark:text-gray-400' },
};

const fmt = (iso: string) => format(new Date(iso), 'dd MMM yyyy HH:mm', { locale: tr });

function buildMockData() {
  const n = new Date();
  const iso = n.toISOString();

  const semesters: Semester[] = [
    {
      id: 'mock-1', name: '2025-2026-Fall', status: 'active',
      hard_deadline: new Date(n.getTime() + 86400000 * 120).toISOString(),
      activated_at: new Date(n.getTime() - 86400000 * 10).toISOString(),
      completed_at: null,
      created_at: new Date(n.getTime() - 86400000 * 30).toISOString(),
      updated_at: new Date(n.getTime() - 86400000 * 10).toISOString(),
    },
    {
      id: 'mock-2', name: '2025-2026-Spring', status: 'planned',
      hard_deadline: new Date(n.getTime() + 86400000 * 300).toISOString(),
      activated_at: null, completed_at: null,
      created_at: new Date(n.getTime() - 86400000 * 5).toISOString(),
      updated_at: new Date(n.getTime() - 86400000 * 5).toISOString(),
    },
  ];

  const grades: AcademicPeriod[] = [{
    id: 'mg1', semester: '2025-2026-Fall', course_id: null,
    period_start: new Date(n.getTime() - 86400000 * 15).toISOString(),
    period_end: new Date(n.getTime() + 86400000 * 45).toISOString(),
    is_active: true, created_at: iso, updated_at: iso,
  }];

  const enrollment: SimplePeriod[] = [{
    id: 'me1', semester: '2025-2026-Fall',
    period_start: new Date(n.getTime() - 86400000 * 30).toISOString(),
    period_end: new Date(n.getTime() - 86400000 * 5).toISOString(),
    is_active: false, created_at: iso, updated_at: iso,
  }];

  const catalog: SimplePeriod[] = [{
    id: 'mc1', semester: '2025-2026-Fall',
    period_start: new Date(n.getTime() - 86400000 * 45).toISOString(),
    period_end: new Date(n.getTime() - 86400000 * 20).toISOString(),
    is_active: false, created_at: iso, updated_at: iso,
  }];

  const attendance = {
    start: new Date(n.getTime() - 86400000 * 10).toISOString(),
    end: new Date(n.getTime() + 86400000 * 60).toISOString(),
    active: true,
  };

  const closedDays: ClosedDay[] = [
    { id: 'cd1', date: '2025-10-29', reason: 'Cumhuriyet Bayramı', created_at: iso },
    { id: 'cd2', date: '2025-11-10', reason: 'Atatürk\'ü Anma Günü', created_at: iso },
    { id: 'cd3', date: '2026-01-01', reason: 'Yılbaşı Tatili', created_at: iso },
  ];

  return { semesters, grades, enrollment, catalog, attendance, closedDays };
}

export default function SemestersPage() {
  const navigate = useNavigate();
  const [semesters, setSemesters] = useState<Semester[]>([]);
  const [loading, setLoading] = useState(false);

  // Period data (read-only)
  const [gradesPeriods, setGradesPeriods] = useState<AcademicPeriod[]>([]);
  const [enrollmentPeriods, setEnrollmentPeriods] = useState<SimplePeriod[]>([]);
  const [catalogPeriods, setCatalogPeriods] = useState<SimplePeriod[]>([]);
  const [attendancePeriods, setAttendancePeriods] = useState<SimplePeriod[]>([]);
  const [closedDays, setClosedDays] = useState<ClosedDay[]>([]);
  const [periodsLoading, setPeriodsLoading] = useState(false);
  const [mockMode, setMockMode] = useState(false);
  const [mockAttendance, setMockAttendance] = useState<{ start: string; end: string; active: boolean } | null>(null);

  const [toast, setToast] = useState<{
    message: string;
    type: 'error' | 'warning' | 'success' | 'info';
    isVisible: boolean;
  }>({ message: '', type: 'info', isVisible: false });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  const fetchSemesters = useCallback(async () => {
    if (mockMode) return;
    setLoading(true);
    try {
      setSemesters(await listSemesters());
    } catch {
      showToast('Dönemler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast, mockMode]);

  useEffect(() => { fetchSemesters(); }, [fetchSemesters]);

  const activeSemester = semesters.find((s) => s.status === 'active' && new Date() < new Date(s.hard_deadline));
  const expiredActiveSemester = semesters.find((s) => s.status === 'active' && new Date() >= new Date(s.hard_deadline));
  const visibleSemesters = semesters.filter(s => s.status !== 'completed' && new Date() < new Date(s.hard_deadline));

  // Fetch periods when active semester changes
  const fetchPeriods = useCallback(async () => {
    if (mockMode) return;
    if (!activeSemester) {
      setGradesPeriods([]);
      setEnrollmentPeriods([]);
      setCatalogPeriods([]);
      setAttendancePeriods([]);
      setClosedDays([]);
      setMockAttendance(null);
      return;
    }
    setPeriodsLoading(true);
    try {
      const [grades, enrollment, catalog, attendance, meals] = await Promise.allSettled([
        listGradesPeriods(activeSemester.name),
        listSimplePeriods('enrollment', activeSemester.name),
        listSimplePeriods('catalog', activeSemester.name),
        listSimplePeriods('attendance', activeSemester.name),
        listClosedDays(),
      ]);
      if (grades.status === 'fulfilled') setGradesPeriods(grades.value);
      if (enrollment.status === 'fulfilled') setEnrollmentPeriods(enrollment.value);
      if (catalog.status === 'fulfilled') setCatalogPeriods(catalog.value);
      if (attendance.status === 'fulfilled') setAttendancePeriods(attendance.value);
      if (meals.status === 'fulfilled') setClosedDays(meals.value);
    } finally {
      setPeriodsLoading(false);
    }
  }, [activeSemester, mockMode]);

  useEffect(() => { fetchPeriods(); }, [fetchPeriods]);

  const toggleMockMode = useCallback(() => {
    if (!mockMode) {
      const mock = buildMockData();
      setSemesters(mock.semesters);
      setGradesPeriods(mock.grades);
      setEnrollmentPeriods(mock.enrollment);
      setCatalogPeriods(mock.catalog);
      setMockAttendance(mock.attendance);
      setClosedDays(mock.closedDays);
      setMockMode(true);
    } else {
      setMockMode(false);
      setMockAttendance(null);
      // Real fetch will trigger via useEffect
    }
  }, [mockMode]);

  // Reset real data when exiting mock mode
  useEffect(() => {
    if (!mockMode) {
      fetchSemesters();
      fetchPeriods();
    }
  }, [mockMode]); // eslint-disable-line react-hooks/exhaustive-deps

  // Delete planned semester
  const [deleteTarget, setDeleteTarget] = useState<Semester | null>(null);
  const [deleting, setDeleting] = useState(false);

  const handleDeleteSemester = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteSemester(deleteTarget.id);
      showToast(`${deleteTarget.name} dönemi silindi`, 'success');
      setDeleteTarget(null);
      fetchSemesters();
    } catch {
      showToast('Dönem silinemedi', 'error');
    } finally {
      setDeleting(false);
    }
  }, [deleteTarget, fetchSemesters, showToast]);


  // Helper: get the first (main) period for a simple period list
  const mainGradesPeriod = gradesPeriods.find(p => !p.course_id) || gradesPeriods[0];
  const mainEnrollmentPeriod = enrollmentPeriods[0];
  const mainCatalogPeriod = catalogPeriods[0];
  const mainAttendancePeriod = attendancePeriods[0];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dönem Yönetimi</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Aktif ve planlanan dönemlerin durumunu görüntüleyin. Yeni dönem açmak için wizard'ı kullanın.
        </p>
      </div>

      {mockMode && (
        <div className="rounded-lg border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-3">
          <p className="text-sm text-amber-700 dark:text-amber-300 font-medium flex items-center gap-2">
            <Eye className="h-4 w-4" />
            Önizleme modu — aşağıdaki veriler örnek (mock) verilerdir.
          </p>
        </div>
      )}

      {expiredActiveSemester && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20 p-4">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-red-600 dark:text-red-400" />
            <span className="font-medium text-red-800 dark:text-red-300">
              Süresi Dolan Dönem: {expiredActiveSemester.name}
            </span>
          </div>
          <p className="mt-1 text-sm text-red-700 dark:text-red-400">
            Bu dönemin hard deadline süresi dolmuştur. Tüm veriler kilitlenmiştir.
          </p>
        </div>
      )}

      {activeSemester ? (
        <div className="rounded-lg border border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20 p-4">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-green-600 dark:text-green-400" />
            <span className="font-medium text-green-800 dark:text-green-300">
              Aktif Dönem: {activeSemester.name}
            </span>
          </div>
          <p className="mt-1 text-sm text-green-700 dark:text-green-400">
            Hard Deadline: {format(new Date(activeSemester.hard_deadline), 'dd MMMM yyyy HH:mm', { locale: tr })}
          </p>
        </div>
      ) : !expiredActiveSemester && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20 p-4">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400" />
            <span className="font-medium text-amber-800 dark:text-amber-300">
              Aktif dönem bulunamadı
            </span>
          </div>
          <p className="mt-1 text-sm text-amber-700 dark:text-amber-400">
            Dönem işlemleri (ders açma, kayıt, not girişi vb.) sadece aktif bir dönem varken çalışır.
          </p>
        </div>
      )}

      {/* Semester List */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CalendarRange className="h-5 w-5 text-indigo-600" />
              <CardTitle>Dönemler</CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={toggleMockMode}
                className={mockMode ? 'border-amber-400 text-amber-600 bg-amber-50 dark:bg-amber-900/20' : ''}
              >
                <Eye className="h-4 w-4 mr-1" />
                {mockMode ? 'Mock Kapat' : 'Önizleme'}
              </Button>
              <Button variant="outline" size="sm" onClick={() => { fetchSemesters(); fetchPeriods(); }} disabled={loading || mockMode}>
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              </Button>
              <Button onClick={() => navigate('/system/semesters/new')} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                <Wand2 className="h-4 w-4 mr-2" />
                Yeni Dönem Wizard'ı
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Dönem Adı</TableHead>
                  <TableHead>Durum</TableHead>
                  <TableHead>Hard Deadline</TableHead>
                  <TableHead>Aktifleştirilme</TableHead>
                  <TableHead>Oluşturulma</TableHead>
                  <TableHead className="text-right">İşlemler</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-6">
                      <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : visibleSemesters.length === 0 && !expiredActiveSemester ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-6 text-gray-500">
                      Sistemde açık veya planlanan bir dönem bulunmuyor.
                    </TableCell>
                  </TableRow>
                ) : (
                  <>
                    {expiredActiveSemester && (
                      <TableRow key={expiredActiveSemester.id} className="bg-red-50 dark:bg-red-900/10">
                        <TableCell className="font-medium text-red-700 dark:text-red-400">{expiredActiveSemester.name}</TableCell>
                        <TableCell><Badge variant="outline" className="border-red-500 text-red-600">Süresi Doldu</Badge></TableCell>
                        <TableCell className="text-sm line-through text-gray-400">{fmt(expiredActiveSemester.hard_deadline)}</TableCell>
                        <TableCell className="text-sm text-gray-500">{expiredActiveSemester.activated_at ? fmt(expiredActiveSemester.activated_at) : '—'}</TableCell>
                        <TableCell className="text-sm text-gray-500">{fmt(expiredActiveSemester.created_at)}</TableCell>
                        <TableCell />
                      </TableRow>
                    )}
                    {visibleSemesters.map((sem) => {
                      const badge = STATUS_BADGE[sem.status];
                      return (
                        <TableRow key={sem.id}>
                          <TableCell className="font-medium">{sem.name}</TableCell>
                          <TableCell><Badge variant="outline" className={badge.className}>{badge.label}</Badge></TableCell>
                          <TableCell className="text-sm">{fmt(sem.hard_deadline)}</TableCell>
                          <TableCell className="text-sm text-gray-500 dark:text-gray-400">{sem.activated_at ? fmt(sem.activated_at) : '—'}</TableCell>
                          <TableCell className="text-sm text-gray-500 dark:text-gray-400">{fmt(sem.created_at)}</TableCell>
                          <TableCell className="text-right">
                            {sem.status === 'planned' && (
                              <div className="flex items-center justify-end gap-1">
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => navigate(`/system/semesters/new?edit=${sem.id}`)}
                                  className="h-8 w-8 p-0 text-gray-500 hover:text-blue-600"
                                >
                                  <Pencil className="h-4 w-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => setDeleteTarget(sem)}
                                  className="h-8 w-8 p-0 text-gray-500 hover:text-red-600"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                            )}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Service Periods — read-only cards */}
      {activeSemester && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-indigo-600" />
              <CardTitle>Servis Tarih Aralıkları — {activeSemester.name}</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            {periodsLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <PeriodCard
                  icon={<GraduationCap className="h-4 w-4" />}
                  label="Not Giriş"
                  color="violet"
                  period={mainGradesPeriod ? { start: mainGradesPeriod.period_start, end: mainGradesPeriod.period_end, active: mainGradesPeriod.is_active } : null}
                />
                <PeriodCard
                  icon={<BookOpen className="h-4 w-4" />}
                  label="Ders Kayıt"
                  color="blue"
                  period={mainEnrollmentPeriod ? { start: mainEnrollmentPeriod.period_start, end: mainEnrollmentPeriod.period_end, active: mainEnrollmentPeriod.is_active } : null}
                />
                <PeriodCard
                  icon={<CalendarRange className="h-4 w-4" />}
                  label="Ders Açma (Katalog)"
                  color="emerald"
                  period={mainCatalogPeriod ? { start: mainCatalogPeriod.period_start, end: mainCatalogPeriod.period_end, active: mainCatalogPeriod.is_active } : null}
                />
                <PeriodCard
                  icon={<ClipboardCheck className="h-4 w-4" />}
                  label="Yoklama"
                  color="amber"
                  period={mainAttendancePeriod
                    ? { start: mainAttendancePeriod.period_start, end: mainAttendancePeriod.period_end, active: mainAttendancePeriod.is_active }
                    : mockAttendance}
                />

                {/* Closed Days — full width */}
                <div className="md:col-span-2 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-orange-100 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400">
                        <CalendarOff className="h-4 w-4" />
                      </div>
                      <h4 className="font-medium text-sm">Yemekhane Kapalı Günler</h4>
                    </div>
                    {closedDays.length > 0 && (
                      <Badge variant="outline" className="text-xs">{closedDays.length} gün</Badge>
                    )}
                  </div>
                  {closedDays.length === 0 ? (
                    <p className="text-sm text-gray-500 dark:text-gray-400">Tanımlı kapalı gün bulunamadı</p>
                  ) : (
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 divide-y divide-gray-100 dark:divide-gray-800">
                      {[...closedDays]
                        .sort((a, b) => a.date.localeCompare(b.date))
                        .map((day) => {
                          const d = new Date(day.date + 'T00:00:00');
                          const isPast = d < new Date();
                          return (
                            <div
                              key={day.id}
                              className={`flex items-center px-3 py-2.5 text-sm ${isPast ? 'opacity-50' : ''}`}
                            >
                              <span className={`font-mono text-xs w-[85px] flex-shrink-0 ${
                                isPast ? 'text-gray-400' : 'text-gray-600 dark:text-gray-300'
                              }`}>
                                {format(d, 'dd MMM yyyy', { locale: tr })}
                              </span>
                              <span className={`text-xs w-[80px] flex-shrink-0 ${
                                isPast ? 'text-gray-400' : 'text-gray-400 dark:text-gray-500'
                              }`}>
                                {format(d, 'EEEE', { locale: tr })}
                              </span>
                              <span className={`flex-1 ${
                                isPast ? 'text-gray-400' : 'text-gray-800 dark:text-gray-200'
                              }`}>
                                {day.reason}
                              </span>
                            </div>
                          );
                        })}
                    </div>
                  )}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Dönemi Sil</AlertDialogTitle>
            <AlertDialogDescription>
              Bu dönem ve tüm içeriği (dersler, periyotlar, kapalı günler) kalıcı olarak silinecek. Emin misiniz?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>İptal</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteSemester}
              disabled={deleting}
              className="bg-red-600 hover:bg-red-700 text-white"
            >
              {deleting ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              Sil
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>



      <Toast
        message={toast.message}
        type={toast.type}
        isVisible={toast.isVisible}
        onClose={() => setToast((prev) => ({ ...prev, isVisible: false }))}
        duration={5000}
      />
    </div>
  );
}

// Read-only period display card
const COLOR_MAP: Record<string, { bg: string; icon: string }> = {
  violet: { bg: 'bg-violet-100 dark:bg-violet-900/30', icon: 'text-violet-600 dark:text-violet-400' },
  blue: { bg: 'bg-blue-100 dark:bg-blue-900/30', icon: 'text-blue-600 dark:text-blue-400' },
  emerald: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', icon: 'text-emerald-600 dark:text-emerald-400' },
  amber: { bg: 'bg-amber-100 dark:bg-amber-900/30', icon: 'text-amber-600 dark:text-amber-400' },
};

function PeriodCard({
  icon,
  label,
  color,
  period,
  note,
}: {
  icon: React.ReactNode;
  label: string;
  color: string;
  period: { start: string; end: string; active: boolean } | null;
  note?: string;
}) {
  const c = COLOR_MAP[color] || COLOR_MAP.blue;
  const now = new Date();
  const isActive = period && new Date(period.start) <= now && now <= new Date(period.end);

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${c.bg} ${c.icon}`}>
            {icon}
          </div>
          <h4 className="font-medium text-sm">{label}</h4>
        </div>
        {period && (
          isActive ? (
            <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400 text-xs">Aktif</Badge>
          ) : (
            <Badge variant="outline" className="border-gray-400 text-gray-500 text-xs">Pasif</Badge>
          )
        )}
      </div>
      {period ? (
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-xs text-gray-500 dark:text-gray-400">Başlangıç</span>
            <p className="font-medium">{fmt(period.start)}</p>
          </div>
          <div>
            <span className="text-xs text-gray-500 dark:text-gray-400">Bitiş</span>
            <p className="font-medium">{fmt(period.end)}</p>
          </div>
        </div>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">{note || 'Tanımlı periyot bulunamadı'}</p>
      )}
    </div>
  );
}
