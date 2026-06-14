import { mealApi } from '../api-client';

// Types
export interface Reservation {
  id: string;
  date: string;
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan';
  cafeteria_name: string;
  cafeteria?: {
    id: string;
    name: string;
    location: string;
  };
  status: 'pending' | 'confirmed' | 'cancelled' | 'expired';
  is_used: boolean;
  created_at: string;
}

export interface ReservationSummary {
  total: number;
  confirmed: number;
  pending: number;
  used: number;
  cancelled: number;
}

export interface PaginationInfo {
  page: number;
  limit: number;
  total_items: number;
  total_pages: number;
}

export interface MyReservationsResponse {
  reservations: Reservation[];
  summary: ReservationSummary;
  pagination?: PaginationInfo;
}

export interface GetMyReservationsParams {
  from_date?: string;
  to_date?: string;
  status?: string;
  page?: number;
  limit?: number;
}

export interface Cafeteria {
  id: string;
  name: string;
  location: string;
  is_active: boolean;
}

export interface CafeteriaListResponse {
  cafeterias: Cafeteria[];
}

export interface CreateReservationRequest {
  cafeteria_id: string;
  date: string; // YYYY-MM-DD
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan';
}

export interface BatchReservationRequest {
  reservations: CreateReservationRequest[];
}

export interface CreateReservationResponse {
  reservation_id: string;
  payment_url: string;
  amount: number;
  currency: string;
  expires_at: string;
  reservation: Reservation;
}

export interface CreateBatchReservationResponse {
  batch_id: string;
  payment_url: string;
  total_amount: number;
  currency: string;
  expires_at: string;
  reservations: Reservation[];
}

export interface CancelReservationResponse {
  reservation_id: string;
  refund_amount: number;
  currency: string;
  refund_status: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// API Functions
export async function getMyReservations(params?: GetMyReservationsParams): Promise<MyReservationsResponse> {
  const searchParams = new URLSearchParams();
  if (params?.from_date) searchParams.set('from_date', params.from_date);
  if (params?.to_date) searchParams.set('to_date', params.to_date);
  if (params?.status) searchParams.set('status', params.status);
  if (params?.page) searchParams.set('page', params.page.toString());
  if (params?.limit) searchParams.set('limit', params.limit.toString());

  const queryString = searchParams.toString();
  const url = queryString ? `reservations/my?${queryString}` : 'reservations/my';

  const response = await mealApi
    .get(url)
    .json<ApiResponse<MyReservationsResponse>>();
  return response.data;
}

export async function getCafeterias(): Promise<CafeteriaListResponse> {
  const response = await mealApi
    .get('cafeterias')
    .json<ApiResponse<CafeteriaListResponse>>();
  return response.data;
}

export async function createReservation(
  request: CreateReservationRequest
): Promise<CreateReservationResponse> {
  const response = await mealApi
    .post('reservations', { json: request })
    .json<ApiResponse<CreateReservationResponse>>();
  return response.data;
}

export async function createBatchReservation(
  request: BatchReservationRequest
): Promise<CreateBatchReservationResponse> {
  const response = await mealApi
    .post('reservations/batch', { json: request })
    .json<ApiResponse<CreateBatchReservationResponse>>();
  return response.data;
}

export async function cancelReservation(
  reservationId: string
): Promise<CancelReservationResponse> {
  const response = await mealApi
    .delete(`reservations/${reservationId}`)
    .json<ApiResponse<CancelReservationResponse>>();
  return response.data;
}

// Helper function to get the start of the current week (Monday)
export function getStartOfCurrentWeek(): Date {
  const now = new Date();
  const dayOfWeek = now.getDay();
  const diff = dayOfWeek === 0 ? -6 : 1 - dayOfWeek; // Monday = start of week
  const monday = new Date(now);
  monday.setDate(now.getDate() + diff);
  monday.setHours(0, 0, 0, 0);
  return monday;
}

// Helper function to get end of previous week (Sunday before current week)
export function getEndOfPreviousWeek(): Date {
  const weekStart = getStartOfCurrentWeek();
  const sunday = new Date(weekStart);
  sunday.setDate(weekStart.getDate() - 1);
  sunday.setHours(23, 59, 59, 999);
  return sunday;
}

// Helper function to format date as YYYY-MM-DD
export function formatDateForApi(date: Date): string {
  return date.toISOString().split('T')[0];
}

// Helper functions for frontend filtering (week-based)
export function filterActiveReservations(reservations: Reservation[]): Reservation[] {
  const weekStart = formatDateForApi(getStartOfCurrentWeek());
  return reservations.filter(
    (r) => r.status === 'confirmed' && !r.is_used && r.date >= weekStart
  );
}

export function filterPastReservations(reservations: Reservation[]): Reservation[] {
  const weekStart = formatDateForApi(getStartOfCurrentWeek());
  return reservations.filter(
    (r) => r.is_used || r.date < weekStart || r.status === 'cancelled' || r.status === 'expired'
  );
}

// Map backend status to frontend display status
export function getDisplayStatus(reservation: Reservation): 'upcoming' | 'completed' | 'missed' | 'cancelled' {
  const weekStart = formatDateForApi(getStartOfCurrentWeek());

  if (reservation.status === 'cancelled') {
    return 'cancelled';
  }

  if (reservation.is_used) {
    return 'completed';
  }

  if (reservation.status === 'confirmed' && reservation.date >= weekStart) {
    return 'upcoming';
  }

  // Past date, confirmed but not used
  if (reservation.status === 'confirmed' && reservation.date < weekStart) {
    return 'missed';
  }

  // Pending or expired
  return 'cancelled';
}

