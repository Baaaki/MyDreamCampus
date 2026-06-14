import { StyleSheet, ScrollView, View } from 'react-native';
import { Text, Surface, Chip, Divider, ActivityIndicator, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useMyReservations } from '@/hooks/useMeals';
import type { Reservation } from '@/types/meal.types';
import { SectionHeader, StatusChip } from '@/components/ui';
import { spacing, radius } from '@/constants/tokens';
import { formatDate, formatDateTR, fullDayNames, mealColors } from './helpers';

export function HistoryTab() {
  const { colors } = useTheme();
  const now = new Date();
  const todayStr = formatDate(now);

  const { data: activeData, isLoading: loadingActive } = useMyReservations({ from_date: todayStr });
  const { data: pastData, isLoading: loadingPast } = useMyReservations({ to_date: todayStr, limit: 20 });

  const activeRes = activeData?.reservations?.filter((r) => r.status === 'confirmed' && !r.is_used) ?? [];
  const pastRes = pastData?.reservations ?? [];

  return (
    <ScrollView style={styles.flex1} contentContainerStyle={styles.scrollContent}>
      {/* Active Reservations */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <FontAwesome name="calendar-check-o" size={16} color={mealColors.pay} />
          <Text variant="titleMedium" style={{ color: colors.onBackground, fontWeight: '600' }}>Aktif Randevular</Text>
          {loadingActive && <ActivityIndicator animating color={colors.primary} size={16} style={{ marginLeft: spacing.sm }} />}
        </View>
        {!loadingActive && activeRes.length === 0 && (
          <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant, textAlign: 'center', paddingVertical: spacing.md }}>
            Aktif randevunuz bulunmamaktadir.
          </Text>
        )}
        {activeRes.map((r) => (
          <ReservationCard key={r.id} reservation={r} isPast={false} />
        ))}
      </View>

      {/* Past Records */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <FontAwesome name="history" size={16} color={colors.onSurfaceVariant} />
          <Text variant="titleMedium" style={{ color: colors.onBackground, fontWeight: '600' }}>Gecmis Kayitlar</Text>
          {loadingPast && <ActivityIndicator animating color={colors.primary} size={16} style={{ marginLeft: spacing.sm }} />}
        </View>
        {!loadingPast && pastRes.length === 0 && (
          <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant, textAlign: 'center', paddingVertical: spacing.md }}>
            Gecmis kayit bulunamadi.
          </Text>
        )}
        {pastRes.map((r) => (
          <ReservationCard key={r.id} reservation={r} isPast />
        ))}

        {pastData?.pagination && pastData.pagination.total_pages > 1 && (
          <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant, textAlign: 'center', marginTop: spacing.sm }}>
            Toplam {pastData.pagination.total_items} kayit ({pastData.pagination.page}/{pastData.pagination.total_pages} sayfa)
          </Text>
        )}
      </View>
    </ScrollView>
  );
}

// ─── ReservationCard ───────────────────────────────────────

function ReservationCard({ reservation: res, isPast }: { reservation: Reservation; isPast: boolean }) {
  const { colors } = useTheme();

  const statusVariant = isPast
    ? (res.is_used ? 'success' : 'danger')
    : 'warning';
  const statusLabel = isPast
    ? (res.is_used ? 'Kullanildi' : 'Kullanilmadi')
    : 'Aktif';

  return (
    <Surface
      style={styles.historyCard}
      elevation={1}
      accessibilityLabel={`${formatDateTR(res.date)}, ${statusLabel}, ${res.menu_type === 'vegan' ? 'vegan' : 'normal'}`}
    >
      <View style={styles.historyCardTop}>
        <View>
          <Text variant="bodyMedium" style={{ color: colors.onSurface, fontWeight: '600' }}>{formatDateTR(res.date)}</Text>
          <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
            {fullDayNames[new Date(res.date).getDay()]} - {res.meal_time === 'lunch' ? 'Ogle' : 'Aksam'}
          </Text>
        </View>
        <StatusChip label={statusLabel} variant={statusVariant} />
      </View>
      <Divider style={{ backgroundColor: colors.outlineVariant }} />
      <View style={styles.historyCardBottom}>
        <View style={styles.historyMeta}>
          <FontAwesome name="map-marker" size={12} color={colors.onSurfaceVariant} />
          <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>{res.cafeteria_name || res.cafeteria?.name || '-'}</Text>
        </View>
        <View style={styles.historyMeta}>
          <View style={[styles.menuTypeDot, { backgroundColor: res.menu_type === 'vegan' ? mealColors.vegan : mealColors.normal }]} />
          <Text variant="labelSmall" style={{ color: res.menu_type === 'vegan' ? mealColors.veganDark : mealColors.normalDark }}>
            {res.menu_type === 'vegan' ? 'Vegan' : 'Normal'}
          </Text>
        </View>
      </View>
    </Surface>
  );
}

const styles = StyleSheet.create({
  flex1: { flex: 1 },
  scrollContent: { padding: spacing.md, paddingBottom: spacing.xl },
  section: { marginBottom: spacing.lg },
  sectionHeader: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, marginBottom: spacing.md },
  historyCard: { borderRadius: radius.md, marginBottom: spacing.sm, overflow: 'hidden' },
  historyCardTop: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: 14 },
  historyCardBottom: { flexDirection: 'row', justifyContent: 'space-between', paddingHorizontal: 14, paddingVertical: spacing.sm },
  historyMeta: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  menuTypeDot: { width: 7, height: 7, borderRadius: 4 },
});
