// Backend: enrollment-service DTO'lari (ogrenci kapsami). Ogrencinin kendi
// programlarinda ogrenci adi/numarasi/bolumu yok — onlar danisman gorunumunde;
// ogrenci bilgisi icin bkz. student.types.ts.

// Catalog's weekday names, as enrollment passes them on.
export type DayOfWeek =
  | 'monday'
  | 'tuesday'
  | 'wednesday'
  | 'thursday'
  | 'friday'
  | 'saturday'
  | 'sunday';

export interface ScheduleSession {
  day_of_week: DayOfWeek;
  slot_numbers: number[];
  session_type: 'theory' | 'lab';
}

export interface EnrollmentCourse {
  id: string;
  course_code: string;
  course_name: string;
  credits: number;
  instructor?: string;
  schedule_sessions: ScheduleSession[];
}

export interface EnrollmentProgramResponse {
  id: string;
  student_id: string;
  semester: string;
  status: 'pending' | 'approved' | 'rejected';
  courses: EnrollmentCourse[];
  created_at: string;
}

export interface MyEnrollmentsResponse {
  student_id: string;
  programs: EnrollmentProgramResponse[];
}
