import Ionicons from '@expo/vector-icons/Ionicons';
import React, { useMemo } from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ScreenHeader } from '@/components/ScreenHeader';
import { Badge, Button, Card, Text } from '@/components/ui';
import { DAYS_OF_WEEK, SESSION_TYPE_LABEL, slotRange, timeToMinutes } from '@/constants/schedule';
import { useTheme } from '@/contexts/ThemeContext';
import { useMyEnrollments } from '@/hooks/useEnrollment';
import { COLORS } from '@/lib/theme';
import type { EnrollmentCourse } from '@/types/enrollment.types';

type DaySlot = {
  course: EnrollmentCourse;
  type: 'theory' | 'lab';
  start: string;
  end: string;
  startMin: number;
};

export default function ScheduleScreen() {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const { data, isLoading, isError, refetch, isRefetching } = useMyEnrollments(undefined, 'approved');

  const program = data?.programs?.[0];
  const courses = useMemo(() => program?.courses ?? [], [program]);

  // Gunlere gore oturumlar (1=Pazartesi..7=Pazar). Seed'de slot bos olabilir;
  // o durumda gunluk program bos gorunur ama kayitli dersler yine listelenir.
  const byDay = useMemo(() => {
    const map: Record<number, DaySlot[]> = {};
    for (const course of courses) {
      for (const session of course.schedule_sessions ?? []) {
        const range = slotRange(session.slot_numbers ?? []);
        if (!range) continue;
        const day = session.day_of_week;
        (map[day] ??= []).push({
          course,
          type: session.session_type,
          start: range.start,
          end: range.end,
          startMin: timeToMinutes(range.start),
        });
      }
    }
    for (const day of Object.keys(map)) {
      map[Number(day)].sort((a, b) => a.startMin - b.startMin);
    }
    return map;
  }, [courses]);

  const hasAnySlot = Object.keys(byDay).length > 0;

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScreenHeader title="Ders Programı" subtitle={program?.semester} />

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center gap-4 px-8">
          <Ionicons name="cloud-offline-outline" size={44} color={colors.mutedForeground} />
          <Text className="text-center text-base text-muted-foreground">
            Ders programın yüklenemedi. Bağlantını kontrol edip tekrar dene.
          </Text>
          <Button variant="outline" onPress={() => refetch()} accessibilityLabel="Tekrar dene">
            <Text>Tekrar Dene</Text>
          </Button>
        </View>
      ) : (
        <ScrollView
          contentContainerClassName="px-5 pb-10 pt-1"
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={() => refetch()}
              tintColor={colors.primary}
            />
          }
        >
          {/* Haftalik program */}
          <Animated.View entering={FadeInDown.duration(400)} className="mb-6">
            <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Haftalık program
            </Text>
            {hasAnySlot ? (
              <View className="gap-4">
                {[1, 2, 3, 4, 5, 6, 7]
                  .filter((day) => byDay[day]?.length)
                  .map((day) => (
                    <View key={day}>
                      <Text className="mb-2 text-sm font-bold text-foreground">
                        {DAYS_OF_WEEK[day]}
                      </Text>
                      <View className="gap-2">
                        {byDay[day].map((slot, i) => (
                          <Card
                            key={`${slot.course.id}-${slot.type}-${i}`}
                            className="flex-row items-center gap-3 p-3"
                          >
                            <View className="items-center">
                              <Text className="font-mono text-sm font-bold text-foreground">
                                {slot.start}
                              </Text>
                              <Text className="font-mono text-xs text-muted-foreground">
                                {slot.end}
                              </Text>
                            </View>
                            <View className="h-9 w-0.5 rounded-full bg-primary" />
                            <View className="flex-1">
                              <Text className="font-semibold text-foreground" numberOfLines={1}>
                                {slot.course.course_name}
                              </Text>
                              <Text className="font-mono text-xs text-muted-foreground">
                                {slot.course.course_code} · {SESSION_TYPE_LABEL[slot.type]}
                              </Text>
                            </View>
                          </Card>
                        ))}
                      </View>
                    </View>
                  ))}
              </View>
            ) : (
              <Card className="items-center p-6">
                <Ionicons name="time-outline" size={32} color={colors.mutedForeground} />
                <Text className="mt-2 text-center text-sm text-muted-foreground">
                  Derslerin için saat bilgisi henüz tanımlanmamış.
                </Text>
              </Card>
            )}
          </Animated.View>

          {/* Kayitli dersler */}
          <Animated.View entering={FadeInDown.delay(100).duration(400)}>
            <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Kayıtlı derslerim ({courses.length})
            </Text>
            {courses.map((course) => (
              <Card key={course.id} className="mb-3 flex-row items-center justify-between p-4">
                <View className="flex-1 pr-3">
                  <Text className="text-base font-bold text-card-foreground" numberOfLines={2}>
                    {course.course_name}
                  </Text>
                  <Text className="font-mono text-xs text-muted-foreground">
                    {course.course_code}
                    {course.instructor ? ` · ${course.instructor}` : ''}
                  </Text>
                </View>
                <Badge variant="secondary">
                  <Text className="text-xs font-semibold text-secondary-foreground">
                    {course.credits} kredi
                  </Text>
                </Badge>
              </Card>
            ))}
          </Animated.View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
