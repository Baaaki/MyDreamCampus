// Backend: catalog-service/internal/dto/semester_dto.go

export interface ScheduleSession {
  day_of_week: number;
  slot_numbers: number[];
  session_type: 'theory' | 'lab';
}

export interface AssessmentItem {
  slug: string;
  name: string;
  weight: number;
}

export interface Prerequisite {
  id: string;
  course_code: string;
  course_name: string;
}

export interface SemesterCourseResponse {
  id: string;
  semester: string;
  course_code: string;
  course_name: string;
  credits: number;
  class_level: number;
  instructor_id: string;
  instructor_fullname: string;
  classroom_location: string;
  max_capacity: number;
  assessment_schema: AssessmentItem[];
  schedule_sessions: ScheduleSession[];
  prerequisites: Prerequisite[];
  created_at: string;
  updated_at: string;
}

export interface SemesterCourseListResponse {
  data: SemesterCourseResponse[];
  pagination: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface TeacherScheduleSession {
  day: string;
  time: string;
  room: string;
  session_type: 'theory' | 'lab';
}

export interface TeacherCourseItem {
  id: string;
  course_code: string;
  course_name: string;
  faculty: string;
  department: string;
  semester: string;
  credits: number;
  theoretical_hours: number;
  lab_hours: number;
  classroom_location: string;
  max_capacity: number;
  schedule: TeacherScheduleSession[];
}

export interface TeacherCoursesResponse {
  courses: TeacherCourseItem[];
  total_courses: number;
}

export interface SemesterResponse {
  id: string;
  name: string;
  status: string;
  hard_deadline: string;
  created_at: string;
}
