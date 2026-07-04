import Ionicons from '@expo/vector-icons/Ionicons';
import { useRouter } from 'expo-router';
import React, { useMemo } from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Badge, Button, Card, Text } from '@/components/ui';
import {
  DAYS_OF_WEEK,
  SESSION_TYPE_LABEL,
  jsDayToBackendDay,
  slotRange,
  timeToMinutes,
} from '@/constants/schedule';
import { useAuthContext } from '@/contexts/AuthContext';
import { useTheme } from '@/contexts/ThemeContext';
import { useMyAttendance } from '@/hooks/useAttendance';
import { useActiveSemester } from '@/hooks/useCatalog';
import { useMyEnrollments } from '@/hooks/useEnrollment';
import { COLORS } from '@/lib/theme';
import type { EnrollmentCourse, ScheduleSession } from '@/types/enrollment.types';

const MONTHS_TR = [
  'Ocak', 'Subat', 'Mart', 'Nisan', 'Mayis', 'Haziran',
  'Temmuz', 'Agustos', 'Eylul', 'Ekim', 'Kasim', 'Aralik',
];

type TodaySession = {
  course: EnrollmentCourse;
  session: ScheduleSession;
  start: string;
  end: string;
  startMin: number;
  endMin: number;
};

function greetingByHour(hour: number): string {
  if (hour < 12) return 'Gunaydin';
  if (hour < 18) return 'Iyi gunler';
  return 'Iyi aksamlar';
}

function initialsOf(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

export default function HomeScreen() {
  const router = useRouter();
  const { user } = useAuthContext();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const semesterQuery = useActiveSemester();
  const semester = semesterQuery.data?.name ?? '';
  const enrollmentsQuery = useMyEnrollments(semester || undefined, 'approved');
  const attendanceQuery = useMyAttendance(semester || undefined);

  const program = enrollmentsQuery.data?.programs?.[0];
  const courses = useMemo(() => program?.courses ?? [], [program]);

  const now = new Date();
  const backendDay = jsDayToBackendDay(now.getDay());
  const nowMin = now.getHours() * 60 + now.getMinutes();

  const todaySessions = useMemo<TodaySession[]>(() => {
    return courses
      .flatMap((course) =>
        course.schedule_sessions
          .filter((session) => session.day_of_week === backendDay)
          .map((session) => {
            const range = slotRange(session.slot_numbers);
            if (!range) return null;
            return {
              course,
              session,
              start: range.start,
              end: range.end,
              startMin: timeToMinutes(range.start),
              endMin: timeToMinutes(range.end),
            };
          })
          .filter((s): s is TodaySession => s !== null)
      )
      .sort((a, b) => a.startMin - b.startMin);
  }, [courses, backendDay]);

  const nextSession = todaySessions.find((s) => s.endMin > nowMin);
  const nextIsLive = !!nextSession && nextSession.startMin <= nowMin;

  const firstName =
    program?.student_name?.split(/\s+/)[0] ?? user?.email?.split('@')[0] ?? 'Ogrenci';
  const displayName = program?.student_name ?? user?.email ?? '';

  const isLoading = semesterQuery.isLoading || enrollmentsQuery.isLoading;
  const isError = semesterQuery.isError || enrollmentsQuery.isError;

  const refetchAll = () => {
    semesterQuery.refetch();
    enrollmentsQuery.refetch();
    attendanceQuery.refetch();
  };

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 items-center justify-center bg-background" edges={['top']}>
        <ActivityIndicator size="large" color={colors.primary} />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView
        className="flex-1 items-center justify-center gap-4 bg-background px-8"
        edges={['top']}
      >
        <Ionicons name="cloud-offline-outline" size={44} color={colors.mutedForeground} />
        <Text className="text-center text-base text-muted-foreground">
          Program bilgisi yuklenemedi. Baglantini kontrol edip tekrar dene.
        </Text>
        <Button variant="outline" onPress={refetchAll} accessibilityLabel="Tekrar dene">
          <Text>Tekrar Dene</Text>
        </Button>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScrollView
        contentContainerClassName="px-5 pb-10 pt-4"
        refreshControl={
          <RefreshControl
            refreshing={enrollmentsQuery.isRefetching}
            onRefresh={refetchAll}
            tintColor={colors.primary}
          />
        }
      >
        {/* Baslik: tarih (mono, makine sesi) + selamlama + avatar */}
        <Animated.View
          entering={FadeInDown.duration(400)}
          className="mb-6 flex-row items-center justify-between"
        >
          <View className="flex-1">
            <Text className="mb-1 font-mono text-xs uppercase tracking-widest text-primary">
              {DAYS_OF_WEEK[backendDay]} · {now.getDate()} {MONTHS_TR[now.getMonth()]}
            </Text>
            <Text className="text-3xl font-extrabold text-foreground">
              {greetingByHour(now.getHours())}, {firstName}
            </Text>
          </View>
          <View className="h-12 w-12 items-center justify-center rounded-full bg-secondary">
            <Text className="text-base font-bold text-secondary-foreground">
              {initialsOf(displayName) || '?'}
            </Text>
          </View>
        </Animated.View>

        {/* Siradaki ders — gunun kahramani */}
        <Animated.View entering={FadeInDown.delay(80).duration(400)} className="mb-8">
          {nextSession ? (
            <View className="rounded-4xl bg-primary p-6">
              <View className="mb-4 flex-row items-center justify-between">
                <Text className="font-mono text-xs uppercase tracking-widest text-primary-foreground/80">
                  {nextIsLive ? '● Su an derste' : 'Siradaki ders'}
                </Text>
                <Badge variant="secondary" className="bg-primary-foreground/20">
                  <Text className="text-xs font-semibold text-primary-foreground">
                    {SESSION_TYPE_LABEL[nextSession.session.session_type]}
                  </Text>
                </Badge>
              </View>
              <Text className="mb-1 text-2xl font-extrabold text-primary-foreground">
                {nextSession.course.course_name}
              </Text>
              <Text className="mb-5 font-mono text-sm text-primary-foreground/80">
                {nextSession.course.course_code} · {nextSession.course.instructor}
              </Text>
              <View className="flex-row items-end justify-between">
                <Text className="font-mono text-3xl font-bold text-primary-foreground">
                  {nextSession.start}
                  <Text className="font-mono text-base text-primary-foreground/70">
                    {' '}– {nextSession.end}
                  </Text>
                </Text>
                <Button
                  size="sm"
                  className="bg-primary-foreground"
                  onPress={() => router.push('/(tabs)/scan')}
                  accessibilityLabel="QR yoklama okut"
                >
                  <Ionicons name="qr-code" size={14} color={colors.primary} />
                  <Text className="text-sm font-bold text-primary">QR Okut</Text>
                </Button>
              </View>
            </View>
          ) : (
            <Card className="items-center p-8">
              <Ionicons name="cafe-outline" size={36} color={colors.mutedForeground} />
              <Text className="mt-3 text-lg font-bold text-foreground">
                Bugun icin ders kalmadi
              </Text>
              <Text className="mt-1 text-center text-sm text-muted-foreground">
                {todaySessions.length > 0
                  ? 'Gunun tum dersleri tamamlandi. Iyi dinlenmeler!'
                  : 'Bugun programinda ders yok.'}
              </Text>
            </Card>
          )}
        </Animated.View>

        {/* Bugunun programi */}
        {todaySessions.length > 0 && (
          <Animated.View entering={FadeInDown.delay(160).duration(400)} className="mb-8">
            <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Bugunun programi
            </Text>
            <View className="gap-3">
              {todaySessions.map((item) => {
                const isPast = item.endMin <= nowMin;
                const isLive = item.startMin <= nowMin && item.endMin > nowMin;
                return (
                  <Card
                    key={`${item.course.id}-${item.session.session_type}-${item.start}`}
                    className={`flex-row items-center gap-4 p-4 ${
                      isLive ? 'border-primary' : ''
                    } ${isPast ? 'opacity-50' : ''}`}
                  >
                    <View className="items-center">
                      <Text className="font-mono text-sm font-bold text-foreground">
                        {item.start}
                      </Text>
                      <Text className="font-mono text-xs text-muted-foreground">{item.end}</Text>
                    </View>
                    <View
                      className={`h-10 w-0.5 rounded-full ${isLive ? 'bg-primary' : 'bg-border'}`}
                    />
                    <View className="flex-1">
                      <Text className="font-semibold text-foreground" numberOfLines={1}>
                        {item.course.course_name}
                      </Text>
                      <Text className="font-mono text-xs text-muted-foreground">
                        {item.course.course_code} · {SESSION_TYPE_LABEL[item.session.session_type]}
                      </Text>
                    </View>
                    {isLive && (
                      <Badge>
                        <Text className="text-xs font-bold text-primary-foreground">Canli</Text>
                      </Badge>
                    )}
                  </Card>
                );
              })}
            </View>
          </Animated.View>
        )}

        {/* Devam durumu */}
        <Animated.View entering={FadeInDown.delay(240).duration(400)}>
          <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
            Devam durumu
          </Text>
          {attendanceQuery.isLoading ? (
            <Card className="items-center p-6">
              <ActivityIndicator color={colors.primary} />
            </Card>
          ) : (attendanceQuery.data?.courses?.length ?? 0) > 0 ? (
            <Card className="gap-4 p-5">
              {attendanceQuery.data!.courses.map((course) => {
                const present = course.theory.present_count + course.lab.present_count;
                const total = course.theory.total_sessions + course.lab.total_sessions;
                const pct = total > 0 ? Math.round((present / total) * 100) : null;
                const barClass =
                  pct === null
                    ? 'bg-muted'
                    : pct >= 80
                      ? 'bg-success'
                      : pct >= 60
                        ? 'bg-warning'
                        : 'bg-destructive';
                return (
                  <View key={course.course_id}>
                    <View className="mb-1.5 flex-row items-center justify-between">
                      <Text
                        className="flex-1 text-sm font-semibold text-foreground"
                        numberOfLines={1}
                      >
                        {course.course_name}
                      </Text>
                      <Text className="ml-3 font-mono text-sm font-bold text-foreground">
                        {pct === null ? '—' : `%${pct}`}
                      </Text>
                    </View>
                    <View className="h-2 overflow-hidden rounded-full bg-muted">
                      <View
                        className={`h-2 rounded-full ${barClass}`}
                        style={{ width: `${pct ?? 0}%` }}
                      />
                    </View>
                  </View>
                );
              })}
            </Card>
          ) : (
            <Card className="items-center p-6">
              <Text className="text-sm text-muted-foreground">
                Bu donem icin henuz yoklama kaydi yok.
              </Text>
            </Card>
          )}
        </Animated.View>
      </ScrollView>
    </SafeAreaView>
  );
}
