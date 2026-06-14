import { accentColors, semanticColors } from '@/constants/tokens';

// ─── Types ───────────────────────────────────────────────────

export type MealType = 'none' | 'normal' | 'vegan';
export type Tab = 'select' | 'menu' | 'history';

// ─── Constants ───────────────────────────────────────────────

export const MEAL_PRICE = 25;

export const weekDays = [
  { key: 'monday', label: 'Pazartesi', short: 'Pzt', dow: 1 },
  { key: 'tuesday', label: 'Sali', short: 'Sal', dow: 2 },
  { key: 'wednesday', label: 'Carsamba', short: 'Car', dow: 3 },
  { key: 'thursday', label: 'Persembe', short: 'Per', dow: 4 },
  { key: 'friday', label: 'Cuma', short: 'Cum', dow: 5 },
];

export const monthNames = ['Oca', 'Sub', 'Mar', 'Nis', 'May', 'Haz', 'Tem', 'Agu', 'Eyl', 'Eki', 'Kas', 'Ara'];
export const fullDayNames = ['Pazar', 'Pazartesi', 'Sali', 'Carsamba', 'Persembe', 'Cuma', 'Cumartesi'];
export const shortDayNames = ['Paz', 'Pzt', 'Sal', 'Car', 'Per', 'Cum', 'Cmt'];
export const menuCategories = ['Corba', 'Ana Yemek', 'Yan Yemek', 'Tatli', 'Diger'];

export const categoryColors: Record<string, string> = {
  Corba: accentColors.amber,
  'Ana Yemek': accentColors.red,
  'Yan Yemek': semanticColors.warning,
  Tatli: accentColors.violet,
  Diger: '#9ca3af',
};

export const dayKeyMap: Record<number, string> = {
  1: 'monday', 2: 'tuesday', 3: 'wednesday', 4: 'thursday', 5: 'friday',
};

export const mealColors = {
  normal: '#f97316',
  normalLight: '#fff7ed',
  normalDark: '#ea580c',
  vegan: '#22c55e',
  veganLight: '#dcfce7',
  veganDark: '#16a34a',
  pay: '#10b981',
} as const;

export const fallbackWeeklyMenu: Record<string, { normal: string[]; vegan: string[] }> = {
  monday: { normal: ['Mercimek Corbasi', 'Etli Nohut', 'Pirinc Pilavi', 'Ayran'], vegan: ['Mercimek Corbasi', 'Zeytinyagli Fasulye', 'Bulgur Pilavi', 'Ayran'] },
  tuesday: { normal: ['Ezogelin Corbasi', 'Tavuk Sote', 'Makarna', 'Cacik'], vegan: ['Ezogelin Corbasi', 'Sebzeli Guvec', 'Makarna', 'Cacik'] },
  wednesday: { normal: ['Domates Corbasi', 'Kofte', 'Patates Puresi', 'Salata'], vegan: ['Domates Corbasi', 'Mercimek Koftesi', 'Patates Puresi', 'Salata'] },
  thursday: { normal: ['Yayla Corbasi', 'Kuru Fasulye', 'Pirinc Pilavi', 'Tursu'], vegan: ['Yayla Corbasi', 'Barbunya Pilaki', 'Bulgur Pilavi', 'Tursu'] },
  friday: { normal: ['Tarhana Corbasi', 'Balik', 'Patates Kizartmasi', 'Salata'], vegan: ['Tarhana Corbasi', 'Ispanakli Borek', 'Patates Firin', 'Salata'] },
};

// ─── Helpers ─────────────────────────────────────────────────

export function getWeekDates() {
  const today = new Date();
  const dow = today.getDay();
  const monday = new Date(today);
  monday.setDate(today.getDate() - (dow === 0 ? 6 : dow - 1));
  return weekDays.map((d, i) => {
    const dt = new Date(monday);
    dt.setDate(monday.getDate() + i);
    return { ...d, date: dt, day: dt.getDate(), month: dt.getMonth(), fullDate: formatDate(dt) };
  });
}

export function formatDate(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function formatDateTR(dateStr: string) {
  const d = new Date(dateStr);
  return `${d.getDate()} ${monthNames[d.getMonth()]} ${d.getFullYear()}`;
}

export function getMonthDays(year: number, month: number) {
  const count = new Date(year, month + 1, 0).getDate();
  return Array.from({ length: count }, (_, i) => new Date(year, month, i + 1));
}

export function getWeekOfMonth(date: Date): number {
  const first = new Date(date.getFullYear(), date.getMonth(), 1);
  const firstDow = first.getDay() === 0 ? 6 : first.getDay() - 1;
  return Math.floor((date.getDate() + firstDow - 1) / 7);
}
