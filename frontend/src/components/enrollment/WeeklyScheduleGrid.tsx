import type { AvailableCourse } from '@/lib/types';
import { TIME_SLOTS } from '@/lib/constants';

interface WeeklyScheduleGridProps {
  sessions: AvailableCourse[];
  selectedCourseIds: string[];
  courseColorMap: Record<string, number>;
  onRemoveCourse: (courseId: string) => void;
  onEnroll?: () => void;
  readOnly?: boolean;
  submitButtonText?: string;
  showSubmitButton?: boolean;
}

// Days of week for the grid
const DAYS_OF_WEEK = [
  { key: 1, label: 'Pazartesi' },
  { key: 2, label: 'Salı' },
  { key: 3, label: 'Çarşamba' },
  { key: 4, label: 'Perşembe' },
  { key: 5, label: 'Cuma' },
];

// Color palette for different courses
const COURSE_COLORS = [
  { bg: 'bg-blue-500', hover: 'hover:bg-blue-600', text: 'text-white' },
  { bg: 'bg-green-500', hover: 'hover:bg-green-600', text: 'text-white' },
  { bg: 'bg-purple-500', hover: 'hover:bg-purple-600', text: 'text-white' },
  { bg: 'bg-orange-500', hover: 'hover:bg-orange-600', text: 'text-white' },
  { bg: 'bg-pink-500', hover: 'hover:bg-pink-600', text: 'text-white' },
  { bg: 'bg-indigo-500', hover: 'hover:bg-indigo-600', text: 'text-white' },
  { bg: 'bg-red-500', hover: 'hover:bg-red-600', text: 'text-white' },
  { bg: 'bg-teal-500', hover: 'hover:bg-teal-600', text: 'text-white' },
  { bg: 'bg-yellow-500', hover: 'hover:bg-yellow-600', text: 'text-gray-900' },
  { bg: 'bg-cyan-500', hover: 'hover:bg-cyan-600', text: 'text-white' },
  { bg: 'bg-lime-500', hover: 'hover:bg-lime-600', text: 'text-gray-900' },
  { bg: 'bg-rose-500', hover: 'hover:bg-rose-600', text: 'text-white' },
];

interface Conflict {
  day: number;
  dayLabel: string;
  slot: number;
  timeRange: string;
  courses: string[];
}

export default function WeeklyScheduleGrid({ sessions, selectedCourseIds, courseColorMap, onRemoveCourse, onEnroll, readOnly = false, submitButtonText = "Ders Kaydını Tamamla", showSubmitButton = true }: WeeklyScheduleGridProps) {
  // Day string to key mapping
  const dayMapping: Record<string, number> = {
    'monday': 1,
    'tuesday': 2,
    'wednesday': 3,
    'thursday': 4,
    'friday': 5,
    'saturday': 6,
    'sunday': 7
  };

  // Create a Map: "1-1" (day-slot) -> array of courses (to detect conflicts)
  const slotSessionsMap = new Map<string, AvailableCourse[]>();
  sessions.forEach(course => {
    course.schedule_sessions.forEach(scheduleItem => {
      const dayKey = dayMapping[scheduleItem.day_of_week.toLowerCase()];
      if (!dayKey) return;

      scheduleItem.slot_numbers.forEach(slotNum => {
        const key = `${dayKey}-${slotNum}`;
        const existing = slotSessionsMap.get(key) || [];
        // Avoid adding same course multiple times to same slot if backend data has duplicates (shouldn't happen but safe)
        if (!existing.some(c => c.id === course.id)) {
           slotSessionsMap.set(key, [...existing, course]);
        }
      });
    });
  });

  // Find all conflicts
  const conflicts: Conflict[] = [];
  slotSessionsMap.forEach((courseList, key) => {
    if (courseList.length > 1) {
      const [day, slot] = key.split('-').map(Number);
      const dayInfo = DAYS_OF_WEEK.find(d => d.key === day);
      const slotInfo = TIME_SLOTS[slot as keyof typeof TIME_SLOTS];
      conflicts.push({
        day,
        dayLabel: dayInfo?.label || String(day),
        slot,
        timeRange: `${slotInfo.label} (${slotInfo.time})`,
        courses: courseList.map(c => c.course_code)
      });
    }
  });

  // Get color for a course based on its assigned color index
  const getCourseColor = (courseId: string) => {
    const colorIndex = courseColorMap[courseId] || 0;
    return COURSE_COLORS[colorIndex % COURSE_COLORS.length];
  };

  const totalSessions = sessions.reduce((sum, course) => 
    sum + course.schedule_sessions.reduce((s, session) => s + session.slot_numbers.length, 0), 0);

  return (
    <div className="h-full flex flex-col bg-white">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 bg-gray-50">
        <h2 className="text-xl font-bold text-gray-800">Haftalık Program</h2>
        <p className="text-sm text-gray-600 mt-1">
          {selectedCourseIds.length > 0
            ? `${selectedCourseIds.length} ders seçildi • ${totalSessions} oturum`
            : 'Sol taraftan ders seçin'
          }
        </p>
      </div>

      {/* Conflict Warning Box */}
      {conflicts.length > 0 && (
        <div className="mx-4 mt-4 p-4 bg-red-50 border border-red-300 rounded-lg">
          <div className="flex items-start gap-3">
            <svg className="w-6 h-6 text-red-600 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div className="flex-1">
              <h3 className="font-bold text-red-800 text-lg">Ders Çakışması Tespit Edildi!</h3>
              <p className="text-red-700 text-sm mt-1 mb-3">
                Aşağıdaki zaman dilimlerinde birden fazla ders çakışıyor:
              </p>
              <div className="space-y-2">
                {conflicts.map((conflict, index) => (
                  <div
                    key={index}
                    className="flex items-center gap-2 bg-red-100 px-3 py-2 rounded-md"
                  >
                    <span className="font-semibold text-red-900">
                      {conflict.dayLabel}
                    </span>
                    <span className="text-red-700">
                      {conflict.timeRange}
                    </span>
                    <span className="text-red-600">→</span>
                    <span className="font-bold text-red-900">
                      {conflict.courses.join(' & ')}
                    </span>
                  </div>
                ))}
              </div>
              <p className="text-red-600 text-xs mt-3">
                Çakışan derslerden birini kaldırmak için tablodaki hücreye veya soldaki listeden derse tıklayın.
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="flex-1 overflow-auto p-4">
        <div className="min-w-max">
          <table className="w-full border-collapse">
            <thead>
              <tr>
                <th className="sticky top-0 left-0 z-20 bg-gray-100 border border-gray-300 p-3 text-sm font-semibold text-gray-700 min-w-[120px]">
                  Saat
                </th>
                {DAYS_OF_WEEK.map((day) => (
                  <th
                    key={day.key}
                    className="sticky top-0 z-10 bg-gray-100 border border-gray-300 p-3 text-sm font-semibold text-gray-700 min-w-[140px]"
                  >
                    {day.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Object.entries(TIME_SLOTS).map(([slotNum, slotInfo]) => (
                <tr key={slotNum}>
                  <td className="sticky left-0 z-10 bg-gray-50 border border-gray-300 p-3 text-sm font-medium text-gray-700 text-center">
                    <div className="flex flex-col">
                      <span className="text-sm font-semibold text-gray-700">{slotInfo.label}</span>
                      <span className="font-mono text-xs text-gray-600">{slotInfo.time}</span>
                    </div>
                  </td>
                  {DAYS_OF_WEEK.map((day) => {
                    const key = `${day.key}-${slotNum}`;
                    const coursesInSlot = slotSessionsMap.get(key) || [];
                    const hasConflict = coursesInSlot.length > 1;

                    if (coursesInSlot.length > 0) {
                      if (hasConflict) {
                        // Conflict cell - show warning
                        return (
                          <td
                            key={key}
                            className="border-2 border-red-500 p-2 bg-red-100 transition-all duration-200"
                          >
                            <div className="flex flex-col items-center justify-center h-12 gap-1">
                              <svg className="w-5 h-5 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                              </svg>
                              <div className="text-center">
                                <div className="font-bold text-xs text-red-800">
                                  {coursesInSlot.map(c => c.course_code).join(' & ')}
                                </div>
                                <div className="text-xs text-red-600 font-semibold">ÇAKIŞMA!</div>
                              </div>
                            </div>
                          </td>
                        );
                      } else {
                        // Single course cell
                        const course = coursesInSlot[0];
                        const color = getCourseColor(course.id);

                        return (
                          <td
                            key={key}
                            onClick={() => !readOnly && onRemoveCourse(course.id)}
                            className={`border border-gray-300 p-2 transition-all duration-200 ${
                              readOnly ? 'cursor-default' : 'cursor-pointer hover:opacity-90'
                            } ${color.bg} ${color.hover}`}
                          >
                            <div className="flex items-center justify-center min-h-[56px]">
                              <div className={`text-center ${color.text}`}>
                                <div className="font-bold text-sm">{course.course_code}</div>
                                {course.instructor && (
                                  <div className="text-xs opacity-80 mt-0.5">{course.instructor}</div>
                                )}
                                {!readOnly && <div className="text-xs opacity-70 mt-1">Kaldırmak için tıkla</div>}
                              </div>
                            </div>
                          </td>
                        );
                      }
                    }

                    return (
                      <td
                        key={key}
                        className="border border-gray-300 p-3 bg-white hover:bg-gray-50 transition-all duration-200"
                      />
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Empty state overlay */}
      {sessions.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-50 bg-opacity-75 pointer-events-none">
          <div className="text-center">
            <svg className="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p className="text-gray-600 font-medium">Sol taraftan bir veya birden fazla ders seçin</p>
            <p className="text-gray-500 text-sm mt-1">Haftalık program burada görünecek</p>
            <p className="text-gray-500 text-sm mt-1">Her ders farklı renkle gösterilecek</p>
          </div>
        </div>
      )}

      {/* Enrollment button - always visible unless explicitly hidden or readOnly without override */}
      {(showSubmitButton && (!readOnly || showSubmitButton)) && (
        <div className="p-4 border-t border-gray-200 bg-white mt-auto">
          <button
            onClick={onEnroll}
            disabled={conflicts.length > 0}
            className={`w-full py-4 px-6 rounded-lg font-bold text-lg transition-all duration-200 ${
              conflicts.length > 0
                ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                : 'bg-indigo-600 text-white hover:bg-indigo-700 active:bg-indigo-800 shadow-lg hover:shadow-xl'
            }`}
          >
            {conflicts.length > 0 ? (
              <span className="flex items-center justify-center gap-2">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                Ders Çakışması Var - Kaydı Tamamlayamazsınız
              </span>
            ) : (
              <span className="flex items-center justify-center gap-2">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {submitButtonText} ({selectedCourseIds.length} Ders)
              </span>
            )}
          </button>
        </div>
      )}
    </div>
  );
}
