import { useState, useCallback } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import { Text, Surface, Banner, ActivityIndicator, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { useQueryClient } from '@tanstack/react-query';

import { useMyAttendance, useScanQR } from '@/hooks/useAttendance';
import type { CourseAttendance, QRPayload, WeeklyRecord } from '@/types/attendance.types';
import { QRScannerModal } from '@/components/QRScannerModal';
import {
  SectionHeader,
  StatusChip,
  SessionTypeChip,
  AttendanceChip,
  SkeletonList,
  ScreenWrapper,
} from '@/components/ui';
import {
  spacing,
  radius,
  semanticColors,
} from '@/constants/tokens';

export default function QRAttendanceScreen() {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const { data: attendance, isLoading, error } = useMyAttendance();
  const scanMutation = useScanQR();
  const [lastScan, setLastScan] = useState<{ course: string; message: string } | null>(null);
  const [scannerOpen, setScannerOpen] = useState(false);

  const handleScanned = useCallback(
    (payload: QRPayload) => {
      setScannerOpen(false);
      scanMutation.mutate(
        { qr_payload: payload },
        {
          onSuccess: (res) => {
            setLastScan({
              course: `${res.course_code} ${res.course_name}`,
              message: res.message,
            });
          },
        },
      );
    },
    [scanMutation],
  );

  const recentRecords: (WeeklyRecord & { courseName: string; courseCode: string })[] = [];
  if (attendance?.courses) {
    for (const course of attendance.courses) {
      for (const rec of course.weekly_records) {
        recentRecords.push({ ...rec, courseName: course.course_name, courseCode: course.course_code });
      }
    }
  }
  recentRecords.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
  const latestRecords = recentRecords.slice(0, 6);

  const activeCourse: CourseAttendance | null = attendance?.courses?.[0] ?? null;

  const handleRefresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ['my-attendance'] });
  }, [queryClient]);

  return (
    <ScreenWrapper onRefresh={handleRefresh}>
      <QRScannerModal
        visible={scannerOpen}
        onClose={() => setScannerOpen(false)}
        onScanned={handleScanned}
      />

      {/* QR Scan Area */}
      <Surface style={[styles.qrArea, { backgroundColor: colors.surface }]} elevation={2}>
        <Pressable
          onPress={() => setScannerOpen(true)}
          disabled={scanMutation.isPending}
          style={({ pressed }) => [
            styles.qrIconBox,
            { borderColor: colors.primary, opacity: pressed ? 0.7 : 1 },
          ]}
          accessibilityRole="button"
          accessibilityLabel="QR kod tarayiciyi ac"
        >
          <FontAwesome name="qrcode" size={72} color={colors.primary} />
        </Pressable>
        <Text variant="bodyLarge" style={{ color: colors.onSurfaceVariant, marginTop: spacing.md }}>
          Yoklama icin QR kodu tarayin
        </Text>
        {scanMutation.isPending && (
          <ActivityIndicator animating style={{ marginTop: spacing.md }} color={colors.primary} />
        )}
        {lastScan && (
          <Surface style={[styles.scanResult, { backgroundColor: semanticColors.successLight }]} elevation={0}>
            <FontAwesome name="check-circle" size={16} color={semanticColors.success} />
            <Text variant="bodySmall" style={{ color: semanticColors.success, flex: 1 }}>
              {lastScan.course} - {lastScan.message}
            </Text>
          </Surface>
        )}
        {scanMutation.isError && (
          <Surface
            style={[styles.scanResult, { backgroundColor: colors.errorContainer }]}
            elevation={0}
            accessibilityRole="alert"
          >
            <FontAwesome name="exclamation-circle" size={16} color={colors.error} />
            <Text variant="bodySmall" style={{ color: colors.error, flex: 1 }}>Yoklama alinamadi. Tekrar deneyin.</Text>
          </Surface>
        )}
      </Surface>

      {/* Loading */}
      {isLoading && <SkeletonList count={2} lines={3} />}

      {/* Error */}
      {error && !isLoading && (
        <Banner
          visible
          icon="alert-circle-outline"
          style={[styles.errorBanner, { backgroundColor: colors.errorContainer }]}
          accessibilityRole="alert"
        >
          <Text variant="bodySmall" style={{ color: colors.onErrorContainer }}>
            Backend baglantisi kurulamadi. Veriler yuklenemedi.
          </Text>
        </Banner>
      )}

      {/* Active Course Info */}
      {activeCourse && (
        <View style={styles.section}>
          <SectionHeader title="Aktif Ders Devam Durumu" />
          <Surface style={[styles.classCard, { backgroundColor: colors.surface }]} elevation={1}>
            <View style={styles.classHeader}>
              <Surface style={[styles.courseIconBg, { backgroundColor: colors.primaryContainer }]} elevation={0}>
                <FontAwesome name="book" size={18} color={colors.primary} />
              </Surface>
              <View style={{ flex: 1 }}>
                <Text variant="titleSmall" style={{ color: colors.onSurface }}>{activeCourse.course_name}</Text>
                <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>{activeCourse.course_code}</Text>
              </View>
            </View>

            {/* Theory stats */}
            <View style={[styles.statsRow, { borderTopColor: colors.outlineVariant }]}>
              <SessionTypeChip type="Teori" />
              <View style={styles.statsValues}>
                <Text variant="bodySmall" style={{ color: semanticColors.success, fontWeight: '600' }}>
                  {activeCourse.theory.present_count} katilim
                </Text>
                <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>/</Text>
                <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>
                  {activeCourse.theory.total_sessions} toplam
                </Text>
              </View>
              <StatusChip
                label={activeCourse.theory.passed ? 'Gecti' : 'Risk'}
                variant={activeCourse.theory.passed ? 'success' : 'danger'}
              />
            </View>

            {/* Lab stats */}
            {activeCourse.lab.total_sessions > 0 && (
              <View style={[styles.statsRow, { borderTopColor: colors.outlineVariant }]}>
                <SessionTypeChip type="Lab" />
                <View style={styles.statsValues}>
                  <Text variant="bodySmall" style={{ color: semanticColors.success, fontWeight: '600' }}>
                    {activeCourse.lab.present_count} katilim
                  </Text>
                  <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>/</Text>
                  <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>
                    {activeCourse.lab.total_sessions} toplam
                  </Text>
                </View>
                <StatusChip
                  label={activeCourse.lab.passed ? 'Gecti' : 'Risk'}
                  variant={activeCourse.lab.passed ? 'success' : 'danger'}
                />
              </View>
            )}

            <View style={styles.classDetail}>
              <FontAwesome name="user" size={14} color={colors.onSurfaceVariant} />
              <Text variant="bodyMedium" style={{ color: colors.onSurfaceVariant }}>{activeCourse.instructor}</Text>
            </View>
          </Surface>
        </View>
      )}

      {/* All Courses Summary */}
      {attendance?.courses && attendance.courses.length > 1 && (
        <View style={styles.section}>
          <SectionHeader title="Tum Dersler" />
          {attendance.courses.map((course) => {
            const theoryPct = course.theory.total_sessions > 0
              ? Math.round((course.theory.present_count / course.theory.total_sessions) * 100)
              : 100;
            return (
              <Surface
                key={course.course_id}
                style={[styles.courseSummaryRow, { backgroundColor: colors.surface }]}
                elevation={1}
                accessibilityLabel={`${course.course_code} ${course.course_name}, devam: yuzde ${theoryPct}`}
              >
                <View style={{ flex: 1 }}>
                  <Text variant="labelSmall" style={{ color: colors.primary, fontWeight: '600' }}>{course.course_code}</Text>
                  <Text variant="bodyMedium" style={{ color: colors.onSurface }}>{course.course_name}</Text>
                </View>
                <AttendanceChip percentage={theoryPct} />
              </Surface>
            );
          })}
        </View>
      )}

      {/* Recent Attendance */}
      {latestRecords.length > 0 && (
        <View style={styles.section}>
          <SectionHeader title="Son Yoklamalar" />
          {latestRecords.map((item, index) => (
            <Surface
              key={`${item.courseCode}-${item.week}-${item.session_type}-${index}`}
              style={[styles.attendanceRow, { backgroundColor: colors.surface }]}
              elevation={1}
              accessibilityLabel={`${item.courseName}, ${item.date}, hafta ${item.week}`}
            >
              <View style={styles.attendanceInfo}>
                <Text variant="bodyMedium" style={{ color: colors.onSurface, fontWeight: '500' }}>{item.courseName}</Text>
                <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                  {item.date} - Hafta {item.week} ({item.session_type === 'theory' ? 'Teori' : 'Lab'})
                </Text>
              </View>
              <StatusChip
                label={item.marked_via === 'qr' ? 'QR' : 'Manuel'}
                variant="success"
              />
            </Surface>
          ))}
        </View>
      )}
    </ScreenWrapper>
  );
}

const styles = StyleSheet.create({
  qrArea: {
    borderRadius: radius.xl,
    padding: spacing.xl,
    alignItems: 'center',
    marginBottom: spacing.lg,
  },
  qrIconBox: {
    width: 150,
    height: 150,
    borderRadius: radius.xl,
    borderWidth: 3,
    borderStyle: 'dashed',
    alignItems: 'center',
    justifyContent: 'center',
  },
  scanResult: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    marginTop: spacing.md,
    padding: spacing.md,
    borderRadius: radius.sm,
    width: '100%',
  },
  errorBanner: { borderRadius: radius.md, marginBottom: spacing.md },
  section: { marginBottom: spacing.lg },
  classCard: {
    borderRadius: radius.lg,
    padding: spacing.md,
  },
  classHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: spacing.md,
    gap: spacing.md,
  },
  courseIconBg: {
    width: 40,
    height: 40,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  classDetail: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    marginTop: spacing.sm,
  },
  statsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    gap: spacing.sm,
  },
  statsValues: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
  courseSummaryRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: radius.md,
    padding: 14,
    marginBottom: spacing.sm,
  },
  attendanceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    borderRadius: radius.md,
    padding: 14,
    marginBottom: spacing.sm,
  },
  attendanceInfo: { flex: 1 },
});
