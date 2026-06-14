import type { MyAttendanceResponse, CourseAttendanceDetail } from '@/lib/types';

// Mock Attendance Response (from attendance-service)
export const mockMyAttendanceResponse: MyAttendanceResponse = {
  student_id: '550e8400-e29b-41d4-a716-446655440005',
  student_number: '20210101001',
  semester: '2025-fall',
  courses: [
    {
      course_id: '550e8400-e29b-41d4-a716-446655440301',
      course_code: 'BİL 3001-1',
      course_name: 'Veri Yapıları ve Algoritmalar',
      instructor: 'Prof. Dr. Ahmet Yılmaz',
      total_weeks: 14,
      completed_weeks: 8,
      present_count: 7,
      absent_count: 1,
      absent_weeks: [5],
      weekly_records: [
        {
          week: 1,
          date: '2025-09-15',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 2,
          date: '2025-09-22',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 3,
          date: '2025-09-29',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 4,
          date: '2025-10-06',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 5,
          date: '2025-10-13',
          is_present: false,
          marked_via: 'auto',
        },
        {
          week: 6,
          date: '2025-10-20',
          is_present: true,
          marked_via: 'manual',
          note: 'Geç geldi',
        },
        {
          week: 7,
          date: '2025-10-27',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 8,
          date: '2025-11-03',
          is_present: true,
          marked_via: 'qr',
        },
      ],
    },
    {
      course_id: '550e8400-e29b-41d4-a716-446655440302',
      course_code: 'BİL 3002-1',
      course_name: 'Veritabanı Sistemleri',
      instructor: 'Doç. Dr. Mehmet Kaya',
      total_weeks: 14,
      completed_weeks: 8,
      present_count: 8,
      absent_count: 0,
      absent_weeks: [],
      weekly_records: [
        {
          week: 1,
          date: '2025-09-16',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 2,
          date: '2025-09-23',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 3,
          date: '2025-09-30',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 4,
          date: '2025-10-07',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 5,
          date: '2025-10-14',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 6,
          date: '2025-10-21',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 7,
          date: '2025-10-28',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 8,
          date: '2025-11-04',
          is_present: true,
          marked_via: 'qr',
        },
      ],
    },
    {
      course_id: '550e8400-e29b-41d4-a716-446655440303',
      course_code: 'BİL 3003-1',
      course_name: 'İşletim Sistemleri',
      instructor: 'Prof. Dr. Ahmet Yılmaz',
      total_weeks: 14,
      completed_weeks: 8,
      present_count: 5,
      absent_count: 3,
      absent_weeks: [2, 4, 7],
      weekly_records: [
        {
          week: 1,
          date: '2025-09-15',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 2,
          date: '2025-09-22',
          is_present: false,
          marked_via: 'auto',
        },
        {
          week: 3,
          date: '2025-09-29',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 4,
          date: '2025-10-06',
          is_present: false,
          marked_via: 'auto',
        },
        {
          week: 5,
          date: '2025-10-13',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 6,
          date: '2025-10-20',
          is_present: true,
          marked_via: 'qr',
        },
        {
          week: 7,
          date: '2025-10-27',
          is_present: false,
          marked_via: 'auto',
        },
        {
          week: 8,
          date: '2025-11-03',
          is_present: true,
          marked_via: 'qr',
        },
      ],
    },
  ],
};

// Mock Course Attendance Details (can be used individually)
export const mockCourseAttendanceDetails: CourseAttendanceDetail[] = mockMyAttendanceResponse.courses;
