import type { Theme } from '@react-navigation/native';
import { DarkTheme, DefaultTheme } from '@react-navigation/native';

// global.css'teki CSS degiskenlerinin JS karsiligi. className alamayan
// prop'lar icin (navigation temasi, ikon rengi, ActivityIndicator vs.)
// buradan okunur — iki kaynak birbiriyle senkron tutulmali.
export const COLORS = {
  light: {
    background: 'hsl(30, 40%, 97%)',
    foreground: 'hsl(24, 22%, 11%)',
    card: 'hsl(0, 0%, 100%)',
    primary: 'hsl(16, 88%, 54%)',
    primaryForeground: 'hsl(20, 100%, 98%)',
    mutedForeground: 'hsl(25, 10%, 42%)',
    border: 'hsl(28, 22%, 86%)',
    success: 'hsl(158, 74%, 32%)',
    destructive: 'hsl(0, 74%, 48%)',
  },
  dark: {
    background: 'hsl(24, 16%, 8%)',
    foreground: 'hsl(30, 28%, 93%)',
    card: 'hsl(24, 14%, 12%)',
    primary: 'hsl(16, 95%, 60%)',
    primaryForeground: 'hsl(20, 60%, 8%)',
    mutedForeground: 'hsl(28, 12%, 62%)',
    border: 'hsl(25, 10%, 19%)',
    success: 'hsl(158, 60%, 44%)',
    destructive: 'hsl(0, 78%, 58%)',
  },
} as const;

export const NAV_THEME: Record<'light' | 'dark', Theme> = {
  light: {
    ...DefaultTheme,
    colors: {
      ...DefaultTheme.colors,
      background: COLORS.light.background,
      card: COLORS.light.card,
      text: COLORS.light.foreground,
      primary: COLORS.light.primary,
      border: COLORS.light.border,
    },
  },
  dark: {
    ...DarkTheme,
    colors: {
      ...DarkTheme.colors,
      background: COLORS.dark.background,
      card: COLORS.dark.card,
      text: COLORS.dark.foreground,
      primary: COLORS.dark.primary,
      border: COLORS.dark.border,
    },
  },
};
