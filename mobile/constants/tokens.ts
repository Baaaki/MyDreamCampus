/**
 * Design Tokens — Figma Variables ile birebir eşleşir.
 *
 * Figma'da "Local Variables" panelinde aynı isimlerle tanımlanır.
 * Böylece tasarımcı spacing/md kullandığında geliştirici tokens.spacing.md kullanır.
 */

export const spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
} as const;

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  full: 9999,
} as const;

export const fontSize = {
  xs: 11,
  sm: 13,
  md: 15,
  lg: 17,
  xl: 20,
  xxl: 24,
  xxxl: 32,
} as const;

export const fontWeight = {
  regular: '400' as const,
  medium: '500' as const,
  semibold: '600' as const,
  bold: '700' as const,
};

/**
 * Semantic spacing — Belirli UI bölgeleri için.
 * Figma'da Auto Layout padding değerlerine karşılık gelir.
 */
export const layout = {
  screenPadding: spacing.md,
  cardPadding: spacing.md,
  cardGap: spacing.sm,
  sectionGap: spacing.lg,
  listItemGap: spacing.sm,
} as const;

/**
 * Semantic Colors — Tema-bağımsız sabit renkler.
 *
 * Paper theme primary/secondary/error gibi MD3 renkleri tema üzerinden gelir.
 * Aşağıdakiler tema dışında kalan, iş mantığına bağlı semantic renklerdir.
 * Figma'da "Semantic Colors" koleksiyonuna karşılık gelir.
 */
export const semanticColors = {
  success: '#16a34a',
  successLight: '#dcfce7',
  successDark: '#15803d',

  warning: '#f59e0b',
  warningLight: '#fef3c7',
  warningDark: '#d97706',

  danger: '#ef4444',
  dangerLight: '#fee2e2',
  dangerDark: '#dc2626',

  info: '#6366f1',
  infoLight: '#e0e7ff',
  infoDark: '#4f46e5',
} as const;

/**
 * Grade Colors — Harf notu renk haritası.
 * Grades, Courses ve Attendance ekranlarında kullanılır.
 */
export const gradeColors: Record<string, string> = {
  AA: '#16a34a',
  BA: '#22c55e',
  AB: '#22c55e',
  BB: '#84cc16',
  CB: '#eab308',
  CC: '#f59e0b',
  DC: '#f97316',
  DD: '#ef4444',
  FF: '#dc2626',
};

/**
 * Quick Action renkleri — Home ekranında kullanılır.
 * Figma'da her action kartının kendi accent rengi vardır.
 */
export const accentColors = {
  indigo: '#6366f1',
  violet: '#8b5cf6',
  pink: '#ec4899',
  amber: '#f59e0b',
  teal: '#14b8a6',
  red: '#ef4444',
} as const;

/**
 * Year Colors — Akademik yıl accordion renkleri.
 */
export const yearColors = [
  accentColors.indigo,
  accentColors.violet,
  accentColors.pink,
  accentColors.teal,
] as const;

/**
 * Yardımcı fonksiyonlar — Renk hesaplamaları.
 */
export function getGradeColor(grade: string, fallback = '#9ca3af'): string {
  return gradeColors[grade] ?? fallback;
}

export function getAttendanceColor(pct: number): string {
  if (pct >= 80) return semanticColors.success;
  if (pct >= 60) return semanticColors.warning;
  return semanticColors.danger;
}

export function withOpacity(color: string, opacity: number): string {
  const hex = Math.round(opacity * 255)
    .toString(16)
    .padStart(2, '0');
  return `${color}${hex}`;
}
