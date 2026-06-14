// QR Scan
export interface QRPayload {
  sid: string;
  sig: string;
}

export interface ScanQRRequest {
  qr_payload: QRPayload;
}

export interface ScanQRResponse {
  message: string;
  course_code: string;
  course_name: string;
  week_number: number;
  session_type: 'theory' | 'lab';
  marked_at: string;
}

// My Attendance
export interface AttendanceStats {
  present_count: number;
  absent_count: number;
  total_sessions: number;
  min_required: number;
  passed: boolean;
}

export interface WeeklyRecord {
  week: number;
  session_type: 'theory' | 'lab';
  date: string;
  marked_via: 'qr' | 'manual';
  note: string;
}

export interface CourseAttendance {
  course_id: string;
  course_code: string;
  course_name: string;
  instructor: string;
  total_weeks: number;
  theory: AttendanceStats;
  lab: AttendanceStats;
  weekly_records: WeeklyRecord[];
}

export interface MyAttendanceResponse {
  student_id: string;
  student_number: string;
  semester: string;
  courses: CourseAttendance[];
}
