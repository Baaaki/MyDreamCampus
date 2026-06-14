import { useQuery } from '@tanstack/react-query';
import { catalogService } from '@/services/catalogService';

export const useActiveSemester = () => {
  return useQuery({
    queryKey: ['active-semester'],
    queryFn: () => catalogService.getActiveSemester(),
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useSemesterCourses = (
  semesterId: string,
  params?: { page?: number; limit?: number; department?: string }
) => {
  return useQuery({
    queryKey: ['semester-courses', semesterId, params],
    queryFn: () => catalogService.getSemesterCourses(semesterId, params),
    enabled: !!semesterId,
  });
};

export const useTeacherCourses = (semester?: string) => {
  return useQuery({
    queryKey: ['teacher-courses', semester],
    queryFn: () => catalogService.getTeacherCourses(semester),
  });
};
