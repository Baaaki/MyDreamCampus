import type { AdminSessionItem, SessionRecordsResponse } from '@/lib/types';

const generateMockSessions = (): AdminSessionItem[] => {
  const sessions: AdminSessionItem[] = [];
  const today = new Date();
  const year = today.getFullYear();
  const month = today.getMonth();

  // Helper to generate a date string for the current month
  const getDateStr = (day: number) => {
    return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
  };
  
  // Helper to generate a full date-time string
  const getDateTimeStr = (day: number, hour: number, isEnd = false) => {
    const d = new Date(year, month, day, hour + (isEnd ? 2 : 0), 0, 0);
    // Correct local offset
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
    return d.toISOString();
  };

  sessions.push({
    session_id: 'sess-mock-1',
    course_id: 'bil-1-ata1001',
    course_code: 'ATA 1001',
    course_name: 'Atatürk İlkeleri ve İnkılap Tarihi I',
    instructor_id: 'inst-1',
    semester: '2023-2024 Güz',
    week_number: 5,
    session_type: 'theory',
    session_date: getDateStr(3),
    is_active: false,
    started_at: getDateTimeStr(3, 9),
    expires_at: getDateTimeStr(3, 9, true),
    present_count: 45,
    enrolled_count: 50,
  });

  sessions.push({
    session_id: 'sess-mock-2',
    course_id: 'bil-2-bil1012',
    course_code: 'BİL 1012',
    course_name: 'Bilgisayar Bilimlerine Giriş II',
    instructor_id: 'inst-1',
    semester: '2023-2024 Güz',
    week_number: 6,
    session_type: 'theory',
    session_date: getDateStr(3),
    is_active: false,
    started_at: getDateTimeStr(3, 13),
    expires_at: getDateTimeStr(3, 13, true),
    present_count: 30,
    enrolled_count: 40,
  });

  sessions.push({
    session_id: 'sess-mock-3',
    course_id: 'bil-2-mat1010',
    course_code: 'MAT 1010',
    course_name: 'Matematik II',
    instructor_id: 'inst-2',
    semester: '2023-2024 Güz',
    week_number: 6,
    session_type: 'theory',
    session_date: getDateStr(today.getDate()),
    is_active: true,
    started_at: getDateTimeStr(today.getDate(), 9),
    expires_at: getDateTimeStr(today.getDate(), 9, true),
    present_count: 35,
    enrolled_count: 50,
  });

  sessions.push({
    session_id: 'sess-mock-4',
    course_id: 'bil-2-bil1012',
    course_code: 'BİL 1012',
    course_name: 'Bilgisayar Bilimlerine Giriş II',
    instructor_id: 'inst-1',
    semester: '2023-2024 Güz',
    week_number: 6,
    session_type: 'lab',
    session_date: getDateStr(today.getDate()),
    is_active: true,
    started_at: getDateTimeStr(today.getDate(), 13),
    expires_at: getDateTimeStr(today.getDate(), 13, true),
    present_count: 38,
    enrolled_count: 40,
  });

  // Adding some random sessions for yesterday
  const yesterdayDay = Math.max(1, today.getDate() - 1);
  sessions.push({
    session_id: 'sess-mock-5',
    course_id: 'bil-3-bil2011',
    course_code: 'BİL 2011',
    course_name: 'Algoritmalar ve Veri Yapıları',
    instructor_id: 'inst-3',
    semester: '2023-2024 Güz',
    week_number: 6,
    session_type: 'theory',
    session_date: getDateStr(yesterdayDay),
    is_active: false,
    started_at: getDateTimeStr(yesterdayDay, 10),
    expires_at: getDateTimeStr(yesterdayDay, 10, true),
    present_count: 22,
    enrolled_count: 45,
  });

  return sessions;
};

export const mockAdminSessionsResponse = {
  sessions: generateMockSessions(),
  total: 5,
};

const sessionRecordsCache: Record<string, SessionRecordsResponse> = {};

export const generateMockSessionRecords = (sessionId: string): SessionRecordsResponse => {
  if (sessionRecordsCache[sessionId]) {
    return sessionRecordsCache[sessionId];
  }

  const data: SessionRecordsResponse = {
    session_id: sessionId,
    week_number: 6,
    total_count: 50,
    present_count: 2,
    records: [
      {
        id: 'rec-1',
        student_id: 'std-1',
        student_number: '20230001',
        student_name: 'Ahmet Yılmaz',
        is_present: true,
        marked_via: 'qr',
        marked_at: new Date().toISOString(),
      },
      {
        id: 'rec-2',
        student_id: 'std-2',
        student_number: '20230002',
        student_name: 'Ayşe Kaya',
        is_present: true,
        marked_via: 'manual',
        marked_at: new Date().toISOString(),
        note: 'Geç kaldı',
      },
      {
        id: 'rec-3',
        student_id: 'std-3',
        student_number: '20230003',
        student_name: 'Mehmet Demir',
        is_present: false,
        marked_via: '',
      },
      {
        id: 'rec-4',
        student_id: 'std-4',
        student_number: '20230004',
        student_name: 'Fatma Çelik',
        is_present: false,
        marked_via: '',
      },
      {
        id: 'rec-5',
        student_id: 'std-5',
        student_number: '20230005',
        student_name: 'Ali Şahin',
        is_present: false,
        marked_via: '',
      }
    ]
  };

  sessionRecordsCache[sessionId] = data;
  return data;
};

export const markMockStudentPresent = (sessionId: string, studentId: string) => {
  const session = sessionRecordsCache[sessionId];
  if (session) {
    const student = session.records.find(r => r.student_id === studentId);
    if (student && !student.is_present) {
      student.is_present = true;
      student.marked_via = 'manual';
      student.marked_at = new Date().toISOString();
      session.present_count += 1;
    }
  }
};
