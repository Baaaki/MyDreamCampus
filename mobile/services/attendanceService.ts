import api from './api';
import type {
  ScanQRRequest,
  ScanQRResponse,
  MyAttendanceResponse,
} from '@/types/attendance.types';

export const attendanceService = {
  async scanQR(data: ScanQRRequest): Promise<ScanQRResponse> {
    const response = await api.post<ScanQRResponse>('/attendance/scan', data);
    return response.data;
  },

  async getMyAttendance(semester?: string): Promise<MyAttendanceResponse> {
    const params = semester ? { semester } : {};
    const response = await api.get<MyAttendanceResponse>('/attendance/my', { params });
    return response.data;
  },
};

export default attendanceService;
