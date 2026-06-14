import { gradesApiSafe, enrollmentApiSafe, mealApiSafe, catalogApiSafe, authApiSafe, attendanceApiSafe, studentApiSafe, staffApiSafe } from '@/lib/api-client';
import type {
  TimeStatus,
  ServiceTimeStatus,
  AcademicPeriod,
  SimplePeriod,
  CreatePeriodRequest,
  SimpleCreatePeriodRequest,
  UpdatePeriodRequest,
  ClosedDay,
  CreateClosedDayRequest,
  Semester,
  CreateSemesterRequest,
  UpdateSemesterRequest,
  AuditLogEntry,
  AuditLogListResponse,
} from '@/lib/types';

export type ServiceKey = 'grades' | 'enrollment' | 'meal' | 'catalog' | 'auth' | 'attendance' | 'student' | 'staff';

interface ServiceConfig {
  label: string;
  timePath: string;
  api: typeof gradesApiSafe;
}

const SERVICES: Record<ServiceKey, ServiceConfig> = {
  grades: { label: 'Notlar', timePath: 'admin/time', api: gradesApiSafe },
  enrollment: { label: 'Kayıt', timePath: 'admin/time', api: enrollmentApiSafe },
  meal: { label: 'Yemekhane', timePath: 'time', api: mealApiSafe },
  catalog: { label: 'Ders Kataloğu', timePath: 'admin/time', api: catalogApiSafe },
  auth: { label: 'Kimlik Doğrulama', timePath: 'admin/time', api: authApiSafe },
  attendance: { label: 'Yoklama', timePath: 'admin/time', api: attendanceApiSafe },
  student: { label: 'Öğrenci', timePath: 'admin/time', api: studentApiSafe },
  staff: { label: 'Personel', timePath: 'admin/time', api: staffApiSafe },
};

export const SERVICE_KEYS: ServiceKey[] = ['grades', 'enrollment', 'meal', 'catalog', 'auth', 'attendance', 'student', 'staff'];

export function getServiceLabel(key: ServiceKey): string {
  return SERVICES[key].label;
}

const MOCK_DELAY = () => new Promise(resolve => setTimeout(resolve, 300));

// ============================================================================
// MOCK DATA
// ============================================================================

export async function getAllTimeStatuses(): Promise<ServiceTimeStatus[]> {
  await MOCK_DELAY();
  return SERVICE_KEYS.map(key => ({
    service: key,
    label: SERVICES[key].label,
    status: { current_time: new Date().toISOString() },
    error: null,
  }));
}

export async function simulateTimeAll(time: string): Promise<{ success: string[]; failed: string[] }> {
  await MOCK_DELAY();
  return { success: SERVICE_KEYS.map(k => SERVICES[k].label), failed: [] };
}

export async function resetTimeAll(): Promise<{ success: string[]; failed: string[] }> {
  await MOCK_DELAY();
  return { success: SERVICE_KEYS.map(k => SERVICES[k].label), failed: [] };
}

// Grades Periods
export async function listGradesPeriods(semester?: string): Promise<AcademicPeriod[]> {
  const searchParams: Record<string, string> = {};
  if (semester) searchParams.semester = semester;
  return gradesApiSafe.get('admin/periods', { searchParams }).json<AcademicPeriod[]>();
}
export async function createGradesPeriod(data: CreatePeriodRequest): Promise<AcademicPeriod> {
  return gradesApiSafe.post('admin/periods', { json: data }).json<AcademicPeriod>();
}
export async function updateGradesPeriod(id: string, data: UpdatePeriodRequest): Promise<AcademicPeriod> {
  return gradesApiSafe.put(`admin/periods/${id}`, { json: data }).json<AcademicPeriod>();
}
export async function deleteGradesPeriod(id: string): Promise<void> {
  await gradesApiSafe.delete(`admin/periods/${id}`);
}

// Simple Periods
export type SimplePeriodServiceKey = 'enrollment' | 'catalog' | 'attendance';

const simplePeriodApi: Record<SimplePeriodServiceKey, typeof enrollmentApiSafe> = {
  enrollment: enrollmentApiSafe,
  catalog: catalogApiSafe,
  attendance: attendanceApiSafe,
};

export async function listSimplePeriods(service: SimplePeriodServiceKey, semester?: string): Promise<SimplePeriod[]> {
  const searchParams: Record<string, string> = {};
  if (semester) searchParams.semester = semester;
  return simplePeriodApi[service].get('admin/periods', { searchParams }).json<SimplePeriod[]>();
}
export async function createSimplePeriod(service: SimplePeriodServiceKey, data: SimpleCreatePeriodRequest): Promise<SimplePeriod> {
  return simplePeriodApi[service].post('admin/periods', { json: data }).json<SimplePeriod>();
}
export async function updateSimplePeriod(service: SimplePeriodServiceKey, id: string, data: UpdatePeriodRequest): Promise<SimplePeriod> {
  return simplePeriodApi[service].put(`admin/periods/${id}`, { json: data }).json<SimplePeriod>();
}
export async function deleteSimplePeriod(service: SimplePeriodServiceKey, id: string): Promise<void> {
  await simplePeriodApi[service].delete(`admin/periods/${id}`);
}

// Closed Days
export async function listClosedDays(): Promise<ClosedDay[]> {
  return mealApiSafe.get('admin/closed-days').json<ClosedDay[]>();
}
export async function createClosedDay(data: CreateClosedDayRequest): Promise<ClosedDay> {
  return mealApiSafe.post('admin/closed-days', { json: data }).json<ClosedDay>();
}
export async function deleteClosedDay(id: string): Promise<void> {
  await mealApiSafe.delete(`admin/closed-days/${id}`);
}

// Semesters
export async function listSemesters(): Promise<Semester[]> {
  return catalogApiSafe.get('admin/semesters').json<Semester[]>();
}

export async function createSemester(data: CreateSemesterRequest): Promise<Semester> {
  const body = await catalogApiSafe.post('admin/semesters', { json: data }).json<any>();
  // Backend returns { semester, period_errors } when period distribution partially fails,
  // but returns the Semester directly when everything succeeds.
  if (body.semester) return body.semester as Semester;
  return body as Semester;
}

export async function getActiveSemester(): Promise<Semester | null> {
  try {
    return await catalogApiSafe.get('admin/semesters/active').json<Semester>();
  } catch {
    return null;
  }
}

export async function activateSemester(id: string): Promise<Semester> {
  return catalogApiSafe.put(`admin/semesters/${id}/activate`).json<Semester>();
}

export async function completeSemester(id: string): Promise<Semester> {
  return catalogApiSafe.put(`admin/semesters/${id}/complete`).json<Semester>();
}

export async function deleteSemester(id: string): Promise<void> {
  await catalogApiSafe.delete(`admin/semesters/${id}`);
}

export async function updateSemester(
  id: string,
  data: UpdateSemesterRequest
): Promise<Semester> {
  const body = await catalogApiSafe
    .put(`admin/semesters/${id}`, { json: data })
    .json<any>();
  if (body.semester) return body.semester as Semester;
  return body as Semester;
}

// Audit Log Filters
export interface AuditLogFilters {
  service?: string; action?: string; actor_id?: string; limit?: number; offset?: number;
}
export async function listAuditLog(filters: AuditLogFilters = {}): Promise<AuditLogListResponse> {
  await MOCK_DELAY();
  const entries: AuditLogEntry[] = [
    { id: 'aud1', timestamp: new Date(Date.now() - 3600000).toISOString(), service: 'catalog', action: 'semester.activated', resource_type: 'semester', resource_id: 'sem2', actor_role: 'admin', actor_id: 'admin-123', details: { note: 'Mock data' } },
    { id: 'aud2', timestamp: new Date(Date.now() - 7200000).toISOString(), service: 'grades', action: 'period.created', resource_type: 'period', resource_id: 'gp1', actor_role: 'admin', actor_id: 'admin-123', details: { course_id: 'CS101' } },
    { id: 'aud3', timestamp: new Date(Date.now() - 86400000).toISOString(), service: 'meal', action: 'closed_day.created', resource_type: 'meal', resource_id: 'cd1', actor_role: 'admin', actor_id: 'admin-456', details: { date: '2025-01-01' } },
  ];
  return { entries, total: 3 };
}
