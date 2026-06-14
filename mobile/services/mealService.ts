import api from './api';
import type {
  CafeteriaListResponse,
  MonthlyMenuResponse,
  MyReservationsResponse,
  GetMyReservationsParams,
  BatchReservationRequest,
  CreateBatchReservationResponse,
  CancelReservationResponse,
  UseReservationRequest,
  UseReservationResponse,
} from '@/types/meal.types';

export const mealService = {
  async getCafeterias(): Promise<CafeteriaListResponse> {
    const response = await api.get<CafeteriaListResponse>('/meals/cafeterias');
    return response.data;
  },

  async getMonthlyMenu(year: number, month: number): Promise<MonthlyMenuResponse> {
    const response = await api.get<MonthlyMenuResponse>('/meals/menu/monthly', {
      params: { year, month },
    });
    return response.data;
  },

  async getMyReservations(params?: GetMyReservationsParams): Promise<MyReservationsResponse> {
    const response = await api.get<MyReservationsResponse>('/meals/reservations/my', { params });
    return response.data;
  },

  async createBatchReservation(data: BatchReservationRequest): Promise<CreateBatchReservationResponse> {
    const response = await api.post<CreateBatchReservationResponse>('/meals/reservations/batch', data);
    return response.data;
  },

  async cancelReservation(reservationId: string): Promise<CancelReservationResponse> {
    const response = await api.delete<CancelReservationResponse>(`/meals/reservations/${reservationId}`);
    return response.data;
  },

  async useReservation(data: UseReservationRequest): Promise<UseReservationResponse> {
    const response = await api.post<UseReservationResponse>('/meals/reservations/use', data);
    return response.data;
  },
};

export default mealService;
