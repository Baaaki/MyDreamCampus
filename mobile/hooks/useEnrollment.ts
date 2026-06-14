import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { enrollmentService } from '@/services/enrollmentService';
import type { CreateEnrollmentRequest } from '@/types/enrollment.types';

export const useAvailableCourses = (semester: string) => {
  return useQuery({
    queryKey: ['available-courses', semester],
    queryFn: () => enrollmentService.getAvailableCourses(semester),
    enabled: !!semester,
  });
};

export const useMyEnrollments = (semester?: string, status?: string) => {
  return useQuery({
    queryKey: ['my-enrollments', semester, status],
    queryFn: () => enrollmentService.getMyEnrollments(semester, status),
  });
};

export const useCreateEnrollment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateEnrollmentRequest) => enrollmentService.createEnrollment(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-enrollments'] });
      queryClient.invalidateQueries({ queryKey: ['available-courses'] });
    },
  });
};

export const useCancelEnrollment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (semester: string) => enrollmentService.cancelEnrollment(semester),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-enrollments'] });
    },
  });
};

export const useLatestRejection = (semester: string) => {
  return useQuery({
    queryKey: ['latest-rejection', semester],
    queryFn: () => enrollmentService.getLatestRejection(semester),
    enabled: !!semester,
  });
};
