import Ionicons from '@expo/vector-icons/Ionicons';
import React, { useMemo, useState } from 'react';
import { ActivityIndicator, Modal, Pressable, RefreshControl, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ScreenHeader } from '@/components/ScreenHeader';
import { Badge, Button, Card, Text } from '@/components/ui';
import { MONTHS_TR, WEEKDAYS_SHORT_TR, formatDateLongTR, nextDays, toISODate } from '@/constants/datetime';
import { useTheme } from '@/contexts/ThemeContext';
import {
  useCafeterias,
  useCancelReservation,
  useCreateReservation,
  useMonthlyMenu,
  useMyReservations,
} from '@/hooks/useMeals';
import { useHaptic } from '@/hooks/useHaptic';
import { COLORS } from '@/lib/theme';
import type { Cafeteria, MealTime, MenuType, Reservation } from '@/types/meal.types';

const MEAL_LABEL: Record<MealTime, string> = { lunch: 'Öğle', dinner: 'Akşam' };
const MENU_LABEL: Record<MenuType, string> = { normal: 'Normal', vegan: 'Vegan' };
const STATUS_LABEL: Record<string, string> = {
  confirmed: 'Onaylandı',
  pending: 'Ödeme bekleniyor',
  used: 'Kullanıldı',
  cancelled: 'İptal edildi',
};

function statusVariant(status: string): 'success' | 'warning' | 'secondary' | 'destructive' {
  if (status === 'confirmed') return 'success';
  if (status === 'pending') return 'warning';
  if (status === 'cancelled') return 'destructive';
  return 'secondary';
}

function Chip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
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

function ReservationCard({
  reservation,
  onCancel,
  canceling,
}: {
  reservation: Reservation;
  onCancel: () => void;
  canceling: boolean;
}) {
  const active = reservation.status === 'confirmed' || reservation.status === 'pending';
  return (
    <Card className="mb-3 p-4">
      <View className="flex-row items-start justify-between">
        <View className="flex-1 pr-3">
          <Text className="text-base font-bold text-card-foreground">
            {formatDateLongTR(reservation.date)}
          </Text>
          <Text className="mt-0.5 text-sm text-muted-foreground">
            {MEAL_LABEL[reservation.meal_time]} · {MENU_LABEL[reservation.menu_type]}
            {reservation.cafeteria?.name ? ` · ${reservation.cafeteria.name}` : ''}
          </Text>
        </View>
        <Badge variant={statusVariant(reservation.status)}>
          <Text className="text-xs font-semibold text-primary-foreground">
            {STATUS_LABEL[reservation.status] ?? reservation.status}
          </Text>
        </Badge>
      </View>
      {active && (
        <Button
          variant="outline"
          size="sm"
          className="mt-3 self-start"
          onPress={onCancel}
          loading={canceling}
          accessibilityLabel="Randevuyu iptal et"
        >
          <Text className="text-sm font-semibold text-destructive">İptal Et</Text>
        </Button>
      )}
    </Card>
  );
}

function NewReservationModal({
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
  const createMutation = useCreateReservation();

  const days = useMemo(() => nextDays(14), []);
  const [cafeteriaId, setCafeteriaId] = useState(cafeterias[0]?.id ?? '');
  const [date, setDate] = useState(toISODate(days[0]));
  const [mealTime, setMealTime] = useState<MealTime>('lunch');
  const [menuType, setMenuType] = useState<MenuType>('normal');
  const [error, setError] = useState<string | null>(null);

  const cafeteria = cafeterias.find((c) => c.id === cafeteriaId) ?? cafeterias[0];
  const selected = new Date(date);
  const menuQuery = useMonthlyMenu(selected.getFullYear(), selected.getMonth() + 1);
  const dayMenu = menuQuery.data?.menu_data?.[date];
  const dishes = mealTime === 'dinner' ? dayMenu?.dinner : dayMenu?.lunch;

  const canDinner = cafeteria?.serves_dinner ?? false;
  const canVegan = cafeteria?.has_vegan_menu ?? false;

  const submit = () => {
    if (!cafeteriaId) {
      setError('Lütfen bir yemekhane seç');
      return;
    }
    setError(null);
    haptic.light();
    createMutation.mutate(
      {
        cafeteria_id: cafeteriaId,
        date,
        meal_time: canDinner ? mealTime : 'lunch',
        menu_type: canVegan ? menuType : 'normal',
      },
      {
        onSuccess: () => {
          haptic.success();
          onClose();
        },
        onError: (err: any) => {
          haptic.error();
          setError(err.response?.data?.message ?? 'Randevu alınamadı. Tekrar dene.');
        },
      }
    );
  };

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <View className="flex-1 justify-end bg-black/50">
        <View className="max-h-[88%] rounded-t-4xl bg-background">
          <View className="flex-row items-center justify-between px-5 pb-2 pt-4">
            <Text className="text-xl font-extrabold text-foreground">Yeni Randevu</Text>
            <Pressable
              onPress={onClose}
              accessibilityRole="button"
              accessibilityLabel="Kapat"
              className="h-9 w-9 items-center justify-center rounded-full bg-secondary active:opacity-70"
            >
              <Ionicons name="close" size={20} color={colors.foreground} />
            </Pressable>
          </View>

          <ScrollView contentContainerClassName="px-5 pb-8 pt-2" keyboardShouldPersistTaps="handled">
            {/* Yemekhane */}
            <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Yemekhane
            </Text>
            <View className="mb-5 gap-2">
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
                  </View>
                  {c.id === cafeteriaId && (
                    <Ionicons name="checkmark-circle" size={22} color={colors.primary} />
                  )}
                </Pressable>
              ))}
            </View>

            {/* Tarih */}
            <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Tarih
            </Text>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerClassName="gap-2 pb-1"
              className="mb-5"
            >
              {days.map((d) => {
                const iso = toISODate(d);
                const isActive = iso === date;
                return (
                  <Pressable
                    key={iso}
                    onPress={() => {
                      haptic.selection();
                      setDate(iso);
                    }}
                    accessibilityRole="button"
                    className={`w-16 items-center rounded-2xl border py-3 active:opacity-70 ${
                      isActive ? 'border-primary bg-primary' : 'border-border bg-card'
                    }`}
                  >
                    <Text
                      className={`text-xs ${isActive ? 'text-primary-foreground/80' : 'text-muted-foreground'}`}
                    >
                      {WEEKDAYS_SHORT_TR[d.getDay()]}
                    </Text>
                    <Text
                      className={`text-lg font-extrabold ${isActive ? 'text-primary-foreground' : 'text-foreground'}`}
                    >
                      {d.getDate()}
                    </Text>
                    <Text
                      className={`text-[10px] ${isActive ? 'text-primary-foreground/80' : 'text-muted-foreground'}`}
                    >
                      {MONTHS_TR[d.getMonth()].slice(0, 3)}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>

            {/* Ogun */}
            <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Öğün
            </Text>
            <View className="mb-5 flex-row gap-2">
              <Chip label="Öğle" active={mealTime === 'lunch'} onPress={() => setMealTime('lunch')} />
              {canDinner && (
                <Chip label="Akşam" active={mealTime === 'dinner'} onPress={() => setMealTime('dinner')} />
              )}
            </View>

            {/* Menu tipi */}
            {canVegan && (
              <>
                <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                  Menü
                </Text>
                <View className="mb-5 flex-row gap-2">
                  <Chip label="Normal" active={menuType === 'normal'} onPress={() => setMenuType('normal')} />
                  <Chip label="Vegan" active={menuType === 'vegan'} onPress={() => setMenuType('vegan')} />
                </View>
              </>
            )}

            {/* Menu onizleme */}
            <Card className="mb-5 p-4">
              <Text className="mb-2 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                {formatDateLongTR(date)} · {MEAL_LABEL[mealTime]}
              </Text>
              {menuQuery.isLoading ? (
                <ActivityIndicator color={colors.primary} />
              ) : dishes && dishes.length > 0 ? (
                dishes.map((dish, i) => (
                  <View key={i} className="flex-row items-center gap-2 py-0.5">
                    <View className="h-1.5 w-1.5 rounded-full bg-primary" />
                    <Text className="text-sm text-foreground">{dish}</Text>
                  </View>
                ))
              ) : (
                <Text className="text-sm text-muted-foreground">Bu gün için menü henüz girilmedi.</Text>
              )}
            </Card>

            {error && (
              <View className="mb-4 flex-row items-center gap-2 rounded-2xl border border-destructive/30 bg-destructive/10 p-3">
                <Ionicons name="alert-circle" size={18} color={colors.destructive} />
                <Text className="flex-1 text-sm font-medium text-destructive">{error}</Text>
              </View>
            )}

            <Button
              size="lg"
              onPress={submit}
              loading={createMutation.isPending}
              accessibilityLabel="Randevuyu oluştur"
            >
              <Text>Randevu Al</Text>
            </Button>
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

export default function CafeteriaScreen() {
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const [modalVisible, setModalVisible] = useState(false);

  const cafeteriasQuery = useCafeterias();
  const reservationsQuery = useMyReservations();
  const cancelMutation = useCancelReservation();

  const reservations = reservationsQuery.data?.reservations ?? [];
  const cafeterias = cafeteriasQuery.data?.cafeterias ?? [];

  const handleCancel = (id: string) => {
    haptic.medium();
    cancelMutation.mutate(id);
  };

  return (
    <SafeAreaView className="flex-1 bg-background" edges={['top']}>
      <ScreenHeader title="Yemekhane" subtitle="Randevularını yönet" />

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
            <Text className="mb-3 font-mono text-xs uppercase tracking-widest text-muted-foreground">
              Randevularım
            </Text>
            {reservations.map((r) => (
              <ReservationCard
                key={r.id}
                reservation={r}
                onCancel={() => handleCancel(r.id)}
                canceling={cancelMutation.isPending && cancelMutation.variables === r.id}
              />
            ))}
          </Animated.View>
        ) : (
          <View className="items-center py-16">
            <Ionicons name="fast-food-outline" size={44} color={colors.mutedForeground} />
            <Text className="mt-3 text-center text-muted-foreground">
              Henüz randevun yok. Aşağıdan yeni randevu al.
            </Text>
          </View>
        )}
      </ScrollView>

      {/* Sabit alt aksiyon butonu */}
      <View className="absolute inset-x-0 bottom-0 border-t border-border bg-background px-5 pb-8 pt-3">
        <Button
          size="lg"
          onPress={() => {
            haptic.light();
            setModalVisible(true);
          }}
          disabled={cafeterias.length === 0}
          accessibilityLabel="Yeni randevu al"
        >
          <Ionicons name="add" size={20} color={colors.primaryForeground} />
          <Text>Yeni Randevu Al</Text>
        </Button>
      </View>

      {cafeterias.length > 0 && (
        <NewReservationModal
          visible={modalVisible}
          onClose={() => setModalVisible(false)}
          cafeterias={cafeterias}
        />
      )}
    </SafeAreaView>
  );
}
