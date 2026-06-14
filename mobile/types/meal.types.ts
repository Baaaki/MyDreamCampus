// Cafeteria
export interface Cafeteria {
  id: string;
  name: string;
  location: string;
  has_vegan_menu: boolean;
  serves_dinner: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CafeteriaListResponse {
  cafeterias: Cafeteria[];
}

// Menu
export interface MonthlyMenuResponse {
  success?: boolean;
  data: {
    year: number;
    month: number;
    menu_data: any;
    created_at: string;
    updated_at: string;
  };
}

// Reservations
export interface CafeteriaInfo {
  id: string;
  name: string;
  location: string;
}

export interface Reservation {
  id: string;
  date: string;
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan';
  cafeteria_name: string;
  cafeteria?: CafeteriaInfo;
  status: string;
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

export interface CreateReservationRequest {
  cafeteria_id: string;
  date: string;
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan';
}

export interface BatchReservationRequest {
  reservations: CreateReservationRequest[];
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

export interface UseReservationRequest {
  qr_payload: string;
}

export interface UseReservationResponse {
  message: string;
  reservation_id: string;
  cafeteria_name: string;
  meal_time: 'lunch' | 'dinner';
  menu_type: 'normal' | 'vegan';
}
