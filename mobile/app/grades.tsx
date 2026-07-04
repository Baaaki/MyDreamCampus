import Ionicons from '@expo/vector-icons/Ionicons';
import React from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ScreenHeader } from '@/components/ScreenHeader';
import { Badge, Button, Card, Text } from '@/components/ui';
import { assessmentLabel, gradePointColor } from '@/constants/grades';
import { useTheme } from '@/contexts/ThemeContext';
import { useMyGrades } from '@/hooks/useGrades';
import { COLORS } from '@/lib/theme';
import type { ActiveCourse, CompletedCourse, ScoreDetail } from '@/types/grades.types';

function ScoreRow({ slug, detail }: { slug: string; detail: ScoreDetail }) {
  return (
    <View className="flex-row items-center justify-between py-1">
      <Text className="text-sm text-muted-foreground">{assessmentLabel(slug)}</Text>
      <Text className="font-mono text-sm font-semibold text-foreground">
        {detail.is_absent ? 'Girmedi' : (detail.score ?? '—')}
      </Text>
    </View>
  );
}

function ActiveCourseCard({ course }: { course: ActiveCourse }) {
  const entries = Object.entries(course.scores ?? {});
  return (
    <Card className="mb-3 p-4">
      <View className="mb-2 flex-row items-start justify-between">
        <View className="flex-1 pr-3">
          <Text className="text-base font-bold text-card-foreground" numberOfLines={2}>
            {course.course_name}
          </Text>
          <Text className="font-mono text-xs text-muted-foreground">
            {course.course_code} · {course.credits} kredi
          </Text>
        </View>
        <Badge variant="secondary">
          <Text className="text-xs font-semibold text-secondary-foreground">Devam ediyor</Text>
        </Badge>
      </View>
      {entries.length > 0 ? (
        <View className="mt-1 border-t border-border pt-2">
          {entries.map(([slug, detail]) => (
            <ScoreRow key={slug} slug={slug} detail={detail} />
          ))}
        </View>
      ) : (
        <Text className="mt-1 text-sm text-muted-foreground">Henüz not girilmedi.</Text>
      )}
    </Card>
  );
}

function CompletedCourseCard({ course }: { course: CompletedCourse }) {
  return (
    <Card className="mb-3 flex-row items-center justify-between p-4">
      <View className="flex-1 pr-3">
        <Text className="text-base font-bold text-card-foreground" numberOfLines={2}>
          {course.course_name}
        </Text>
        <Text className="font-mono text-xs text-muted-foreground">
          {course.course_code} · {course.credits} kredi · Ort. {course.weighted_average.toFixed(1)}
        </Text>
      </View>
      <Badge variant={gradePointColor(course.grade_point)}>
        <Text className="text-sm font-extrabold text-primary-foreground">{course.grade_point}</Text>
      </Badge>
    </Card>
  );
}

export default function GradesScreen() {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const { data, isLoading, isError, refetch, isRefetching } = useMyGrades();

  const active = data?.active_courses ?? [];
  const completed = data?.completed_courses ?? [];

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScreenHeader title="Notlarım" subtitle={data?.student_number} />

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center gap-4 px-8">
          <Ionicons name="cloud-offline-outline" size={44} color={colors.mutedForeground} />
          <Text className="text-center text-base text-muted-foreground">
            Notların yüklenemedi. Bağlantını kontrol edip tekrar dene.
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
          {/* GNO ozet karti */}
          <Animated.View entering={FadeInDown.duration(400)}>
            <View className="mb-6 flex-row gap-3">
              <View className="flex-1 rounded-3xl bg-primary p-5">
                <Text className="font-mono text-xs uppercase tracking-widest text-primary-foreground/70">
                  GNO
                </Text>
                <Text className="mt-2 text-3xl font-extrabold text-primary-foreground">
                  {(data?.cumulative_gpa ?? 0).toFixed(2)}
                </Text>
              </View>
              <View className="flex-1 rounded-3xl border border-border bg-card p-5">
                <Text className="font-mono text-xs uppercase tracking-widest text-muted-foreground">
                  Toplam Kredi
                </Text>
                <Text className="mt-2 text-3xl font-extrabold text-card-foreground">
                  {data?.total_credits ?? 0}
                </Text>
              </View>
            </View>
          </Animated.View>

          {active.length > 0 && (
            <Animated.View entering={FadeInDown.delay(80).duration(400)} className="mb-6">
              <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                Devam eden dersler
              </Text>
              {active.map((course) => (
                <ActiveCourseCard key={course.course_code} course={course} />
              ))}
            </Animated.View>
          )}

          {completed.length > 0 && (
            <Animated.View entering={FadeInDown.delay(160).duration(400)}>
              <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                Tamamlanan dersler
              </Text>
              {completed.map((course) => (
                <CompletedCourseCard key={course.course_code} course={course} />
              ))}
            </Animated.View>
          )}

          {active.length === 0 && completed.length === 0 && (
            <View className="items-center py-16">
              <Ionicons name="document-text-outline" size={44} color={colors.mutedForeground} />
              <Text className="mt-3 text-center text-muted-foreground">
                Henüz not kaydın yok.
              </Text>
            </View>
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
