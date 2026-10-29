import { currentProgram } from './enrollment';
import type { EnrollmentProgramResponse } from '@/types/enrollment.types';

function program(semester: string, status: EnrollmentProgramResponse['status']): EnrollmentProgramResponse {
  return {
    id: `p-${semester}-${status}`,
    student_id: 's-1',
    semester,
    status,
    courses: [],
    created_at: '2026-10-01T00:00:00Z',
  };
}

describe('currentProgram', () => {
  it("keeps the running semester's program when the next one is approved", () => {
    // The backend's order: newest submission first.
    const programs = [program('2026-2027-Spring', 'approved'), program('2026-2027-Fall', 'approved')];

    expect(currentProgram(programs, '2026-2027-Fall')?.semester).toBe('2026-2027-Fall');
  });

  it('falls back to the latest approved semester', () => {
    const programs = [program('2025-2026-Spring', 'approved'), program('2026-2027-Fall', 'approved')];

    expect(currentProgram(programs)?.semester).toBe('2026-2027-Fall');
    expect(currentProgram(programs, '2027-2028-Fall')?.semester).toBe('2026-2027-Fall');
  });

  it('skips programs still waiting on the advisor', () => {
    const programs = [program('2026-2027-Fall', 'pending')];

    expect(currentProgram(programs, '2026-2027-Fall')).toBeNull();
  });
});
