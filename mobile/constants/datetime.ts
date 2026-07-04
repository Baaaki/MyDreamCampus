// Turkce tarih yardimcilari — cihaz locale'ine bagli kalmadan tutarli cikti.

export const WEEKDAYS_SHORT_TR = ['Paz', 'Pzt', 'Sal', 'Çar', 'Per', 'Cum', 'Cmt'];
export const MONTHS_TR = [
  'Ocak', 'Şubat', 'Mart', 'Nisan', 'Mayıs', 'Haziran',
  'Temmuz', 'Ağustos', 'Eylül', 'Ekim', 'Kasım', 'Aralık',
];

// "YYYY-MM-DD" (yerel saat, UTC kaymasi olmadan).
export function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

// Bugunden itibaren `count` gunluk liste (randevu tarih secimi icin).
export function nextDays(count: number): Date[] {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return Array.from({ length: count }, (_, i) => {
    const d = new Date(today);
    d.setDate(today.getDate() + i);
    return d;
  });
}

// "YYYY-MM-DD" -> "6 Temmuz Pazar" gibi okunur metin.
export function formatDateLongTR(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number);
  if (!y || !m || !d) return iso;
  const date = new Date(y, m - 1, d);
  return `${d} ${MONTHS_TR[m - 1]} ${WEEKDAYS_FULL_TR[date.getDay()]}`;
}

export const WEEKDAYS_FULL_TR = [
  'Pazar', 'Pazartesi', 'Salı', 'Çarşamba', 'Perşembe', 'Cuma', 'Cumartesi',
];
