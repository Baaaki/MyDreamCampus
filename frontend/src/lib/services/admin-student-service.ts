export interface CreateStudentForm {
  student_number: string
  first_name: string
  last_name: string
  email: string
  faculty: string
  department: string
  enrollment_year: number
  class_level: number
  advisor_id: string
}

/**
 * The backend parses advisor_id as a UUID, so the empty string of an unpicked
 * advisor is a 400; leaving the field out creates the student advisor-less.
 */
export function toCreateStudentPayload(form: CreateStudentForm) {
  const { advisor_id, ...rest } = form
  return advisor_id ? { ...rest, advisor_id } : rest
}
