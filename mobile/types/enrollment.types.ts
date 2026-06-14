// Backend: enrollment-service/internal/dto/enrollment_dto.go

export interface ScheduleSession {
  day_of_week: number;
  slot_numbers: number[];
  session_type: 'theory' | 'lab';
}

export interface AvailableCourse {
  id: string;
  course_code: string;
  course_name: string;
  credits: number;
  schedule_sessions: ScheduleSession[];
  max_capacity: number;
  current_enrollment: number;
  available_seats: number;
  instructor: string;
}

export interface AvailableCoursesResponse {
  student_id: string;
  department: string;
  class_level: number;
  semester: string;
  available_courses: AvailableCourse[];
}

export interface CreateEnrollmentRequest {
  semester: string;
  course_ids: string[];
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

export interface RejectedCourseDetail {
  course_id: string;
  course_code: string;
  course_name: string;
  credits: number;
  instructor: string;
}

export interface RejectionDetail {
  id: string;
  advisor_id: string;
  advisor_fullname: string;
  rejection_reason: string;
  rejected_courses: RejectedCourseDetail[];
  rejected_at: string;
}

export interface LatestRejectionResponse {
  student_id: string;
  semester: string;
  has_rejection: boolean;
  latest_rejection?: RejectionDetail;
  total_rejections: number;
}

export interface MyEnrollmentsResponse {
  programs: EnrollmentProgramResponse[];
  total_count: number;
}

export interface MyRejectionsResponse {
  rejections: RejectionDetail[];
  total_count: number;
}
