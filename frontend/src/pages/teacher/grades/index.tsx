
import { useState, useEffect } from 'react';
import { Calendar, Users, BookOpen, Loader2, AlertCircle } from 'lucide-react';
import { semesterApi } from '@/lib/api-client';
import type { TeacherCoursesResponse, TeacherCourse } from '@/lib/types';
import { AssessmentSelectDialog } from '@/components/grades/assessment-select-dialog';

export default function GradesPage() {
  const [courses, setCourses] = useState<TeacherCourse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedCourse, setSelectedCourse] = useState<TeacherCourse | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  useEffect(() => {
    const fetchCourses = async () => {
      try {
        const response = await semesterApi.get('teacher/courses').json<TeacherCoursesResponse>();
        setCourses(response.courses || []);
      } catch (err: any) {
        console.error('Failed to fetch courses:', err);
        setError('Dersler yüklenirken bir hata oluştu.');
      } finally {
        setLoading(false);
      }
    };

    fetchCourses();
  }, []);

  const handleGradeEntryClick = (course: TeacherCourse) => {
    setSelectedCourse(course);
    setIsDialogOpen(true);
  };

  const handleDialogClose = () => {
    setIsDialogOpen(false);
    setSelectedCourse(null);
  };

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
      </div>
    );
  }

  const currentSemester = courses.length > 0 ? courses[0].semester : '';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Not Girme
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {currentSemester} döneminde verdiğiniz dersler için not girin
          </p>
        </div>
        {currentSemester && (
          <div className="flex items-center gap-2 rounded-lg bg-blue-50 px-4 py-2 dark:bg-blue-900/30">
            <Calendar className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            <span className="text-sm font-medium text-blue-700 dark:text-blue-300">
              {currentSemester}
            </span>
          </div>
        )}
      </div>

      {/* Course List */}
      <div className="grid gap-4">
        {courses.map((course) => (
          <div
            key={course.id}
            className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"
          >
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <span className="rounded-md bg-blue-100 px-2.5 py-1 text-sm font-semibold text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
                    {course.course_code}
                  </span>
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                    {course.course_name}
                  </h3>
                </div>
                <div className="mt-2 flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400">
                  <div className="flex items-center gap-1.5">
                    <Users className="h-4 w-4" />
                    <span>{course.max_capacity} Kontenjan</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <BookOpen className="h-4 w-4" />
                    <span>{course.credits} Kredi</span>
                  </div>
                </div>
              </div>
              <button
                className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600"
                onClick={() => handleGradeEntryClick(course)}
              >
                Not Gir
              </button>
            </div>
          </div>
        ))}
      </div>

      {courses.length === 0 && (
        <div className="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-12 text-center dark:border-gray-700 dark:bg-gray-900">
          <p className="text-gray-500 dark:text-gray-400">
            Bu dönemde verdiğiniz ders bulunmamaktadır.
          </p>
        </div>
      )}

      {/* Assessment Selection Dialog */}
      <AssessmentSelectDialog
        courseId={selectedCourse?.id ?? null}
        courseName={selectedCourse?.course_name ?? ''}
        courseCode={selectedCourse?.course_code ?? ''}
        isOpen={isDialogOpen}
        onClose={handleDialogClose}
      />
    </div>
  );
}
