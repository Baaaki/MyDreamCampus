// Ders programi sabitleri — frontend/src/lib/constants.ts ile ayni kaynak
// (backend slot semasi). Degisirse ikisini birden guncelle.

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

export const DAYS_OF_WEEK: Record<number, string> = {
  1: 'Pazartesi',
  2: 'Sali',
  3: 'Carsamba',
  4: 'Persembe',
  5: 'Cuma',
  6: 'Cumartesi',
  7: 'Pazar',
};

export const SESSION_TYPE_LABEL: Record<'theory' | 'lab', string> = {
  theory: 'Teori',
  lab: 'Lab',
};

// JS getDay(): 0=Pazar..6=Cumartesi -> backend day_of_week: 1=Pazartesi..7=Pazar
export function jsDayToBackendDay(jsDay: number): number {
  return jsDay === 0 ? 7 : jsDay;
}

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
