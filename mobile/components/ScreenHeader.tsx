import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter } from 'expo-router';
import React from 'react';
import { Pressable, View } from 'react-native';

import { Text } from '@/components/ui';
import { useTheme } from '@/contexts/ThemeContext';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';

// Stack detay ekranlari icin tema-tutarli ust cubuk: geri butonu + baslik.
// Root Stack headerShown:false; her detay ekrani kendi basligini bununla cizer.
export function ScreenHeader({ title, subtitle }: { title: string; subtitle?: string }) {
  const router = useRouter();
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  return (
    <View className="flex-row items-center gap-3 px-5 pb-3 pt-2">
      <Pressable
        onPress={() => {
          haptic.light();
          router.back();
        }}
        accessibilityRole="button"
        accessibilityLabel="Geri"
        className="h-10 w-10 items-center justify-center rounded-full bg-secondary active:opacity-70"
      >
        <Ionicons name="chevron-back" size={22} color={colors.foreground} />
      </Pressable>
      <View className="flex-1">
        <Text className="text-xl font-extrabold text-foreground">{title}</Text>
        {subtitle ? (
          <Text className="text-xs text-muted-foreground">{subtitle}</Text>
        ) : null}
      </View>
    </View>
  );
}
