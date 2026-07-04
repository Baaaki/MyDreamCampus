import React, { createContext, useContext } from 'react';
import { useColorScheme } from 'nativewind';

// NativeWind'in global colorScheme state'i uzerine ince bir sarmalayici:
// ekranlar tema durumunu tek API'den (useTheme) okur, dark class'larini
// NativeWind kendisi uygular.
interface ThemeContextType {
  theme: 'light' | 'dark';
  isDark: boolean;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType>({
  theme: 'light',
  isDark: false,
  toggleTheme: () => {},
});

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const { colorScheme, toggleColorScheme } = useColorScheme();
  const theme = colorScheme === 'dark' ? 'dark' : 'light';

  return (
    <ThemeContext.Provider
      value={{ theme, isDark: theme === 'dark', toggleTheme: toggleColorScheme }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

export const useTheme = () => useContext(ThemeContext);
