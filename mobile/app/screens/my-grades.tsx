import { useMemo, useCallback } from 'react';
import { StyleSheet, View } from 'react-native';
import { Text, Surface, Chip, Banner, useTheme } from 'react-native-paper';
import { useQueryClient } from '@tanstack/react-query';

import { useMyGrades } from '@/hooks/useGrades';
import type { ActiveCourse, CompletedCourse } from '@/types/grades.types';
import {
  HeroCard,
  YearAccordion,
  SemesterAccordion,
  EmptyState,
  GradeBadge,
  SkeletonList,
  ScreenWrapper,
} from '@/components/ui';
import {
  spacing,
  radius,
  yearColors,
  withOpacity,
} from '@/constants/tokens';

// ─── Helpers ───────────────────────────────────────────────

function parseSemester(sem: string): { year: number; term: number; display: string } {
  const lower = sem.toLowerCase();
  const match = lower.match(/(\d{4})-(\d{4})-(fall|spring|guz|bahar)/);
  if (match) {
    const startYear = parseInt(match[1]);
    const isFall = match[3] === 'fall' || match[3] === 'guz';
    return {
      year: startYear,
      term: isFall ? 1 : 2,
      display: `${match[1]}-${match[2]} ${isFall ? 'Guz' : 'Bahar'}`,
    };
  }
  const numMatch = lower.match(/^(\d+)$/);
  if (numMatch) {
    const n = parseInt(numMatch[1]);
    return {
      year: Math.ceil(n / 2),
      term: ((n - 1) % 2) + 1,
      display: `${Math.ceil(n / 2)}. Sinif ${((n - 1) % 2) === 0 ? 'Guz' : 'Bahar'}`,
    };
  }
  return { year: 1, term: 1, display: sem };
}

type SemesterGroup = {
  semester: string;
  display: string;
  term: number;
  gpa: number | null;
  courses: {
    code: string;
    name: string;
    credits: number;
    weightedAvg: number | null;
    gradePoint: string;
    isActive: boolean;
  }[];
};

type YearGroup = { year: number; semesters: SemesterGroup[] };

function buildYearGroups(active: ActiveCourse[], completed: CompletedCourse[]): YearGroup[] {
  const semMap = new Map<string, SemesterGroup>();

  for (const c of completed) {
    const parsed = parseSemester(c.semester);
    const key = `${parsed.year}-${parsed.term}`;
    if (!semMap.has(key)) {
      semMap.set(key, { semester: c.semester, display: parsed.display, term: parsed.term, gpa: null, courses: [] });
    }
    semMap.get(key)!.courses.push({
      code: c.course_code, name: c.course_name, credits: c.credits,
      weightedAvg: c.weighted_average, gradePoint: c.grade_point, isActive: false,
    });
  }

  for (const c of active) {
    const parsed = parseSemester(c.semester);
    const key = `${parsed.year}-${parsed.term}`;
    if (!semMap.has(key)) {
      semMap.set(key, { semester: c.semester, display: parsed.display, term: parsed.term, gpa: null, courses: [] });
    }
    semMap.get(key)!.courses.push({
      code: c.course_code, name: c.course_name, credits: c.credits,
      weightedAvg: null, gradePoint: '-', isActive: true,
    });
  }

  const yearMap = new Map<number, YearGroup>();
  for (const [key, sem] of semMap) {
    const yearNum = parseInt(key.split('-')[0]);
    if (!yearMap.has(yearNum)) yearMap.set(yearNum, { year: yearNum, semesters: [] });
    yearMap.get(yearNum)!.semesters.push(sem);
  }

  const result = Array.from(yearMap.values()).sort((a, b) => a.year - b.year);
  for (const y of result) y.semesters.sort((a, b) => a.term - b.term);
  return result;
}

// ─── Screen ────────────────────────────────────────────────

export default function MyGradesScreen() {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useMyGrades();

  const yearGroups = useMemo(() => {
    if (!data) return [];
    return buildYearGroups(data.active_courses || [], data.completed_courses || []);
  }, [data]);

  const handleRefresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ['my-grades'] });
  }, [queryClient]);

  return (
    <ScreenWrapper onRefresh={handleRefresh}>
      {/* Loading */}
      {isLoading && <SkeletonList count={3} lines={4} />}

      {/* Error */}
      {error && !isLoading && (
        <Banner
          visible
          icon="alert-circle-outline"
          style={[styles.errorBanner, { backgroundColor: colors.errorContainer }]}
          accessibilityRole="alert"
        >
          <Text variant="bodySmall" style={{ color: colors.onErrorContainer }}>
            Backend baglantisi kurulamadi.
          </Text>
        </Banner>
      )}

      {/* GPA Card */}
      {data && (
        <HeroCard
          items={[
            { value: data.cumulative_gpa?.toFixed(2) ?? '-', label: 'Genel GPA' },
            { value: `${data.total_credits ?? 0} KR`, label: 'Tamamlanan' },
          ]}
        />
      )}

      {/* Year > Semester > Grades */}
      {yearGroups.map((yearData) => {
        const totalCourses = yearData.semesters.reduce((s, sem) => s + sem.courses.length, 0);
        const yColor = yearColors[(yearData.year - 1) % yearColors.length];

        return (
          <YearAccordion key={yearData.year} year={yearData.year} courseCount={totalCourses}>
            {yearData.semesters.map((sem) => (
              <SemesterAccordion
                key={sem.term}
                year={yearData.year}
                term={sem.term}
                extra={
                  sem.gpa !== null ? (
                    <Chip
                      compact
                      mode="flat"
                      style={{ backgroundColor: withOpacity(yColor, 0.09) }}
                      textStyle={{ color: yColor, fontSize: 12, fontWeight: '700' }}
                    >
                      GPA {sem.gpa.toFixed(2)}
                    </Chip>
                  ) : undefined
                }
              >
                {sem.courses.length === 0 ? (
                  <EmptyState message="Henuz not girilmedi" />
                ) : (
                  <>
                    <View style={[styles.tableHeader, { borderBottomColor: colors.outlineVariant }]}>
                      <Text variant="labelSmall" style={[styles.cellWide, { color: colors.onSurfaceVariant }]}>Ders</Text>
                      <Text variant="labelSmall" style={[styles.cellNarrow, { color: colors.onSurfaceVariant }]}>KR</Text>
                      <Text variant="labelSmall" style={[styles.cellNarrow, { color: colors.onSurfaceVariant }]}>Ort.</Text>
                      <Text variant="labelSmall" style={[styles.cellNarrow, { color: colors.onSurfaceVariant }]}>Not</Text>
                    </View>
                    {sem.courses.map((course) => (
                      <Surface
                        key={course.code}
                        style={[styles.tableRow, { backgroundColor: colors.surfaceVariant }]}
                        elevation={0}
                        accessibilityLabel={`${course.code} ${course.name}, ${course.credits} kredi, not: ${course.gradePoint}`}
                      >
                        <View style={styles.cellWide}>
                          <Text variant="labelSmall" style={{ color: colors.primary, fontWeight: '600' }}>{course.code}</Text>
                          <Text variant="bodySmall" style={{ color: colors.onSurface }} numberOfLines={1}>{course.name}</Text>
                        </View>
                        <Text variant="bodySmall" style={[styles.cellNarrow, styles.cellText, { color: colors.onSurfaceVariant }]}>
                          {course.credits}
                        </Text>
                        <Text variant="bodySmall" style={[styles.cellNarrow, styles.cellText, { color: colors.onSurface, fontWeight: '600' }]}>
                          {course.weightedAvg !== null ? course.weightedAvg.toFixed(0) : '-'}
                        </Text>
                        <View style={styles.cellNarrow}>
                          <GradeBadge grade={course.gradePoint} />
                        </View>
                      </Surface>
                    ))}
                  </>
                )}
              </SemesterAccordion>
            ))}
          </YearAccordion>
        );
      })}
    </ScreenWrapper>
  );
}

// ─── Styles ────────────────────────────────────────────────

const styles = StyleSheet.create({
  errorBanner: { borderRadius: radius.md, marginBottom: spacing.md },
  tableHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.sm,
    marginTop: spacing.sm,
    borderBottomWidth: 1,
  },
  tableRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.sm,
    borderRadius: radius.sm,
    marginVertical: 2,
  },
  cellWide: { flex: 2 },
  cellNarrow: { flex: 1, alignItems: 'center' as const },
  cellText: { textAlign: 'center' },
});
