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

// Verilen tarihin ait oldugu haftanin Pazartesi'si (yerel, 00:00).
function mondayOf(d: Date): Date {
  const x = new Date(d);
  x.setHours(0, 0, 0, 0);
  const daysSinceMonday = (x.getDay() + 6) % 7; // getDay: Pazar=0
  x.setDate(x.getDate() - daysSinceMonday);
  return x;
}

// Randevular her zaman GELECEK hafta icin alinir: bir sonraki haftanin
// Pazartesi-Cuma gunleri.
export function nextWeekWeekdays(): Date[] {
  const nextMonday = mondayOf(new Date());
  nextMonday.setDate(nextMonday.getDate() + 7);
  return Array.from({ length: 5 }, (_, i) => {
    const d = new Date(nextMonday);
    d.setDate(nextMonday.getDate() + i);
    return d;
  });
}

// Bir randevunun iptal kilidi: rezervasyon haftasindan ONCEKI cuma 23:59:59.
// O ana kadar iptal serbest; sonrasinda hafta sabitlenir (backend ile ayni kural).
export function cancelDeadline(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number);
  const monday = mondayOf(new Date(y, m - 1, d));
  const friday = new Date(monday);
  friday.setDate(monday.getDate() - 3); // onceki haftanin cumasi
  friday.setHours(23, 59, 59, 0);
  return friday;
}

// Randevu su an iptal edilebilir mi? (cuma kilidinden once mi)
export function isCancellable(iso: string, now: Date = new Date()): boolean {
  return now <= cancelDeadline(iso);
}
