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
  AuditLogEntry,
  AuditLogListResponse,
} from '@/lib/types';

// Service definitions with their API paths
// Uses "safe" API clients that don't auto-redirect on 401 — this page calls
// 4 services in parallel and a single 401 from any service would clear tokens
// before Promise.allSettled can catch it.
export type ServiceKey = 'grades' | 'enrollment' | 'meal' | 'catalog' | 'auth' | 'attendance' | 'student' | 'staff';

interface ServiceConfig {
  label: string;
  timePath: string;
  api: typeof gradesApiSafe;
}

const SERVICES: Record<ServiceKey, ServiceConfig> = {
  grades: {
    label: 'Notlar',
    timePath: 'admin/time',
    api: gradesApiSafe,
  },
  enrollment: {
    label: 'Kayıt',
    timePath: 'admin/time',
    api: enrollmentApiSafe,
  },
  meal: {
    label: 'Yemekhane',
    timePath: 'time',
    api: mealApiSafe,
  },
  catalog: {
    label: 'Ders Kataloğu',
    timePath: 'admin/time',
    api: catalogApiSafe,
  },
  auth: {
    label: 'Kimlik Doğrulama',
    timePath: 'admin/time',
    api: authApiSafe,
  },
  attendance: {
    label: 'Yoklama',
    timePath: 'admin/time',
    api: attendanceApiSafe,
  },
  student: {
    label: 'Öğrenci',
    timePath: 'admin/time',
    api: studentApiSafe,
  },
  staff: {
    label: 'Personel',
    timePath: 'admin/time',
    api: staffApiSafe,
  },
};

export const SERVICE_KEYS: ServiceKey[] = ['grades', 'enrollment', 'meal', 'catalog', 'auth', 'attendance', 'student', 'staff'];

export function getServiceLabel(key: ServiceKey): string {
  return SERVICES[key].label;
}

// ============================================================================
// TIME MACHINE
// ============================================================================

export async function getAllTimeStatuses(): Promise<ServiceTimeStatus[]> {
  const results = await Promise.allSettled(
    SERVICE_KEYS.map(async (key) => {
      const cfg = SERVICES[key];
      const status = await cfg.api.get(`${cfg.timePath}/status`).json<TimeStatus>();
      return { service: key, label: cfg.label, status, error: null };
    })
  );

  return results.map((result, i) => {
    if (result.status === 'fulfilled') {
      return result.value;
    }
    return {
      service: SERVICE_KEYS[i],
      label: SERVICES[SERVICE_KEYS[i]].label,
      status: null,
      error: 'Bağlantı hatası',
    };
  });
}

export async function simulateTimeAll(time: string): Promise<{ success: string[]; failed: string[] }> {
  const results = await Promise.allSettled(
    SERVICE_KEYS.map(async (key) => {
      const cfg = SERVICES[key];
      await cfg.api.post(`${cfg.timePath}/simulate`, { json: { time } }).json();
      return key;
    })
  );

  const success: string[] = [];
  const failed: string[] = [];

  results.forEach((result, i) => {
    if (result.status === 'fulfilled') {
      success.push(SERVICES[SERVICE_KEYS[i]].label);
    } else {
      failed.push(SERVICES[SERVICE_KEYS[i]].label);
    }
  });

  return { success, failed };
}

export async function resetTimeAll(): Promise<{ success: string[]; failed: string[] }> {
  const results = await Promise.allSettled(
    SERVICE_KEYS.map(async (key) => {
      const cfg = SERVICES[key];
      await cfg.api.post(`${cfg.timePath}/reset`).json();
      return key;
    })
  );

  const success: string[] = [];
  const failed: string[] = [];

  results.forEach((result, i) => {
    if (result.status === 'fulfilled') {
      success.push(SERVICES[SERVICE_KEYS[i]].label);
    } else {
      failed.push(SERVICES[SERVICE_KEYS[i]].label);
    }
  });

  return { success, failed };
}

// ============================================================================
// ACADEMIC PERIODS — Grades service (with course_id)
// ============================================================================

const GRADES_PERIOD_PATH = 'admin/periods';

export async function listGradesPeriods(semester?: string): Promise<AcademicPeriod[]> {
  const searchParams: Record<string, string> = {};
  if (semester) searchParams.semester = semester;
  return gradesApiSafe.get(GRADES_PERIOD_PATH, { searchParams }).json<AcademicPeriod[]>();
}

export async function createGradesPeriod(data: CreatePeriodRequest): Promise<AcademicPeriod> {
  return gradesApiSafe.post(GRADES_PERIOD_PATH, { json: data }).json<AcademicPeriod>();
}

export async function updateGradesPeriod(id: string, data: UpdatePeriodRequest): Promise<AcademicPeriod> {
  return gradesApiSafe.put(`${GRADES_PERIOD_PATH}/${id}`, { json: data }).json<AcademicPeriod>();
}

export async function deleteGradesPeriod(id: string): Promise<void> {
  await gradesApiSafe.delete(`${GRADES_PERIOD_PATH}/${id}`).json();
}

// ============================================================================
// ACADEMIC PERIODS — Catalog & Enrollment (no course_id)
// ============================================================================

export type SimplePeriodServiceKey = 'enrollment' | 'catalog';

const SIMPLE_PERIOD_PATHS: Record<SimplePeriodServiceKey, { api: typeof gradesApiSafe; path: string }> = {
  enrollment: { api: enrollmentApiSafe, path: 'admin/periods' },
  catalog: { api: catalogApiSafe, path: 'admin/periods' },
};

export async function listSimplePeriods(service: SimplePeriodServiceKey, semester?: string): Promise<SimplePeriod[]> {
  const cfg = SIMPLE_PERIOD_PATHS[service];
  const searchParams: Record<string, string> = {};
  if (semester) searchParams.semester = semester;
  return cfg.api.get(cfg.path, { searchParams }).json<SimplePeriod[]>();
}

export async function createSimplePeriod(service: SimplePeriodServiceKey, data: SimpleCreatePeriodRequest): Promise<SimplePeriod> {
  const cfg = SIMPLE_PERIOD_PATHS[service];
  return cfg.api.post(cfg.path, { json: data }).json<SimplePeriod>();
}

export async function updateSimplePeriod(service: SimplePeriodServiceKey, id: string, data: UpdatePeriodRequest): Promise<SimplePeriod> {
  const cfg = SIMPLE_PERIOD_PATHS[service];
  return cfg.api.put(`${cfg.path}/${id}`, { json: data }).json<SimplePeriod>();
}

export async function deleteSimplePeriod(service: SimplePeriodServiceKey, id: string): Promise<void> {
  const cfg = SIMPLE_PERIOD_PATHS[service];
  await cfg.api.delete(`${cfg.path}/${id}`).json();
}

// ============================================================================
// CLOSED DAYS — Meal service (holidays)
// ============================================================================

const CLOSED_DAYS_PATH = 'closed-days';

export async function listClosedDays(from?: string, to?: string): Promise<ClosedDay[]> {
  const searchParams: Record<string, string> = {};
  if (from) searchParams.from = from;
  if (to) searchParams.to = to;
  return mealApiSafe.get(CLOSED_DAYS_PATH, { searchParams }).json<ClosedDay[]>();
}

export async function createClosedDay(data: CreateClosedDayRequest): Promise<ClosedDay> {
  return mealApiSafe.post(CLOSED_DAYS_PATH, { json: data }).json<ClosedDay>();
}

export async function deleteClosedDay(id: string): Promise<void> {
  await mealApiSafe.delete(`${CLOSED_DAYS_PATH}/${id}`).json();
}

// ============================================================================
// SEMESTERS — Catalog service (Zero Trust State Machine)
// ============================================================================

const SEMESTERS_PATH = 'admin/semesters';

export async function listSemesters(): Promise<Semester[]> {
  return catalogApiSafe.get(SEMESTERS_PATH).json<Semester[]>();
}

export async function createSemester(data: CreateSemesterRequest): Promise<Semester> {
  return catalogApiSafe.post(SEMESTERS_PATH, { json: data }).json<Semester>();
}

export async function getActiveSemester(): Promise<Semester | null> {
  try {
    return await catalogApiSafe.get(`${SEMESTERS_PATH}/active`).json<Semester>();
  } catch {
    return null;
  }
}

export async function activateSemester(id: string): Promise<Semester> {
  return catalogApiSafe.put(`${SEMESTERS_PATH}/${id}/activate`).json<Semester>();
}

export async function completeSemester(id: string): Promise<Semester> {
  return catalogApiSafe.put(`${SEMESTERS_PATH}/${id}/complete`).json<Semester>();
}

// ============================================================================
// AUDIT LOG — Catalog service (immutable log)
// ============================================================================

const AUDIT_LOG_PATH = 'admin/audit-log';

export interface AuditLogFilters {
  service?: string;
  action?: string;
  actor_id?: string;
  limit?: number;
  offset?: number;
}

export async function listAuditLog(filters: AuditLogFilters = {}): Promise<AuditLogListResponse> {
  const searchParams: Record<string, string> = {};
  if (filters.service) searchParams.service = filters.service;
  if (filters.action) searchParams.action = filters.action;
  if (filters.actor_id) searchParams.actor_id = filters.actor_id;
  if (filters.limit) searchParams.limit = String(filters.limit);
  if (filters.offset) searchParams.offset = String(filters.offset);
  return catalogApiSafe.get(AUDIT_LOG_PATH, { searchParams }).json<AuditLogListResponse>();
}
