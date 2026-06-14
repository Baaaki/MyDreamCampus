import { StyleSheet, ScrollView, View, RefreshControl } from 'react-native';
import { Text, Surface, TouchableRipple, useTheme, ActivityIndicator } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { useRouter } from 'expo-router';

import { useAuthContext } from '@/contexts/AuthContext';
import { useMyGrades } from '@/hooks/useGrades';
import { useMyAttendance } from '@/hooks/useAttendance';
import { useMyEnrollments } from '@/hooks/useEnrollment';
import { SectionHeader, StatCard, SessionTypeChip } from '@/components/ui';
import { spacing, radius, layout, accentColors, withOpacity } from '@/constants/tokens';

const quickActions = [
  { icon: 'qrcode' as const, label: 'QR Yoklama', color: accentColors.indigo, route: '/screens/qr-attendance' as const },
  { icon: 'calendar' as const, label: 'Ders Kaydi', color: accentColors.violet, route: '/screens/enrollment' as const },
  { icon: 'bar-chart' as const, label: 'Notlarim', color: accentColors.pink, route: '/screens/my-grades' as const },
  { icon: 'cutlery' as const, label: 'Yemekhane', color: accentColors.amber, route: '/screens/cafeteria' as const },
];

export default function HomeScreen() {
  const { colors } = useTheme();
  const router = useRouter();
  const { user } = useAuthContext();

  const gradesQuery = useMyGrades();
  const attendanceQuery = useMyAttendance();
  const enrollmentsQuery = useMyEnrollments();

  const isRefreshing = gradesQuery.isRefetching || attendanceQuery.isRefetching;

  const handleRefresh = () => {
    gradesQuery.refetch();
    attendanceQuery.refetch();
    enrollmentsQuery.refetch();
  };

  // Derive stats from real data
  const gpa = gradesQuery.data?.cumulative_gpa?.toFixed(2) ?? '-';
  const courseCount = (() => {
    const approved = enrollmentsQuery.data?.programs?.find(p => p.status === 'approved');
    return approved?.courses?.length ?? gradesQuery.data?.active_courses?.length ?? '-';
  })();

  const attendancePercent = (() => {
    const courses = attendanceQuery.data?.courses;
    if (!courses?.length) return '-';
    const totalPresent = courses.reduce((sum, c) => sum + c.theory.present_count + c.lab.present_count, 0);
    const totalSessions = courses.reduce((sum, c) => sum + c.theory.total_sessions + c.lab.total_sessions, 0);
    if (totalSessions === 0) return '-';
    return `%${Math.round((totalPresent / totalSessions) * 100)}`;
  })();

  const displayName = user?.email?.split('@')[0] ?? 'Ogrenci';
  const department = user?.department ?? '';

  return (
    <ScrollView
      style={[styles.scrollView, { backgroundColor: colors.background }]}
      refreshControl={
        <RefreshControl refreshing={isRefreshing} onRefresh={handleRefresh} />
      }
    >
      <View style={styles.container}>
        {/* Header */}
        <View style={styles.header}>
          <Text variant="bodyLarge" style={{ color: colors.onSurfaceVariant }}>Merhaba,</Text>
          <Text variant="headlineMedium" style={[styles.name, { color: colors.onBackground }]}>
            {displayName}
          </Text>
          {department ? (
            <Text variant="bodyMedium" style={{ color: colors.onSurfaceVariant }}>
              {department}
            </Text>
          ) : null}
        </View>

        {/* Quick Actions */}
        <View style={styles.section}>
          <SectionHeader title="Hizli Erisim" />
          <View style={styles.actionsGrid}>
            {quickActions.map((action) => (
              <Surface
                key={action.label}
                style={[styles.actionCard, { backgroundColor: colors.surface }]}
                elevation={1}
              >
                <TouchableRipple
                  onPress={() => router.push(action.route)}
                  borderless
                  style={styles.actionCardInner}
                  rippleColor={withOpacity(action.color, 0.19)}
                >
                  <View style={styles.actionCardContent}>
                    <View style={[styles.actionIcon, { backgroundColor: withOpacity(action.color, 0.08) }]}>
                      <FontAwesome name={action.icon} size={24} color={action.color} />
                    </View>
                    <Text variant="labelMedium" style={{ color: colors.onSurface }}>
                      {action.label}
                    </Text>
                  </View>
                </TouchableRipple>
              </Surface>
            ))}
          </View>
        </View>

        {/* Enrolled Courses */}
        <View style={styles.section}>
          <SectionHeader title="Kayitli Derslerim" />
          {enrollmentsQuery.isLoading ? (
            <ActivityIndicator style={{ marginVertical: spacing.lg }} />
          ) : (() => {
            const approved = enrollmentsQuery.data?.programs?.find(p => p.status === 'approved');
            if (!approved?.courses?.length) {
              return (
                <Surface style={[styles.scheduleCard, { backgroundColor: colors.surface }]} elevation={1}>
                  <View style={styles.scheduleContent}>
                    <Text variant="bodyMedium" style={{ color: colors.onSurfaceVariant }}>
                      Henuz onaylanmis ders kaydiniz bulunmuyor
                    </Text>
                  </View>
                </Surface>
              );
            }
            return approved.courses.slice(0, 4).map((course) => (
              <Surface
                key={course.id}
                style={[styles.scheduleCard, { backgroundColor: colors.surface }]}
                elevation={1}
              >
                <View style={styles.scheduleContent}>
                  <View style={styles.scheduleTime}>
                    <Text variant="labelSmall" style={{ color: colors.primary }}>{course.course_code}</Text>
                  </View>
                  <View style={styles.scheduleInfo}>
                    <Text variant="bodyLarge" style={[styles.courseName, { color: colors.onSurface }]}>
                      {course.course_name}
                    </Text>
                    <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>
                      {course.instructor} · {course.credits} KR
                    </Text>
                  </View>
                  <FontAwesome name="chevron-right" size={14} color={colors.onSurfaceVariant} />
                </View>
              </Surface>
            ));
          })()}
        </View>

        {/* Stats */}
        <View style={styles.section}>
          <SectionHeader title="Donem Ozeti" />
          <View style={styles.statsRow}>
            <StatCard icon="line-chart" value={gpa} label="GPA" />
            <StatCard icon="book" value={String(courseCount)} label="Ders" />
            <StatCard icon="check-circle" value={attendancePercent} label="Devam" />
          </View>
        </View>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scrollView: { flex: 1 },
  container: { flex: 1, padding: layout.screenPadding },
  header: { marginBottom: spacing.lg },
  name: { fontWeight: 'bold', marginTop: 2 },
  section: { marginBottom: spacing.lg },
  actionsGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.md },
  actionCard: {
    width: '47%',
    borderRadius: radius.lg,
    overflow: 'hidden',
  },
  actionCardInner: {
    padding: spacing.md,
  },
  actionCardContent: {
    alignItems: 'center',
  },
  actionIcon: {
    width: 52,
    height: 52,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: spacing.sm,
  },
  scheduleCard: {
    borderRadius: radius.md,
    marginBottom: spacing.sm,
    overflow: 'hidden',
  },
  scheduleContent: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 14,
  },
  scheduleTime: { alignItems: 'center', marginRight: 14, minWidth: 50 },
  scheduleInfo: { flex: 1 },
  courseName: { fontWeight: '500' },
  statsRow: { flexDirection: 'row', gap: spacing.md },
});
