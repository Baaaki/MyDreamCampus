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
  // Bir ogunun fiyati (backend MEAL_PRICE_TRY config'i); yalnizca odemeden
  // ONCE tahmini tutar icin, gercek tahsilat batch yanitindaki total_amount.
  meal_price: number;
  currency: string;
}

// Bir ayin menusu iki bicimde gelebilir (seed ikisini birden yazar):
// - web: { normalMenus: [hafta...], veganMenus: [hafta...] }; i'inci hafta ayi
//   kesen i'inci Pazartesi-Pazar haftasi, her gun { items: [5 kalem], calories }.
//   Ogun ayrimi yok. Web admin menuyu yalnizca bu bicimde kaydeder.
// - eski: { "2026-07-06": { lunch: string[], dinner: string[] }, ... };
//   yalnizca normal menu.
export type DailyMenu = { lunch?: string[]; dinner?: string[] };

export interface MenuDay {
  items: string[];
  calories: number;
}

export type MenuWeekday = 'monday' | 'tuesday' | 'wednesday' | 'thursday' | 'friday';

export type MenuWeek = Partial<Record<MenuWeekday, MenuDay>>;

export interface WebMenuData {
  normalMenus: MenuWeek[];
  veganMenus: MenuWeek[];
}

export type DatedMenuData = Record<string, DailyMenu>;

export type MenuData = WebMenuData | DatedMenuData;

export interface MonthlyMenuData {
  year: number;
  month: number;
  menu_data: MenuData | null;
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
  payment_id: string;
  amount: number;
  currency: string;
  expires_at: string;
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

// payment_id kart formunda onaylanir (/payments/:id/confirm); randevular
// odeme onaylanana kadar `pending` kalir.
export interface CreateBatchReservationData {
  batch_id: string;
  payment_id: string;
  total_amount: number;
  currency: string;
  expires_at: string;
  reservations: Reservation[];
}

export interface UseReservationRequest {
  qr_payload: string;
}

export interface UseReservationData {
  message: string;
  reservation_id: string;
  cafeteria_name: string;
  meal_time: MealTime;
  menu_type: MenuType;
}
