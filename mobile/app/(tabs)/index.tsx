import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter, type Href } from 'expo-router';
import React from 'react';
import { ActivityIndicator, Pressable, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { QuoteRotator } from '@/components/QuoteRotator';
import { Text } from '@/components/ui';
import { useAuthContext } from '@/contexts/AuthContext';
import { useTheme } from '@/contexts/ThemeContext';
import { useMyEnrollments } from '@/hooks/useEnrollment';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';

type IconName = React.ComponentProps<typeof Ionicons>['name'];

type Action = {
  key: string;
  label: string;
  hint: string;
  icon: IconName;
  href: Href;
  tint: string;
  tintBg: string;
};

function greetingByHour(hour: number): string {
  if (hour < 12) return 'Günaydın';
  if (hour < 18) return 'Merhaba';
  return 'İyi akşamlar';
}

function ActionTile({ action, index }: { action: Action; index: number }) {
  const router = useRouter();
  const haptic = useHaptic();
  return (
    <Animated.View entering={FadeInDown.delay(120 + index * 70).duration(400)} className="w-[48%]">
      <Pressable
        onPress={() => {
          haptic.light();
          router.push(action.href);
        }}
        accessibilityRole="button"
        accessibilityLabel={action.label}
        className="mb-4 rounded-3xl border border-border bg-card p-4 active:opacity-80"
      >
        <View
          className="mb-8 h-12 w-12 items-center justify-center rounded-2xl"
          style={{ backgroundColor: action.tintBg }}
        >
          <Ionicons name={action.icon} size={24} color={action.tint} />
        </View>
        <Text className="text-base font-bold text-card-foreground">{action.label}</Text>
        <Text className="mt-0.5 text-xs text-muted-foreground">{action.hint}</Text>
      </Pressable>
    </Animated.View>
  );
}

export default function HomeScreen() {
  const { user } = useAuthContext();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const enrollmentsQuery = useMyEnrollments(undefined, 'approved');
  const program = enrollmentsQuery.data?.programs?.[0];
  const hasProgram = (program?.courses?.length ?? 0) > 0;

  const firstName =
    program?.student_name?.split(/\s+/)[0] ?? user?.email?.split('@')[0] ?? 'öğrenci';

  // Ikon tile renkleri mavi ailesinden + yemekhane icin sicak bir aksan;
  // zemin beyaz/mavi kalirken kartlar Getir'deki gibi ayirt edilebilir olur.
  const actions: Action[] = [
    {
      key: 'scan',
      label: 'Yoklama Al',
      hint: 'QR okut, kendini yazdır',
      icon: 'qr-code',
      href: '/(tabs)/scan',
      tint: colors.primary,
      tintBg: isDark ? 'rgba(59,130,246,0.18)' : 'rgba(37,99,235,0.10)',
    },
    {
      key: 'grades',
      label: 'Notlarım',
      hint: 'Sınav sonuçların',
      icon: 'ribbon',
      href: '/grades',
      tint: isDark ? '#818cf8' : '#4f46e5',
      tintBg: isDark ? 'rgba(129,140,248,0.18)' : 'rgba(79,70,229,0.10)',
    },
    {
      key: 'cafeteria',
      label: 'Yemekhane',
      hint: 'Menü ve randevu',
      icon: 'restaurant',
      href: '/cafeteria',
      tint: isDark ? '#fbbf24' : '#d97706',
      tintBg: isDark ? 'rgba(251,191,36,0.16)' : 'rgba(217,119,6,0.10)',
    },
  ];

  if (hasProgram) {
    actions.push({
      key: 'schedule',
      label: 'Ders Programı',
      hint: 'Haftalık derslerin',
      icon: 'calendar',
      href: '/schedule',
      tint: isDark ? '#2dd4bf' : '#0d9488',
      tintBg: isDark ? 'rgba(45,212,191,0.16)' : 'rgba(13,148,136,0.10)',
    });
  }

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScrollView
        contentContainerClassName="px-5 pb-10 pt-4"
        refreshControl={
          <RefreshControl
            refreshing={enrollmentsQuery.isRefetching}
            onRefresh={() => enrollmentsQuery.refetch()}
            tintColor={colors.primary}
          />
        }
      >
        <Animated.View entering={FadeInDown.duration(400)}>
          <QuoteRotator />
        </Animated.View>

        <Animated.View entering={FadeInDown.delay(80).duration(400)} className="mb-5 mt-7">
          <Text className="text-2xl font-extrabold text-foreground">
            {greetingByHour(new Date().getHours())}, {firstName}!
          </Text>
          <Text className="mt-1 text-sm text-muted-foreground">Bugün ne yapmak istersin?</Text>
        </Animated.View>

        {enrollmentsQuery.isLoading ? (
          <View className="items-center py-10">
            <ActivityIndicator color={colors.primary} />
          </View>
        ) : (
          <View className="flex-row flex-wrap justify-between">
            {actions.map((action, i) => (
              <ActionTile key={action.key} action={action} index={i} />
            ))}
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
