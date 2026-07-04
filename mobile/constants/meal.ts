// Yemekhane sabitleri. Fiyat backend config'i (MEAL_PRICE_TRY) ile ayni tutulur;
// gercek tahsilat batch yanitindaki total_amount'tan gelir, bu deger yalnizca
// odemeden ONCE ozet ekraninda tahmini tutari gostermek icin.
import type { MealTime, MenuType } from '@/types/meal.types';

export const MEAL_PRICE_TRY = 15;

export const MEAL_LABEL: Record<MealTime, string> = { lunch: 'Öğle', dinner: 'Akşam' };
export const MENU_LABEL: Record<MenuType, string> = { normal: 'Normal', vegan: 'Vegan' };

export const STATUS_LABEL: Record<string, string> = {
  confirmed: 'Onaylandı',
  pending: 'Onaylanıyor',
  used: 'Kullanıldı',
  cancelled: 'İptal edildi',
  expired: 'Süresi doldu',
};

export function statusVariant(
  status: string
): 'success' | 'warning' | 'secondary' | 'destructive' {
  if (status === 'confirmed') return 'success';
  if (status === 'pending') return 'warning';
  if (status === 'cancelled' || status === 'expired') return 'destructive';
  return 'secondary';
}

export function formatTRY(amount: number): string {
  return `${amount.toLocaleString('tr-TR')} ₺`;
}
