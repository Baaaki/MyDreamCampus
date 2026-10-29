import Ionicons from '@expo/vector-icons/Ionicons';
import React from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ScreenHeader } from '@/components/ScreenHeader';
import { Badge, Button, Card, Text } from '@/components/ui';
import { formatDateLongTR } from '@/constants/datetime';
import { useTheme } from '@/contexts/ThemeContext';
import { useMyAttendance } from '@/hooks/useAttendance';
import { sessionRows, type SessionRow } from '@/lib/attendance';
import { semesterLabel } from '@/lib/semester';
import { COLORS } from '@/lib/theme';
import type { AttendanceStats, CourseAttendance, WeeklyRecord } from '@/types/attendance.types';

const SESSION_LABEL = { theory: 'Teori', lab: 'Lab' } as const;

// Shown only when someone other than the QR marked the student present.
const MARKED_BY_LABEL: Partial<Record<WeeklyRecord['marked_via'], string>> = {
  manual: 'hoca işaretledi',
  admin: 'yönetici işaretledi',
};

function StatsRow({ label, stats }: { label: string; stats: AttendanceStats }) {
  return (
    <View className="flex-row items-center justify-between py-1.5">
      <View>
        <Text className="text-sm font-semibold text-card-foreground">{label}</Text>
        <Text className="font-mono text-xs text-muted-foreground">
          {stats.present_count} katıldı · {stats.absent_count} devamsız · en az {stats.min_required}{' '}
          / {stats.total_sessions}
        </Text>
      </View>
      <Badge variant={stats.passed ? 'success' : 'destructive'}>
        <Text className="text-xs font-semibold">{stats.passed ? 'Yeterli' : 'Yetersiz'}</Text>
      </Badge>
    </View>
  );
}

function SessionRowItem({ row }: { row: SessionRow }) {
  const markedBy = row.attended ? MARKED_BY_LABEL[row.marked_via] : undefined;
  return (
    <View className="flex-row items-center justify-between py-1.5">
      <View className="flex-1 pr-3">
        <Text className="text-sm text-foreground">
          {row.week}. hafta · {SESSION_LABEL[row.session_type] ?? row.session_type}
        </Text>
        <Text className="font-mono text-xs text-muted-foreground">
          {formatDateLongTR(row.date)}
          {markedBy ? ` · ${markedBy}` : ''}
        </Text>
      </View>
      <Badge variant={row.attended ? 'success' : 'destructive'}>
        <Text className="text-xs font-semibold">{row.attended ? 'Katıldın' : 'Katılmadın'}</Text>
      </Badge>
    </View>
  );
}

function CourseCard({ course }: { course: CourseAttendance }) {
  const rows = sessionRows(course);
  return (
    <Card className="mb-3 p-4">
      <Text className="text-base font-bold text-card-foreground" numberOfLines={2}>
        {course.course_name}
      </Text>
      <Text className="font-mono text-xs text-muted-foreground">
        {course.course_code}
        {course.instructor ? ` · ${course.instructor}` : ''}
      </Text>

      <View className="mt-2 border-t border-border pt-1">
        {course.theory && <StatsRow label="Teori" stats={course.theory} />}
        {course.lab && <StatsRow label="Lab" stats={course.lab} />}
      </View>

      <View className="mt-2 border-t border-border pt-2">
        <Text className="mb-1 font-mono text-xs uppercase tracking-widest text-muted-foreground">
          Oturumlar
        </Text>
        {rows.length > 0 ? (
          rows.map((row) => <SessionRowItem key={`${row.week}-${row.session_type}`} row={row} />)
        ) : (
          <Text className="py-1 text-sm text-muted-foreground">
            Bu derste henüz yoklama alınmadı.
          </Text>
        )}
      </View>
    </Card>
  );
}

export default function AttendanceScreen() {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const { data, isLoading, isError, refetch, isRefetching } = useMyAttendance();

  const courses = data?.courses ?? [];

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScreenHeader
        title="Devamsızlık"
        subtitle={data?.semester ? semesterLabel(data.semester) : undefined}
      />

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center gap-4 px-8">
          <Ionicons name="cloud-offline-outline" size={44} color={colors.mutedForeground} />
          <Text className="text-center text-base text-muted-foreground">
            Yoklama bilgilerin yüklenemedi. Bağlantını kontrol edip tekrar dene.
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
          {courses.length > 0 ? (
            courses.map((course, index) => (
              <Animated.View
                key={course.course_id}
                entering={FadeInDown.delay(index * 60).duration(400)}
              >
                <CourseCard course={course} />
              </Animated.View>
            ))
          ) : (
            <View className="items-center py-16">
              <Ionicons name="calendar-clear-outline" size={44} color={colors.mutedForeground} />
              <Text className="mt-3 text-center text-muted-foreground">
                Bu dönem için yoklama bilgisi yok.
              </Text>
            </View>
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
