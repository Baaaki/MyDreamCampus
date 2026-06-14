import { useState } from 'react';
import { StyleSheet, ScrollView, Pressable, View, Alert } from 'react-native';
import { Text, Surface, Button, Chip, ActivityIndicator, Portal, Modal, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';

import { useCafeterias, useCreateBatchReservation } from '@/hooks/useMeals';
import type { CreateReservationRequest } from '@/types/meal.types';
import { SectionHeader } from '@/components/ui';
import { spacing, radius, semanticColors } from '@/constants/tokens';
import {
  type MealType,
  MEAL_PRICE,
  monthNames,
  getWeekDates,
  fallbackWeeklyMenu,
  mealColors,
} from './helpers';

export function SelectTab() {
  const { colors } = useTheme();
  const { data: cafeteriaData, isLoading: loadingCafeterias } = useCafeterias();
  const batchMutation = useCreateBatchReservation();

  const cafeterias = cafeteriaData?.cafeterias?.filter((c) => c.is_active) ?? [];

  const [selectedCafeteria, setSelectedCafeteria] = useState<string | null>(null);
  const [selections, setSelections] = useState<Record<string, MealType>>({
    monday: 'none', tuesday: 'none', wednesday: 'none', thursday: 'none', friday: 'none',
  });
  const [showPayment, setShowPayment] = useState(false);
  const weekDates = getWeekDates();

  const selectedCount = Object.values(selections).filter((v) => v !== 'none').length;
  const totalPrice = selectedCount * MEAL_PRICE;

  const setMeal = (day: string, type: MealType) => {
    setSelections((prev) => ({ ...prev, [day]: type }));
  };

  const handleConfirm = async () => {
    if (!selectedCafeteria) return;
    const reservations: CreateReservationRequest[] = weekDates
      .filter((day) => selections[day.key] !== 'none')
      .map((day) => ({
        cafeteria_id: selectedCafeteria,
        date: day.fullDate,
        meal_time: 'lunch' as const,
        menu_type: selections[day.key] as 'normal' | 'vegan',
      }));

    try {
      const response = await batchMutation.mutateAsync({ reservations });
      setShowPayment(false);
      Alert.alert('Basarili', `Rezervasyonunuz olusturuldu. ${response.payment_url ? 'Odeme sayfasina yonlendirileceksiniz.' : ''}`);
      setSelections({ monday: 'none', tuesday: 'none', wednesday: 'none', thursday: 'none', friday: 'none' });
    } catch {
      Alert.alert('Hata', 'Rezervasyon olusturulurken bir hata olustu.');
    }
  };

  const getMenuItems = (dayKey: string, type: MealType) => {
    if (type === 'none') return fallbackWeeklyMenu[dayKey]?.normal ?? [];
    return type === 'vegan' ? (fallbackWeeklyMenu[dayKey]?.vegan ?? []) : (fallbackWeeklyMenu[dayKey]?.normal ?? []);
  };

  return (
    <>
      <ScrollView style={styles.flex1} contentContainerStyle={styles.scrollContent}>
        <SectionHeader title="Yemekhane Secimi" />

        {loadingCafeterias ? (
          <ActivityIndicator animating color={colors.primary} style={{ marginVertical: 20 }} />
        ) : cafeterias.length === 0 ? (
          <Surface style={[styles.infoBanner, { backgroundColor: semanticColors.warningLight }]} elevation={0} accessibilityRole="alert">
            <FontAwesome name="info-circle" size={14} color={semanticColors.warningDark} />
            <Text variant="bodySmall" style={{ color: '#92400e', flex: 1 }}>Yemekhane verisi yuklenemedi.</Text>
          </Surface>
        ) : (
          <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.cafeteriaScroll}>
            {cafeterias.map((cafe) => {
              const active = selectedCafeteria === cafe.id;
              return (
                <Chip
                  key={cafe.id}
                  selected={active}
                  onPress={() => setSelectedCafeteria(cafe.id)}
                  icon="map-marker"
                  mode={active ? 'flat' : 'outlined'}
                  style={[styles.cafeteriaChip, active && { backgroundColor: colors.primary }]}
                  textStyle={active ? { color: '#fff' } : undefined}
                  selectedColor={active ? '#fff' : colors.onSurface}
                  accessibilityRole="button"
                  accessibilityLabel={`${cafe.name} yemekhane${active ? ', secili' : ''}`}
                >
                  {cafe.name}
                </Chip>
              );
            })}
          </ScrollView>
        )}

        {selectedCafeteria && (
          <>
            <View style={styles.legendRow} accessibilityRole="text" accessibilityLabel="Renk aciklamasi">
              <View style={styles.legendItem}><View style={[styles.legendDot, { backgroundColor: '#9ca3af' }]} /><Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>Secilmedi</Text></View>
              <View style={styles.legendItem}><View style={[styles.legendDot, { backgroundColor: mealColors.normal }]} /><Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>Normal</Text></View>
              <View style={styles.legendItem}><View style={[styles.legendDot, { backgroundColor: mealColors.vegan }]} /><Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>Vegan</Text></View>
            </View>

            {weekDates.map((day) => {
              const sel = selections[day.key];
              const items = getMenuItems(day.key, sel);
              const borderColor = sel === 'normal' ? mealColors.normal : sel === 'vegan' ? mealColors.vegan : colors.outlineVariant;
              return (
                <Surface key={day.key} style={[styles.dayCard, { borderColor }]} elevation={1} accessibilityLabel={`${day.label}, ${sel === 'none' ? 'secilmedi' : sel}`}>
                  <View style={styles.dayHeader}>
                    <View>
                      <Text variant="titleSmall" style={{ color: colors.onSurface }}>{day.label}</Text>
                      <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>{day.day} {monthNames[day.month]}</Text>
                    </View>
                    {sel !== 'none' && (
                      <Pressable onPress={() => setMeal(day.key, 'none')} style={styles.cancelBtn} accessibilityRole="button" accessibilityLabel="Secimi kaldir">
                        <FontAwesome name="times-circle" size={20} color={colors.error} />
                      </Pressable>
                    )}
                  </View>
                  <View style={[styles.dayMenuItems, { opacity: sel === 'none' ? 0.4 : 1 }]}>
                    {items.map((item, idx) => (
                      <View key={idx} style={styles.dayMenuItem}>
                        <View style={[styles.dayMenuDot, { backgroundColor: sel === 'vegan' ? mealColors.vegan : sel === 'normal' ? mealColors.normal : '#d1d5db' }]} />
                        <Text variant="bodySmall" style={{ color: colors.onSurface }}>{item}</Text>
                      </View>
                    ))}
                  </View>
                  <View style={styles.mealTypeRow}>
                    <Button
                      mode={sel === 'normal' ? 'contained' : 'outlined'}
                      onPress={() => setMeal(day.key, 'normal')}
                      compact
                      buttonColor={sel === 'normal' ? mealColors.normal : undefined}
                      textColor={sel === 'normal' ? '#fff' : colors.onSurfaceVariant}
                      style={styles.mealTypeBtn}
                      accessibilityLabel="Normal menu sec"
                    >
                      Normal
                    </Button>
                    <Button
                      mode={sel === 'vegan' ? 'contained' : 'outlined'}
                      onPress={() => setMeal(day.key, 'vegan')}
                      compact
                      buttonColor={sel === 'vegan' ? mealColors.vegan : undefined}
                      textColor={sel === 'vegan' ? '#fff' : colors.onSurfaceVariant}
                      style={styles.mealTypeBtn}
                      accessibilityLabel="Vegan menu sec"
                    >
                      Vegan
                    </Button>
                  </View>
                </Surface>
              );
            })}
            <View style={{ height: 100 }} />
          </>
        )}
      </ScrollView>

      {selectedCafeteria && (
        <Surface style={[styles.bottomBar, { borderTopColor: colors.outlineVariant }]} elevation={3}>
          <View>
            <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>{selectedCount} ogun secildi</Text>
            <Text variant="headlineSmall" style={{ color: colors.primary, fontWeight: '700' }}>{totalPrice.toFixed(2)} TL</Text>
          </View>
          <Button
            mode="contained"
            onPress={() => setShowPayment(true)}
            disabled={selectedCount === 0}
            icon="credit-card"
            buttonColor={mealColors.pay}
            style={styles.payBtn}
            contentStyle={{ paddingVertical: 4 }}
            accessibilityLabel={`Odeme yap, ${totalPrice} TL`}
          >
            Odeme Yap
          </Button>
        </Surface>
      )}

      <Portal>
        <Modal visible={showPayment} onDismiss={() => setShowPayment(false)} contentContainerStyle={[styles.modalContent, { backgroundColor: colors.surface }]}>
          <Text variant="headlineSmall" style={{ color: colors.onSurface, fontWeight: '700' }}>Odeme Onayi</Text>
          <Text variant="bodyMedium" style={{ color: colors.onSurfaceVariant, marginTop: spacing.xs, marginBottom: spacing.md }}>
            Secilen yemekleri kontrol edin
          </Text>
          <Surface style={[styles.modalSummary, { backgroundColor: colors.surfaceVariant }]} elevation={0}>
            {weekDates.map((day) => {
              const sel = selections[day.key];
              if (sel === 'none') return null;
              return (
                <View key={day.key} style={styles.modalRow}>
                  <Text variant="bodyMedium" style={{ color: colors.onSurface, fontWeight: '500' }}>{day.label}</Text>
                  <Chip compact mode="flat" style={{ backgroundColor: sel === 'vegan' ? mealColors.veganLight : mealColors.normalLight }} textStyle={{ color: sel === 'vegan' ? mealColors.veganDark : mealColors.normalDark, fontSize: 12, fontWeight: '600' }}>
                    {sel === 'vegan' ? 'Vegan' : 'Normal'}
                  </Chip>
                </View>
              );
            })}
          </Surface>
          <Surface style={[styles.modalTotal, { backgroundColor: semanticColors.successLight }]} elevation={0}>
            <Text variant="bodyMedium" style={{ color: '#065f46', fontWeight: '500' }}>Toplam Tutar</Text>
            <Text variant="headlineSmall" style={{ color: semanticColors.success, fontWeight: '700' }}>{totalPrice.toFixed(2)} TL</Text>
          </Surface>
          <View style={styles.modalBtns}>
            <Button mode="outlined" onPress={() => setShowPayment(false)} style={styles.modalCancelBtn} accessibilityLabel="Iptal">Iptal</Button>
            <Button
              mode="contained"
              onPress={handleConfirm}
              loading={batchMutation.isPending}
              disabled={batchMutation.isPending}
              icon="credit-card"
              buttonColor={mealColors.pay}
              style={styles.modalConfirmBtn}
              accessibilityLabel="Odemeyi onayla"
            >
              Odemeyi Onayla
            </Button>
          </View>
        </Modal>
      </Portal>
    </>
  );
}

const styles = StyleSheet.create({
  flex1: { flex: 1 },
  scrollContent: { padding: spacing.md, paddingBottom: spacing.xl },
  infoBanner: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, borderRadius: radius.md, padding: spacing.md, marginBottom: spacing.md },
  cafeteriaScroll: { marginBottom: spacing.md },
  cafeteriaChip: { marginRight: spacing.sm },
  legendRow: { flexDirection: 'row', gap: spacing.md, marginBottom: spacing.md },
  legendItem: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  legendDot: { width: 10, height: 10, borderRadius: 5 },
  dayCard: { borderRadius: radius.lg, borderWidth: 2, padding: 14, marginBottom: spacing.md },
  dayHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: spacing.sm },
  cancelBtn: { padding: 6 },
  dayMenuItems: { marginBottom: spacing.sm },
  dayMenuItem: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, paddingVertical: 3 },
  dayMenuDot: { width: 5, height: 5, borderRadius: 3 },
  mealTypeRow: { flexDirection: 'row', gap: spacing.sm },
  mealTypeBtn: { flex: 1 },
  bottomBar: { position: 'absolute', bottom: 0, left: 0, right: 0, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: spacing.lg, paddingVertical: 14, borderTopWidth: 1 },
  payBtn: { borderRadius: radius.md },
  modalContent: { borderTopLeftRadius: radius.xl, borderTopRightRadius: radius.xl, padding: spacing.lg, marginTop: 'auto' },
  modalSummary: { borderRadius: radius.md, padding: 14, marginBottom: spacing.md },
  modalRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 6 },
  modalTotal: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', borderRadius: radius.md, padding: spacing.md, marginBottom: spacing.lg },
  modalBtns: { flexDirection: 'row', gap: spacing.md },
  modalCancelBtn: { flex: 1 },
  modalConfirmBtn: { flex: 2 },
});
