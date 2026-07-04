import api from './api';
import type {
  CafeteriaListData,
  CreateReservationData,
  CreateReservationRequest,
  MonthlyMenuData,
  MyReservationsData,
} from '@/types/meal.types';

// meal endpoint'leri yaniti {success, data} zarfiyla dondurur. Diger
// modullerden (grades/attendance) farkli; burada zarfi tek yerde aciyoruz.
type Envelope<T> = { success: boolean; data: T };

export const mealService = {
  async getCafeterias(): Promise<CafeteriaListData> {
    const response = await api.get<Envelope<CafeteriaListData>>('/meals/cafeterias');
    return response.data.data;
  },

  async getMonthlyMenu(year: number, month: number): Promise<MonthlyMenuData> {
    const response = await api.get<Envelope<MonthlyMenuData>>('/meals/menu/monthly', {
      params: { year, month },
    });
    return response.data.data;
  },

  async getMyReservations(): Promise<MyReservationsData> {
    const response = await api.get<Envelope<MyReservationsData>>('/meals/reservations/my');
    return response.data.data;
  },

  async createReservation(data: CreateReservationRequest): Promise<CreateReservationData> {
    const response = await api.post<Envelope<CreateReservationData>>('/meals/reservations', data);
    return response.data.data;
  },

  async cancelReservation(reservationId: string): Promise<void> {
    await api.delete(`/meals/reservations/${reservationId}`);
  },
};

export default mealService;
