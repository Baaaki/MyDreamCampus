// Backend: payment service (/payments/*). Yanitlar meal'in aksine zarfsiz
// duz JSON. Kart numarasi ve CVC hic geri donmez; yalniz marka ve son 4 hane.

export type PaymentStatus = 'pending' | 'completed' | 'failed' | 'expired' | 'refunded';

export interface Payment {
  id: string;
  status: PaymentStatus;
  amount: number;
  currency: string;
  card_brand: string | null;
  card_last4: string | null;
  failure_reason: string | null;
  expires_at: string;
  completed_at: string | null;
}

export interface ConfirmPaymentRequest {
  card_number: string;
  exp_month: number;
  exp_year: number;
  cvc: string;
  cardholder_name: string;
}
