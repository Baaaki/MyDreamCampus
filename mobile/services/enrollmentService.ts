import api from './api';
import type { MyEnrollmentsResponse } from '@/types/enrollment.types';

export const enrollmentService = {
  async getMyEnrollments(semester?: string, status?: string): Promise<MyEnrollmentsResponse> {
    const params: Record<string, string> = {};
    if (semester) params.semester = semester;
    if (status) params.status = status;
    const response = await api.get<MyEnrollmentsResponse>('/enrollment/my-enrollments', { params });
    return response.data;
  },
};

export default enrollmentService;
