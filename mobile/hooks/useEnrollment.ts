import { useQuery } from '@tanstack/react-query';
import { useMyAttendance } from '@/hooks/useAttendance';
import { currentProgram } from '@/lib/enrollment';
import { enrollmentService } from '@/services/enrollmentService';

export const useMyEnrollments = (semester?: string, status?: string) => {
  return useQuery({
    queryKey: ['my-enrollments', semester, status],
    queryFn: () => enrollmentService.getMyEnrollments(semester, status),
    staleTime: 5 * 60 * 1000,
  });
};

// The approved program of the semester running now. The student app cannot
// read the catalog's active semester (admin-only); attendance answers with
// the semester whose attendance period holds, so that one decides. While it
// loads the screen waits rather than flashing another semester's program; if
// it fails, currentProgram falls back to the latest approved semester.
export const useCurrentProgram = () => {
  const programs = useMyEnrollments(undefined, 'approved');
  const attendance = useMyAttendance();
  return {
    program: currentProgram(programs.data?.programs ?? [], attendance.data?.semester),
    isLoading: programs.isLoading || attendance.isLoading,
    isError: programs.isError,
    isRefetching: programs.isRefetching,
    refetch: () => Promise.all([programs.refetch(), attendance.refetch()]),
  };
};
