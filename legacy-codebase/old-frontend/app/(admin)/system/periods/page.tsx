'use client';

import { useCallback, useEffect, useState } from 'react';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import { CalendarRange, Plus, Pencil, Trash2, RefreshCw, Loader2, CalendarOff, ShieldCheck, AlertTriangle } from 'lucide-react';

// Returns the 2 semesters that physically occur in the current calendar year.
// 2025 → ["2024-2025-Spring", "2025-2026-Fall"]
// 2026 → ["2025-2026-Spring", "2026-2027-Fall"]
function getSemesterOptions(): string[] {
  const year = new Date().getFullYear();
  return [
    `${year - 1}-${year}-Spring`,
    `${year}-${year + 1}-Fall`,
  ];
}

const SEMESTER_OPTIONS = getSemesterOptions();

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import Toast from '@/components/enrollment/Toast';

import type {
  AcademicPeriod,
  SimplePeriod,
  CreatePeriodRequest,
  UpdatePeriodRequest,
  ClosedDay,
  Semester,
} from '@/lib/types';
import {
  listGradesPeriods,
  createGradesPeriod,
  updateGradesPeriod,
  deleteGradesPeriod,
  listSimplePeriods,
  createSimplePeriod,
  updateSimplePeriod,
  deleteSimplePeriod,
  listClosedDays,
  createClosedDay,
  deleteClosedDay,
  SERVICE_KEYS,
  getServiceLabel,
  getActiveSemester,
} from '@/lib/services/system-service';
import type { SimplePeriodServiceKey } from '@/lib/services/system-service';
import { catalogApiSafe } from '@/lib/api-client';

// Ders kodu → UUID lookup (catalog public endpoint)
async function lookupCourseUUID(courseCode: string): Promise<string> {
  const res = await catalogApiSafe.get(`courses/${courseCode.trim().toUpperCase()}`).json<{ id: string }>();
  return res.id;
}

export default function PeriodsPage() {
  const [toast, setToast] = useState<{
    message: string;
    type: 'error' | 'warning' | 'success' | 'info';
    isVisible: boolean;
  }>({ message: '', type: 'info', isVisible: false });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  const [activeTab, setActiveTab] = useState<string>('grades');
  const [activeSemester, setActiveSemester] = useState<Semester | null>(null);
  const [semesterLoading, setSemesterLoading] = useState(true);

  useEffect(() => {
    getActiveSemester()
      .then(setActiveSemester)
      .finally(() => setSemesterLoading(false));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dönem Yönetimi</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Servis bazlı akademik dönem deadline&apos;larını ve yemekhane kapalı günlerini yönet
        </p>
      </div>

      {/* Active semester banner */}
      {!semesterLoading && (
        activeSemester ? (
          <div className="rounded-lg border border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20 p-3">
            <div className="flex items-center gap-2 text-sm">
              <ShieldCheck className="h-4 w-4 text-green-600 dark:text-green-400" />
              <span className="text-green-800 dark:text-green-300">
                Aktif dönem: <strong>{activeSemester.name}</strong> — Period CRUD işlemleri bu döneme ait semester aktif olduğu sürece çalışır.
              </span>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20 p-3">
            <div className="flex items-center gap-2 text-sm">
              <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
              <span className="text-amber-800 dark:text-amber-300">
                Aktif dönem bulunamadı — Period oluşturma/güncelleme/silme işlemleri çalışmayabilir. Önce <a href="/system/semesters" className="underline font-medium">Dönem Durumu</a> sayfasından bir dönem aktif edin.
              </span>
            </div>
          </div>
        )
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <CalendarRange className="h-5 w-5 text-indigo-600" />
            <CardTitle>Dönem & Kapalı Gün Yönetimi</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-4">
              {SERVICE_KEYS.map((key) => (
                <TabsTrigger key={key} value={key}>
                  {getServiceLabel(key)}
                </TabsTrigger>
              ))}
            </TabsList>

            {/* Grades tab — academic periods with course_id */}
            <TabsContent value="grades">
              <GradesPeriodsTab showToast={showToast} />
            </TabsContent>

            {/* Enrollment tab — simple academic periods */}
            <TabsContent value="enrollment">
              <SimplePeriodsTab serviceKey="enrollment" showToast={showToast} />
            </TabsContent>

            {/* Meal tab — closed days */}
            <TabsContent value="meal">
              <ClosedDaysTab showToast={showToast} />
            </TabsContent>

            {/* Catalog tab — simple academic periods */}
            <TabsContent value="catalog">
              <SimplePeriodsTab serviceKey="catalog" showToast={showToast} />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

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
// GRADES PERIODS TAB — with course_id support
// ============================================================================

function GradesPeriodsTab({ showToast }: { showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void }) {
  const [periods, setPeriods] = useState<AcademicPeriod[]>([]);
  const [loading, setLoading] = useState(false);
  const [semesterFilter, setSemesterFilter] = useState('');

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<AcademicPeriod | null>(null);

  const [formSemester, setFormSemester] = useState('');
  const [formStart, setFormStart] = useState('');
  const [formEnd, setFormEnd] = useState('');
  const [courseCodeInput, setCourseCodeInput] = useState('');
  const [editData, setEditData] = useState<UpdatePeriodRequest>({});
  const [formLoading, setFormLoading] = useState(false);

  const fetchPeriods = useCallback(async () => {
    setLoading(true);
    try {
      setPeriods(await listGradesPeriods(semesterFilter || undefined));
    } catch {
      showToast('Dönemler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [semesterFilter, showToast]);

  useEffect(() => { fetchPeriods(); }, [fetchPeriods]);

  const handleCreate = async () => {
    if (!formSemester || !formStart || !formEnd) {
      showToast('Lütfen tüm zorunlu alanları doldurun', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      const payload: CreatePeriodRequest = {
        semester: formSemester,
        period_start: new Date(formStart).toISOString(),
        period_end: new Date(formEnd).toISOString(),
      };

      if (courseCodeInput.trim()) {
        try {
          payload.course_id = await lookupCourseUUID(courseCodeInput);
        } catch {
          showToast(`"${courseCodeInput}" kodu bulunamadı`, 'error');
          setFormLoading(false);
          return;
        }
      }

      await createGradesPeriod(payload);
      showToast('Dönem başarıyla oluşturuldu', 'success');
      setCreateDialogOpen(false);
      setFormSemester(''); setFormStart(''); setFormEnd(''); setCourseCodeInput('');
      await fetchPeriods();
    } catch {
      showToast('Dönem oluşturulamadı (aynı dönem zaten var olabilir)', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      const payload: UpdatePeriodRequest = {};
      if (editData.period_end) payload.period_end = new Date(editData.period_end).toISOString();
      if (editData.is_active !== undefined) payload.is_active = editData.is_active;
      await updateGradesPeriod(selectedPeriod.id, payload);
      showToast('Dönem başarıyla güncellendi', 'success');
      setEditDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Dönem güncellenemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      await deleteGradesPeriod(selectedPeriod.id);
      showToast('Dönem başarıyla silindi', 'success');
      setDeleteDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Dönem silinemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const openEdit = (period: AcademicPeriod) => {
    setSelectedPeriod(period);
    setEditData({
      period_end: period.period_end ? new Date(period.period_end).toISOString().slice(0, 16) : '',
      is_active: period.is_active,
    });
    setEditDialogOpen(true);
  };

  return (
    <>
      <div className="flex items-end gap-3 mb-4">
        <div className="flex-1 max-w-xs">
          <Label>Semester Filtresi</Label>
          <select
            value={semesterFilter}
            onChange={(e) => setSemesterFilter(e.target.value)}
            className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          >
            <option value="">Tümü</option>
            {SEMESTER_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <Button variant="outline" size="sm" onClick={fetchPeriods} disabled={loading}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
        <Button onClick={() => setCreateDialogOpen(true)} className="bg-indigo-600 hover:bg-indigo-700 text-white">
          <Plus className="h-4 w-4 mr-2" />
          Yeni Dönem
        </Button>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Semester</TableHead>
              <TableHead>Başlangıç</TableHead>
              <TableHead>Bitiş</TableHead>
              <TableHead>Ders ID</TableHead>
              <TableHead>Durum</TableHead>
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
            ) : periods.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-6 text-gray-500">
                  Tanımlı dönem bulunamadı
                </TableCell>
              </TableRow>
            ) : (
              periods.map((period) => (
                <TableRow key={period.id}>
                  <TableCell className="font-medium">{period.semester}</TableCell>
                  <TableCell className="text-sm">
                    {format(new Date(period.period_start), 'dd MMM yyyy HH:mm', { locale: tr })}
                  </TableCell>
                  <TableCell className="text-sm">
                    {format(new Date(period.period_end), 'dd MMM yyyy HH:mm', { locale: tr })}
                  </TableCell>
                  <TableCell className="text-sm text-gray-500">
                    {period.course_id ? (
                      <span className="font-mono text-xs">{period.course_id.slice(0, 8)}...</span>
                    ) : (
                      <Badge variant="outline">Global</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {period.is_active ? (
                      <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400">Aktif</Badge>
                    ) : (
                      <Badge variant="outline" className="border-gray-400 text-gray-500">Pasif</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(period)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost" size="sm"
                        onClick={() => { setSelectedPeriod(period); setDeleteDialogOpen(true); }}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormSemester(''); setFormStart(''); setFormEnd(''); setCourseCodeInput(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yeni Dönem — Notlar</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Semester *</Label>
              <select
                value={formSemester}
                onChange={(e) => setFormSemester(e.target.value)}
                className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <option value="">Dönem seçin...</option>
                {SEMESTER_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <Label>Başlangıç *</Label>
              <Input type="datetime-local" value={formStart} onChange={(e) => setFormStart(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>Bitiş *</Label>
              <Input type="datetime-local" value={formEnd} onChange={(e) => setFormEnd(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>Ders Kodu <span className="text-gray-400 font-normal">(boş = tüm dersler için geçerli)</span></Label>
              <Input
                placeholder="CS101, MAT201..."
                value={courseCodeInput}
                onChange={(e) => setCourseCodeInput(e.target.value)}
                className="mt-1"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>İptal</Button>
              <Button onClick={handleCreate} disabled={formLoading} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Oluştur
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Dönemi Düzenle</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Semester</Label>
              <Input value={selectedPeriod?.semester || ''} disabled className="mt-1" />
            </div>
            <div>
              <Label>Bitiş Tarihi</Label>
              <Input
                type="datetime-local"
                value={typeof editData.period_end === 'string' ? editData.period_end : ''}
                onChange={(e) => setEditData({ ...editData, period_end: e.target.value })}
                className="mt-1"
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="grades-edit-active"
                checked={editData.is_active ?? true}
                onChange={(e) => setEditData({ ...editData, is_active: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300"
              />
              <Label htmlFor="grades-edit-active">Aktif</Label>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setEditDialogOpen(false)}>İptal</Button>
              <Button onClick={handleEdit} disabled={formLoading} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Güncelle
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Dönemi Sil</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{selectedPeriod?.semester}</strong> dönemini silmek istediğinize emin misiniz? Bu işlem geri alınamaz.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-red-600 hover:bg-red-700 text-white">
              {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Sil
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

// ============================================================================
// SIMPLE PERIODS TAB — Enrollment & Catalog (no course_id)
// ============================================================================

function SimplePeriodsTab({ serviceKey, showToast }: { serviceKey: SimplePeriodServiceKey; showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void }) {
  const [periods, setPeriods] = useState<SimplePeriod[]>([]);
  const [loading, setLoading] = useState(false);
  const [semesterFilter, setSemesterFilter] = useState('');

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<SimplePeriod | null>(null);

  const [formSemester, setFormSemester] = useState('');
  const [formStart, setFormStart] = useState('');
  const [formEnd, setFormEnd] = useState('');
  const [editData, setEditData] = useState<UpdatePeriodRequest>({});
  const [formLoading, setFormLoading] = useState(false);

  const fetchPeriods = useCallback(async () => {
    setLoading(true);
    try {
      setPeriods(await listSimplePeriods(serviceKey, semesterFilter || undefined));
    } catch {
      showToast('Dönemler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [serviceKey, semesterFilter, showToast]);

  useEffect(() => { fetchPeriods(); }, [fetchPeriods]);

  const handleCreate = async () => {
    if (!formSemester || !formStart || !formEnd) {
      showToast('Lütfen tüm zorunlu alanları doldurun', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      await createSimplePeriod(serviceKey, {
        semester: formSemester,
        period_start: new Date(formStart).toISOString(),
        period_end: new Date(formEnd).toISOString(),
      });
      showToast('Dönem başarıyla oluşturuldu', 'success');
      setCreateDialogOpen(false);
      setFormSemester(''); setFormStart(''); setFormEnd('');
      await fetchPeriods();
    } catch {
      showToast('Dönem oluşturulamadı (aynı dönem zaten var olabilir)', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      const payload: UpdatePeriodRequest = {};
      if (editData.period_end) payload.period_end = new Date(editData.period_end).toISOString();
      if (editData.is_active !== undefined) payload.is_active = editData.is_active;
      await updateSimplePeriod(serviceKey, selectedPeriod.id, payload);
      showToast('Dönem başarıyla güncellendi', 'success');
      setEditDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Dönem güncellenemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      await deleteSimplePeriod(serviceKey, selectedPeriod.id);
      showToast('Dönem başarıyla silindi', 'success');
      setDeleteDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Dönem silinemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const openEdit = (period: SimplePeriod) => {
    setSelectedPeriod(period);
    setEditData({
      period_end: period.period_end ? new Date(period.period_end).toISOString().slice(0, 16) : '',
      is_active: period.is_active,
    });
    setEditDialogOpen(true);
  };

  const serviceLabel = getServiceLabel(serviceKey);

  return (
    <>
      <div className="flex items-end gap-3 mb-4">
        <div className="flex-1 max-w-xs">
          <Label>Semester Filtresi</Label>
          <select
            value={semesterFilter}
            onChange={(e) => setSemesterFilter(e.target.value)}
            className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          >
            <option value="">Tümü</option>
            {SEMESTER_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <Button variant="outline" size="sm" onClick={fetchPeriods} disabled={loading}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
        <Button onClick={() => setCreateDialogOpen(true)} className="bg-indigo-600 hover:bg-indigo-700 text-white">
          <Plus className="h-4 w-4 mr-2" />
          Yeni Dönem
        </Button>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Semester</TableHead>
              <TableHead>Başlangıç</TableHead>
              <TableHead>Bitiş</TableHead>
              <TableHead>Durum</TableHead>
              <TableHead className="text-right">İşlemler</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-6">
                  <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                </TableCell>
              </TableRow>
            ) : periods.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-6 text-gray-500">
                  Tanımlı dönem bulunamadı
                </TableCell>
              </TableRow>
            ) : (
              periods.map((period) => (
                <TableRow key={period.id}>
                  <TableCell className="font-medium">{period.semester}</TableCell>
                  <TableCell className="text-sm">
                    {format(new Date(period.period_start), 'dd MMM yyyy HH:mm', { locale: tr })}
                  </TableCell>
                  <TableCell className="text-sm">
                    {format(new Date(period.period_end), 'dd MMM yyyy HH:mm', { locale: tr })}
                  </TableCell>
                  <TableCell>
                    {period.is_active ? (
                      <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400">Aktif</Badge>
                    ) : (
                      <Badge variant="outline" className="border-gray-400 text-gray-500">Pasif</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(period)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost" size="sm"
                        onClick={() => { setSelectedPeriod(period); setDeleteDialogOpen(true); }}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormSemester(''); setFormStart(''); setFormEnd(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yeni Dönem — {serviceLabel}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Semester *</Label>
              <select
                value={formSemester}
                onChange={(e) => setFormSemester(e.target.value)}
                className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                <option value="">Dönem seçin...</option>
                {SEMESTER_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <Label>Başlangıç *</Label>
              <Input type="datetime-local" value={formStart} onChange={(e) => setFormStart(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>Bitiş *</Label>
              <Input type="datetime-local" value={formEnd} onChange={(e) => setFormEnd(e.target.value)} className="mt-1" />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>İptal</Button>
              <Button onClick={handleCreate} disabled={formLoading} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Oluştur
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Dönemi Düzenle</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Semester</Label>
              <Input value={selectedPeriod?.semester || ''} disabled className="mt-1" />
            </div>
            <div>
              <Label>Bitiş Tarihi</Label>
              <Input
                type="datetime-local"
                value={typeof editData.period_end === 'string' ? editData.period_end : ''}
                onChange={(e) => setEditData({ ...editData, period_end: e.target.value })}
                className="mt-1"
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id={`${serviceKey}-edit-active`}
                checked={editData.is_active ?? true}
                onChange={(e) => setEditData({ ...editData, is_active: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300"
              />
              <Label htmlFor={`${serviceKey}-edit-active`}>Aktif</Label>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setEditDialogOpen(false)}>İptal</Button>
              <Button onClick={handleEdit} disabled={formLoading} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Güncelle
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Dönemi Sil</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{selectedPeriod?.semester}</strong> dönemini silmek istediğinize emin misiniz? Bu işlem geri alınamaz.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-red-600 hover:bg-red-700 text-white">
              {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Sil
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

// ============================================================================
// CLOSED DAYS TAB — Meal service (holidays)
// ============================================================================

function ClosedDaysTab({ showToast }: { showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void }) {
  const [closedDays, setClosedDays] = useState<ClosedDay[]>([]);
  const [loading, setLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDay, setSelectedDay] = useState<ClosedDay | null>(null);

  const [formDate, setFormDate] = useState('');
  const [formReason, setFormReason] = useState('');
  const [formLoading, setFormLoading] = useState(false);

  const fetchClosedDays = useCallback(async () => {
    setLoading(true);
    try {
      setClosedDays(await listClosedDays());
    } catch {
      showToast('Kapalı günler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => { fetchClosedDays(); }, [fetchClosedDays]);

  const handleCreate = async () => {
    if (!formDate || !formReason.trim()) {
      showToast('Lütfen tarih ve sebep girin', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      await createClosedDay({ date: formDate, reason: formReason.trim() });
      showToast('Kapalı gün başarıyla eklendi', 'success');
      setCreateDialogOpen(false);
      setFormDate(''); setFormReason('');
      await fetchClosedDays();
    } catch {
      showToast('Kapalı gün eklenemedi (bu tarih zaten kayıtlı olabilir)', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedDay) return;
    setFormLoading(true);
    try {
      await deleteClosedDay(selectedDay.id);
      showToast('Kapalı gün başarıyla silindi', 'success');
      setDeleteDialogOpen(false);
      setSelectedDay(null);
      await fetchClosedDays();
    } catch {
      showToast('Kapalı gün silinemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  return (
    <>
      <div className="mb-4 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
        <p className="text-sm text-amber-700 dark:text-amber-300">
          <CalendarOff className="h-4 w-4 inline mr-1.5 -mt-0.5" />
          Yemekhane randevu penceresi sabittir (Pazartesi-Cuma, bir sonraki hafta). Burada sadece yemekhanenin kapalı olduğu günleri (resmi tatiller, bayramlar vb.) tanımlayın. Bu günlerde öğrenciler randevu alamaz.
        </p>
      </div>

      <div className="flex items-end gap-3 mb-4">
        <Button variant="outline" size="sm" onClick={fetchClosedDays} disabled={loading}>
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
        <Button onClick={() => setCreateDialogOpen(true)} className="bg-amber-600 hover:bg-amber-700 text-white">
          <Plus className="h-4 w-4 mr-2" />
          Kapalı Gün Ekle
        </Button>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Tarih</TableHead>
              <TableHead>Sebep</TableHead>
              <TableHead>Eklenme Tarihi</TableHead>
              <TableHead className="text-right">İşlemler</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-6">
                  <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                </TableCell>
              </TableRow>
            ) : closedDays.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-6 text-gray-500">
                  Tanımlı kapalı gün bulunamadı
                </TableCell>
              </TableRow>
            ) : (
              closedDays.map((day) => (
                <TableRow key={day.id}>
                  <TableCell className="font-medium">
                    {format(new Date(day.date + 'T00:00:00'), 'dd MMMM yyyy, EEEE', { locale: tr })}
                  </TableCell>
                  <TableCell>{day.reason}</TableCell>
                  <TableCell className="text-sm text-gray-500">
                    {format(new Date(day.created_at), 'dd MMM yyyy', { locale: tr })}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost" size="sm"
                      onClick={() => { setSelectedDay(day); setDeleteDialogOpen(true); }}
                      className="text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-900/20"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormDate(''); setFormReason(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Kapalı Gün Ekle — Yemekhane</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Tarih *</Label>
              <Input type="date" value={formDate} onChange={(e) => setFormDate(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>Sebep *</Label>
              <Input
                placeholder="Cumhuriyet Bayramı, Ramazan Bayramı..."
                value={formReason}
                onChange={(e) => setFormReason(e.target.value)}
                className="mt-1"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>İptal</Button>
              <Button onClick={handleCreate} disabled={formLoading} className="bg-amber-600 hover:bg-amber-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Ekle
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Kapalı Günü Sil</AlertDialogTitle>
            <AlertDialogDescription>
              <strong>{selectedDay?.date && format(new Date(selectedDay.date + 'T00:00:00'), 'dd MMMM yyyy', { locale: tr })}</strong> — <em>{selectedDay?.reason}</em> kaydını silmek istediğinize emin misiniz?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-red-600 hover:bg-red-700 text-white">
              {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Sil
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
