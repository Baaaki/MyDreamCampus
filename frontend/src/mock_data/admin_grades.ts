import type { CourseStatusResponse, StudentGrades } from '@/lib/types';

export const mockAdminCourseStatus: CourseStatusResponse = {
  course_id: 'mock-course-123',
  course_code: 'BIL 2011',
  course_name: 'Algoritmalar ve Veri Yapıları',
  semester: '2025-2026 Güz',
  total_students: 5,
  is_finalized: false,
  assessments: [
    {
      slug: 'vize-1',
      name: '1. Vize',
      weight: 30,
      graded_count: 5,
      pending_count: 0,
      is_complete: true,
    },
    {
      slug: 'proje',
      name: 'Dönem Projesi',
      weight: 30,
      graded_count: 4,
      pending_count: 1,
      is_complete: false,
    },
    {
      slug: 'final',
      name: 'Final',
      weight: 40,
      graded_count: 0,
      pending_count: 5,
      is_complete: false,
    },
  ],
};

export const mockAdminStudents: StudentGrades[] = [
  {
    registration_id: 'reg-001',
    student_id: 'std-001',
    student_number: '20210001',
    first_name: 'Ahmet',
    last_name: 'Yılmaz',
    current_average: null,
    scores: {
      'vize-1': { score: 85, is_absent: false, is_locked: true },
      'proje': { score: 90, is_absent: false, is_locked: false },
    },
  },
  {
    registration_id: 'reg-002',
    student_id: 'std-002',
    student_number: '20210002',
    first_name: 'Ayşe',
    last_name: 'Kaya',
    current_average: null,
    scores: {
      'vize-1': { score: null, is_absent: true, is_locked: true },
      'proje': { score: 95, is_absent: false, is_locked: false },
    },
  },
  {
    registration_id: 'reg-003',
    student_id: 'std-003',
    student_number: '20210003',
    first_name: 'Mehmet',
    last_name: 'Demir',
    current_average: null,
    scores: {
      'vize-1': { score: 65, is_absent: false, is_locked: false },
      'proje': { score: 70, is_absent: false, is_locked: false },
    },
  },
  {
    registration_id: 'reg-004',
    student_id: 'std-004',
    student_number: '20210004',
    first_name: 'Fatma',
    last_name: 'Çelik',
    current_average: null,
    scores: {
      'vize-1': { score: 100, is_absent: false, is_locked: true },
      'proje': { score: null, is_absent: false, is_locked: false },
    },
  },
  {
    registration_id: 'reg-005',
    student_id: 'std-005',
    student_number: '20210005',
    first_name: 'Can',
    last_name: 'Öztürk',
    current_average: null,
    scores: {
      'vize-1': { score: 45, is_absent: false, is_locked: false },
      'proje': { score: 55, is_absent: false, is_locked: false },
    },
  },
];
