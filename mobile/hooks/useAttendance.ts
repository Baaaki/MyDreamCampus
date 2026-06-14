import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { attendanceService } from '@/services/attendanceService';
import type { ScanQRRequest } from '@/types/attendance.types';

export const useMyAttendance = (semester?: string) => {
  return useQuery({
    queryKey: ['my-attendance', semester],
    queryFn: () => attendanceService.getMyAttendance(semester),
    staleTime: 5 * 60 * 1000,
  });
};

export const useScanQR = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: ScanQRRequest) => attendanceService.scanQR(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-attendance'] });
    },
  });
};
