import api from './api';
import type { ConfirmPaymentRequest, Payment } from '@/types/payment.types';

export const paymentService = {
  // Reddedilen kart hata degildir: odeme status "failed" ve bir sebeple doner.
  async confirm(paymentId: string, data: ConfirmPaymentRequest): Promise<Payment> {
    const response = await api.post<Payment>(`/payments/${paymentId}/confirm`, data);
    return response.data;
  },

  async get(paymentId: string): Promise<Payment> {
    const response = await api.get<Payment>(`/payments/${paymentId}`);
    return response.data;
  },
};

export default paymentService;
