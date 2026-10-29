// Backend: monolith grades module — ogrenci kapsami (/grades/my/grades).

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
  // 4-point coefficient as text, e.g. "2.75".
  grade_point: string;
  // Final scores only; absent when the transcript row has none.
  assessment_scores?: Record<string, number>;
}

export interface MyGradesResponse {
  student_id: string;
  student_number: string;
  active_courses: ActiveCourse[] | null;
  completed_courses: CompletedCourse[] | null;
  cumulative_gpa: number;
  total_credits: number;
}
