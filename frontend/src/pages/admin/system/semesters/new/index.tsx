import { useState, useCallback, useMemo, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import {
  ArrowLeft,
  ArrowRight,
  CalendarRange,
  CheckCircle2,
  Clock,
  GraduationCap,
  Loader2,
  Play,
  AlertCircle,
  BookOpen,
  UtensilsCrossed,
  Trash2,
  Plus,
} from 'lucide-react';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
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
import CourseHierarchyView from '@/components/course-hierarchy-view';

import type { Semester, SemesterPeriods, Department, Faculty } from '@/lib/types';
import {
  createSemester, activateSemester, listSemesters,
  listGradesPeriods, listSimplePeriods, listClosedDays, updateSemester
} from '@/lib/services/system-service';

const STEPS = [
  { label: 'Dönem Bilgileri', icon: CalendarRange },
  { label: 'Servis Tarihleri', icon: Clock },
  { label: 'Ders Ekleme', icon: BookOpen },
  { label: 'Önizleme', icon: CheckCircle2 },
];

const PERIOD_SERVICES = [
  { key: 'catalog' as const, label: 'Ders Açma (Katalog)', description: 'Dönemlik ders açılış dönemi' },
  { key: 'enrollment' as const, label: 'Ders Kayıt', description: 'Öğrenci ders kayıt dönemi' },
  { key: 'grading' as const, label: 'Not Giriş', description: 'Hoca not giriş dönemi' },
  { key: 'attendance' as const, label: 'Yoklama', description: 'Yoklama alma dönemi' },
];

// Wizard executes 3 separate API calls (not one atomic call):
// 1. POST /semesters — create semester + distribute periods
// 2. POST /semesters/:s/courses — add each course (existing page handles this)
// 3. PUT /semesters/:id/activate — activate semester
//
// Why separate calls instead of one atomic endpoint?
// Future extensibility: when department_head role is added,
// step 2 will be done by department heads (not admin).
// See: docs/semester-wizard-plan.md "Gelecek Uyumluluk"

export default function SemesterWizardPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const editId = searchParams.get('edit');
  const isEditMode = !!editId;

  const [step, setStep] = useState(0);

  // Step 1: Semester info
  const [semesterName, setSemesterName] = useState('');
  const [hardDeadline, setHardDeadline] = useState('');

  // Step 2: Periods
  const [periods, setPeriods] = useState<{
    catalog: { start: string; end: string };
    enrollment: { start: string; end: string };
    grading: { start: string; end: string };
    attendance: { start: string; end: string };
  }>({
    catalog: { start: '', end: '' },
    enrollment: { start: '', end: '' },
    grading: { start: '', end: '' },
    attendance: { start: '', end: '' },
  });

  // Step 2: Closed days (meal service)
  const [closedDays, setClosedDays] = useState<Array<{ date: string; reason: string }>>([]);

  // Created semester reference
  const [createdSemester, setCreatedSemester] = useState<Semester | null>(null);

  // Loading & UI state
  const [loading, setLoading] = useState(false);
  const [initialLoad, setInitialLoad] = useState(isEditMode);
  const [activateConfirmOpen, setActivateConfirmOpen] = useState(false);
  const [toast, setToast] = useState<{
    message: string;
    type: 'error' | 'warning' | 'success' | 'info';
    isVisible: boolean;
  }>({ message: '', type: 'info', isVisible: false });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  // Fetch data if editing
  useEffect(() => {
    if (!editId) return;

    const loadEditData = async () => {
      try {
        const semesters = await listSemesters();
        const sem = semesters.find((s) => s.id === editId);
        if (!sem) {
          showToast('Dönem bulunamadı', 'error');
          navigate('/system/semesters');
          return;
        }

        setCreatedSemester(sem);
        setSemesterName(sem.name);
        setHardDeadline(new Date(sem.hard_deadline).toISOString().slice(0, 16));

        const [grades, enrollment, catalog, attendance, meals] = await Promise.allSettled([
          listGradesPeriods(sem.name),
          listSimplePeriods('enrollment', sem.name),
          listSimplePeriods('catalog', sem.name),
          listSimplePeriods('attendance', sem.name),
          listClosedDays(),
        ]);

        const getPeriod = (res: any) => {
          if (res.status === 'fulfilled' && res.value && res.value.length > 0) {
            const main = res.value.find((x: any) => !x.course_id) || res.value[0];
            if (main && main.period_start && main.period_end) {
              return {
                start: new Date(main.period_start).toISOString().slice(0, 16),
                end: new Date(main.period_end).toISOString().slice(0, 16),
              };
            }
          }
          return { start: '', end: '' };
        };

        setPeriods({
          grading: getPeriod(grades),
          enrollment: getPeriod(enrollment),
          catalog: getPeriod(catalog),
          attendance: getPeriod(attendance),
        });

        if (meals.status === 'fulfilled' && meals.value) {
          let days = meals.value;
          if (days.some((d: any) => d.semester)) {
            days = days.filter((d: any) => d.semester === sem.name);
          }
          setClosedDays(days.map((d: any) => ({
            date: new Date(d.date).toISOString().split('T')[0],
            reason: d.reason,
          })));
        }
      } catch {
        showToast('Veriler yüklenirken hata oluştu', 'error');
      } finally {
        setInitialLoad(false);
      }
    };

    loadEditData();
  }, [editId, showToast, navigate]);

  // Semester name suggestions
  const year = new Date().getFullYear();
  const suggestions = [
    `${year - 1}-${year}-Spring`,
    `${year}-${year + 1}-Fall`,
    `${year}-${year + 1}-Spring`,
  ];

  // Validation
  const isStep1Valid = useMemo(() => {
    return /^\d{4}-\d{4}-(Fall|Spring)$/.test(semesterName) && hardDeadline !== '';
  }, [semesterName, hardDeadline]);

  const isStep2Valid = useMemo(() => {
    const deadline = hardDeadline ? new Date(hardDeadline) : null;
    if (!deadline) return false;

    for (const svc of PERIOD_SERVICES) {
      const p = periods[svc.key];
      if (!p.start || !p.end) return false;
      if (new Date(p.start) >= new Date(p.end)) return false;
      if (new Date(p.end) > deadline) return false;
    }
    return true;
  }, [periods, hardDeadline]);

  // Resolved semester name for steps 3 & 4
  const resolvedSemesterName = createdSemester?.name || semesterName;

  // Handle semester creation (after Step 2)
  const handleCreateSemester = async () => {
    setLoading(true);
    try {
      const periodsPayload: SemesterPeriods = {};
      for (const svc of PERIOD_SERVICES) {
        const p = periods[svc.key];
        if (p.start && p.end) {
          periodsPayload[svc.key] = {
            start: new Date(p.start).toISOString(),
            end: new Date(p.end).toISOString(),
          };
        }
      }

      const semester = await createSemester({
        name: semesterName,
        hard_deadline: new Date(hardDeadline).toISOString(),
        periods: periodsPayload,
        closed_days: closedDays.length > 0 ? closedDays : undefined,
      });

      setCreatedSemester(semester);
      showToast('Dönem ve servis tarihleri başarıyla oluşturuldu', 'success');
      setStep(2);
    } catch (err: any) {
      const status = err.response?.status;
      if (status === 409) {
        // Dönem zaten var — mevcut semester'ı bul ve devam et
        try {
          const semesters = await listSemesters();
          const existing = semesters.find((s) => s.name === semesterName);
          if (existing) {
            setCreatedSemester(existing);
            showToast('Dönem zaten mevcut, devam ediliyor', 'info');
            setStep(2);
            return;
          }
        } catch { /* listSemesters failed, fall through to generic error */ }
      }
      let message = 'Dönem oluşturulamadı';
      try {
        const body = await err.response?.json();
        if (body?.error) message = body.error;
      } catch { /* ignore */ }
      showToast(message, 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateSemester = async () => {
    if (!createdSemester) return;
    setLoading(true);
    try {
      const periodsPayload: SemesterPeriods = {};
      for (const svc of PERIOD_SERVICES) {
        const p = periods[svc.key];
        if (p.start && p.end) {
          periodsPayload[svc.key] = {
            start: new Date(p.start).toISOString(),
            end: new Date(p.end).toISOString(),
          };
        }
      }

      const updated = await updateSemester(createdSemester.id, {
        hard_deadline: new Date(hardDeadline).toISOString(),
        periods: Object.keys(periodsPayload).length > 0 ? periodsPayload : undefined,
        closed_days: closedDays.length > 0 ? closedDays : undefined,
      });

      setCreatedSemester(updated);
      showToast('Dönem başarıyla güncellendi', 'success');
      setStep(2);
    } catch (err: any) {
      let message = 'Dönem güncellenemedi';
      try {
        const body = await err.response?.json();
        if (body?.error) message = body.error;
      } catch { /* ignore */ }
      showToast(message, 'error');
    } finally {
      setLoading(false);
    }
  };

  // Handle activation (Step 4)
  // INVARIANT: Only one semester can be active at any given time.
  // Backend returns ACTIVE_SEMESTER_EXISTS (409) if another semester is already active.
  // The error message is displayed to the user via showToast.
  const handleActivate = async () => {
    if (!createdSemester) return;
    setLoading(true);
    try {
      await activateSemester(createdSemester.id);
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
      setLoading(false);
      setActivateConfirmOpen(false);
    }
  };

  // Navigation
  const canGoNext = () => {
    if (step === 0) return isStep1Valid;
    if (step === 1) return isStep2Valid;
    return true;
  };

  const handleNext = () => {
    if (step === 1) {
      if (isEditMode) {
        handleUpdateSemester();
        return;
      } else if (!createdSemester) {
        handleCreateSemester();
        return;
      }
    }
    setStep((s) => Math.min(s + 1, 3));
  };

  const handleBack = () => {
    setStep((s) => Math.max(s - 1, 0));
  };

  if (initialLoad) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => navigate('/system/semesters')}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          Geri
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            {isEditMode ? 'Dönemi Düzenle' : "Yeni Dönem Wizard'ı"}
          </h1>
          <p className="text-gray-600 dark:text-gray-400 text-sm">
            {isEditMode
              ? 'Dönem bilgilerini, tarihleri veya açılan dersleri güncelleyin'
              : 'Dönem oluşturma, tarih belirleme ve ders açılışını tek akışta yapın'}
          </p>
        </div>
      </div>

      {/* Stepper */}
      <div className="flex items-center gap-2">
        {STEPS.map((s, i) => {
          const Icon = s.icon;
          const isActive = i === step;
          const isCompleted = i < step;
          return (
            <div key={i} className="flex items-center gap-2 flex-1">
              <div
                className={`flex items-center gap-2 px-3 py-2 rounded-lg flex-1 transition-colors ${
                  isActive
                    ? 'bg-indigo-50 dark:bg-indigo-950/30 border border-indigo-300 dark:border-indigo-700'
                    : isCompleted
                      ? 'bg-green-50 dark:bg-green-950/20 border border-green-300 dark:border-green-700'
                      : 'bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700'
                }`}
              >
                <div
                  className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold ${
                    isActive
                      ? 'bg-indigo-600 text-white'
                      : isCompleted
                        ? 'bg-green-600 text-white'
                        : 'bg-gray-300 dark:bg-gray-600 text-gray-600 dark:text-gray-300'
                  }`}
                >
                  {isCompleted ? <CheckCircle2 className="h-4 w-4" /> : i + 1}
                </div>
                <div className="hidden md:block">
                  <div className="flex items-center gap-1 text-sm font-medium">
                    <Icon className="h-3.5 w-3.5" />
                    {s.label}
                  </div>
                </div>
              </div>
              {i < STEPS.length - 1 && (
                <ArrowRight className="h-4 w-4 text-gray-400 flex-shrink-0" />
              )}
            </div>
          );
        })}
      </div>

      {/* Step Content */}
      {step === 0 && (
        <StepSemesterInfo
          semesterName={semesterName}
          setSemesterName={setSemesterName}
          hardDeadline={hardDeadline}
          setHardDeadline={setHardDeadline}
          suggestions={suggestions}
          isEditMode={isEditMode}
        />
      )}

      {step === 1 && (
        <StepPeriods
          periods={periods}
          setPeriods={setPeriods}
          hardDeadline={hardDeadline}
          closedDays={closedDays}
          setClosedDays={setClosedDays}
        />
      )}

      {step === 2 && (
        <StepCourses semesterName={resolvedSemesterName} />
      )}

      {step === 3 && (
        <StepPreview
          semesterName={semesterName}
          hardDeadline={hardDeadline}
          periods={periods}
          resolvedSemesterName={resolvedSemesterName}
          closedDays={closedDays}
        />
      )}

      {/* Navigation Buttons */}
      <div className="flex justify-between">
        <Button
          variant="outline"
          onClick={handleBack}
          disabled={step === 0 || loading}
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Geri
        </Button>

        <div className="flex gap-2">
          {step === 3 ? (
            <>
              <Button
                variant="outline"
                onClick={() => navigate('/system/semesters')}
              >
                Daha Sonra Aktifleştir
              </Button>
              <Button
                onClick={() => setActivateConfirmOpen(true)}
                disabled={loading}
                className="bg-green-600 hover:bg-green-700 text-white"
              >
                {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                <Play className="h-4 w-4 mr-2" />
                Dönemi Aktifleştir
              </Button>
            </>
          ) : (
            <Button
              onClick={handleNext}
              disabled={!canGoNext() || loading}
              className="bg-indigo-600 hover:bg-indigo-700 text-white"
            >
              {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              {step === 1
                ? isEditMode ? 'Kaydet ve Devam Et' : !createdSemester ? 'Dönemi Oluştur ve Devam Et' : 'Devam Et'
                : 'Devam Et'}
              <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          )}
        </div>
      </div>

      {/* Activate Confirmation */}
      <AlertDialog open={activateConfirmOpen} onOpenChange={setActivateConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Dönemi Aktifleştir</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{semesterName}</strong> dönemini aktifleştirmek istediğinize emin misiniz?
              <span className="block mt-2 text-amber-600 font-medium">
                Aktifleştirmeden sonra dönemlik ders yapısı donar. Ders ekleme/silme/değiştirme yapılamaz.
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleActivate}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Aktifleştir
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

// ============================================================================
// Step 1: Semester Info
// ============================================================================

function StepSemesterInfo({
  semesterName,
  setSemesterName,
  hardDeadline,
  setHardDeadline,
  suggestions,
  isEditMode,
}: {
  semesterName: string;
  setSemesterName: (v: string) => void;
  hardDeadline: string;
  setHardDeadline: (v: string) => void;
  suggestions: string[];
  isEditMode: boolean;
}) {
  const isNameValid = /^\d{4}-\d{4}-(Fall|Spring)$/.test(semesterName);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CalendarRange className="h-5 w-5" />
          Dönem Bilgileri
        </CardTitle>
        <CardDescription>
          Yeni dönemin adını ve son geçerlilik tarihini belirleyin
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-2">
          <Label>Dönem Adı *</Label>
          <Input
            placeholder="2025-2026-Fall"
            value={semesterName}
            onChange={(e) => setSemesterName(e.target.value)}
            disabled={isEditMode}
            className={isEditMode ? "bg-gray-100 dark:bg-gray-800 text-gray-500" : ""}
          />
          {!isEditMode && (
            <div className="flex gap-1 flex-wrap">
              {suggestions.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => setSemesterName(s)}
                  className="text-xs px-2 py-0.5 rounded bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300"
                >
                  {s}
                </button>
              ))}
            </div>
          )}
          {semesterName && !isNameValid && !isEditMode && (
            <p className="text-xs text-red-500 flex items-center gap-1">
              <AlertCircle className="h-3 w-3" />
              Format: YYYY-YYYY-Fall veya YYYY-YYYY-Spring
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label>Hard Deadline (Son Geçerlilik) *</Label>
          <Input
            type="datetime-local"
            value={hardDeadline}
            onChange={(e) => setHardDeadline(e.target.value)}
          />
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Bu tarihten sonra dönem otomatik olarak kilitlenir. Kimse (admin dahil) veri değiştiremez.
          </p>
        </div>

        <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 p-3">
          <p className="text-sm text-blue-700 dark:text-blue-300">
            Hard deadline, dönemin mutlak kilit tarihidir. Bu tarihten sonra not, yoklama, kayıt dahil hiçbir veri değiştirilemez.
            Admin bile müdahale edemez.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Step 2: Service Periods
// ============================================================================

function StepPeriods({
  periods,
  setPeriods,
  hardDeadline,
  closedDays,
  setClosedDays,
}: {
  periods: Record<string, { start: string; end: string }>;
  setPeriods: (v: any) => void;
  hardDeadline: string;
  closedDays: Array<{ date: string; reason: string }>;
  setClosedDays: React.Dispatch<React.SetStateAction<Array<{ date: string; reason: string }>>>;
}) {
  const deadline = hardDeadline ? new Date(hardDeadline) : null;

  const updatePeriod = (key: string, field: 'start' | 'end', value: string) => {
    setPeriods((prev: any) => ({
      ...prev,
      [key]: { ...prev[key], [field]: value },
    }));
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Clock className="h-5 w-5" />
          Servis Tarih Aralıkları
        </CardTitle>
        <CardDescription>
          Her servisin çalışma dönemini belirleyin. Bitiş tarihleri hard deadline'ı ({deadline ? format(deadline, 'dd MMM yyyy', { locale: tr }) : '—'}) aşamaz.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {PERIOD_SERVICES.map((svc) => {
          const p = periods[svc.key];
          const endExceedsDeadline = deadline && p.end && new Date(p.end) > deadline;
          const startAfterEnd = p.start && p.end && new Date(p.start) >= new Date(p.end);

          return (
            <div
              key={svc.key}
              className="rounded-lg border border-gray-200 dark:border-gray-700 p-4 space-y-3"
            >
              <div className="flex items-center justify-between">
                <div>
                  <h4 className="font-medium text-sm">{svc.label}</h4>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{svc.description}</p>
                </div>
                {p.start && p.end && !endExceedsDeadline && !startAfterEnd && (
                  <Badge variant="outline" className="border-green-500 text-green-600">
                    <CheckCircle2 className="h-3 w-3 mr-1" /> Geçerli
                  </Badge>
                )}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs">Başlangıç</Label>
                  <Input
                    type="datetime-local"
                    value={p.start}
                    onChange={(e) => updatePeriod(svc.key, 'start', e.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Bitiş</Label>
                  <Input
                    type="datetime-local"
                    value={p.end}
                    onChange={(e) => updatePeriod(svc.key, 'end', e.target.value)}
                  />
                </div>
              </div>
              {endExceedsDeadline && (
                <p className="text-xs text-red-500 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  Bitiş tarihi hard deadline'ı aşamaz
                </p>
              )}
              {startAfterEnd && (
                <p className="text-xs text-red-500 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  Başlangıç tarihi bitiş tarihinden önce olmalıdır
                </p>
              )}
            </div>
          );
        })}

        {/* Meal Service — Closed Days */}
        <ClosedDaysCard
          closedDays={closedDays}
          setClosedDays={setClosedDays}
          hardDeadline={hardDeadline}
        />

        <div className="rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-3">
          <p className="text-sm text-amber-700 dark:text-amber-300">
            "Kaydet ve Devam Et" butonuna basıldığında girdiğiniz tarihler kaydedilecektir.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Closed Days Card (used in Step 2)
// ============================================================================

function ClosedDaysCard({
  closedDays,
  setClosedDays,
  hardDeadline,
}: {
  closedDays: Array<{ date: string; reason: string }>;
  setClosedDays: React.Dispatch<React.SetStateAction<Array<{ date: string; reason: string }>>>;
  hardDeadline: string;
}) {
  const [newDate, setNewDate] = useState('');
  const [newReason, setNewReason] = useState('');

  const deadline = hardDeadline ? new Date(hardDeadline) : null;

  const addClosedDay = () => {
    if (!newDate || !newReason.trim()) return;
    if (closedDays.some((d) => d.date === newDate)) return;

    setClosedDays((prev) => [...prev, { date: newDate, reason: newReason.trim() }]);
    setNewDate('');
    setNewReason('');
  };

  const removeClosedDay = (date: string) => {
    setClosedDays((prev) => prev.filter((d) => d.date !== date));
  };

  const isDuplicate = closedDays.some((d) => d.date === newDate);
  const exceedsDeadline = deadline && newDate && new Date(newDate + 'T23:59:59') > deadline;

  return (
    <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50/30 dark:bg-orange-950/10 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="font-medium text-sm flex items-center gap-1.5">
            <UtensilsCrossed className="h-4 w-4 text-orange-600" />
            Yemekhane Kapalı Günler
          </h4>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Yemekhanenin kapalı olacağı özel günler (resmi tatiller vb.)
          </p>
        </div>
        {closedDays.length > 0 && (
          <Badge variant="outline" className="border-orange-500 text-orange-600">
            {closedDays.length} gün
          </Badge>
        )}
      </div>

      {/* Add new closed day */}
      <div className="flex gap-2 items-end">
        <div className="space-y-1 flex-shrink-0">
          <Label className="text-xs">Tarih</Label>
          <Input
            type="date"
            value={newDate}
            onChange={(e) => setNewDate(e.target.value)}
            className="w-[160px]"
          />
        </div>
        <div className="space-y-1 flex-1">
          <Label className="text-xs">Sebep</Label>
          <Input
            type="text"
            placeholder="ör. Cumhuriyet Bayramı"
            value={newReason}
            onChange={(e) => setNewReason(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addClosedDay();
              }
            }}
          />
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={addClosedDay}
          disabled={!newDate || !newReason.trim() || isDuplicate || !!exceedsDeadline}
          className="flex-shrink-0"
        >
          <Plus className="h-4 w-4 mr-1" />
          Ekle
        </Button>
      </div>

      {isDuplicate && (
        <p className="text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          Bu tarih zaten eklenmiş
        </p>
      )}
      {exceedsDeadline && (
        <p className="text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          Tarih hard deadline'ı aşamaz
        </p>
      )}

      {/* List of added closed days */}
      {closedDays.length > 0 && (
        <div className="rounded-md border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {closedDays
            .sort((a, b) => a.date.localeCompare(b.date))
            .map((day) => (
              <div key={day.date} className="flex items-center justify-between px-3 py-2 text-sm">
                <div className="flex items-center gap-3">
                  <span className="font-mono text-xs text-gray-500">{day.date}</span>
                  <span className="text-xs text-gray-400">
                    {format(new Date(day.date + 'T00:00:00'), 'EEEE', { locale: tr })}
                  </span>
                  <span>{day.reason}</span>
                </div>
                <button
                  type="button"
                  onClick={() => removeClosedDay(day.date)}
                  className="text-red-400 hover:text-red-600 p-1"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Step 3: Course Addition (delegates to existing page)
// ============================================================================

function StepCourses({ semesterName }: { semesterName: string }) {
  const buildDepartmentHref = (dept: Department, faculty: Faculty) => {
    const params = new URLSearchParams({
      semester: semesterName,
      from: 'wizard',
      faculty_id: faculty.id,
      department_id: dept.id,
    });
    return `/semester-courses?${params.toString()}`;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GraduationCap className="h-5 w-5" />
          Dönemlik Ders Ekleme
        </CardTitle>
        <CardDescription>
          <strong>{semesterName}</strong> dönemine ders ekleyin. Bir bölüm seçerek ders ekleme sayfasına gidin.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <CourseHierarchyView
          semesterName={semesterName}
          showWizardActions={true}
          onAddCourse={(dept, faculty) => {
            window.open(buildDepartmentHref(dept, faculty), '_blank', 'noopener,noreferrer');
          }}
        />
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Step 4: Preview & Activate
// ============================================================================

function StepPreview({
  semesterName,
  hardDeadline,
  periods,
  resolvedSemesterName,
  closedDays,
}: {
  semesterName: string;
  hardDeadline: string;
  periods: Record<string, { start: string; end: string }>;
  resolvedSemesterName: string;
  closedDays: Array<{ date: string; reason: string }>;
}) {
  const deadline = new Date(hardDeadline);

  return (
    <div className="space-y-4">
      {/* Semester Info Summary */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <CalendarRange className="h-5 w-5" />
            Dönem Özeti
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="text-sm text-gray-500 dark:text-gray-400">Dönem Adı</span>
              <p className="font-medium text-lg">{semesterName}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500 dark:text-gray-400">Hard Deadline</span>
              <p className="font-medium text-lg">
                {format(deadline, 'dd MMMM yyyy HH:mm', { locale: tr })}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Periods Summary */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Clock className="h-5 w-5" />
            Servis Tarih Aralıkları
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Servis</TableHead>
                  <TableHead>Başlangıç</TableHead>
                  <TableHead>Bitiş</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {PERIOD_SERVICES.map((svc) => {
                  const p = periods[svc.key];
                  return (
                    <TableRow key={svc.key}>
                      <TableCell className="font-medium">{svc.label}</TableCell>
                      <TableCell>
                        {p.start ? format(new Date(p.start), 'dd MMM yyyy HH:mm', { locale: tr }) : '—'}
                      </TableCell>
                      <TableCell>
                        {p.end ? format(new Date(p.end), 'dd MMM yyyy HH:mm', { locale: tr }) : '—'}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Closed Days Summary */}
      {closedDays.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <UtensilsCrossed className="h-5 w-5" />
              Yemekhane Kapalı Günler
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="rounded-lg border border-gray-200 dark:border-gray-700">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tarih</TableHead>
                    <TableHead>Gün</TableHead>
                    <TableHead>Sebep</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {closedDays
                    .sort((a, b) => a.date.localeCompare(b.date))
                    .map((day) => (
                      <TableRow key={day.date}>
                        <TableCell className="font-mono text-sm">{day.date}</TableCell>
                        <TableCell>
                          {format(new Date(day.date + 'T00:00:00'), 'EEEE', { locale: tr })}
                        </TableCell>
                        <TableCell>{day.reason}</TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Courses Summary - Hierarchy View */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <GraduationCap className="h-5 w-5" />
            Açılan Dersler
          </CardTitle>
        </CardHeader>
        <CardContent>
          <CourseHierarchyView semesterName={resolvedSemesterName} readOnly />
        </CardContent>
      </Card>

      {/* Warning */}
      <div className="rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-amber-600 mt-0.5" />
          <div>
            <h4 className="font-medium text-amber-800 dark:text-amber-300">Aktifleştirme Uyarısı</h4>
            <p className="text-sm text-amber-700 dark:text-amber-400 mt-1">
              Dönemi aktifleştirdiğinizde ders yapısı (ders ekleme, silme, hoca değişikliği vb.) tamamen donar.
              Aktifleştirmeden önce tüm derslerin eklendiğinden emin olun.
              Aktifleştirmek istemiyorsanız "Daha Sonra Aktifleştir" butonunu kullanabilirsiniz.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
