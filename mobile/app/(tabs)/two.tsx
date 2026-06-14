import { StyleSheet, ScrollView, View, RefreshControl } from 'react-native';
import { Text, Surface, Chip, Divider, useTheme, ActivityIndicator } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useMyEnrollments } from '@/hooks/useEnrollment';
import { useMyGrades } from '@/hooks/useGrades';
import {
  HeroCard,
  SectionHeader,
  GradeBadge,
  EmptyState,
  Skeleton,
} from '@/components/ui';
import {
  spacing,
  radius,
  layout,
  accentColors,
} from '@/constants/tokens';

const courseColors = [
  accentColors.indigo,
  accentColors.violet,
  accentColors.pink,
  accentColors.teal,
  accentColors.amber,
  accentColors.red,
];

export default function CoursesScreen() {
  const { colors } = useTheme();

  const enrollmentsQuery = useMyEnrollments();
  const gradesQuery = useMyGrades();

  const isRefreshing = enrollmentsQuery.isRefetching || gradesQuery.isRefetching;
  const isLoading = enrollmentsQuery.isLoading || gradesQuery.isLoading;

  const handleRefresh = () => {
    enrollmentsQuery.refetch();
    gradesQuery.refetch();
  };

  // Get approved program courses
  const approvedProgram = enrollmentsQuery.data?.programs?.find(p => p.status === 'approved');
  const enrolledCourses = approvedProgram?.courses ?? [];

  // Grade lookup from grades API
  const activeGrades = gradesQuery.data?.active_courses ?? [];
  const completedGrades = gradesQuery.data?.completed_courses ?? [];
  const gradeMap = new Map<string, { grade?: string; average?: number }>();
  for (const c of activeGrades) {
    gradeMap.set(c.course_code, {});
  }
  for (const c of completedGrades) {
    gradeMap.set(c.course_code, { grade: c.grade_point, average: c.weighted_average });
  }

  const totalCredits = enrolledCourses.reduce((sum, c) => sum + c.credits, 0);
  const gpa = gradesQuery.data?.cumulative_gpa;
  const semester = approvedProgram?.semester ?? '';

  return (
    <ScrollView
      style={[styles.scrollView, { backgroundColor: colors.background }]}
      refreshControl={
        <RefreshControl refreshing={isRefreshing} onRefresh={handleRefresh} />
      }
    >
      <View style={styles.container}>
        {isLoading ? (
          <View style={{ marginTop: spacing.lg }}>
            <Skeleton width="100%" height={80} />
            <Skeleton width="100%" height={100} style={{ marginTop: spacing.md }} />
            <Skeleton width="100%" height={100} style={{ marginTop: spacing.md }} />
          </View>
        ) : enrolledCourses.length === 0 ? (
          <EmptyState message="Henuz onaylanmis ders kaydiniz bulunmuyor" />
        ) : (
          <>
            {/* Summary */}
            <HeroCard
              items={[
                { value: enrolledCourses.length, label: 'Ders' },
                { value: totalCredits, label: 'Kredi' },
                { value: gpa?.toFixed(2) ?? '-', label: 'GPA' },
              ]}
            />

            {semester ? (
              <View style={styles.semesterBadge}>
                <Chip icon="calendar" compact mode="flat">
                  {semester}
                </Chip>
              </View>
            ) : null}

            {/* Course Cards */}
            <SectionHeader title="Kayitli Dersler" />
            {enrolledCourses.map((course, index) => {
              const gradeInfo = gradeMap.get(course.course_code);
              const color = courseColors[index % courseColors.length];

              return (
                <Surface
                  key={course.id}
                  style={[styles.courseCard, { backgroundColor: colors.surfaceVariant }]}
                  elevation={0}
                >
                  <View style={styles.courseHeader}>
                    <View style={[styles.colorBar, { backgroundColor: color }]} />
                    <View style={styles.courseHeaderInfo}>
                      <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                        {course.course_code}
                      </Text>
                      <Text variant="bodyMedium" style={[styles.courseNameText, { color: colors.onSurface }]}>
                        {course.course_name}
                      </Text>
                      <View style={styles.instructorRow}>
                        <FontAwesome name="user" size={10} color={colors.onSurfaceVariant} />
                        <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                          {course.instructor}
                        </Text>
                      </View>
                    </View>
                    <Chip
                      compact
                      mode="flat"
                      style={{ backgroundColor: colors.outline + '40' }}
                      textStyle={{ fontSize: 11, color: colors.onSurfaceVariant }}
                    >
                      {course.credits} KR
                    </Chip>
                  </View>

                  {gradeInfo?.grade && (
                    <>
                      <Divider style={{ backgroundColor: colors.outlineVariant }} />
                      <View style={styles.courseFooter}>
                        <View style={styles.footerItem}>
                          <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>Not</Text>
                          <GradeBadge grade={gradeInfo.grade} />
                        </View>
                        {gradeInfo.average != null && (
                          <>
                            <View style={[styles.footerDivider, { backgroundColor: colors.outlineVariant }]} />
                            <View style={styles.footerItem}>
                              <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>Ortalama</Text>
                              <Text variant="bodyMedium" style={{ color: colors.onSurface, fontWeight: '600' }}>
                                {gradeInfo.average.toFixed(1)}
                              </Text>
                            </View>
                          </>
                        )}
                      </View>
                    </>
                  )}
                </Surface>
              );
            })}
          </>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scrollView: { flex: 1 },
  container: { flex: 1, padding: layout.screenPadding },
  semesterBadge: {
    flexDirection: 'row',
    marginBottom: spacing.md,
  },
  courseCard: {
    borderRadius: radius.md,
    marginTop: spacing.sm,
    overflow: 'hidden',
  },
  courseHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: spacing.md,
  },
  colorBar: {
    width: 4,
    height: 40,
    borderRadius: 2,
    marginRight: spacing.sm,
  },
  courseHeaderInfo: { flex: 1 },
  courseNameText: { fontWeight: '600', marginTop: 1 },
  instructorRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
    marginTop: 3,
  },
  courseFooter: { flexDirection: 'row' },
  footerItem: {
    flex: 1,
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    gap: spacing.sm,
  },
  footerDivider: {
    width: 1,
    marginVertical: spacing.sm,
  },
});
