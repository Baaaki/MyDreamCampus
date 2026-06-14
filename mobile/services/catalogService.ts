import api from './api';
import type {
  SemesterCourseListResponse,
  SemesterCourseResponse,
  TeacherCoursesResponse,
  SemesterResponse,
} from '@/types/catalog.types';

export const catalogService = {
  async getSemesterCourses(
    semesterId: string,
    params?: { page?: number; limit?: number; department?: string }
  ): Promise<SemesterCourseListResponse> {
    const response = await api.get<SemesterCourseListResponse>(
      `/catalog/semesters/${semesterId}/courses`,
      { params }
    );
    return response.data;
  },

  async getSemesterCourse(semesterId: string, courseId: string): Promise<SemesterCourseResponse> {
    const response = await api.get<SemesterCourseResponse>(
      `/catalog/semesters/${semesterId}/courses/${courseId}`
    );
    return response.data;
  },

  async getTeacherCourses(semester?: string): Promise<TeacherCoursesResponse> {
    const params = semester ? { semester } : {};
    const response = await api.get<TeacherCoursesResponse>(
      '/catalog/semesters/teacher/courses',
      { params }
    );
    return response.data;
  },

  async getActiveSemester(): Promise<SemesterResponse> {
    const response = await api.get<SemesterResponse>('/catalog/semesters/active');
    return response.data;
  },

  async getSemesters(): Promise<SemesterResponse[]> {
    const response = await api.get<SemesterResponse[]>('/catalog/semesters');
    return response.data;
  },
};

export default catalogService;
