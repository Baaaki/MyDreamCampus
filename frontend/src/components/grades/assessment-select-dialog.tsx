
import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { gradesService } from '@/lib/services/grades-service';
import type { CourseStatusResponse } from '@/lib/types';
import {
  Loader2,
  ChevronRight,
  CheckCircle2,
  Clock,
  FileText,
  Info,
} from 'lucide-react';

interface AssessmentSelectDialogProps {
  courseId: string | null;
  courseName: string;
  courseCode: string;
  isOpen: boolean;
  onClose: () => void;
}

export function AssessmentSelectDialog({
  courseId,
  courseName,
  courseCode,
  isOpen,
  onClose,
}: AssessmentSelectDialogProps) {
  const navigate = useNavigate();
  const [courseStatus, setCourseStatus] = useState<CourseStatusResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen && courseId) {
      setLoading(true);
      setError('');
      setCourseStatus(null);

      gradesService
        .getCourseStatus(courseId)
        .then((status) => {
          setCourseStatus(status);
        })
        .catch(() => {
          setError('Ders durumu yüklenirken bir hata oluştu.');
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [isOpen, courseId]);

  const handleAssessmentClick = (slug: string) => {
    if (!courseId) return;
    onClose();
    navigate(`/teacher/grades/${courseId}/${slug}`);
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-xl font-bold">
            <FileText className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Not Girilecek Sınavı Seçin
          </DialogTitle>
          <DialogDescription>
            <span className="font-medium text-gray-700 dark:text-gray-300">{courseCode}</span>
            {' - '}
            {courseName}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
            </div>
          )}

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-center text-sm text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
              {error}
            </div>
          )}

          {courseStatus && (
            <>
              {courseStatus.is_finalized && (
                <div className="mb-4 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300">
                  <Info className="h-4 w-4 shrink-0" />
                  Bu dersin notları kesinleştirilmiştir.
                </div>
              )}

              <div className="space-y-2">
                {courseStatus.assessments.map((assessment) => (
                  <button
                    key={assessment.slug}
                    onClick={() => handleAssessmentClick(assessment.slug)}
                    disabled={courseStatus.is_finalized}
                    className="flex w-full items-center gap-3 rounded-lg border border-gray-200 p-4 text-left transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:bg-gray-800/50"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-gray-900 dark:text-white">
                          {assessment.name}
                        </span>
                        <Badge variant="secondary" className="text-xs">
                          %{assessment.weight}
                        </Badge>
                      </div>
                      <div className="mt-1 flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
                        <span>
                          {assessment.graded_count}/{courseStatus.total_students} öğrenci notlandırıldı
                        </span>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      {assessment.is_complete ? (
                        <Badge className="bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                          <CheckCircle2 className="mr-1 h-3 w-3" />
                          Tamamlandı
                        </Badge>
                      ) : (
                        <Badge className="bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">
                          <Clock className="mr-1 h-3 w-3" />
                          Bekliyor
                        </Badge>
                      )}
                      <ChevronRight className="h-5 w-5 text-gray-400" />
                    </div>
                  </button>
                ))}
              </div>

              {courseStatus.assessments.length === 0 && (
                <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  Bu ders için tanımlı sınav bulunmamaktadır.
                </div>
              )}
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
