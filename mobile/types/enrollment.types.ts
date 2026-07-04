// Backend: monolith enrollment module DTO'lari (ogrenci kapsami).

export interface ScheduleSession {
  day_of_week: number;
  slot_numbers: number[];
  session_type: 'theory' | 'lab';
}

export interface EnrollmentCourse {
  id: string;
  course_code: string;
  course_name: string;
  credits: number;
  instructor: string;
  schedule_sessions: ScheduleSession[];
}

export interface EnrollmentProgramResponse {
  id: string;
  student_id: string;
  student_number: string;
  student_name: string;
  department: string;
  class_level: number;
  semester: string;
  status: 'pending' | 'approved' | 'rejected';
  courses: EnrollmentCourse[];
  created_at: string;
}

export interface MyEnrollmentsResponse {
  programs: EnrollmentProgramResponse[];
  total_count: number;
}
