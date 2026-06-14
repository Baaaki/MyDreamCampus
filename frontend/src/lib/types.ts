// Common types used across the application

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'teacher' | 'student';
  department?: string;
}

export interface AuthResponse {
  access_token: string;
  expires_in: number;
  user: User;
  force_password_change: boolean;
  message?: string;
}

export interface Session {
  id: string;
  device_info?: string;
  ip_address?: string;
  created_at: string;
  last_used_at: string;
  is_current: boolean;
}

// Staff types
export interface Staff {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  faculty?: string;
  department?: string;
  phone?: string;
  office_location?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// Student types
export interface Student {
  id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  email: string;
  faculty: string;
  department: string;
  enrollment_year: number;
  class_level: number;
  advisor_id?: string | null;
  advisor?: AdvisorInfo;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface AdvisorInfo {
  id: string;
  first_name: string;
  last_name: string;
  email?: string;
}

// Course Catalog types
export interface WeeklyTopic {
  week: number;
  topic: string;
  description?: string;
}

export interface CourseCoordinator {
  title: string;
  name: string;
  email?: string;
  phone?: string;
  office?: string;
}

export interface CourseCatalog {
  id: string;
  course_code: string;
  name: string;
  faculty: string;
  department: string;
  offering_unit?: string; // Dersi veren birim
  class_level: number;
  semester?: number;
  credits: number;
  theoretical_hours: number;
  lab_hours?: number;
  ects?: number;
  course_type: string;
  education_level?: string; // Lisans, Yüksek Lisans, vb.
  teaching_type?: string; // Örgün Öğretim, Uzaktan Öğretim, vb.
  language?: string; // Türkçe, İngilizce, vb.
  coordinator?: CourseCoordinator;
  purpose?: string; // Dersin amacı
  learning_outcomes_list?: string[]; // Öğrenme kazanımları listesi
  weekly_topics?: WeeklyTopic[]; // Haftalık konular
  recommended_sources?: string[]; // Önerilen kaynaklar
  prerequisites: Prerequisite[];
  description?: string;
  learning_outcomes?: string;
  syllabus?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// Faculty type
export interface Faculty {
  id: string;
  name: string;
  code: string;
  departments: Department[];
}

// Department type
export interface Department {
  id: string;
  name: string;
  facultyId: string;
  code: string;
  description?: string;
}

export interface Prerequisite {
  id: string;
  course_code: string;
  course_name: string;
}

export interface ScheduleSession {
  day: number;
  slot: number;
}

export interface CourseOffering {
  id: string;
  course_code: string;
  course_name: string;
  instructor: string;
  classroom: string;
  schedule: ScheduleSession[];
}

export interface AvailableCourse {
  id: string;
  course_code: string;
  course_name: string;
  credits: number;
  schedule_sessions: AvailableCourseSlot[];
  max_capacity: number;
  current_enrollment: number;
  available_seats: number;
  instructor: string;
}

export interface AvailableCourseSlot {
  day_of_week: string;
  slot_numbers: number[];
  session_type?: 'theory' | 'lab';
}

export interface CourseBasic {
  id: string;
  course_code: string;
  course_name: string;
  credits: number;
  instructor?: string;
  schedule_sessions?: ScheduleSessionDTO[];
}

// Enrollment types
export interface EnrollmentProgramResponse {
  id: string;
  student_id: string;
  student_number?: string;
  student_name?: string;
  department?: string;
  class_level?: number;
  semester: string;
  status: string;
  courses: CourseBasic[];
  created_at: string;
}

export interface MyEnrollmentsResponse {
  programs: EnrollmentProgramResponse[];
}

export interface AdvisorPendingProgramsResponse {
  advisor_id: string;
  programs: EnrollmentProgramResponse[];
}

export interface RejectedCourseDetail {
  course_id: string;
  course_code: string;
  course_name: string;
  credits: number;
  instructor: string;
}

export interface RejectedCoursesData {
  courses: RejectedCourseDetail[];
  total_credits: number;
  submitted_at: string;
}

export interface RejectionDetail {
  id: string;
  advisor_id: string;
  advisor_fullname: string;
  rejection_reason: string;
  rejected_courses: RejectedCoursesData;
  rejected_at: string;
}

export interface LatestRejectionResponse {
  student_id: string;
  semester: string;
  has_rejection: boolean;
  latest_rejection: RejectionDetail | null;
  total_rejections: number;
}

export interface MyRejectionsResponse {
  student_id: string;
  rejections: RejectionDetail[];
}

// Attendance types
export interface QRPayload {
  sid: string;
  ts: number;
  sig: string;
}

export interface SessionListItem {
  session_id?: string;
  week_number: number;
  session_type: string;
  session_date?: string;
  present_count?: number;
  absent_count?: number;
  is_active?: boolean;
  status?: string;
}

export interface WeeklyAttendanceRecord {
  week: number;
  date: string;
  is_present: boolean;
  marked_via: string;
  note?: string;
}

export interface CourseAttendanceDetail {
  course_id: string;
  course_code: string;
  course_name: string;
  instructor: string;
  total_weeks: number;
  completed_weeks: number;
  present_count: number;
  absent_count: number;
  absent_weeks: number[];
  weekly_records: WeeklyAttendanceRecord[];
}

export interface MyAttendanceResponse {
  student_id: string;
  student_number: string;
  semester: string;
  courses: CourseAttendanceDetail[];
}

// Grades types
export interface ScoreDetail {
  score: number | null;
  is_absent: boolean;
  is_locked: boolean;
}

export interface ActiveCourse {
  course_code: string;
  course_name: string;
  semester: string;
  credits: number;
  scores: Record<string, ScoreDetail>;
}

export interface CompletedCourse {
  course_code: string;
  course_name: string;
  semester: string;
  credits: number;
  weighted_average: number;
  grade_point: string;
  assessment_scores?: Record<string, number>;
}

export interface MyGradesResponse {
  student_id: string;
  student_number: string;
  active_courses: ActiveCourse[];
  completed_courses: CompletedCourse[];
  cumulative_gpa: number;
  total_credits: number;
}

export interface StudentInfo {
  student_number: string;
  first_name: string;
  last_name: string;
  department: string;
  enrollment_year: number;
}

export interface CourseGrade {
  course_code: string;
  course_name: string;
  credits: number;
  grade_point: string;
}

export interface SemesterGrades {
  semester: string;
  semester_display: string;
  courses: CourseGrade[];
  semester_credits: number;
  semester_gpa: number;
}

export interface TranscriptSummary {
  total_credits: number;
  cumulative_gpa: number;
}

export interface TranscriptResponse {
  student: StudentInfo;
  semesters: SemesterGrades[];
  summary: TranscriptSummary;
  generated_at: string;
}

// Teacher Grade Entry types
export interface AssessmentStatus {
  slug: string;
  name: string;
  weight: number;
  graded_count: number;
  pending_count: number;
  is_complete: boolean;
}

export interface ClassStatistics {
  mean: number;
  stddev: number;
  passing_count: number;
  failing_count: number;
  attendance_failed_count?: number;
}

export interface CourseStatusResponse {
  course_id: string;
  course_code: string;
  course_name: string;
  semester: string;
  total_students: number;
  assessments: AssessmentStatus[];
  is_finalized: boolean;
  pending_message?: string;
  finalized_at?: string;
  grading_type?: string;
  class_statistics?: ClassStatistics;
}

export interface StudentGrades {
  registration_id: string;
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  scores: Record<string, ScoreDetail>;
  current_average: number | null;
  is_attendance_failed?: boolean;
}

export interface CourseStudentsResponse {
  course_id: string;
  course_code: string;
  students: StudentGrades[];
}

export interface SubmitScoreRequest {
  registration_id: string;
  slug: string;
  score: number | null;
  is_absent: boolean;
}

export interface SubmitScoreResponse {
  id: string;
  student_number: string;
  slug: string;
  score: number | null;
  is_absent: boolean;
  graded_at: string;
  auto_finalized?: boolean;
  finalize_result?: FinalizeResult;
}

export interface BulkScoreEntry {
  registration_id: string;
  score: number | null;
  is_absent: boolean;
}

export interface BulkSubmitScoresRequest {
  slug: string;
  scores: BulkScoreEntry[];
}

export interface BulkSubmitScoresResponse {
  slug: string;
  success_count: number;
  auto_finalized?: boolean;
  finalize_result?: FinalizeResult;
}

export interface FinalizeResult {
  grading_type: string;
  class_mean: number;
  total_students: number;
  passing_count: number;
  failing_count: number;
  attendance_failed_count?: number;
}

// Meal service types
export interface Cafeteria {
  id: string;
  name: string;
  location: string;
  has_vegan_menu: boolean;
  serves_dinner: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CafeteriaInfo {
  id: string;
  name: string;
  location: string;
}

export interface ReservationResponse {
  id: string;
  date: string;
  meal_time: string;
  menu_type: string;
  cafeteria_name: string;
  cafeteria?: CafeteriaInfo;
  status: string;
  is_used: boolean;
  created_at: string;
}

export interface ReservationSummary {
  total: number;
  confirmed: number;
  pending: number;
  used: number;
  cancelled: number;
}

export interface MyReservationsResponse {
  reservations: ReservationResponse[];
  summary: ReservationSummary;
}

export interface ValidTimeWindow {
  start: string;
  end: string;
}

export interface QRResponse {
  cafeteria_id: string;
  cafeteria_name: string;
  date: string;
  meal_time: string;
  qr_payload: string;
  valid_time_window: ValidTimeWindow;
}

// Menu types
export interface MenuItem {
  name: string;
  calories?: number;
}

export interface DailyMenu {
  id: string;
  cafeteria_id: string;
  cafeteria_name?: string;
  date: string;
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan' | 'diet';
  items: MenuItem[];
  created_at: string;
  updated_at: string;
}

// Pagination
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

// Semester Course types (Dönemlik Açılan Dersler)
export interface ScheduleSessionDTO {
  day_of_week: string; // 'monday', 'tuesday', etc.
  slot_numbers: number[]; // [1, 2] for 1st and 2nd slots
  session_type: 'theory' | 'lab'; // required by backend
}

export interface AssessmentItem {
  slug: string; // 'midterm', 'final', 'quiz', 'homework', 'project', etc.
  name: string; // 'Vize', 'Final', 'Quiz', 'Ödev', 'Proje', etc.
  weight: number; // 0-100
}

export interface CreateSemesterCourseRequest {
  course_code: string;
  class_level: number;
  instructor_id: string;
  instructor_fullname: string;
  classroom_location: string;
  max_capacity: number;
  assessment_schema: AssessmentItem[];
  schedule_sessions: ScheduleSessionDTO[];
}

export interface SemesterCourse {
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
  schedule_sessions: ScheduleSessionDTO[];
  prerequisites?: Prerequisite[];
  created_at: string;
  updated_at: string;
}

// Teacher types
export interface TeacherScheduleSession {
  day: string;
  time: string;
  room: string;
}

export interface TeacherCourse {
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
  instructor_id: string;
  total_courses: number;
  courses: TeacherCourse[];
}

// Attendance Session types
export interface CreateSessionRequest {
  course_id: string;
  week_number: number;
  duration_minutes: number;
  session_type: 'theory' | 'lab';
}

export interface CreateSessionResponse {
  session_id: string;
  course_id: string;
  course_code: string;
  course_name: string;
  week_number: number;
  session_type: string;
  session_date: string;
  qr_rotation_interval: number;
  started_at: string;
  expires_at: string;
  enrolled_student_count: number;
}

export interface SessionDetailsResponse {
  session_id: string;
  course_id: string;
  course_code: string;
  course_name: string;
  week_number: number;
  session_type: string;
  session_date: string;
  semester: string;
  is_active: boolean;
  qr_rotation_interval: number;
  started_at: string;
  expires_at: string;
  enrolled_student_count: number;
  present_count: number;
  absent_count: number;
}

export interface QRCodeResponse {
  session_id: string;
  qr_payload: QRPayload;
  valid_until: string;
  rotation_interval: number;
}

export interface AttendanceRecordItem {
  id: string;
  student_id: string;
  student_number: string;
  student_name: string;
  is_present: boolean;
  marked_via: string;
  marked_at?: string;
  note?: string;
}

export interface SessionRecordsResponse {
  session_id: string;
  week_number: number;
  total_count: number;
  present_count: number;
  records: AttendanceRecordItem[];
}

export interface EnrolledStudentItem {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  email: string;
  is_marked: boolean;
}

export interface SessionStudentsResponse {
  session_id: string;
  course_id: string;
  total_enrolled: number;
  marked_count: number;
  students: EnrolledStudentItem[];
}

export interface ManualAttendanceRequest {
  student_id: string;
  is_present: boolean;
  note?: string;
}

export interface ManualAttendanceResponse {
  id: string;
  session_id: string;
  student_id: string;
  student_number: string;
  student_name: string;
  is_present: boolean;
  marked_via: string;
  note?: string;
  marked_at?: string;
}

export interface CloseSessionResponse {
  session_id: string;
  closed_at: string;
  summary: {
    total_enrolled: number;
    present_count: number;
    absent_count: number;
  };
  newly_marked_absent: {
    student_id: string;
    student_number: string;
    student_name: string;
  }[];
}

// System Management types (Time Machine & Academic Periods)
export interface TimeStatus {
  mode: 'real' | 'simulated';
  current_time: string;
  simulated_time: string | null;
}

export interface ServiceTimeStatus {
  service: string;
  label: string;
  status: TimeStatus | null;
  error: string | null;
}

// Grades service: has course_id for course-specific deadline overrides
export interface AcademicPeriod {
  id: string;
  semester: string;
  period_start: string;
  period_end: string;
  course_id: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Catalog & Enrollment services: no course_id
export interface SimplePeriod {
  id: string;
  semester: string;
  period_start: string;
  period_end: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePeriodRequest {
  semester: string;
  period_start: string;
  period_end: string;
  course_id?: string;
}

export interface SimpleCreatePeriodRequest {
  semester: string;
  period_start: string;
  period_end: string;
}

export interface UpdatePeriodRequest {
  period_end?: string;
  is_active?: boolean;
}

// Meal service: closed days (holidays) instead of academic periods
export interface ClosedDay {
  id: string;
  date: string;
  reason: string;
  semester?: string;
  created_at: string;
}

export interface CreateClosedDayRequest {
  date: string;
  reason: string;
}

// Semester State Machine types (Zero Trust)
export type SemesterStatus = 'planned' | 'active' | 'completed';

export interface Semester {
  id: string;
  name: string;
  status: SemesterStatus;
  hard_deadline: string;
  activated_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface PeriodTimeRange {
  start: string;
  end: string;
}

export interface SemesterPeriods {
  enrollment?: PeriodTimeRange;
  grading?: PeriodTimeRange;
  attendance?: PeriodTimeRange;
  catalog?: PeriodTimeRange;
}

export interface CreateSemesterRequest {
  name: string;
  hard_deadline: string;
  periods?: SemesterPeriods;
  closed_days?: Array<{ date: string; reason: string }>;
}

export interface UpdateSemesterRequest {
  hard_deadline: string;
  periods?: SemesterPeriods;
  closed_days?: Array<{ date: string; reason: string }>;
}

// Audit Log types
export interface AuditLogEntry {
  id: string;
  timestamp: string;
  service: string;
  actor_id: string;
  actor_role: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details: Record<string, unknown>;
}

export interface AuditLogListResponse {
  entries: AuditLogEntry[];
  total: number;
}

// Admin Attendance
export interface AdminSessionItem {
  session_id: string;
  course_id: string;
  course_code: string;
  course_name: string;
  instructor_id: string;
  semester: string;
  week_number: number;
  session_type: 'theory' | 'lab';
  session_date: string;
  is_active: boolean;
  started_at: string;
  expires_at: string;
  present_count: number;
  enrolled_count: number;
}

export interface AdminSessionsResponse {
  sessions: AdminSessionItem[];
  total: number;
}
