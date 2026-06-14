'use client';

import { useState } from 'react';
import { EnrollmentProgramResponse } from '@/lib/types';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { AlertCircle, Check, X, BookOpen, User } from 'lucide-react';

interface EnrollmentReviewDialogProps {
  program: EnrollmentProgramResponse | null;
  isOpen: boolean;
  onClose: () => void;
  onApprove: (programId: string) => Promise<void>;
  onReject: (programId: string, reason: string) => Promise<void>;
}

export function EnrollmentReviewDialog({
  program,
  isOpen,
  onClose,
  onApprove,
  onReject,
}: EnrollmentReviewDialogProps) {
  const [rejectReason, setRejectReason] = useState('');
  const [isRejecting, setIsRejecting] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!program) return null;

  const handleApprove = async () => {
    try {
      setIsSubmitting(true);
      await onApprove(program.id);
      onClose();
    } catch (error) {
      console.error('Onaylama hatası:', error);
      alert('Onaylama işlemi başarısız oldu.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!rejectReason.trim()) {
      alert('Lütfen ret nedeni giriniz.');
      return;
    }

    try {
      setIsSubmitting(true);
      await onReject(program.id, rejectReason);
      onClose();
    } catch (error) {
      console.error('Reddetme hatası:', error);
      alert('Reddetme işlemi başarısız oldu.');
    } finally {
      setIsSubmitting(false);
      setIsRejecting(false);
      setRejectReason('');
    }
  };

  const totalCredits = program.courses.reduce((sum, course) => sum + course.credits, 0);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold flex items-center gap-2">
            <BookOpen className="h-6 w-6 text-blue-600" />
            Ders Kaydı İnceleme
          </DialogTitle>
          <DialogDescription>
            Öğrencinin ders seçimlerini inceleyin ve onaylayın veya reddedin.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-6 py-4">
          {/* Student Info */}
          <div className="flex items-start gap-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-100 dark:border-gray-700">
             <div className="p-2 bg-white dark:bg-gray-700 rounded-full shadow-sm">
                <User className="h-6 w-6 text-blue-600 dark:text-blue-400" />
             </div>
             <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {program.student_name || 'Öğrenci Adı Yok'}
                </h3>
                <div className="flex flex-wrap gap-x-6 gap-y-1 mt-1 text-sm text-gray-500 dark:text-gray-400">
                  <span className="flex items-center gap-1">
                     <span className="font-medium text-gray-700 dark:text-gray-300">Numara:</span> {program.student_number}
                  </span>
                   <span className="flex items-center gap-1">
                     <span className="font-medium text-gray-700 dark:text-gray-300">Dönem:</span> {program.semester}
                  </span>
                   <span className="flex items-center gap-1">
                     <span className="font-medium text-gray-700 dark:text-gray-300">Sınıf:</span> {program.class_level}. Sınıf
                  </span>
                  <span className="flex items-center gap-1">
                     <span className="font-medium text-gray-700 dark:text-gray-300">Bölüm:</span> {program.department}
                  </span>
                </div>
             </div>
          </div>

          {/* Courses Table */}
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Ders Kodu</TableHead>
                  <TableHead>Ders Adı</TableHead>
                  <TableHead>Kredi</TableHead>
                  <TableHead>Öğretim Elemanı</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {program.courses.map((course) => (
                  <TableRow key={course.id}>
                    <TableCell className="font-medium">{course.course_code}</TableCell>
                    <TableCell>{course.course_name}</TableCell>
                    <TableCell>{course.credits}</TableCell>
                    <TableCell>{course.instructor || '-'}</TableCell>
                  </TableRow>
                ))}
                <TableRow className="bg-gray-50 dark:bg-gray-800/50 font-medium">
                  <TableCell colSpan={2} className="text-right">Toplam:</TableCell>
                  <TableCell>{totalCredits}</TableCell>
                  <TableCell></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          {/* Rejection UI */}
          {isRejecting && (
            <div className="space-y-2 animate-in fade-in slide-in-from-top-2">
              <div className="flex items-center gap-2 text-red-600 font-medium">
                <AlertCircle className="h-4 w-4" />
                Ret Nedeni
              </div>
              <Textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Öğrenciye gösterilecek ret nedenini yazınız..."
                className="min-h-[100px]"
              />
            </div>
          )}
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          {!isRejecting ? (
            <>
              <Button
                variant="outline"
                onClick={() => onClose()}
                disabled={isSubmitting}
              >
                İptal
              </Button>
              <Button
                variant="destructive"
                onClick={() => setIsRejecting(true)}
                disabled={isSubmitting}
                className="gap-2"
              >
                <X className="h-4 w-4" />
                Reddet
              </Button>
              <Button
                onClick={handleApprove}
                disabled={isSubmitting}
                className="bg-green-600 hover:bg-green-700 gap-2"
              >
                <Check className="h-4 w-4" />
                Onayla
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="ghost"
                onClick={() => {
                  setIsRejecting(false);
                  setRejectReason('');
                }}
                disabled={isSubmitting}
              >
                Geri Dön
              </Button>
              <Button
                variant="destructive"
                onClick={handleReject}
                disabled={isSubmitting || !rejectReason.trim()}
                className="gap-2"
              >
                <X className="h-4 w-4" />
                Reddetmeyi Onayla
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
