// Backend: shared/contracts StudentResponse (student-service). A student may
// read only their own record, at their login user id.

export interface StudentProfile {
  id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  email: string;
  faculty: string;
  department: string;
  enrollment_year: number;
  class_level: number;
  advisor_id?: string;
  advisor_name?: string;
  status: string;
}
