// Backend: monolith meal module (/meals/*). API yanitlari {success, data}
// zarfiyla gelir; mealService zarfi acar, buradaki tipler ic veriyi tanimlar.

export interface Cafeteria {
  id: string;
  name: string;
  location: string;
  has_vegan_menu: boolean;
  serves_dinner: boolean;
  is_active: boolean;
}

export interface CafeteriaListData {
  cafeterias: Cafeteria[];
}

// menu_data: { "2026-07-06": { lunch: string[], dinner: string[] }, ... }
export type DailyMenu = { lunch?: string[]; dinner?: string[] };

export interface MonthlyMenuData {
  year: number;
  month: number;
  menu_data: Record<string, DailyMenu> | null;
}

export interface CafeteriaInfo {
  id: string;
  name: string;
  location: string;
}

export type MealTime = 'lunch' | 'dinner';
export type MenuType = 'normal' | 'vegan';

export interface Reservation {
  id: string;
  date: string;
  meal_time: MealTime;
  menu_type: MenuType;
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

export interface MyReservationsData {
  reservations: Reservation[] | null;
  summary: ReservationSummary;
}

export interface CreateReservationRequest {
  cafeteria_id: string;
  date: string;
  meal_time: MealTime;
  menu_type: MenuType;
}

export interface CreateReservationData {
  reservation_id: string;
  payment_url: string;
  amount: number;
  currency: string;
  reservation: Reservation;
}

// Batch: bir kombinasyon (secili gunler x secili ogunler) tek istekte alinir.
export interface BatchReservationItem {
  cafeteria_id: string;
  date: string; // YYYY-MM-DD
  meal_time: MealTime;
  menu_type: MenuType;
}

export interface BatchReservationRequest {
  reservations: BatchReservationItem[];
}

export interface CreateBatchReservationData {
  batch_id: string;
  payment_url: string;
  total_amount: number;
  currency: string;
  reservations: Reservation[];
}
