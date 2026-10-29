import type { CourseAttendance, MissedSession, WeeklyRecord } from '@/types/attendance.types';

export type SessionRow = (WeeklyRecord & { attended: true }) | (MissedSession & { attended: false });

// Attended and missed sessions in one list, by week and then session type —
// the order the web attendance page shows them in.
export function sessionRows(course: CourseAttendance): SessionRow[] {
  return [
    ...(course.weekly_records ?? []).map((r) => ({ ...r, attended: true as const })),
    ...(course.missed_sessions ?? []).map((m) => ({ ...m, attended: false as const })),
  ].sort((a, b) => a.week - b.week || a.session_type.localeCompare(b.session_type));
}
