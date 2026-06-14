import React, { useState, useMemo } from 'react';
import { StyleSheet, ScrollView, View, RefreshControl } from 'react-native';
import {
  Text, Surface, Button, Checkbox, Chip, Divider, Banner,
  useTheme, ActivityIndicator,
} from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useActiveSemester } from '@/hooks/useCatalog';
import { useAvailableCourses, useMyEnrollments, useCreateEnrollment, useCancelEnrollment } from '@/hooks/useEnrollment';
import { useToast } from '@/contexts/ToastContext';
import { useHaptic } from '@/hooks/useHaptic';
import { SectionHeader, StatusChip } from '@/components/ui';
import { spacing, radius, layout, accentColors, withOpacity } from '@/constants/tokens';
import type { AvailableCourse } from '@/types/enrollment.types';

const DAY_NAMES = ['', 'Pazartesi', 'Sali', 'Carsamba', 'Persembe', 'Cuma', 'Cumartesi', 'Pazar'];

function formatSchedule(course: AvailableCourse): string {
  if (!course.schedule_sessions?.length) return '';
  return course.schedule_sessions
    .map(s => `${DAY_NAMES[s.day_of_week] || ''} ${s.session_type === 'lab' ? '(Lab)' : ''}`)
    .join(', ');
}

export default function EnrollmentScreen() {
  const { colors } = useTheme();
  const toast = useToast();
  const haptic = useHaptic();

  const semesterQuery = useActiveSemester();
  const semester = semesterQuery.data?.name ?? '';

  const availableQuery = useAvailableCourses(semester);
  const enrollmentsQuery = useMyEnrollments(semester);
  const createMutation = useCreateEnrollment();
  const cancelMutation = useCancelEnrollment();

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const existingProgram = enrollmentsQuery.data?.programs?.[0];
  const hasPending = existingProgram?.status === 'pending';
  const hasApproved = existingProgram?.status === 'approved';

  const availableCourses = availableQuery.data?.available_courses ?? [];

  const totalCredits = useMemo(() => {
    return availableCourses
      .filter(c => selectedIds.has(c.id))
      .reduce((sum, c) => sum + c.credits, 0);
  }, [selectedIds, availableCourses]);

  const isLoading = semesterQuery.isLoading || availableQuery.isLoading || enrollmentsQuery.isLoading;
  const isRefreshing = availableQuery.isRefetching || enrollmentsQuery.isRefetching;

  const handleRefresh = () => {
    availableQuery.refetch();
    enrollmentsQuery.refetch();
  };

  const toggleCourse = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const handleSubmit = () => {
    if (selectedIds.size === 0) {
      haptic.error();
      toast.show({ message: 'En az bir ders secin', type: 'warning' });
      return;
    }

    haptic.light();
    createMutation.mutate(
      { semester, course_ids: Array.from(selectedIds) },
      {
        onSuccess: () => {
          haptic.success();
          toast.show({ message: 'Ders kaydiniz gonderildi', type: 'success' });
          setSelectedIds(new Set());
        },
        onError: (error: any) => {
          haptic.error();
          const msg = error.response?.data?.message || 'Ders kaydi gonderilemedi';
          toast.show({ message: msg, type: 'error' });
        },
      }
    );
  };

  const handleCancel = () => {
    haptic.light();
    cancelMutation.mutate(semester, {
      onSuccess: () => {
        haptic.success();
        toast.show({ message: 'Ders kaydiniz iptal edildi', type: 'success' });
      },
      onError: () => {
        haptic.error();
        toast.show({ message: 'Iptal islemi basarisiz', type: 'error' });
      },
    });
  };

  return (
    <View style={[styles.screen, { backgroundColor: colors.background }]}>
      <ScrollView
        style={styles.scrollView}
        refreshControl={<RefreshControl refreshing={isRefreshing} onRefresh={handleRefresh} />}
      >
        <View style={styles.container}>
          {!semester ? (
            <Banner visible icon="information-outline">
              Aktif donem bulunamadi
            </Banner>
          ) : isLoading ? (
            <ActivityIndicator style={{ marginTop: spacing.xxl }} />
          ) : (
            <>
              {/* Current status */}
              {existingProgram && (
                <Surface style={[styles.statusCard, { backgroundColor: colors.surface }]} elevation={1}>
                  <View style={styles.statusHeader}>
                    <Text variant="titleSmall" style={{ color: colors.onSurface }}>
                      Mevcut Kayit
                    </Text>
                    <StatusChip
                      variant={existingProgram.status === 'approved' ? 'success' : existingProgram.status === 'rejected' ? 'danger' : 'warning'}
                      label={existingProgram.status === 'approved' ? 'Onaylandi' : existingProgram.status === 'rejected' ? 'Reddedildi' : 'Beklemede'}
                    />
                  </View>
                  <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant, marginTop: spacing.xs }}>
                    {existingProgram.courses.length} ders · {existingProgram.courses.reduce((s, c) => s + c.credits, 0)} kredi
                  </Text>
                  {hasPending && (
                    <Button
                      mode="outlined"
                      onPress={handleCancel}
                      loading={cancelMutation.isPending}
                      style={{ marginTop: spacing.md }}
                      textColor={accentColors.red}
                    >
                      Kaydi Iptal Et
                    </Button>
                  )}
                </Surface>
              )}

              {/* Available courses */}
              {!hasApproved && !hasPending && (
                <>
                  <SectionHeader title={`Alinabilir Dersler (${semester})`} />
                  {availableCourses.length === 0 ? (
                    <Text variant="bodyMedium" style={{ color: colors.onSurfaceVariant, marginTop: spacing.md }}>
                      Alinabilir ders bulunamadi
                    </Text>
                  ) : (
                    availableCourses.map((course) => {
                      const isSelected = selectedIds.has(course.id);
                      return (
                        <Surface
                          key={course.id}
                          style={[
                            styles.courseCard,
                            { backgroundColor: isSelected ? withOpacity(colors.primary, 0.08) : colors.surface },
                          ]}
                          elevation={1}
                        >
                          <View style={styles.courseRow}>
                            <Checkbox
                              status={isSelected ? 'checked' : 'unchecked'}
                              onPress={() => toggleCourse(course.id)}
                            />
                            <View style={styles.courseInfo}>
                              <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                                {course.course_code}
                              </Text>
                              <Text variant="bodyMedium" style={[styles.courseName, { color: colors.onSurface }]}>
                                {course.course_name}
                              </Text>
                              <View style={styles.metaRow}>
                                <Chip compact textStyle={{ fontSize: 10 }}>{course.credits} KR</Chip>
                                <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                                  {course.instructor}
                                </Text>
                              </View>
                              {course.schedule_sessions?.length > 0 && (
                                <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant, marginTop: 2 }}>
                                  {formatSchedule(course)}
                                </Text>
                              )}
                              <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
                                Kontenjan: {course.available_seats}/{course.max_capacity}
                              </Text>
                            </View>
                          </View>
                        </Surface>
                      );
                    })
                  )}
                </>
              )}
            </>
          )}
        </View>
      </ScrollView>

      {/* Bottom bar */}
      {!hasApproved && !hasPending && availableCourses.length > 0 && selectedIds.size > 0 && (
        <Surface style={[styles.bottomBar, { backgroundColor: colors.surface }]} elevation={3}>
          <View style={styles.bottomInfo}>
            <Text variant="bodyMedium" style={{ color: colors.onSurface }}>
              {selectedIds.size} ders · {totalCredits} kredi
            </Text>
          </View>
          <Button
            mode="contained"
            onPress={handleSubmit}
            loading={createMutation.isPending}
            disabled={createMutation.isPending}
          >
            Kaydi Gonder
          </Button>
        </Surface>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  scrollView: { flex: 1 },
  container: { flex: 1, padding: layout.screenPadding, paddingBottom: 100 },
  statusCard: { borderRadius: radius.lg, padding: spacing.md, marginBottom: spacing.md },
  statusHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  courseCard: { borderRadius: radius.md, marginTop: spacing.sm, overflow: 'hidden' },
  courseRow: { flexDirection: 'row', alignItems: 'center', padding: spacing.sm },
  courseInfo: { flex: 1, marginLeft: spacing.xs },
  courseName: { fontWeight: '600', marginTop: 1 },
  metaRow: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, marginTop: 4 },
  bottomBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderTopWidth: 1,
    borderTopColor: '#e0e0e0',
  },
  bottomInfo: { flex: 1 },
});
