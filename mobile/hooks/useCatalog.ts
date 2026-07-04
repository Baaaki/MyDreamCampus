import { useQuery } from '@tanstack/react-query';
import { catalogService } from '@/services/catalogService';

export const useActiveSemester = () => {
  return useQuery({
    queryKey: ['active-semester'],
    queryFn: () => catalogService.getActiveSemester(),
    staleTime: 10 * 60 * 1000,
  });
};
