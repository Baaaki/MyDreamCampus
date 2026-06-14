'use client';

import { useCallback, useEffect, useState } from 'react';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import { ShieldCheck, Plus, Play, CheckCircle2, RefreshCw, Loader2, AlertTriangle } from 'lucide-react';

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
import Toast from '@/components/enrollment/Toast';

import type { Semester, SemesterStatus } from '@/lib/types';
import {
  listSemesters,
  createSemester,
  activateSemester,
  completeSemester,
} from '@/lib/services/system-service';

const STATUS_BADGE: Record<SemesterStatus, { label: string; className: string }> = {
  planned: { label: 'Planlandı', className: 'border-blue-500 text-blue-600 dark:text-blue-400' },
  active: { label: 'Aktif', className: 'border-green-500 text-green-600 dark:text-green-400' },
  completed: { label: 'Tamamlandı', className: 'border-gray-400 text-gray-500 dark:text-gray-400' },
};

export default function SemestersPage() {
  const [semesters, setSemesters] = useState<Semester[]>([]);
  const [loading, setLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{ type: 'activate' | 'complete'; semester: Semester } | null>(null);

  const [formName, setFormName] = useState('');
  const [formDeadline, setFormDeadline] = useState('');
  const [formLoading, setFormLoading] = useState(false);

  const [toast, setToast] = useState<{
    message: string;
    type: 'error' | 'warning' | 'success' | 'info';
    isVisible: boolean;
  }>({ message: '', type: 'info', isVisible: false });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  const fetchSemesters = useCallback(async () => {
    setLoading(true);
    try {
      setSemesters(await listSemesters());
    } catch {
      showToast('Donemler yuklenemedi', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => { fetchSemesters(); }, [fetchSemesters]);

  const activeSemester = semesters.find((s) => s.status === 'active');

  const handleCreate = async () => {
    if (!formName || !formDeadline) {
      showToast('Lutfen tum alanlari doldurun', 'warning');
      return;
    }
    // Validate name format: YYYY-YYYY-(Fall|Spring)
    if (!/^\d{4}-\d{4}-(Fall|Spring)$/.test(formName)) {
      showToast('Donem adi formati: YYYY-YYYY-Fall veya YYYY-YYYY-Spring', 'warning');
      return;
    }
    setFormLoading(true);
    try {
      await createSemester({
        name: formName,
        hard_deadline: new Date(formDeadline).toISOString(),
      });
      showToast('Donem basariyla olusturuldu', 'success');
      setCreateDialogOpen(false);
      setFormName('');
      setFormDeadline('');
      await fetchSemesters();
    } catch {
      showToast('Donem olusturulamadi (ayni isimde donem olabilir)', 'error');
    } finally {
      setFormLoading(false);
    }
  };

  const handleConfirmAction = async () => {
    if (!confirmAction) return;
    setFormLoading(true);
    try {
      if (confirmAction.type === 'activate') {
        await activateSemester(confirmAction.semester.id);
        showToast('Donem aktif edildi', 'success');
      } else {
        await completeSemester(confirmAction.semester.id);
        showToast('Donem tamamlandi olarak isaretlendi', 'success');
      }
      setConfirmAction(null);
      await fetchSemesters();
    } catch {
      showToast(
        confirmAction.type === 'activate'
          ? 'Donem aktif edilemedi (zaten aktif bir donem olabilir)'
          : 'Donem tamamlanamadi',
        'error'
      );
    } finally {
      setFormLoading(false);
    }
  };

  // Generate semester name suggestions
  const year = new Date().getFullYear();
  const suggestions = [
    `${year - 1}-${year}-Spring`,
    `${year}-${year + 1}-Fall`,
    `${year}-${year + 1}-Spring`,
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Donem Durumu</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Akademik donem yasam dongusunu yonet (planned &rarr; active &rarr; completed)
        </p>
      </div>

      {/* Active semester banner */}
      {activeSemester ? (
        <div className="rounded-lg border border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20 p-4">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-green-600 dark:text-green-400" />
            <span className="font-medium text-green-800 dark:text-green-300">
              Aktif Donem: {activeSemester.name}
            </span>
          </div>
          <p className="mt-1 text-sm text-green-700 dark:text-green-400">
            Hard deadline: {format(new Date(activeSemester.hard_deadline), 'dd MMMM yyyy HH:mm', { locale: tr })}
          </p>
        </div>
      ) : (
        <div className="rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20 p-4">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400" />
            <span className="font-medium text-amber-800 dark:text-amber-300">
              Aktif donem bulunamadi
            </span>
          </div>
          <p className="mt-1 text-sm text-amber-700 dark:text-amber-400">
            Donem islemleri (ders acma, kayit, not girisi, period CRUD) aktif donem olmadan calismaz.
          </p>
        </div>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-indigo-600" />
              <CardTitle>Donemler</CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={fetchSemesters} disabled={loading}>
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              </Button>
              <Button onClick={() => setCreateDialogOpen(true)} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                <Plus className="h-4 w-4 mr-2" />
                Yeni Donem
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Donem Adi</TableHead>
                  <TableHead>Durum</TableHead>
                  <TableHead>Hard Deadline</TableHead>
                  <TableHead>Aktif Edilme</TableHead>
                  <TableHead>Tamamlanma</TableHead>
                  <TableHead className="text-right">Islemler</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-6">
                      <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : semesters.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-6 text-gray-500">
                      Henuz donem tanimlanmadi
                    </TableCell>
                  </TableRow>
                ) : (
                  semesters.map((sem) => {
                    const badge = STATUS_BADGE[sem.status];
                    return (
                      <TableRow key={sem.id}>
                        <TableCell className="font-medium">{sem.name}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className={badge.className}>
                            {badge.label}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm">
                          {format(new Date(sem.hard_deadline), 'dd MMM yyyy HH:mm', { locale: tr })}
                        </TableCell>
                        <TableCell className="text-sm text-gray-500">
                          {sem.activated_at
                            ? format(new Date(sem.activated_at), 'dd MMM yyyy HH:mm', { locale: tr })
                            : '-'}
                        </TableCell>
                        <TableCell className="text-sm text-gray-500">
                          {sem.completed_at
                            ? format(new Date(sem.completed_at), 'dd MMM yyyy HH:mm', { locale: tr })
                            : '-'}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            {sem.status === 'planned' && (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setConfirmAction({ type: 'activate', semester: sem })}
                                className="text-green-600 border-green-300 hover:bg-green-50 dark:hover:bg-green-900/20"
                              >
                                <Play className="h-4 w-4 mr-1" />
                                Aktif Et
                              </Button>
                            )}
                            {sem.status === 'active' && (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setConfirmAction({ type: 'complete', semester: sem })}
                                className="text-gray-600 border-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800"
                              >
                                <CheckCircle2 className="h-4 w-4 mr-1" />
                                Tamamla
                              </Button>
                            )}
                            {sem.status === 'completed' && (
                              <span className="text-xs text-gray-400 px-2">Degistirilemez</span>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>

          {/* Info box */}
          <div className="mt-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 p-3">
            <p className="text-sm text-blue-700 dark:text-blue-300">
              <strong>planned &rarr; active:</strong> Donemi baslatir, sistem islemleri acilir.{' '}
              <strong>active &rarr; completed:</strong> Donem kapanir, islemler kilitlenir.{' '}
              <strong>completed</strong> durumu geri alinamaz (DB trigger ile korunur).
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={(open) => { setCreateDialogOpen(open); if (!open) { setFormName(''); setFormDeadline(''); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yeni Donem Olustur</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>Donem Adi *</Label>
              <Input
                placeholder="2025-2026-Fall"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                className="mt-1"
              />
              <div className="mt-1 flex gap-1 flex-wrap">
                {suggestions.map((s) => (
                  <button
                    key={s}
                    type="button"
                    onClick={() => setFormName(s)}
                    className="text-xs px-2 py-0.5 rounded bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-400"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <Label>Hard Deadline *</Label>
              <Input
                type="datetime-local"
                value={formDeadline}
                onChange={(e) => setFormDeadline(e.target.value)}
                className="mt-1"
              />
              <p className="mt-1 text-xs text-gray-500">
                Bu tarihten sonra donem otomatik olarak &quot;completed&quot; durumuna gecer.
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>Iptal</Button>
              <Button onClick={handleCreate} disabled={formLoading} className="bg-indigo-600 hover:bg-indigo-700 text-white">
                {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                Olustur
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirm Action Dialog */}
      <AlertDialog open={!!confirmAction} onOpenChange={(open) => { if (!open) setConfirmAction(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {confirmAction?.type === 'activate' ? 'Donemi Aktif Et' : 'Donemi Tamamla'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {confirmAction?.type === 'activate' ? (
                <>
                  <strong>{confirmAction.semester.name}</strong> donemini aktif etmek istediginize emin misiniz?
                  Baska bir aktif donem varsa bu islem basarisiz olur.
                </>
              ) : (
                <>
                  <strong>{confirmAction?.semester.name}</strong> donemini tamamlamak istediginize emin misiniz?
                  <span className="block mt-2 text-red-600 dark:text-red-400 font-medium">
                    Bu islem geri alinamaz! Tamamlanan donem bir daha aktif edilemez.
                  </span>
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Iptal</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmAction}
              className={
                confirmAction?.type === 'activate'
                  ? 'bg-green-600 hover:bg-green-700 text-white'
                  : 'bg-red-600 hover:bg-red-700 text-white'
              }
            >
              {formLoading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              {confirmAction?.type === 'activate' ? 'Aktif Et' : 'Tamamla'}
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
