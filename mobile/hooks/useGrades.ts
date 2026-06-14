import { useQuery } from '@tanstack/react-query';
import { gradesService } from '@/services/gradesService';

export const useMyGrades = () => {
  return useQuery({
    queryKey: ['my-grades'],
    queryFn: () => gradesService.getMyGrades(),
    staleTime: 5 * 60 * 1000,
  });
};

export const useTranscript = (studentId: string, enabled: boolean = true) => {
  return useQuery({
    queryKey: ['transcript', studentId],
    queryFn: () => gradesService.getTranscript(studentId),
    enabled: !!studentId && enabled,
    staleTime: 5 * 60 * 1000,
  });
};
