import { useCallback, useEffect, useState } from 'react';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import { CalendarOff, Plus, Pencil, Trash2, RefreshCw, Loader2, AlertTriangle } from 'lucide-react';

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
  getServiceLabel,
} from '@/lib/services/system-service';
import type { SimplePeriodServiceKey } from '@/lib/services/system-service';
import { catalogApiSafe } from '@/lib/api-client';

async function lookupCourseUUID(courseCode: string): Promise<string> {
  const res = await catalogApiSafe.get(`courses/${courseCode.trim().toUpperCase()}`).json<{ id: string }>();
  return res.id;
}

// ============================================================================
// GRADES PERIODS TAB — with course_id support
// ============================================================================

export function GradesPeriodsTab({ showToast, activeSemester }: { showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void; activeSemester?: Semester }) {
  const [periods, setPeriods] = useState<AcademicPeriod[]>([]);
  const [loading, setLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<AcademicPeriod | null>(null);

  const [formStart, setFormStart] = useState('');
  const [formEnd, setFormEnd] = useState('');
  const [courseCodeInput, setCourseCodeInput] = useState('');
  const [editData, setEditData] = useState<UpdatePeriodRequest>({});
  const [formLoading, setFormLoading] = useState(false);

  const fetchPeriods = useCallback(async () => {
    if (!activeSemester) {
      setPeriods([]);
      return;
    }
    setLoading(true);
    try {
      setPeriods(await listGradesPeriods(activeSemester.name));
    } catch {
      showToast('Dönemler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [activeSemester, showToast]);

  useEffect(() => { fetchPeriods(); }, [fetchPeriods]);

  if (!activeSemester) {
    return (
      <div className="pt-8 text-center text-gray-500">
        <AlertTriangle className="h-8 w-8 mx-auto mb-3 text-amber-500" />
        <p>Not servisi işlemleri için lütfen aktif bir dönem açılışı yapın.</p>
      </div>
    );
  }

  const handleCreate = async () => {
    if (!formStart || !formEnd) {
      showToast('Lütfen tüm zorunlu alanları doldurun', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      const payload: CreatePeriodRequest = {
        semester: activeSemester.name,
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
      showToast('Deadline başarıyla oluşturuldu', 'success');
      setCreateDialogOpen(false);
      setFormStart(''); setFormEnd(''); setCourseCodeInput('');
      await fetchPeriods();
    } catch {
      showToast('Oluşturulamadı (aynı kayıt zaten var olabilir)', 'error');
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
      showToast('Deadline başarıyla güncellendi', 'success');
      setEditDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Güncellenemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      await deleteGradesPeriod(selectedPeriod.id);
      showToast('Deadline başarıyla silindi', 'success');
      setDeleteDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Silinemedi', 'error');
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
    <div className="pt-4 space-y-4">
      <div className="flex items-end justify-between mb-4">
        <div>
          <h3 className="text-lg font-medium">Not Servisi ({activeSemester.name})</h3>
          <p className="text-sm text-gray-500">Hocaların not girebileceği tarih aralıklarını belirleyin.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchPeriods} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button onClick={() => setCreateDialogOpen(true)} className="bg-indigo-600 hover:bg-indigo-700 text-white">
            <Plus className="h-4 w-4 mr-2" />
            Yeni Deadline
          </Button>
        </div>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
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
                <TableCell colSpan={5} className="text-center py-6">
                  <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                </TableCell>
              </TableRow>
            ) : periods.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-6 text-gray-500">
                  Bu döneme ait tanımlı deadline bulunamadı
                </TableCell>
              </TableRow>
            ) : (
              periods.map((period) => (
                <TableRow key={period.id}>
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
                      <Badge variant="outline">Tüm Dersler</Badge>
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
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormStart(''); setFormEnd(''); setCourseCodeInput(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yeni Deadline — Notlar ({activeSemester.name})</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
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
            <DialogTitle>Deadline Düzenle</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
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
            <AlertDialogTitle>Silimi Onayla</AlertDialogTitle>
            <AlertDialogDescription>
              Bu deadline kaydını silmek istediğinize emin misiniz? Bu işlem geri alınamaz.
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
    </div>
  );
}

// ============================================================================
// SIMPLE PERIODS TAB — Enrollment & Catalog (no course_id)
// ============================================================================

export function SimplePeriodsTab({ serviceKey, showToast, activeSemester }: { serviceKey: SimplePeriodServiceKey; showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void; activeSemester?: Semester }) {
  const [periods, setPeriods] = useState<SimplePeriod[]>([]);
  const [loading, setLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<SimplePeriod | null>(null);

  const [formStart, setFormStart] = useState('');
  const [formEnd, setFormEnd] = useState('');
  const [editData, setEditData] = useState<UpdatePeriodRequest>({});
  const [formLoading, setFormLoading] = useState(false);

  const fetchPeriods = useCallback(async () => {
    if (!activeSemester) {
      setPeriods([]);
      return;
    }
    setLoading(true);
    try {
      setPeriods(await listSimplePeriods(serviceKey, activeSemester.name));
    } catch {
      showToast('Kayıtlar yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [serviceKey, activeSemester, showToast]);

  useEffect(() => { fetchPeriods(); }, [fetchPeriods]);

  if (!activeSemester) {
    return (
      <div className="pt-8 text-center text-gray-500">
        <AlertTriangle className="h-8 w-8 mx-auto mb-3 text-amber-500" />
        <p>İşlem yapabilmek için lütfen aktif bir dönem açılışı yapın.</p>
      </div>
    );
  }

  const handleCreate = async () => {
    if (!formStart || !formEnd) {
      showToast('Lütfen tüm zorunlu alanları doldurun', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      await createSimplePeriod(serviceKey, {
        semester: activeSemester.name,
        period_start: new Date(formStart).toISOString(),
        period_end: new Date(formEnd).toISOString(),
      });
      showToast('Başarıyla oluşturuldu', 'success');
      setCreateDialogOpen(false);
      setFormStart(''); setFormEnd('');
      await fetchPeriods();
    } catch {
      showToast('Oluşturulamadı (zaten var olabilir)', 'error');
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
      showToast('Başarıyla güncellendi', 'success');
      setEditDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Güncellenemedi', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedPeriod) return;
    setFormLoading(true);
    try {
      await deleteSimplePeriod(serviceKey, selectedPeriod.id);
      showToast('Başarıyla silindi', 'success');
      setDeleteDialogOpen(false);
      setSelectedPeriod(null);
      await fetchPeriods();
    } catch {
      showToast('Silinemedi', 'error');
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
    <div className="pt-4 space-y-4">
      <div className="flex items-end justify-between mb-4">
        <div>
          <h3 className="text-lg font-medium">{serviceLabel === 'Katalog/Ders Programı' ? 'Katalog Servisi' : 'Ders Kayıt Sistemi'} ({activeSemester.name})</h3>
          <p className="text-sm text-gray-500">Tarih aralıklarını belirleyin.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchPeriods} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button onClick={() => setCreateDialogOpen(true)} className="bg-indigo-600 hover:bg-indigo-700 text-white">
            <Plus className="h-4 w-4 mr-2" />
            Yeni Deadline
          </Button>
        </div>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Başlangıç</TableHead>
              <TableHead>Bitiş</TableHead>
              <TableHead>Durum</TableHead>
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
            ) : periods.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-6 text-gray-500">
                  Kayıt bulunamadı
                </TableCell>
              </TableRow>
            ) : (
              periods.map((period) => (
                <TableRow key={period.id}>
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
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormStart(''); setFormEnd(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yeni Deadline</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
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
            <DialogTitle>Düzenle</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
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
            <AlertDialogTitle>Onay</AlertDialogTitle>
            <AlertDialogDescription>
              Kaydı silmek istediğinize emin misiniz? Bu işlem geri alınamaz.
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
    </div>
  );
}

// ============================================================================
// CLOSED DAYS TAB — Meal service (holidays)
// ============================================================================

export function ClosedDaysTab({ showToast, activeSemester }: { showToast: (msg: string, type: 'error' | 'warning' | 'success' | 'info') => void; activeSemester?: Semester }) {
  const [closedDays, setClosedDays] = useState<ClosedDay[]>([]);
  const [loading, setLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDay, setSelectedDay] = useState<ClosedDay | null>(null);

  const [formDate, setFormDate] = useState('');
  const [formReason, setFormReason] = useState('');
  const [formLoading, setFormLoading] = useState(false);

  const fetchClosedDays = useCallback(async () => {
    if (!activeSemester) {
      setClosedDays([]);
      return;
    }
    setLoading(true);
    try {
      setClosedDays(await listClosedDays());
    } catch {
      showToast('Kapalı günler yüklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [activeSemester, showToast]);

  useEffect(() => { fetchClosedDays(); }, [fetchClosedDays]);

  if (!activeSemester) {
    return (
      <div className="pt-8 text-center text-gray-500">
        <AlertTriangle className="h-8 w-8 mx-auto mb-3 text-amber-500" />
        <p>Kapalı gün (özel tatil) eklemek için lütfen aktif bir dönem açılışı yapın.</p>
      </div>
    );
  }

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
    <div className="pt-4 space-y-4">
      <div className="flex items-end justify-between mb-4">
        <div>
          <h3 className="text-lg font-medium">Yemekhane Servisi ({activeSemester.name})</h3>
          <p className="text-sm text-gray-500">Öğrencilerin rezervasyon yapamayacağı kapalı günleri tanımlayın.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={fetchClosedDays} disabled={loading}>
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button onClick={() => setCreateDialogOpen(true)} className="bg-amber-600 hover:bg-amber-700 text-white">
            <Plus className="h-4 w-4 mr-2" />
            Kapalı Gün Ekle
          </Button>
        </div>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Tarih</TableHead>
              <TableHead>Sebep</TableHead>
              <TableHead className="text-right">İşlemler</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={3} className="text-center py-6">
                  <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                </TableCell>
              </TableRow>
            ) : closedDays.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} className="text-center py-6 text-gray-500">
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
            <DialogTitle>Kapalı Gün Ekle — Özel Seçili Dönem</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Tarih *</Label>
              <Input type="date" value={formDate} onChange={(e) => setFormDate(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>Sebep *</Label>
              <Input
                placeholder="Örn: Resmi Tatil, Cumhuriyet Bayramı"
                value={formReason}
                onChange={(e) => setFormReason(e.target.value)}
                className="mt-1"
              />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>İptal</Button>
              <Button onClick={handleCreate} disabled={formLoading} className="bg-amber-600 hover:bg-amber-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Kaydet
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
              Bu kaydı silmek istediğinize emin misiniz?
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
    </div>
  );
}
