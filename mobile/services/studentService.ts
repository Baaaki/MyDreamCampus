import api from './api';
import type { StudentProfile } from '@/types/student.types';

export const studentService = {
  // The student's own record: the login user id is the student id.
  async getById(id: string): Promise<StudentProfile> {
    const response = await api.get<StudentProfile>(`/students/${id}`);
    return response.data;
  },
};

export default studentService;
