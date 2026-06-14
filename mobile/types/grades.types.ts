// Student Grades
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
  assessment_scores: Record<string, ScoreDetail>;
}

export interface MyGradesResponse {
  student_id: string;
  student_number: string;
  active_courses: ActiveCourse[];
  completed_courses: CompletedCourse[];
  cumulative_gpa: number;
  total_credits: number;
}

// Transcript
export interface TranscriptCourse {
  course_code: string;
  course_name: string;
  credits: number;
  grade_point: string;
}

export interface TranscriptSemester {
  semester: string;
  semester_display: string;
  courses: TranscriptCourse[];
  semester_credits: number;
  semester_gpa: number;
}

export interface TranscriptStudent {
  student_number: string;
  first_name: string;
  last_name: string;
  department: string;
  enrollment_year: number;
}

export interface TranscriptResponse {
  student: TranscriptStudent;
  semesters: TranscriptSemester[];
  summary: {
    total_credits: number;
    cumulative_gpa: number;
  };
  generated_at: string;
}
