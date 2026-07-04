import api from './api';
import type { MyGradesResponse } from '@/types/grades.types';

export const gradesService = {
  async getMyGrades(): Promise<MyGradesResponse> {
    // Not: dogru yol /grades/my/grades — eski mikroservis /grades/student/my degil.
    const response = await api.get<MyGradesResponse>('/grades/my/grades');
    return response.data;
  },
};

export default gradesService;
