import api from './api';
import type {
  AvailableCoursesResponse,
  CreateEnrollmentRequest,
  EnrollmentProgramResponse,
  MyEnrollmentsResponse,
  LatestRejectionResponse,
  MyRejectionsResponse,
} from '@/types/enrollment.types';

export const enrollmentService = {
  async getAvailableCourses(semester: string): Promise<AvailableCoursesResponse> {
    const response = await api.get<AvailableCoursesResponse>('/enrollment/available-courses', {
      params: { semester },
    });
    return response.data;
  },

  async createEnrollment(data: CreateEnrollmentRequest): Promise<EnrollmentProgramResponse> {
    const response = await api.post<EnrollmentProgramResponse>('/enrollment/programs', data);
    return response.data;
  },

  async getMyEnrollments(semester?: string, status?: string): Promise<MyEnrollmentsResponse> {
    const params: Record<string, string> = {};
    if (semester) params.semester = semester;
    if (status) params.status = status;
    const response = await api.get<MyEnrollmentsResponse>('/enrollment/my-enrollments', { params });
    return response.data;
  },

  async cancelEnrollment(semester: string): Promise<void> {
    await api.delete('/enrollment/programs', { params: { semester } });
  },

  async getLatestRejection(semester: string): Promise<LatestRejectionResponse> {
    const response = await api.get<LatestRejectionResponse>('/enrollment/latest-rejection', {
      params: { semester },
    });
    return response.data;
  },

  async getMyRejections(semester?: string): Promise<MyRejectionsResponse> {
    const params = semester ? { semester } : {};
    const response = await api.get<MyRejectionsResponse>('/enrollment/my-rejections', { params });
    return response.data;
  },
};

export default enrollmentService;
