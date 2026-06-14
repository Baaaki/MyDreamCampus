import type { MyGradesResponse, TranscriptResponse } from '@/lib/types';

// Mock My Grades Response (from grades-service)
// grade_point: DB'de numeric string olarak saklanır ('4.00', '3.50' vb.)
// semester: backend formatı 'YYYY_season' (ör: '2023_fall', '2024_spring')
export const mockMyGradesResponse: MyGradesResponse = {
  student_id: '550e8400-e29b-41d4-a716-446655440005',
  student_number: '20210101001',
  active_courses: [
    {
      course_code: 'BİL 4001',
      course_name: 'Yapay Zeka',
      semester: '2025_fall',
      credits: 4,
      scores: {
        midterm: { score: 78, is_absent: false, is_locked: true },
        final: { score: null, is_absent: false, is_locked: false },
        project: { score: 92, is_absent: false, is_locked: true },
      },
    },
    {
      course_code: 'BİL 4002',
      course_name: 'Bilgisayar Ağları',
      semester: '2025_fall',
      credits: 4,
      scores: {
        midterm: { score: 85, is_absent: false, is_locked: true },
        final: { score: null, is_absent: false, is_locked: false },
        homework: { score: 90, is_absent: false, is_locked: true },
      },
    },
  ],
  completed_courses: [
    // ===== 1. YIL - GÜZ (2022_fall) =====
    {
      course_code: 'BİL 1001',
      course_name: 'Programlama Temelleri',
      semester: '2022_fall',
      credits: 5,
      weighted_average: 92.6,
      grade_point: '4.00',
      assessment_scores: { midterm: 88, final: 95, homework: 94 },
    },
    {
      course_code: 'MAT 1001',
      course_name: 'Matematik I',
      semester: '2022_fall',
      credits: 5,
      weighted_average: 78.0,
      grade_point: '3.00',
      assessment_scores: { midterm: 72, final: 82, quiz: 80 },
    },
    {
      course_code: 'FİZ 1001',
      course_name: 'Fizik I',
      semester: '2022_fall',
      credits: 4,
      weighted_average: 65.3,
      grade_point: '2.00',
      assessment_scores: { midterm: 58, final: 70, lab: 68 },
    },
    // ===== 1. YIL - BAHAR (2023_spring) =====
    {
      course_code: 'BİL 1002',
      course_name: 'İleri Programlama',
      semester: '2023_spring',
      credits: 4,
      weighted_average: 85.4,
      grade_point: '3.75',
      assessment_scores: { midterm: 80, final: 88, project: 90 },
    },
    {
      course_code: 'MAT 1002',
      course_name: 'Matematik II',
      semester: '2023_spring',
      credits: 5,
      weighted_average: 71.2,
      grade_point: '2.75',
      assessment_scores: { midterm: 65, final: 75, quiz: 74 },
    },
    // ===== 2. YIL - GÜZ (2023_fall) =====
    {
      course_code: 'BİL 2001',
      course_name: 'Veri Yapıları',
      semester: '2023_fall',
      credits: 5,
      weighted_average: 88.1,
      grade_point: '3.75',
      assessment_scores: { midterm: 85, final: 90, project: 89 },
    },
    {
      course_code: 'MAT 2001',
      course_name: 'Lineer Cebir',
      semester: '2023_fall',
      credits: 4,
      weighted_average: 55.0,
      grade_point: '1.75',
      assessment_scores: { midterm: 48, final: 60, quiz: 57 },
    },
    // ===== 2. YIL - BAHAR (2024_spring) =====
    {
      course_code: 'BİL 2002',
      course_name: 'Nesne Yönelimli Programlama',
      semester: '2024_spring',
      credits: 4,
      weighted_average: 45.8,
      grade_point: '1.00',
      assessment_scores: { midterm: 40, final: 50, homework: 47 },
    },
    // ===== 3. YIL - GÜZ (2024_fall) =====
    {
      course_code: 'BİL 3001',
      course_name: 'Veritabanı Sistemleri',
      semester: '2024_fall',
      credits: 4,
      weighted_average: 38.5,
      grade_point: '0.50',
      assessment_scores: { midterm: 30, final: 45, project: 40 },
    },
    // ===== 3. YIL - BAHAR (2025_spring) =====
    {
      course_code: 'BİL 3002',
      course_name: 'İşletim Sistemleri',
      semester: '2025_spring',
      credits: 4,
      weighted_average: 28.0,
      grade_point: '0.00',
      assessment_scores: { midterm: 20, final: 35, lab: 30 },
    },
  ],
  cumulative_gpa: 2.85,
  total_credits: 44,
};

// Mock Transcript Response (from grades-service)
export const mockTranscriptResponse: TranscriptResponse = {
  student: {
    student_number: '20210101001',
    first_name: 'Ali',
    last_name: 'Çelik',
    department: 'Bilgisayar Mühendisliği',
    enrollment_year: 2021,
  },
  semesters: [
    {
      semester: '2022_fall',
      semester_display: '2022-2023 Güz',
      courses: [
        { course_code: 'BİL 1001', course_name: 'Programlama Temelleri', credits: 5, grade_point: '4.00' },
        { course_code: 'MAT 1001', course_name: 'Matematik I', credits: 5, grade_point: '3.00' },
        { course_code: 'FİZ 1001', course_name: 'Fizik I', credits: 4, grade_point: '2.00' },
      ],
      semester_credits: 14,
      semester_gpa: 3.07,
    },
    {
      semester: '2023_spring',
      semester_display: '2022-2023 Bahar',
      courses: [
        { course_code: 'BİL 1002', course_name: 'İleri Programlama', credits: 4, grade_point: '3.75' },
        { course_code: 'MAT 1002', course_name: 'Matematik II', credits: 5, grade_point: '2.75' },
      ],
      semester_credits: 9,
      semester_gpa: 3.19,
    },
    {
      semester: '2023_fall',
      semester_display: '2023-2024 Güz',
      courses: [
        { course_code: 'BİL 2001', course_name: 'Veri Yapıları', credits: 5, grade_point: '3.75' },
        { course_code: 'MAT 2001', course_name: 'Lineer Cebir', credits: 4, grade_point: '1.75' },
      ],
      semester_credits: 9,
      semester_gpa: 2.86,
    },
    {
      semester: '2024_spring',
      semester_display: '2023-2024 Bahar',
      courses: [
        { course_code: 'BİL 2002', course_name: 'Nesne Yönelimli Programlama', credits: 4, grade_point: '1.00' },
      ],
      semester_credits: 4,
      semester_gpa: 1.00,
    },
    {
      semester: '2024_fall',
      semester_display: '2024-2025 Güz',
      courses: [
        { course_code: 'BİL 3001', course_name: 'Veritabanı Sistemleri', credits: 4, grade_point: '0.50' },
      ],
      semester_credits: 4,
      semester_gpa: 0.50,
    },
    {
      semester: '2025_spring',
      semester_display: '2024-2025 Bahar',
      courses: [
        { course_code: 'BİL 3002', course_name: 'İşletim Sistemleri', credits: 4, grade_point: '0.00' },
      ],
      semester_credits: 4,
      semester_gpa: 0.00,
    },
  ],
  summary: {
    total_credits: 44,
    cumulative_gpa: 2.85,
  },
  generated_at: new Date().toISOString(),
};

// Letter grade mapping helper
export const letterGradeMap: Record<string, { min: number; max: number; points: number }> = {
  'AA': { min: 90, max: 100, points: 4.0 },
  'AB': { min: 85, max: 89, points: 3.75 },
  'BA': { min: 80, max: 84, points: 3.5 },
  'BB': { min: 75, max: 79, points: 3.0 },
  'BC': { min: 70, max: 74, points: 2.75 },
  'CB': { min: 65, max: 69, points: 2.5 },
  'CC': { min: 60, max: 64, points: 2.0 },
  'CD': { min: 55, max: 59, points: 1.75 },
  'DC': { min: 50, max: 54, points: 1.5 },
  'DD': { min: 45, max: 49, points: 1.0 },
  'FD': { min: 35, max: 44, points: 0.5 },
  'FF': { min: 0, max: 34, points: 0.0 },
};
