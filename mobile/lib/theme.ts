import type { Theme } from '@react-navigation/native';
import { DarkTheme, DefaultTheme } from '@react-navigation/native';

// global.css'teki CSS degiskenlerinin JS karsiligi. className alamayan
// prop'lar icin (navigation temasi, ikon rengi, ActivityIndicator vs.)
// buradan okunur — iki kaynak birbiriyle senkron tutulmali.
export const COLORS = {
  light: {
    background: 'hsl(210, 40%, 98%)',
    foreground: 'hsl(222, 47%, 11%)',
    card: 'hsl(0, 0%, 100%)',
    primary: 'hsl(221, 83%, 53%)',
    primaryForeground: 'hsl(210, 40%, 99%)',
    accent: 'hsl(199, 89%, 48%)',
    mutedForeground: 'hsl(215, 16%, 47%)',
    border: 'hsl(214, 32%, 90%)',
    success: 'hsl(158, 70%, 36%)',
    warning: 'hsl(33, 92%, 48%)',
    destructive: 'hsl(0, 74%, 51%)',
  },
  dark: {
    background: 'hsl(222, 47%, 7%)',
    foreground: 'hsl(210, 40%, 96%)',
    card: 'hsl(222, 40%, 12%)',
    primary: 'hsl(217, 91%, 60%)',
    primaryForeground: 'hsl(222, 47%, 8%)',
    accent: 'hsl(199, 89%, 55%)',
    mutedForeground: 'hsl(215, 20%, 65%)',
    border: 'hsl(222, 25%, 20%)',
    success: 'hsl(158, 62%, 46%)',
    warning: 'hsl(33, 92%, 56%)',
    destructive: 'hsl(0, 74%, 58%)',
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
