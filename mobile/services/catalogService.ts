import api from './api';
import type { SemesterResponse } from '@/types/catalog.types';

export const catalogService = {
  async getActiveSemester(): Promise<SemesterResponse> {
    const response = await api.get<SemesterResponse>('/catalog/semesters/active');
    return response.data;
  },
};

export default catalogService;
