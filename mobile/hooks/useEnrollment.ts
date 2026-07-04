import { useQuery } from '@tanstack/react-query';
import { enrollmentService } from '@/services/enrollmentService';

export const useMyEnrollments = (semester?: string, status?: string) => {
  return useQuery({
    queryKey: ['my-enrollments', semester, status],
    queryFn: () => enrollmentService.getMyEnrollments(semester, status),
    staleTime: 5 * 60 * 1000,
  });
};
