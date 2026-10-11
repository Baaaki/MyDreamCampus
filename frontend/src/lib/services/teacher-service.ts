import { useQuery } from "@tanstack/react-query"
import { semesterApi } from "@/lib/api-client"
import { normalizeTeacherCourse } from "@/lib/normalize-schedule"
import type { TeacherCourse, TeacherCoursesResponse } from "@/lib/types"

export async function getTeacherCourses(): Promise<TeacherCourse[]> {
  const response = await semesterApi
    .get("teacher/courses")
    .json<TeacherCoursesResponse>()
  return (response.courses || []).map(normalizeTeacherCourse)
}

// One key for the attendance list, the grade list and the attendance start
// page, so moving between them reuses the cached list.
export function useTeacherCourses() {
  return useQuery({
    queryKey: ["teacher", "courses"],
    queryFn: getTeacherCourses,
  })
}
