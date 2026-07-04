import Ionicons from '@expo/vector-icons/Ionicons';
import React, { useMemo, useState } from 'react';
import { ActivityIndicator, Alert, Modal, Pressable, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ScreenHeader } from '@/components/ScreenHeader';
import { Badge, Button, Card, Text } from '@/components/ui';
import { formatDateLongTR, isCancellable, nextWeekWeekdays, toISODate } from '@/constants/datetime';
import { MEAL_LABEL, MENU_LABEL, MEAL_PRICE_TRY, STATUS_LABEL, formatTRY, statusVariant } from '@/constants/meal';
import { useTheme } from '@/contexts/ThemeContext';
import { useHaptic } from '@/hooks/useHaptic';
import {
  useCafeterias,
  useCancelReservation,
  useCancelReservations,
  useCreateBatchReservation,
  useMonthlyMenu,
  useMyReservations,
} from '@/hooks/useMeals';
import { COLORS } from '@/lib/theme';
import type { Cafeteria, DailyMenu, MealTime, MenuType, Reservation } from '@/types/meal.types';

// ---------------------------------------------------------------------------
// Yardimci parcalar
// ---------------------------------------------------------------------------

function Chip({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      className={`rounded-full border px-4 py-2 active:opacity-70 ${
        active ? 'border-primary bg-primary' : 'border-border bg-card'
      }`}
    >
      <Text className={active ? 'text-sm font-semibold text-primary-foreground' : 'text-sm text-foreground'}>
        {label}
      </Text>
    </Pressable>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
      {children}
    </Text>
  );
}

// ---------------------------------------------------------------------------
// Randevu karti (liste) — secim modu + tekil iptal
// ---------------------------------------------------------------------------

function ReservationCard({
  reservation,
  selectMode,
  selected,
  onToggleSelect,
  onCancel,
  canceling,
}: {
  reservation: Reservation;
  selectMode: boolean;
  selected: boolean;
  onToggleSelect: () => void;
  onCancel: () => void;
  canceling: boolean;
}) {
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const active = reservation.status === 'confirmed' || reservation.status === 'pending';
  const cancellable = active && !reservation.is_used && isCancellable(reservation.date);

  return (
    <Pressable
      onPress={selectMode && cancellable ? onToggleSelect : undefined}
      accessibilityRole={selectMode && cancellable ? 'checkbox' : undefined}
      className={`mb-3 rounded-3xl border ${
        selected ? 'border-primary bg-primary/5' : 'border-border bg-card'
      }`}
    >
      <View className="flex-row items-start justify-between p-4">
        {selectMode && (
          <View className="mr-3 mt-0.5">
            {cancellable ? (
              <Ionicons
                name={selected ? 'checkbox' : 'square-outline'}
                size={22}
                color={selected ? colors.primary : colors.mutedForeground}
              />
            ) : (
              <Ionicons name="lock-closed" size={20} color={colors.mutedForeground} />
            )}
          </View>
        )}
        <View className="flex-1 pr-3">
          <Text className="text-base font-bold text-card-foreground">
            {formatDateLongTR(reservation.date)}
          </Text>
          <Text className="mt-0.5 text-sm text-muted-foreground">
            {MEAL_LABEL[reservation.meal_time]} · {MENU_LABEL[reservation.menu_type]}
            {reservation.cafeteria?.name ? ` · ${reservation.cafeteria.name}` : ''}
          </Text>
          {active && !cancellable && !reservation.is_used && (
            <Text className="mt-1 text-xs text-muted-foreground">
              İptal süresi doldu (cuma kilidi)
            </Text>
          )}
        </View>
        <Badge variant={statusVariant(reservation.status)}>
          <Text className="text-xs font-semibold text-primary-foreground">
            {STATUS_LABEL[reservation.status] ?? reservation.status}
          </Text>
        </Badge>
      </View>
      {!selectMode && cancellable && (
        <View className="px-4 pb-4">
          <Button
            variant="outline"
            size="sm"
            className="self-start"
            onPress={onCancel}
            loading={canceling}
            accessibilityLabel="Randevuyu iptal et"
          >
            <Text className="text-sm font-semibold text-destructive">İptal Et</Text>
          </Button>
        </View>
      )}
    </Pressable>
  );
}

// ---------------------------------------------------------------------------
// Yeni randevu sihirbazi
// ---------------------------------------------------------------------------

const STEP_TITLES = ['Yemekhane', 'Öğün & Menü', 'Günler', 'Öde'];

function StepDots({ step }: { step: number }) {
  return (
    <View className="flex-row items-center gap-1.5">
      {STEP_TITLES.map((_, i) => (
        <View
          key={i}
          className={`h-1.5 rounded-full ${i === step ? 'w-6 bg-primary' : 'w-1.5 bg-border'}`}
        />
      ))}
    </View>
  );
}

function DayMenuRow({ menu, mealTimes }: { menu?: DailyMenu; mealTimes: MealTime[] }) {
  const dishes = mealTimes.flatMap((mt) => menu?.[mt] ?? []);
  if (dishes.length === 0) {
    return <Text className="text-xs text-muted-foreground">Bu gün için menü henüz girilmedi.</Text>;
  }
  return (
    <Text className="text-xs leading-5 text-muted-foreground" numberOfLines={2}>
      {dishes.join(' · ')}
    </Text>
  );
}

function ReservationWizard({
  visible,
  onClose,
  cafeterias,
}: {
  visible: boolean;
  onClose: () => void;
  cafeterias: Cafeteria[];
}) {
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const batchMutation = useCreateBatchReservation();

  const days = useMemo(() => nextWeekWeekdays(), []);

  const [step, setStep] = useState(0);
  const [cafeteriaId, setCafeteriaId] = useState(cafeterias[0]?.id ?? '');
  const [mealTimes, setMealTimes] = useState<MealTime[]>(['lunch']);
  const [menuType, setMenuType] = useState<MenuType>('normal');
  const [dates, setDates] = useState<string[]>([toISODate(days[0])]);
  const [error, setError] = useState<string | null>(null);

  const cafeteria = cafeterias.find((c) => c.id === cafeteriaId) ?? cafeterias[0];
  const canDinner = cafeteria?.serves_dinner ?? false;
  const canVegan = cafeteria?.has_vegan_menu ?? false;

  // Gelecek hafta iki aya yayilabilir (ay sonu); iki ay menusunu birlestir.
  const menuA = useMonthlyMenu(days[0].getFullYear(), days[0].getMonth() + 1);
  const menuB = useMonthlyMenu(days[4].getFullYear(), days[4].getMonth() + 1);
  const menuByDate: Record<string, DailyMenu> = {
    ...(menuA.data?.menu_data ?? {}),
    ...(menuB.data?.menu_data ?? {}),
  };
  const menuLoading = menuA.isLoading || menuB.isLoading;

  const effectiveMenuType: MenuType = canVegan ? menuType : 'normal';
  const effectiveMeals = mealTimes.filter((mt) => mt === 'lunch' || canDinner);

  // Kombinasyon: secili gunler x secili ogunler.
  const items = dates.flatMap((date) =>
    effectiveMeals.map((mt) => ({
      cafeteria_id: cafeteriaId,
      date,
      meal_time: mt,
      menu_type: effectiveMenuType,
    }))
  );
  const total = items.length * MEAL_PRICE_TRY;

  const reset = () => {
    setStep(0);
    setMealTimes(['lunch']);
    setMenuType('normal');
    setDates([toISODate(days[0])]);
    setError(null);
  };

  const close = () => {
    reset();
    onClose();
  };

  const toggleMeal = (mt: MealTime) => {
    haptic.selection();
    setMealTimes((prev) =>
      prev.includes(mt) ? prev.filter((x) => x !== mt) : [...prev, mt]
    );
  };

  const toggleDate = (iso: string) => {
    haptic.selection();
    setDates((prev) => (prev.includes(iso) ? prev.filter((x) => x !== iso) : [...prev, iso]));
  };

  const next = () => {
    setError(null);
    if (step === 0 && !cafeteriaId) {
      setError('Lütfen bir yemekhane seç.');
      return;
    }
    if (step === 1 && effectiveMeals.length === 0) {
      setError('En az bir öğün seç.');
      return;
    }
    if (step === 2 && dates.length === 0) {
      setError('En az bir gün seç.');
      return;
    }
    haptic.light();
    setStep((s) => Math.min(s + 1, 3));
  };

  const back = () => {
    haptic.light();
    setError(null);
    setStep((s) => Math.max(s - 1, 0));
  };

  const submit = () => {
    if (items.length === 0) {
      setError('Randevu için gün ve öğün seç.');
      return;
    }
    setError(null);
    haptic.medium();
    batchMutation.mutate(
      { reservations: items },
      {
        onSuccess: () => {
          haptic.success();
          close();
        },
        onError: (err: any) => {
          haptic.error();
          const code = err?.response?.data?.error?.code;
          if (code === 'RESERVATION_CONFLICTS') {
            setError('Seçtiğin bazı gün/öğünler için zaten randevun var. Onları çıkar ve tekrar dene.');
          } else {
            setError(err?.response?.data?.error?.message ?? 'Randevu alınamadı. Tekrar dene.');
          }
        },
      }
    );
  };

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={close}>
      <View className="flex-1 justify-end bg-black/50">
        <View className="max-h-[90%] rounded-t-4xl bg-background">
          {/* Baslik */}
          <View className="flex-row items-center justify-between px-5 pb-2 pt-4">
            <View>
              <Text className="text-xl font-extrabold text-foreground">Yeni Randevu</Text>
              <Text className="text-xs text-muted-foreground">
                Gelecek hafta · {STEP_TITLES[step]}
              </Text>
            </View>
            <Pressable
              onPress={close}
              accessibilityRole="button"
              accessibilityLabel="Kapat"
              className="h-9 w-9 items-center justify-center rounded-full bg-secondary active:opacity-70"
            >
              <Ionicons name="close" size={20} color={colors.foreground} />
            </Pressable>
          </View>
          <View className="px-5 pb-2">
            <StepDots step={step} />
          </View>

          <ScrollView contentContainerClassName="px-5 pb-4 pt-2" keyboardShouldPersistTaps="handled">
            {/* Adim 1: Yemekhane */}
            {step === 0 && (
              <Animated.View entering={FadeInDown.duration(250)}>
                <SectionLabel>Yemekhane seç</SectionLabel>
                <View className="gap-2">
                  {cafeterias.map((c) => (
                    <Pressable
                      key={c.id}
                      onPress={() => {
                        haptic.selection();
                        setCafeteriaId(c.id);
                      }}
                      accessibilityRole="button"
                      className={`flex-row items-center justify-between rounded-2xl border p-4 active:opacity-70 ${
                        c.id === cafeteriaId ? 'border-primary bg-primary/5' : 'border-border bg-card'
                      }`}
                    >
                      <View className="flex-1 pr-3">
                        <Text className="font-semibold text-foreground">{c.name}</Text>
                        <Text className="text-xs text-muted-foreground">{c.location}</Text>
                        <View className="mt-1 flex-row gap-1.5">
                          {c.serves_dinner && (
                            <Badge variant="secondary">
                              <Text className="text-[10px] font-semibold text-secondary-foreground">Akşam var</Text>
                            </Badge>
                          )}
                          {c.has_vegan_menu && (
                            <Badge variant="secondary">
                              <Text className="text-[10px] font-semibold text-secondary-foreground">Vegan var</Text>
                            </Badge>
                          )}
                        </View>
                      </View>
                      {c.id === cafeteriaId && (
                        <Ionicons name="checkmark-circle" size={22} color={colors.primary} />
                      )}
                    </Pressable>
                  ))}
                </View>
              </Animated.View>
            )}

            {/* Adim 2: Ogun & Menu */}
            {step === 1 && (
              <Animated.View entering={FadeInDown.duration(250)}>
                <SectionLabel>Öğün (birden fazla seçebilirsin)</SectionLabel>
                <View className="mb-5 flex-row gap-2">
                  <Chip label="Öğle" active={mealTimes.includes('lunch')} onPress={() => toggleMeal('lunch')} />
                  {canDinner ? (
                    <Chip label="Akşam" active={mealTimes.includes('dinner')} onPress={() => toggleMeal('dinner')} />
                  ) : (
                    <View className="rounded-full border border-dashed border-border px-4 py-2">
                      <Text className="text-sm text-muted-foreground">Akşam yok</Text>
                    </View>
                  )}
                </View>

                <SectionLabel>Menü tipi</SectionLabel>
                <View className="flex-row gap-2">
                  <Chip label="Normal" active={effectiveMenuType === 'normal'} onPress={() => setMenuType('normal')} />
                  {canVegan ? (
                    <Chip label="Vegan" active={effectiveMenuType === 'vegan'} onPress={() => setMenuType('vegan')} />
                  ) : (
                    <View className="rounded-full border border-dashed border-border px-4 py-2">
                      <Text className="text-sm text-muted-foreground">Vegan yok</Text>
                    </View>
                  )}
                </View>
              </Animated.View>
            )}

            {/* Adim 3: Gunler + menu onizleme */}
            {step === 2 && (
              <Animated.View entering={FadeInDown.duration(250)}>
                <SectionLabel>Gelecek hafta — günleri seç</SectionLabel>
                {menuLoading && (
                  <View className="items-center py-3">
                    <ActivityIndicator color={colors.primary} />
                  </View>
                )}
                <View className="gap-2">
                  {days.map((d) => {
                    const iso = toISODate(d);
                    const isActive = dates.includes(iso);
                    return (
                      <Pressable
                        key={iso}
                        onPress={() => toggleDate(iso)}
                        accessibilityRole="checkbox"
                        className={`rounded-2xl border p-3 active:opacity-70 ${
                          isActive ? 'border-primary bg-primary/5' : 'border-border bg-card'
                        }`}
                      >
                        <View className="flex-row items-center gap-3">
                          <Ionicons
                            name={isActive ? 'checkbox' : 'square-outline'}
                            size={22}
                            color={isActive ? colors.primary : colors.mutedForeground}
                          />
                          <View className="flex-1">
                            <Text className="font-semibold text-foreground">
                              {formatDateLongTR(iso)}
                            </Text>
                            <View className="mt-1">
                              <DayMenuRow menu={menuByDate[iso]} mealTimes={effectiveMeals} />
                            </View>
                          </View>
                        </View>
                      </Pressable>
                    );
                  })}
                </View>
              </Animated.View>
            )}

            {/* Adim 4: Ozet + fiyat */}
            {step === 3 && (
              <Animated.View entering={FadeInDown.duration(250)}>
                <SectionLabel>Özet</SectionLabel>
                <Card className="mb-4 p-4">
                  <View className="mb-2 flex-row items-center justify-between">
                    <Text className="text-muted-foreground">Yemekhane</Text>
                    <Text className="font-semibold text-card-foreground">{cafeteria?.name}</Text>
                  </View>
                  <View className="mb-2 flex-row items-center justify-between">
                    <Text className="text-muted-foreground">Öğün</Text>
                    <Text className="font-semibold text-card-foreground">
                      {effectiveMeals.map((mt) => MEAL_LABEL[mt]).join(' + ')} · {MENU_LABEL[effectiveMenuType]}
                    </Text>
                  </View>
                  <View className="flex-row items-center justify-between">
                    <Text className="text-muted-foreground">Gün sayısı</Text>
                    <Text className="font-semibold text-card-foreground">{dates.length} gün</Text>
                  </View>
                </Card>

                <SectionLabel>Seçilen günler</SectionLabel>
                <View className="mb-4 gap-1.5">
                  {[...dates].sort().map((iso) => (
                    <View key={iso} className="flex-row items-center gap-2">
                      <View className="h-1.5 w-1.5 rounded-full bg-primary" />
                      <Text className="text-sm text-foreground">{formatDateLongTR(iso)}</Text>
                    </View>
                  ))}
                </View>

                <View className="flex-row items-center justify-between rounded-2xl border border-border bg-card p-4">
                  <View>
                    <Text className="text-sm text-muted-foreground">
                      {items.length} öğün × {formatTRY(MEAL_PRICE_TRY)}
                    </Text>
                    <Text className="text-xs text-muted-foreground">Toplam tutar</Text>
                  </View>
                  <Text className="text-2xl font-extrabold text-foreground">{formatTRY(total)}</Text>
                </View>
              </Animated.View>
            )}

            {error && (
              <View className="mt-4 flex-row items-center gap-2 rounded-2xl border border-destructive/30 bg-destructive/10 p-3">
                <Ionicons name="alert-circle" size={18} color={colors.destructive} />
                <Text className="flex-1 text-sm font-medium text-destructive">{error}</Text>
              </View>
            )}
          </ScrollView>

          {/* Alt navigasyon */}
          <View className="flex-row gap-3 border-t border-border px-5 pb-8 pt-3">
            {step > 0 && (
              <Button variant="outline" className="flex-1" onPress={back} accessibilityLabel="Geri">
                <Text>Geri</Text>
              </Button>
            )}
            {step < 3 ? (
              <Button className="flex-1" onPress={next} accessibilityLabel="Devam">
                <Text>Devam</Text>
              </Button>
            ) : (
              <Button
                className="flex-1"
                onPress={submit}
                loading={batchMutation.isPending}
                accessibilityLabel="Öde ve onayla"
              >
                <Text>{formatTRY(total)} Öde ve Onayla</Text>
              </Button>
            )}
          </View>
        </View>
      </View>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Ekran
// ---------------------------------------------------------------------------

export default function CafeteriaScreen() {
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const [wizardVisible, setWizardVisible] = useState(false);
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  const cafeteriasQuery = useCafeterias();
  const reservationsQuery = useMyReservations();
  const cancelMutation = useCancelReservation();
  const bulkCancelMutation = useCancelReservations();

  const cafeterias = cafeteriasQuery.data?.cafeterias ?? [];
  const reservations = useMemo(
    () => [...(reservationsQuery.data?.reservations ?? [])].sort((a, b) => a.date.localeCompare(b.date)),
    [reservationsQuery.data]
  );

  const cancellableIds = reservations
    .filter((r) => (r.status === 'confirmed' || r.status === 'pending') && !r.is_used && isCancellable(r.date))
    .map((r) => r.id);

  const toggleSelect = (id: string) =>
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const exitSelectMode = () => {
    setSelectMode(false);
    setSelectedIds([]);
  };

  const handleSingleCancel = (id: string) => {
    haptic.medium();
    cancelMutation.mutate(id);
  };

  const handleBulkCancel = () => {
    if (selectedIds.length === 0) return;
    Alert.alert(
      'Randevuları iptal et',
      `${selectedIds.length} randevu iptal edilecek. Emin misin?`,
      [
        { text: 'Vazgeç', style: 'cancel' },
        {
          text: 'İptal Et',
          style: 'destructive',
          onPress: () => {
            haptic.medium();
            bulkCancelMutation.mutate(selectedIds, {
              onSuccess: (res) => {
                exitSelectMode();
                if (res.failed > 0) {
                  Alert.alert(
                    'Kısmi iptal',
                    `${res.total - res.failed} randevu iptal edildi, ${res.failed} tanesi iptal edilemedi (kilit süresi geçmiş olabilir).`
                  );
                }
              },
            });
          },
        },
      ]
    );
  };

  const hasCancellable = cancellableIds.length > 0;

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScreenHeader title="Yemekhane" subtitle="Gelecek hafta randevuların" />

      <ScrollView
        contentContainerClassName="px-5 pb-28 pt-1"
        refreshControl={
          <RefreshControl
            refreshing={reservationsQuery.isRefetching}
            onRefresh={() => reservationsQuery.refetch()}
            tintColor={colors.primary}
          />
        }
      >
        {reservationsQuery.isLoading ? (
          <View className="items-center py-16">
            <ActivityIndicator size="large" color={colors.primary} />
          </View>
        ) : reservations.length > 0 ? (
          <Animated.View entering={FadeInDown.duration(400)}>
            <View className="mb-3 flex-row items-center justify-between">
              <Text className="font-mono text-xs uppercase tracking-widest text-muted-foreground">
                Randevularım
              </Text>
              {hasCancellable &&
                (selectMode ? (
                  <Pressable onPress={exitSelectMode} accessibilityRole="button" className="active:opacity-70">
                    <Text className="text-sm font-semibold text-primary">Bitti</Text>
                  </Pressable>
                ) : (
                  <Pressable
                    onPress={() => {
                      haptic.light();
                      setSelectMode(true);
                    }}
                    accessibilityRole="button"
                    className="active:opacity-70"
                  >
                    <Text className="text-sm font-semibold text-primary">Toplu Seç</Text>
                  </Pressable>
                ))}
            </View>

            {reservations.map((r) => (
              <ReservationCard
                key={r.id}
                reservation={r}
                selectMode={selectMode}
                selected={selectedIds.includes(r.id)}
                onToggleSelect={() => toggleSelect(r.id)}
                onCancel={() => handleSingleCancel(r.id)}
                canceling={cancelMutation.isPending && cancelMutation.variables === r.id}
              />
            ))}
          </Animated.View>
        ) : (
          <View className="items-center py-16">
            <Ionicons name="fast-food-outline" size={44} color={colors.mutedForeground} />
            <Text className="mt-3 text-center text-muted-foreground">
              Gelecek hafta için randevun yok. Aşağıdan yeni randevu al.
            </Text>
          </View>
        )}
      </ScrollView>

      {/* Sabit alt aksiyon */}
      <View className="absolute inset-x-0 bottom-0 border-t border-border bg-background px-5 pb-8 pt-3">
        {selectMode ? (
          <Button
            size="lg"
            variant="destructive"
            onPress={handleBulkCancel}
            loading={bulkCancelMutation.isPending}
            disabled={selectedIds.length === 0}
            accessibilityLabel="Seçili randevuları iptal et"
          >
            <Ionicons name="trash-outline" size={20} color={colors.primaryForeground} />
            <Text>Seçili İptal Et ({selectedIds.length})</Text>
          </Button>
        ) : (
          <Button
            size="lg"
            onPress={() => {
              haptic.light();
              setWizardVisible(true);
            }}
            disabled={cafeterias.length === 0}
            accessibilityLabel="Yeni randevu al"
          >
            <Ionicons name="add" size={20} color={colors.primaryForeground} />
            <Text>Yeni Randevu Al</Text>
          </Button>
        )}
      </View>

      {cafeterias.length > 0 && (
        <ReservationWizard
          visible={wizardVisible}
          onClose={() => setWizardVisible(false)}
          cafeterias={cafeterias}
        />
      )}
    </SafeAreaView>
  );
}
