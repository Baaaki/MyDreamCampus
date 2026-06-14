import FontAwesome from '@expo/vector-icons/FontAwesome';
import { DarkTheme, DefaultTheme, ThemeProvider as NavThemeProvider } from '@react-navigation/native';
import { useFonts } from 'expo-font';
import { Stack, useRouter, useSegments } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { useEffect } from 'react';
import 'react-native-reanimated';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PaperProvider } from 'react-native-paper';

import { useColorScheme } from '@/components/useColorScheme';
import { ThemeProvider } from '@/contexts/ThemeContext';
import { AuthProvider, useAuthContext } from '@/contexts/AuthContext';
import { ToastProvider } from '@/contexts/ToastContext';
import { LightTheme, DarkTheme as PaperDarkTheme } from '@/constants/theme';
import { ErrorBoundary as AppErrorBoundary } from '@/components/ui';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export {
  ErrorBoundary,
} from 'expo-router';

export const unstable_settings = {
  initialRouteName: '(tabs)',
};

SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  const [loaded, error] = useFonts({
    SpaceMono: require('../assets/fonts/SpaceMono-Regular.ttf'),
    ...FontAwesome.font,
  });

  useEffect(() => {
    if (error) throw error;
  }, [error]);

  useEffect(() => {
    if (loaded) {
      SplashScreen.hideAsync();
    }
  }, [loaded]);

  if (!loaded) {
    return null;
  }

  return (
    <ThemeProvider>
      <RootLayoutNav />
    </ThemeProvider>
  );
}

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuthContext();
  const segments = useSegments();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;

    const inAuthGroup = segments[0] === '(auth)';

    if (!isAuthenticated && !inAuthGroup) {
      router.replace('/(auth)/login');
    } else if (isAuthenticated && inAuthGroup) {
      router.replace('/(tabs)');
    }
  }, [isAuthenticated, isLoading, segments]);

  return <>{children}</>;
}

function RootLayoutNav() {
  const colorScheme = useColorScheme();
  const paperTheme = colorScheme === 'dark' ? PaperDarkTheme : LightTheme;

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <PaperProvider theme={paperTheme}>
          <NavThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
            <ToastProvider>
              <AppErrorBoundary>
                <AuthGuard>
                  <Stack>
                    <Stack.Screen name="(auth)" options={{ headerShown: false }} />
                    <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
                    <Stack.Screen name="modal" options={{ presentation: 'modal' }} />
                    <Stack.Screen name="screens/qr-attendance" options={{ title: 'QR Yoklama' }} />
                    <Stack.Screen name="screens/my-grades" options={{ title: 'Notlarim' }} />
                    <Stack.Screen name="screens/cafeteria" options={{ title: 'Yemekhane' }} />
                    <Stack.Screen name="screens/change-password" options={{ title: 'Sifre Degistir' }} />
                    <Stack.Screen name="screens/enrollment" options={{ title: 'Ders Kaydi' }} />
                  </Stack>
                </AuthGuard>
              </AppErrorBoundary>
            </ToastProvider>
          </NavThemeProvider>
        </PaperProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}
