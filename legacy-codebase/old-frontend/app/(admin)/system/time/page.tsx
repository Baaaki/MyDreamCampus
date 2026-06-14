'use client';

import { useCallback, useEffect, useState } from 'react';
import { format } from 'date-fns';
import { tr } from 'date-fns/locale';
import { Clock, Play, RotateCcw, RefreshCw, Loader2 } from 'lucide-react';

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
import Toast from '@/components/enrollment/Toast';

import type { ServiceTimeStatus } from '@/lib/types';
import { getAllTimeStatuses, simulateTimeAll, resetTimeAll } from '@/lib/services/system-service';

export default function TimeMachinePage() {
  const [toast, setToast] = useState<{
    message: string;
    type: 'error' | 'warning' | 'success' | 'info';
    isVisible: boolean;
  }>({ message: '', type: 'info', isVisible: false });

  const showToast = useCallback((message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  }, []);

  const [statuses, setStatuses] = useState<ServiceTimeStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [simulateTime, setSimulateTime] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  const fetchStatuses = useCallback(async () => {
    setLoading(true);
    try {
      const results = await getAllTimeStatuses();
      setStatuses(results);
    } catch {
      showToast('Servis durumları alınamadı', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => {
    fetchStatuses();
  }, [fetchStatuses]);

  const handleSimulate = async () => {
    if (!simulateTime) {
      showToast('Lütfen bir tarih-saat seçin', 'warning');
      return;
    }
    setActionLoading(true);
    try {
      const isoTime = new Date(simulateTime).toISOString();
      const result = await simulateTimeAll(isoTime);
      if (result.failed.length === 0) {
        showToast('Tüm servisler simüle moduna geçirildi', 'success');
      } else if (result.success.length > 0) {
        showToast(`Başarılı: ${result.success.join(', ')} | Hata: ${result.failed.join(', ')}`, 'warning');
      } else {
        showToast('Hiçbir servis simüle edilemedi', 'error');
      }
      await fetchStatuses();
    } catch {
      showToast('Simülasyon başlatılamadı', 'error');
    } finally {
      setActionLoading(false);
    }
  };

  const handleReset = async () => {
    setActionLoading(true);
    try {
      const result = await resetTimeAll();
      if (result.failed.length === 0) {
        showToast('Tüm servisler gerçek zamana döndürüldü', 'success');
      } else {
        showToast(`Başarılı: ${result.success.join(', ')} | Hata: ${result.failed.join(', ')}`, 'warning');
      }
      setSimulateTime('');
      await fetchStatuses();
    } catch {
      showToast('Sıfırlama başarısız', 'error');
    } finally {
      setActionLoading(false);
    }
  };

  const isAnySimulated = statuses.some((s) => s.status?.mode === 'simulated');

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Zaman Makinesi</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Servis saatlerini simüle et — demo ve test amaçlıdır
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-indigo-600" />
              <CardTitle>Servis Saatleri</CardTitle>
            </div>
            <div className="flex items-center gap-2">
              {isAnySimulated ? (
                <Badge variant="outline" className="border-amber-500 text-amber-600 dark:text-amber-400">
                  Simüle Edilmiş
                </Badge>
              ) : (
                <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400">
                  Gerçek Zaman
                </Badge>
              )}
              <Button variant="ghost" size="sm" onClick={fetchStatuses} disabled={loading}>
                <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Servis</TableHead>
                  <TableHead>Mod</TableHead>
                  <TableHead>Anlık Saat</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center py-4">
                      <Loader2 className="h-5 w-5 animate-spin mx-auto text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : (
                  statuses.map((s) => (
                    <TableRow key={s.service}>
                      <TableCell className="font-medium">{s.label}</TableCell>
                      <TableCell>
                        {s.error ? (
                          <Badge variant="destructive">Hata</Badge>
                        ) : s.status?.mode === 'simulated' ? (
                          <Badge variant="outline" className="border-amber-500 text-amber-600 dark:text-amber-400">
                            Simüle
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="border-green-500 text-green-600 dark:text-green-400">
                            Gerçek
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-gray-600 dark:text-gray-400">
                        {s.error
                          ? s.error
                          : s.status
                            ? format(new Date(s.status.current_time), 'dd MMM yyyy HH:mm:ss', { locale: tr })
                            : '—'}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 min-w-[250px]">
              <Label htmlFor="simulate-time">Hedef Tarih-Saat</Label>
              <Input
                id="simulate-time"
                type="datetime-local"
                value={simulateTime}
                onChange={(e) => setSimulateTime(e.target.value)}
                className="mt-1"
              />
            </div>
            <Button
              onClick={handleSimulate}
              disabled={actionLoading || !simulateTime}
              className="bg-indigo-600 hover:bg-indigo-700 text-white"
            >
              {actionLoading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Play className="h-4 w-4 mr-2" />}
              Simüle Et
            </Button>
            <Button variant="outline" onClick={handleReset} disabled={actionLoading}>
              {actionLoading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <RotateCcw className="h-4 w-4 mr-2" />}
              Sıfırla
            </Button>
          </div>
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
