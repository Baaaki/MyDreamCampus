// Ders programi sabitleri — frontend/src/lib/constants.ts ile ayni kaynak
// (backend slot semasi). Degisirse ikisini birden guncelle.

import type { DayOfWeek } from '@/types/enrollment.types';

export const TIME_SLOTS: Record<number, { label: string; start: string; end: string }> = {
  1: { label: 'Ders 1', start: '08:30', end: '09:15' },
  2: { label: 'Ders 2', start: '09:25', end: '10:10' },
  3: { label: 'Ders 3', start: '10:20', end: '11:05' },
  4: { label: 'Ders 4', start: '11:15', end: '12:00' },
  5: { label: 'Ogle Arasi', start: '12:10', end: '12:55' },
  6: { label: 'Ders 5', start: '13:00', end: '13:45' },
  7: { label: 'Ders 6', start: '13:55', end: '14:40' },
  8: { label: 'Ders 7', start: '14:50', end: '15:35' },
  9: { label: 'Ders 8', start: '15:45', end: '16:30' },
};

// Backend day_of_week names in week order, and their labels.
export const WEEK_DAYS: readonly DayOfWeek[] = [
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
  'sunday',
];

export const DAYS_OF_WEEK: Record<DayOfWeek, string> = {
  monday: 'Pazartesi',
  tuesday: 'Salı',
  wednesday: 'Çarşamba',
  thursday: 'Perşembe',
  friday: 'Cuma',
  saturday: 'Cumartesi',
  sunday: 'Pazar',
};

export const SESSION_TYPE_LABEL: Record<'theory' | 'lab', string> = {
  theory: 'Teori',
  lab: 'Lab',
};

// Ardisik slot listesini tek zaman araligina cevirir: [3,4] -> "10:20 – 12:00"
export function slotRange(slotNumbers: number[]): { start: string; end: string } | null {
  if (slotNumbers.length === 0) return null;
  const sorted = [...slotNumbers].sort((a, b) => a - b);
  const first = TIME_SLOTS[sorted[0]];
  const last = TIME_SLOTS[sorted[sorted.length - 1]];
  if (!first || !last) return null;
  return { start: first.start, end: last.end };
}

// "HH:MM" -> gunun dakikasi; ders siralamasi ve "siradaki ders" secimi icin.
export function timeToMinutes(time: string): number {
  const [h, m] = time.split(':').map(Number);
  return h * 60 + m;
}
