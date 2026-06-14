
import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams } from 'react-router';
import { Link } from 'react-router';
import {
  ArrowLeft,
  Save,
  Search,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Users,
  AlertTriangle,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { gradesService } from '@/lib/services/grades-service';
import type {
  CourseStatusResponse,
  CourseStudentsResponse,
  StudentGrades,
  AssessmentStatus,
} from '@/lib/types';

interface LocalScore {
  score: string;
  isAbsent: boolean;
}

export default function GradeEntryPage() {
  const params = useParams<{ courseId: string; slug: string }>();
  const courseId = params.courseId;
  const slug = params.slug;

  const [courseStatus, setCourseStatus] = useState<CourseStatusResponse | null>(null);
  const [students, setStudents] = useState<StudentGrades[]>([]);
  const [localScores, setLocalScores] = useState<Record<string, LocalScore>>({});
  const [dirtyIds, setDirtyIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [saveMessage, setSaveMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const inputRefs = useRef<Record<string, HTMLInputElement | null>>({});

  // Fetch data
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statusRes, studentsRes] = await Promise.all([
          gradesService.getCourseStatus(courseId),
          gradesService.getCourseStudents(courseId),
        ]);

        setCourseStatus(statusRes);
        setStudents(studentsRes.students || []);

        // Initialize local scores from existing data
        const initial: Record<string, LocalScore> = {};
        for (const student of studentsRes.students || []) {
          const existing = student.scores[slug];
          initial[student.registration_id] = {
            score: existing?.score != null ? String(existing.score) : '',
            isAbsent: existing?.is_absent ?? false,
          };
        }
        setLocalScores(initial);
      } catch (err: any) {
        console.error('Failed to fetch grade data:', err);
        setError('Veriler yüklenirken bir hata oluştu.');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [courseId, slug]);

  // Warn on unsaved changes
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (dirtyIds.size > 0) {
        e.preventDefault();
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [dirtyIds.size]);

  // Clear save message after delay
  useEffect(() => {
    if (saveMessage) {
      const timer = setTimeout(() => setSaveMessage(null), 4000);
      return () => clearTimeout(timer);
    }
  }, [saveMessage]);

  const currentAssessment: AssessmentStatus | undefined = courseStatus?.assessments.find(
    (a) => a.slug === slug
  );

  const handleScoreChange = useCallback(
    (registrationId: string, value: string) => {
      setLocalScores((prev) => ({
        ...prev,
        [registrationId]: { ...prev[registrationId], score: value },
      }));
      setDirtyIds((prev) => new Set(prev).add(registrationId));
    },
    []
  );

  const handleAbsentChange = useCallback(
    (registrationId: string, checked: boolean) => {
      setLocalScores((prev) => ({
        ...prev,
        [registrationId]: {
          score: checked ? '' : prev[registrationId]?.score ?? '',
          isAbsent: checked,
        },
      }));
      setDirtyIds((prev) => new Set(prev).add(registrationId));
    },
    []
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>, currentIndex: number) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        const nextStudent = filteredStudents[currentIndex + 1];
        if (nextStudent) {
          inputRefs.current[nextStudent.registration_id]?.focus();
        }
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [searchQuery, students]
  );

  const isScoreValid = (value: string): boolean => {
    if (value === '') return true;
    const num = parseFloat(value);
    return !isNaN(num) && num >= 0 && num <= 100;
  };

  const handleBulkSave = async () => {
    // Validate all dirty scores
    const invalidIds: string[] = [];
    for (const regId of dirtyIds) {
      const local = localScores[regId];
      if (!local.isAbsent && local.score !== '' && !isScoreValid(local.score)) {
        invalidIds.push(regId);
      }
    }

    if (invalidIds.length > 0) {
      setSaveMessage({
        type: 'error',
        text: `${invalidIds.length} öğrencinin notu geçersiz. Notlar 0-100 arasında olmalıdır.`,
      });
      return;
    }

    setSaving(true);
    setSaveMessage(null);

    try {
      const scores = Array.from(dirtyIds).map((regId) => {
        const local = localScores[regId];
        return {
          registration_id: regId,
          score: local.isAbsent || local.score === '' ? null : parseInt(local.score, 10),
          is_absent: local.isAbsent,
        };
      });

      const result = await gradesService.bulkSubmitScores(courseId, { slug, scores });

      setDirtyIds(new Set());

      let message = `${result.success_count} öğrencinin notu başarıyla kaydedildi.`;
      if (result.auto_finalized) {
        message += ' Tüm notlar girildiği için ders notları otomatik olarak kesinleştirildi.';
      }
      setSaveMessage({ type: 'success', text: message });

      // Refresh data
      const [statusRes, studentsRes] = await Promise.all([
        gradesService.getCourseStatus(courseId),
        gradesService.getCourseStudents(courseId),
      ]);
      setCourseStatus(statusRes);
      setStudents(studentsRes.students || []);

      // Re-initialize local scores
      const updated: Record<string, LocalScore> = {};
      for (const student of studentsRes.students || []) {
        const existing = student.scores[slug];
        updated[student.registration_id] = {
          score: existing?.score != null ? String(existing.score) : '',
          isAbsent: existing?.is_absent ?? false,
        };
      }
      setLocalScores(updated);
    } catch (err: any) {
      console.error('Failed to save scores:', err);
      setSaveMessage({ type: 'error', text: 'Notlar kaydedilirken bir hata oluştu.' });
    } finally {
      setSaving(false);
    }
  };

  // Filter students
  const filteredStudents = students.filter((student) => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      student.student_number.toLowerCase().includes(query) ||
      student.first_name.toLowerCase().includes(query) ||
      student.last_name.toLowerCase().includes(query) ||
      `${student.first_name} ${student.last_name}`.toLowerCase().includes(query)
    );
  });

  // Count graded
  const gradedCount = students.filter((s) => {
    const local = localScores[s.registration_id];
    return local && (local.score !== '' || local.isAbsent);
  }).length;

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-96 flex-col items-center justify-center gap-4">
        <AlertCircle className="h-12 w-12 text-red-500" />
        <p className="text-red-600 dark:text-red-400">{error}</p>
        <Link
          to="/teacher/grades"
          className="text-sm text-blue-600 hover:underline dark:text-blue-400"
        >
          Not Girme sayfasına dön
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <Link
            to="/teacher/grades"
            className="inline-flex items-center gap-1 text-sm text-gray-500 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
          >
            <ArrowLeft className="h-4 w-4" />
            Not Girme Sayfasına Dön
          </Link>
          <div className="flex items-center gap-3">
            <span className="rounded-md bg-blue-100 px-2.5 py-1 text-sm font-semibold text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
              {courseStatus?.course_code}
            </span>
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">
              {courseStatus?.course_name}
            </h1>
          </div>
          {currentAssessment && (
            <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <span className="font-medium text-gray-700 dark:text-gray-300">
                {currentAssessment.name}
              </span>
              <Badge variant="secondary" className="text-xs">
                %{currentAssessment.weight}
              </Badge>
              <span>
                {gradedCount}/{students.length} öğrenci notlandırıldı
              </span>
            </div>
          )}
        </div>

        <Button
          onClick={handleBulkSave}
          disabled={dirtyIds.size === 0 || saving}
          className="gap-2 shrink-0"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          {dirtyIds.size > 0 ? `Kaydet (${dirtyIds.size} değişiklik)` : 'Kaydet'}
        </Button>
      </div>

      {/* Save message */}
      {saveMessage && (
        <div
          className={`flex items-center gap-2 rounded-lg border p-3 text-sm ${
            saveMessage.type === 'success'
              ? 'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400'
              : 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400'
          }`}
        >
          {saveMessage.type === 'success' ? (
            <CheckCircle2 className="h-4 w-4 shrink-0" />
          ) : (
            <AlertCircle className="h-4 w-4 shrink-0" />
          )}
          {saveMessage.text}
        </div>
      )}

      {/* Finalized warning */}
      {courseStatus?.is_finalized && (
        <div className="flex items-center gap-2 rounded-lg border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-700 dark:border-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          Bu dersin notları kesinleştirilmiştir. Değişiklik yapılamaz.
        </div>
      )}

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <Input
          type="text"
          placeholder="Öğrenci ara (ad, soyad veya numara)..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
        />
      </div>

      {/* Student Table */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-800">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12">#</TableHead>
              <TableHead className="w-36">Öğrenci No</TableHead>
              <TableHead>Ad Soyad</TableHead>
              <TableHead className="w-32 text-center">Not</TableHead>
              <TableHead className="w-24 text-center">Devamsız</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredStudents.map((student, index) => {
              const local = localScores[student.registration_id] || {
                score: '',
                isAbsent: false,
              };
              const isDirty = dirtyIds.has(student.registration_id);
              const scoreInvalid = local.score !== '' && !isScoreValid(local.score);
              const existingScore = student.scores[slug];
              const isLocked = existingScore?.is_locked ?? false;

              return (
                <TableRow
                  key={student.registration_id}
                  className={isDirty ? 'bg-blue-50/50 dark:bg-blue-900/10' : ''}
                >
                  <TableCell className="text-gray-500">{index + 1}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {student.student_number}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900 dark:text-white">
                        {student.first_name} {student.last_name}
                      </span>
                      {student.is_attendance_failed && (
                        <Badge className="bg-red-100 text-red-700 text-xs dark:bg-red-900/30 dark:text-red-400">
                          Devamsız
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-center">
                    {isLocked ? (
                      <span className="inline-flex h-9 w-24 items-center justify-center rounded-md bg-gray-100 font-medium text-gray-700 dark:bg-gray-800 dark:text-gray-300">
                        {existingScore?.is_absent ? 'G' : (existingScore?.score ?? '—')}
                      </span>
                    ) : (
                      <Input
                        ref={(el) => {
                          inputRefs.current[student.registration_id] = el;
                        }}
                        type="number"
                        min={0}
                        max={100}
                        step="any"
                        placeholder="—"
                        value={local.score}
                        onChange={(e) =>
                          handleScoreChange(student.registration_id, e.target.value)
                        }
                        onKeyDown={(e) => handleKeyDown(e, index)}
                        disabled={local.isAbsent || courseStatus?.is_finalized}
                        aria-invalid={scoreInvalid}
                        className={`h-9 w-24 text-center ${
                          scoreInvalid
                            ? 'border-red-500 focus:ring-red-500'
                            : ''
                        }`}
                      />
                    )}
                  </TableCell>
                  <TableCell className="text-center">
                    {isLocked ? (
                      existingScore?.is_absent && (
                        <Badge variant="secondary" className="text-xs">
                          Devamsız
                        </Badge>
                      )
                    ) : (
                      <Checkbox
                        checked={local.isAbsent}
                        onCheckedChange={(checked) =>
                          handleAbsentChange(
                            student.registration_id,
                            checked === true
                          )
                        }
                        disabled={courseStatus?.is_finalized}
                      />
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
            {filteredStudents.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center text-gray-500">
                  {searchQuery
                    ? 'Arama kriterlerine uygun öğrenci bulunamadı.'
                    : 'Bu derse kayıtlı öğrenci bulunmamaktadır.'}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Footer Summary */}
      <div className="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3 text-sm dark:bg-gray-800/50">
        <div className="flex items-center gap-1.5 text-gray-600 dark:text-gray-400">
          <Users className="h-4 w-4" />
          <span>
            Toplam: <span className="font-medium">{students.length}</span> öğrenci
          </span>
        </div>
        <div className="flex items-center gap-4 text-gray-600 dark:text-gray-400">
          <span>
            Notlanan: <span className="font-medium text-green-600 dark:text-green-400">{gradedCount}</span>
          </span>
          <span>
            Bekleyen:{' '}
            <span className="font-medium text-yellow-600 dark:text-yellow-400">
              {students.length - gradedCount}
            </span>
          </span>
        </div>
      </div>
    </div>
  );
}
