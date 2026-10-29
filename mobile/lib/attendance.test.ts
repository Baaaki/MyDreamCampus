import { sessionRows } from './attendance';
import type { CourseAttendance } from '@/types/attendance.types';

function course(partial: Partial<CourseAttendance>): CourseAttendance {
  return {
    course_id: 'c-1',
    course_code: 'CENG301',
    course_name: 'İşletim Sistemleri',
    instructor: 'Ahmet Yılmaz',
    total_weeks: 14,
    weekly_records: [],
    missed_sessions: [],
    ...partial,
  };
}

describe('sessionRows', () => {
  it('puts the missed weeks between the attended ones', () => {
    const rows = sessionRows(
      course({
        weekly_records: [
          { week: 3, session_type: 'theory', date: '2026-10-14', marked_via: 'qr_scan' },
          { week: 1, session_type: 'theory', date: '2026-09-30', marked_via: 'manual' },
        ],
        missed_sessions: [{ week: 2, session_type: 'theory', date: '2026-10-07' }],
      })
    );

    expect(rows.map((r) => [r.week, r.attended])).toEqual([
      [1, true],
      [2, false],
      [3, true],
    ]);
  });

  it('orders the two session types of a week the same way every time', () => {
    const rows = sessionRows(
      course({
        weekly_records: [{ week: 1, session_type: 'theory', date: '2026-09-30', marked_via: 'qr_scan' }],
        missed_sessions: [{ week: 1, session_type: 'lab', date: '2026-10-01' }],
      })
    );

    expect(rows.map((r) => r.session_type)).toEqual(['lab', 'theory']);
  });

  it('is empty for a course that has held no session yet', () => {
    expect(sessionRows(course({}))).toEqual([]);
  });
});
