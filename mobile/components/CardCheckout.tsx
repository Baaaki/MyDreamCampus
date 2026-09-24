import Ionicons from '@expo/vector-icons/Ionicons';
import React, { useState } from 'react';
import { ActivityIndicator, KeyboardAvoidingView, Platform, ScrollView, View } from 'react-native';
import Animated, { FadeInDown } from 'react-native-reanimated';

import { Button, Input, Text } from '@/components/ui';
import { formatTRY } from '@/constants/meal';
import { useTheme } from '@/contexts/ThemeContext';
import { useHaptic } from '@/hooks/useHaptic';
import { useCheckoutOutcome, useConfirmPayment } from '@/hooks/useMeals';
import { apiErrorMessage } from '@/lib/api-error';
import {
  EMPTY_CARD_FORM,
  TEST_CARDS,
  formatCardNumber,
  formatExpiry,
  toConfirmRequest,
  validateCardForm,
  type CardForm,
  type CardFormErrors,
} from '@/lib/payment-card';
import { COLORS } from '@/lib/theme';
import type { Payment } from '@/types/payment.types';

/**
 * The reservations one batch created and the payment that covers them. It
 * lives in the wizard, not here: the Modal unmounts its content on close, and
 * a checkout reopened after the card was sent must show that result.
 */
export interface Checkout {
  paymentId: string;
  reservationIds: string[];
  amount: number;
  expiresAt: string;
  /** The settled payment and when the card was sent. */
  payment?: Payment;
  confirmedAt?: number;
  /** Why the payment can no longer be completed (it expired or was settled). */
  rejection?: string;
}

function isConflict(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    'response' in err &&
    typeof err.response === 'object' &&
    err.response !== null &&
    'status' in err.response &&
    err.response.status === 409
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <View className="gap-1.5">
      <Text className="text-sm font-medium text-foreground">{label}</Text>
      {children}
      {error && <Text className="text-xs text-destructive">{error}</Text>}
    </View>
  );
}

/**
 * Card form → confirm → wait for meal → result, for a batch that is already
 * reserved. onPaid closes the wizard; onRetry starts a new reservation, since
 * a declined or expired payment cannot be charged again.
 */
export function CardCheckout({
  checkout,
  onChange,
  onPaid,
  onRetry,
  onClose,
}: {
  checkout: Checkout;
  onChange: (next: Checkout) => void;
  onPaid: () => void;
  onRetry: () => void;
  onClose: () => void;
}) {
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];
  const confirm = useConfirmPayment();

  const [form, setForm] = useState<CardForm>(EMPTY_CARD_FORM);
  const [formErrors, setFormErrors] = useState<CardFormErrors>({});
  const [error, setError] = useState<string | null>(null);

  const payment = checkout.payment ?? null;
  const startedAt = checkout.confirmedAt ?? null;
  const { outcome, timedOut } = useCheckoutOutcome(checkout.reservationIds, startedAt);
  const declineReason =
    payment?.status === 'failed' ? (payment.failure_reason ?? 'Ödeme reddedildi') : null;

  type Phase = 'card' | 'waiting' | 'paid' | 'failed' | 'late';
  let phase: Phase = startedAt === null ? 'card' : 'waiting';
  if (checkout.rejection) phase = 'failed';
  else if (startedAt !== null) {
    if (outcome === 'confirmed') phase = 'paid';
    else if (outcome === 'cancelled') phase = 'failed';
    // A declined card needs no confirmation from meal to be reported.
    else if (timedOut) phase = declineReason ? 'failed' : 'late';
  }

  const update = (field: keyof CardForm, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setFormErrors((prev) => ({ ...prev, [field]: undefined }));
  };

  const fillTestCard = (number: string) => {
    haptic.selection();
    setForm((prev) => ({
      number,
      expiry: prev.expiry || '12/30',
      cvc: prev.cvc || '123',
      holder: prev.holder,
    }));
    setFormErrors({});
  };

  const submit = () => {
    const errors = validateCardForm(form);
    setFormErrors(errors);
    setError(null);
    if (Object.keys(errors).length > 0) {
      haptic.error();
      return;
    }
    haptic.medium();
    confirm.mutate(
      { paymentId: checkout.paymentId, card: toConfirmRequest(form) },
      {
        onSuccess: (result) => {
          // The card data has done its job; do not keep it around.
          setForm(EMPTY_CARD_FORM);
          onChange({ ...checkout, payment: result, confirmedAt: Date.now() });
        },
        onError: (err) => {
          haptic.error();
          // 409: the payment expired or was settled elsewhere — this checkout
          // is over. Anything else (a mistyped card) can be fixed and retried.
          if (isConflict(err)) {
            setForm(EMPTY_CARD_FORM);
            onChange({
              ...checkout,
              rejection: apiErrorMessage(err, 'Bu ödeme artık tamamlanamaz'),
            });
            return;
          }
          setError(apiErrorMessage(err, 'Ödeme gönderilemedi. Tekrar dene.'));
        },
      },
    );
  };

  if (phase === 'waiting') {
    return (
      <View className="items-center gap-3 px-5 py-16">
        <ActivityIndicator size="large" color={colors.primary} />
        <Text className="text-center text-muted-foreground">
          Ödeme işleniyor, randevu onayı bekleniyor…
        </Text>
      </View>
    );
  }

  if (phase !== 'card') {
    const success = phase === 'paid' || phase === 'late';
    const message =
      phase === 'paid'
        ? payment?.card_last4
          ? `${payment.card_brand ?? 'Kart'} •••• ${payment.card_last4} ile ${formatTRY(payment.amount)} ödendi.`
          : 'Ödemen alındı.'
        : phase === 'late'
          ? 'Ödemen alındı. Randevu onayı biraz gecikti; durumunu randevu listesinden takip edebilirsin.'
          : `${checkout.rejection ?? declineReason ?? 'Randevu iptal edildi'}. Öğünler ayrılmadı; tekrar deneyebilirsin.`;
    return (
      <Animated.View
        entering={FadeInDown.duration(250)}
        className="items-center gap-4 px-5 pb-8 pt-6"
      >
        <View
          className={`h-16 w-16 items-center justify-center rounded-full ${
            success ? 'bg-primary/10' : 'bg-destructive/10'
          }`}
        >
          <Ionicons
            name={success ? 'checkmark' : 'close'}
            size={32}
            color={success ? colors.primary : colors.destructive}
          />
        </View>
        <Text className="text-xl font-extrabold text-foreground">
          {phase === 'paid'
            ? 'Randevu Onaylandı'
            : phase === 'late'
              ? 'Ödeme Alındı'
              : 'Ödeme Başarısız'}
        </Text>
        <Text className="text-center text-sm text-muted-foreground">{message}</Text>
        <View className="w-full gap-2">
          {success ? (
            <Button onPress={onPaid} accessibilityLabel="Tamam">
              <Text>Tamam</Text>
            </Button>
          ) : (
            <>
              <Button onPress={onRetry} accessibilityLabel="Tekrar dene">
                <Text>Tekrar Dene</Text>
              </Button>
              <Button variant="outline" onPress={onClose} accessibilityLabel="Kapat">
                <Text>Kapat</Text>
              </Button>
            </>
          )}
        </View>
      </Animated.View>
    );
  }

  return (
    <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'}>
      <ScrollView
        contentContainerClassName="gap-4 px-5 pb-4 pt-2"
        keyboardShouldPersistTaps="handled"
      >
        <Text className="text-sm text-muted-foreground">
          {checkout.reservationIds.length} öğün için {formatTRY(checkout.amount)} ödenecek. Ödemeyi{' '}
          {new Date(checkout.expiresAt).toLocaleTimeString('tr-TR', {
            hour: '2-digit',
            minute: '2-digit',
          })}{' '}
          saatine kadar tamamla.
        </Text>

        <View className="gap-2 rounded-2xl border border-border bg-card p-4">
          <Text className="font-semibold text-foreground">Demo ödeme: gerçek para çekilmez.</Text>
          <Text className="text-xs text-muted-foreground">
            Test kartlarından birini kullan (son kullanma ileri bir tarih, CVC herhangi 3 hane):
          </Text>
          {TEST_CARDS.map((card) => (
            <View key={card.number} className="flex-row items-center justify-between gap-2">
              <View className="flex-1">
                <Text className="font-mono text-sm text-foreground">{card.number}</Text>
                <Text className="text-xs text-muted-foreground">{card.result}</Text>
              </View>
              <Button
                size="sm"
                variant="outline"
                onPress={() => fillTestCard(card.number)}
                accessibilityLabel={`${card.number} test kartını kullan`}
              >
                <Text>Kullan</Text>
              </Button>
            </View>
          ))}
        </View>

        <Field label="Kart Numarası" error={formErrors.number}>
          <Input
            value={form.number}
            onChangeText={(v) => update('number', formatCardNumber(v))}
            placeholder="0000 0000 0000 0000"
            keyboardType="number-pad"
            autoComplete="cc-number"
            accessibilityLabel="Kart numarası"
          />
        </Field>
        <View className="flex-row gap-3">
          <View className="flex-1">
            <Field label="Son Kullanma" error={formErrors.expiry}>
              <Input
                value={form.expiry}
                onChangeText={(v) => update('expiry', formatExpiry(v))}
                placeholder="AA/YY"
                keyboardType="number-pad"
                autoComplete="cc-exp"
                accessibilityLabel="Son kullanma tarihi"
              />
            </Field>
          </View>
          <View className="flex-1">
            <Field label="CVC" error={formErrors.cvc}>
              <Input
                value={form.cvc}
                onChangeText={(v) => update('cvc', v.replace(/\D/g, ''))}
                placeholder="123"
                keyboardType="number-pad"
                autoComplete="cc-csc"
                maxLength={4}
                secureTextEntry
                accessibilityLabel="CVC"
              />
            </Field>
          </View>
        </View>
        <Field label="Kart Üzerindeki Ad" error={formErrors.holder}>
          <Input
            value={form.holder}
            onChangeText={(v) => update('holder', v)}
            placeholder="Ad Soyad"
            autoComplete="cc-name"
            autoCapitalize="words"
            returnKeyType="done"
            accessibilityLabel="Kart üzerindeki ad"
          />
        </Field>

        {error && (
          <View className="flex-row items-center gap-2 rounded-2xl border border-destructive/30 bg-destructive/10 p-3">
            <Ionicons name="alert-circle" size={18} color={colors.destructive} />
            <Text className="flex-1 text-sm font-medium text-destructive">{error}</Text>
          </View>
        )}
      </ScrollView>

      <View className="flex-row gap-3 border-t border-border px-5 pb-8 pt-3">
        <Button variant="outline" className="flex-1" onPress={onClose} accessibilityLabel="Kapat">
          <Text>Kapat</Text>
        </Button>
        <Button
          className="flex-1"
          onPress={submit}
          loading={confirm.isPending}
          accessibilityLabel="Kartla öde"
        >
          <Text>{formatTRY(checkout.amount)} Öde</Text>
        </Button>
      </View>
    </KeyboardAvoidingView>
  );
}
