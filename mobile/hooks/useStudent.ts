import { skipToken, useQuery } from '@tanstack/react-query';
import { useAuthContext } from '@/contexts/AuthContext';
import { studentService } from '@/services/studentService';

// Name, number, department and class come from the student's own record;
// the enrollment programs a student reads do not carry them.
export const useMyProfile = () => {
  const { user } = useAuthContext();
  const userId = user?.id;
  return useQuery({
    queryKey: ['student', userId],
    queryFn: userId ? () => studentService.getById(userId) : skipToken,
    staleTime: 5 * 60 * 1000,
  });
};
