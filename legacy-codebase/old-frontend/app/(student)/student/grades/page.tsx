'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { gradesApi } from '@/lib/api-client';
import { mockMyGradesResponse } from '@/mock_data/grades';
import type { MyGradesResponse, ActiveCourse, CompletedCourse, ScoreDetail } from '@/lib/types';

const USE_MOCK = process.env.NEXT_PUBLIC_USE_MOCK_API === 'true';
import {
  BarChart3,
  BookOpen,
  Trophy,
  GraduationCap,
  TrendingUp,
  Clock,
  CheckCircle2,
  AlertCircle,
  Loader2,
  ChevronDown,
} from 'lucide-react';

// Sayısal grade point'i harf notuna çevirir (DB'de '4.00', '3.75' vb. saklanır)
// 12 harf notu: AA, AB, BA, BB, BC, CB, CC, CD, DC, DD, FD, FF
function gradePointToLetter(gp: string): string {
  const map: Record<string, string> = {
    '4.00': 'AA',
    '3.75': 'AB',
    '3.50': 'BA',
    '3.25': 'BB',
    '3.00': 'BB',
    '2.75': 'BC',
    '2.50': 'CB',
    '2.25': 'CB',
    '2.00': 'CC',
    '1.75': 'CD',
    '1.50': 'DC',
    '1.25': 'DC',
    '1.00': 'DD',
    '0.50': 'FD',
    '0.00': 'FF',
  };
  return map[gp] || gp;
}

// Harf notuna göre renk döndürür
function getGradeColor(grade: string): string {
  const letter = gradePointToLetter(grade);
  switch (letter) {
    case 'AA':
      return 'bg-emerald-500 text-white';
    case 'AB':
      return 'bg-emerald-400 text-white';
    case 'BA':
      return 'bg-green-500 text-white';
    case 'BB':
      return 'bg-green-400 text-white';
    case 'BC':
      return 'bg-lime-500 text-white';
    case 'CB':
      return 'bg-yellow-500 text-white';
    case 'CC':
      return 'bg-yellow-600 text-white';
    case 'CD':
      return 'bg-orange-400 text-white';
    case 'DC':
      return 'bg-orange-500 text-white';
    case 'DD':
      return 'bg-orange-600 text-white';
    case 'FD':
      return 'bg-red-400 text-white';
    case 'FF':
      return 'bg-red-500 text-white';
    default:
      return 'bg-gray-500 text-white';
  }
}

// Slug'ı okunabilir isme çevirir
function getAssessmentName(slug: string): string {
  const names: Record<string, string> = {
    'midterm': 'Vize',
    'final': 'Final',
    'quiz': 'Quiz',
    'homework': 'Ödev',
    'project': 'Proje',
    'lab': 'Lab',
    'attendance': 'Devam',
  };
  return names[slug] || slug.charAt(0).toUpperCase() + slug.slice(1);
}

// Semester'dan akademik yılı çıkarır: "2022_fall" → "2022", "2023_spring" → "2022" (bahar = önceki güzün yılı)
function getAcademicYear(semester: string): number {
  const parts = semester.split('_');
  const year = parseInt(parts[0]) || 0;
  if (parts[1] === 'spring' || parts[1] === 'summer') return year - 1;
  return year; // fall
}

// Semester'ın dönem sırasını döndürür (güz=1, bahar=2)
function getSemesterTerm(semester: string): number {
  const parts = semester.split('_');
  if (parts[1] === 'fall') return 1;
  return 2; // spring, summer
}

// Akademik yıl ve dönem etiketini döndürür
function getTermLabel(semester: string): string {
  const parts = semester.split('_');
  if (parts[1] === 'fall') return 'Güz Dönemi';
  if (parts[1] === 'spring') return 'Bahar Dönemi';
  return 'Yaz Dönemi';
}

// Akademik yıl gruplama yapısı
interface AcademicYearGroup {
  academicYear: number; // 2022 = "2022-2023"
  classNumber: number;  // 1, 2, 3, 4
  semesters: { key: string; term: number; label: string; courses: CompletedCourse[] }[];
}

function groupByAcademicYear(courses: CompletedCourse[]): AcademicYearGroup[] {
  // Semester'a göre grupla
  const semesterMap: Record<string, CompletedCourse[]> = {};
  for (const course of courses) {
    const key = course.semester || 'unknown';
    if (!semesterMap[key]) semesterMap[key] = [];
    semesterMap[key].push(course);
  }

  // Akademik yıla göre grupla
  const yearMap: Record<number, { key: string; term: number; label: string; courses: CompletedCourse[] }[]> = {};
  for (const [semKey, semCourses] of Object.entries(semesterMap)) {
    const ay = getAcademicYear(semKey);
    if (!yearMap[ay]) yearMap[ay] = [];
    yearMap[ay].push({ key: semKey, term: getSemesterTerm(semKey), label: getTermLabel(semKey), courses: semCourses });
  }

  // Sırala ve sınıf numarası ata
  const sortedYears = Object.keys(yearMap).map(Number).sort((a, b) => a - b);
  return sortedYears.map((ay, idx) => ({
    academicYear: ay,
    classNumber: idx + 1,
    semesters: yearMap[ay].sort((a, b) => a.term - b.term),
  }));
}

// Score badge rengi
function getScoreBadgeColor(score: number | null, isAbsent: boolean): string {
  if (isAbsent) return 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300';
  if (score === null) return 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400';
  if (score >= 85) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300';
  if (score >= 70) return 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300';
  if (score >= 50) return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/50 dark:text-yellow-300';
  return 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300';
}

export default function StudentGradesPage() {
  const [gradesData, setGradesData] = useState<MyGradesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedCourses, setExpandedCourses] = useState<Set<string>>(new Set());

  const toggleCourse = (key: string) => {
    setExpandedCourses((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  useEffect(() => {
    const fetchGrades = async () => {
      try {
        setLoading(true);
        if (USE_MOCK) {
          setGradesData(mockMyGradesResponse);
        } else {
          const response = await gradesApi.get('student/my').json<MyGradesResponse>();
          setGradesData(response);
        }
        setError(null);
      } catch (err: any) {
        console.error('Failed to fetch grades:', err);
        setError(err.message || 'Notlar yüklenirken bir hata oluştu');
      } finally {
        setLoading(false);
      }
    };

    fetchGrades();
  }, []);

  // Loading state
  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="h-12 w-12 animate-spin text-emerald-600" />
          <p className="text-gray-500 dark:text-gray-400">Notlar yükleniyor...</p>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
            <AlertCircle className="h-8 w-8 text-red-600 dark:text-red-400" />
          </div>
          <div>
            <p className="text-lg font-medium text-gray-900 dark:text-white">Notlar yüklenemedi</p>
            <p className="text-gray-500 dark:text-gray-400 mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  // Empty state
  if (!gradesData || (gradesData.active_courses.length === 0 && gradesData.completed_courses.length === 0)) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
            <BarChart3 className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Ders Notları</h1>
            <p className="text-gray-600 dark:text-gray-400">Derslerinizin not durumunu görüntüleyin</p>
          </div>
        </div>

        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <GraduationCap className="h-16 w-16 text-gray-300 dark:text-gray-600 mb-4" />
            <p className="text-gray-500 dark:text-gray-400 text-center">
              Henüz kayıtlı ders veya not kaydı bulunmuyor.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
          <BarChart3 className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Ders Notları</h1>
          <p className="text-gray-600 dark:text-gray-400">Derslerinizin not durumunu görüntüleyin</p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* GPA Card */}
        <Card className="bg-gradient-to-br from-emerald-500 to-emerald-600 text-white border-0">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-emerald-100 text-sm font-medium">Genel Not Ortalaması</p>
                <p className="text-4xl font-bold mt-1">{gradesData.cumulative_gpa.toFixed(2)}</p>
                <p className="text-emerald-100 text-xs mt-1">4.00 üzerinden</p>
              </div>
              <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                <Trophy className="h-7 w-7" />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Total Credits Card */}
        <Card className="bg-gradient-to-br from-blue-500 to-blue-600 text-white border-0">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-blue-100 text-sm font-medium">Toplam Kredi</p>
                <p className="text-4xl font-bold mt-1">{gradesData.total_credits}</p>
                <p className="text-blue-100 text-xs mt-1">Tamamlanan kredi</p>
              </div>
              <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                <GraduationCap className="h-7 w-7" />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Courses Count Card */}
        <Card className="bg-gradient-to-br from-purple-500 to-purple-600 text-white border-0">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-purple-100 text-sm font-medium">Ders Sayısı</p>
                <p className="text-4xl font-bold mt-1">
                  {gradesData.active_courses.length + gradesData.completed_courses.length}
                </p>
                <p className="text-purple-100 text-xs mt-1">
                  {gradesData.active_courses.length} aktif, {gradesData.completed_courses.length} tamamlanmış
                </p>
              </div>
              <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                <BookOpen className="h-7 w-7" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Active Courses */}
      {gradesData.active_courses.length > 0 && (
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <Clock className="h-5 w-5 text-orange-500" />
              Aktif Dersler
              <Badge variant="outline" className="ml-2">
                {gradesData.active_courses.length} ders
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {gradesData.active_courses.map((course: ActiveCourse, index: number) => (
                <div
                  key={`${course.course_code}-${index}`}
                  className="p-4 rounded-xl bg-gray-50 dark:bg-gray-800 border border-gray-100 dark:border-gray-700"
                >
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="px-2.5 py-1 rounded-md bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 text-sm font-semibold">
                          {course.course_code}
                        </span>
                        <h3 className="font-semibold text-gray-900 dark:text-white">
                          {course.course_name}
                        </h3>
                      </div>
                      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                        {course.credits} Kredi • {course.semester}
                      </p>
                    </div>
                  </div>

                  {/* Scores */}
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(course.scores).length > 0 ? (
                      Object.entries(course.scores).map(([slug, detail]: [string, ScoreDetail]) => (
                        <div
                          key={slug}
                          className={`px-3 py-1.5 rounded-lg text-sm font-medium ${getScoreBadgeColor(detail.score, detail.is_absent)}`}
                        >
                          <span className="font-normal opacity-80">{getAssessmentName(slug)}:</span>{' '}
                          {detail.is_absent ? 'Girmedi' : detail.score !== null ? detail.score : '-'}
                        </div>
                      ))
                    ) : (
                      <p className="text-sm text-gray-400 dark:text-gray-500 italic">
                        Henüz not girilmedi
                      </p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Completed Courses - Grouped by Academic Year & Semester */}
      {gradesData.completed_courses.length > 0 && (() => {
        const yearGroups = groupByAcademicYear(gradesData.completed_courses);

        return (
          <div className="space-y-6">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-emerald-500" />
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Tamamlanan Dersler
              </h2>
              <Badge variant="outline">
                {gradesData.completed_courses.length} ders
              </Badge>
            </div>

            {yearGroups.map((yearGroup) => (
              <Card key={yearGroup.academicYear} className="overflow-hidden">
                {/* Year Header */}
                <div className="bg-gradient-to-r from-emerald-600 to-emerald-700 dark:from-emerald-700 dark:to-emerald-800 px-6 py-4">
                  <h3 className="text-lg font-bold text-white">
                    {yearGroup.classNumber}. Sınıf
                    <span className="text-emerald-200 font-normal text-sm ml-2">
                      ({yearGroup.academicYear}-{yearGroup.academicYear + 1})
                    </span>
                  </h3>
                </div>

                {/* 2 Semester Columns */}
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-0 divide-y lg:divide-y-0 lg:divide-x divide-gray-200 dark:divide-gray-700">
                  {yearGroup.semesters.map((sem) => {
                    const semCredits = sem.courses.reduce((sum, c) => sum + c.credits, 0);
                    const isFall = sem.term === 1;

                    return (
                      <div key={sem.key} className="flex flex-col">
                        {/* Semester Header */}
                        <div className={`px-5 py-3 flex items-center justify-between border-b ${
                          isFall
                            ? 'bg-slate-50 dark:bg-slate-800/40 border-gray-200 dark:border-gray-700'
                            : 'bg-gray-50 dark:bg-gray-800/30 border-gray-200 dark:border-gray-700'
                        }`}>
                          <div className="flex items-center gap-2.5">
                            <div className={`w-1 h-5 rounded-full ${isFall ? 'bg-slate-400 dark:bg-slate-500' : 'bg-gray-300 dark:bg-gray-600'}`} />
                            <h4 className="font-semibold text-sm text-gray-800 dark:text-gray-200">
                              {sem.label}
                            </h4>
                          </div>
                          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                            <span>{sem.courses.length} ders</span>
                            <span className="text-gray-300 dark:text-gray-600">|</span>
                            <span>{semCredits} Kredi</span>
                          </div>
                        </div>

                        <div className="p-3 space-y-2 flex-1">
                          {sem.courses.map((course: CompletedCourse, idx: number) => {
                            const courseKey = `${sem.key}-${course.course_code}-${idx}`;
                            const isExpanded = expandedCourses.has(courseKey);
                            const hasScores = course.assessment_scores && Object.keys(course.assessment_scores).length > 0;

                            return (
                              <div
                                key={courseKey}
                                className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900/50"
                              >
                                <button
                                  onClick={() => hasScores && toggleCourse(courseKey)}
                                  className={`w-full flex items-center justify-between p-3 text-left transition-colors ${
                                    hasScores
                                      ? 'hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer'
                                      : 'cursor-default'
                                  }`}
                                >
                                  <div className="flex items-center gap-3 flex-1 min-w-0">
                                    <span className="px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300 text-xs font-mono shrink-0">
                                      {course.course_code}
                                    </span>
                                    <span className="text-sm font-medium text-gray-900 dark:text-white truncate">
                                      {course.course_name}
                                    </span>
                                  </div>
                                  <div className="flex items-center gap-2 shrink-0">
                                    <span className="text-xs text-gray-500 dark:text-gray-400 hidden sm:block">
                                      {course.credits} Kr
                                    </span>
                                    <span className={`px-2 py-0.5 rounded text-xs font-bold ${getGradeColor(course.grade_point)}`}>
                                      {gradePointToLetter(course.grade_point)}
                                    </span>
                                    {hasScores && (
                                      <ChevronDown
                                        className={`h-4 w-4 text-gray-400 transition-transform duration-200 ${
                                          isExpanded ? 'rotate-180' : ''
                                        }`}
                                      />
                                    )}
                                  </div>
                                </button>

                                {isExpanded && hasScores && (
                                  <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-3 py-2 flex items-center gap-1.5">
                                    <div className="flex flex-wrap gap-1.5 flex-1">
                                      {Object.entries(course.assessment_scores!).map(([slug, score]) => (
                                        <span
                                          key={slug}
                                          className={`inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium ${
                                            score >= 85
                                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300'
                                              : score >= 70
                                              ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300'
                                              : score >= 50
                                              ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/50 dark:text-yellow-300'
                                              : 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300'
                                          }`}
                                        >
                                          <span className="opacity-70">{getAssessmentName(slug)}:</span>
                                          <span className="font-semibold">{score}</span>
                                        </span>
                                      ))}
                                    </div>
                                    <span className="shrink-0 inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                                      <span className="opacity-70">Ort:</span>
                                      <span className="font-semibold">{course.weighted_average.toFixed(1)}</span>
                                    </span>
                                  </div>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </Card>
            ))}
          </div>
        );
      })()}
    </div>
  );
}
