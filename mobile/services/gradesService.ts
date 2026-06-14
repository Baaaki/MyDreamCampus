import api from './api';
import type {
  MyGradesResponse,
  TranscriptResponse,
} from '@/types/grades.types';

export const gradesService = {
  async getMyGrades(): Promise<MyGradesResponse> {
    const response = await api.get<MyGradesResponse>('/grades/student/my');
    return response.data;
  },

  async getTranscript(studentId: string): Promise<TranscriptResponse> {
    const response = await api.get<TranscriptResponse>(`/grades/transcript/${studentId}`);
    return response.data;
  },
};

export default gradesService;
