import Ionicons from '@expo/vector-icons/Ionicons';
import { useIsFocused } from 'expo-router/react-navigation';
import { CameraView, useCameraPermissions } from 'expo-camera';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Modal, StyleSheet, View } from 'react-native';
import Animated, {
  Easing,
  FadeIn,
  FadeInDown,
  ReduceMotion,
  ZoomIn,
  useAnimatedStyle,
  useSharedValue,
  withRepeat,
  withTiming,
} from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Button, Card, Text } from '@/components/ui';
import { MEAL_LABEL, MENU_LABEL } from '@/constants/meal';
import { SESSION_TYPE_LABEL } from '@/constants/schedule';
import { useTheme } from '@/contexts/ThemeContext';
import { useScanQR } from '@/hooks/useAttendance';
import { useHaptic } from '@/hooks/useHaptic';
import { useUseReservation } from '@/hooks/useMeals';
import { apiErrorMessage } from '@/lib/api-error';
import { createScanGate, parseScannedQR } from '@/lib/qr-payload';
import { COLORS } from '@/lib/theme';
import type { ScanQRResponse } from '@/types/attendance.types';
import type { UseReservationData } from '@/types/meal.types';

const FRAME_SIZE = 260;

type ScanResult =
  | { type: 'success'; data: ScanQRResponse }
  | { type: 'meal'; data: UseReservationData }
  | { type: 'error'; title: string; message: string };

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

// Vizorun kose braketleri: iki kenari cizili 4 kucuk kare.
function CornerBracket({ position }: { position: 'tl' | 'tr' | 'bl' | 'br' }) {
  const base = 'absolute h-9 w-9 border-primary';
  const map = {
    tl: 'left-0 top-0 rounded-tl-2xl border-l-4 border-t-4',
    tr: 'right-0 top-0 rounded-tr-2xl border-r-4 border-t-4',
    bl: 'bottom-0 left-0 rounded-bl-2xl border-b-4 border-l-4',
    br: 'bottom-0 right-0 rounded-br-2xl border-b-4 border-r-4',
  } as const;
  return <View className={`${base} ${map[position]}`} />;
}

export default function ScanScreen() {
  const isFocused = useIsFocused();
  const haptic = useHaptic();
  const { isDark } = useTheme();
  const colors = COLORS[isDark ? 'dark' : 'light'];

  const [permission, requestPermission] = useCameraPermissions();
  const [invalid, setInvalid] = useState(false);
  const [result, setResult] = useState<ScanResult | null>(null);
  const gateRef = useRef(createScanGate());
  const scanMutation = useScanQR();
  const mealMutation = useUseReservation();

  // Sekme her odaklandiginda temiz durumla basla.
  useEffect(() => {
    if (isFocused) {
      gateRef.current.release();
      setInvalid(false);
    }
  }, [isFocused]);

  // Tarama cizgisi: cerceve icinde asagi-yukari supurme.
  const sweep = useSharedValue(0);
  useEffect(() => {
    sweep.value = withRepeat(
      withTiming(1, { duration: 2200, easing: Easing.inOut(Easing.quad), reduceMotion: ReduceMotion.System }),
      -1,
      true
    );
  }, [sweep]);

  const scanLineStyle = useAnimatedStyle(() => ({
    transform: [{ translateY: 10 + sweep.value * (FRAME_SIZE - 20) }],
  }));

  const handleBarcode = useCallback(
    ({ data }: { data: string }) => {
      if (!gateRef.current.tryConsume()) return;

      const scanned = parseScannedQR(data);
      if (!scanned) {
        haptic.error();
        setInvalid(true);
        // Kisa kilit: ayni bozuk QR her karede tekrar tetiklenmesin,
        // kullanici modal kapatmadan yeniden nisan alabilsin.
        setTimeout(() => {
          setInvalid(false);
          gateRef.current.release();
        }, 1500);
        return;
      }

      haptic.medium();

      if (scanned.kind === 'meal') {
        mealMutation.mutate(
          { qr_payload: scanned.qrPayload },
          {
            onSuccess: (data) => {
              haptic.success();
              setResult({ type: 'meal', data });
            },
            onError: (error) => {
              haptic.error();
              setResult({
                type: 'error',
                title: 'Yemek kullanilamadi',
                message: apiErrorMessage(error, 'Yemek kullanilamadi. Tekrar dene.'),
              });
            },
          }
        );
        return;
      }

      scanMutation.mutate(
        { qr_payload: scanned.payload },
        {
          onSuccess: (data) => {
            haptic.success();
            setResult({ type: 'success', data });
          },
          onError: (error) => {
            haptic.error();
            setResult({
              type: 'error',
              title: 'Yoklama alinamadi',
              message: apiErrorMessage(error, 'Yoklama alinamadi. Tekrar dene.'),
            });
          },
        }
      );
    },
    [haptic, scanMutation, mealMutation]
  );

  const closeResult = () => {
    setResult(null);
    gateRef.current.release();
  };

  // Kamera izni akisi
  if (!permission || !permission.granted) {
    return (
      <SafeAreaView className="flex-1 bg-background px-6" edges={['top']}>
        <View className="flex-1 items-center justify-center">
          <Card className="w-full items-center p-8">
            <View className="mb-5 h-16 w-16 items-center justify-center rounded-2xl bg-primary/15">
              <Ionicons name="camera-outline" size={30} color={colors.primary} />
            </View>
            <Text className="mb-2 text-xl font-bold text-foreground">Kamera izni gerekli</Text>
            <Text className="mb-6 text-center text-sm text-muted-foreground">
              Dersteki QR kodu tarayip yoklamani alabilmen icin kameraya erisim gerekiyor.
            </Text>
            {permission ? (
              <Button onPress={requestPermission} accessibilityLabel="Kamera izni ver">
                <Text>Izin Ver</Text>
              </Button>
            ) : (
              <Text className="text-sm text-muted-foreground">Kamera hazirlaniyor...</Text>
            )}
          </Card>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <View className="flex-1 bg-black">
      {isFocused && (
        <CameraView
          style={StyleSheet.absoluteFill}
          facing="back"
          barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
          onBarcodeScanned={
            result || scanMutation.isPending || mealMutation.isPending ? undefined : handleBarcode
          }
        />
      )}

      {/* Vizor katmani */}
      <SafeAreaView className="flex-1" edges={['top']} pointerEvents="box-none">
        <Animated.View entering={FadeIn.duration(400)} className="items-center pt-6">
          <Text className="font-mono text-xs uppercase tracking-widest text-white/70">
            Yoklama · Yemekhane
          </Text>
          <Text className="mt-1 text-2xl font-extrabold text-white">QR Tara</Text>
        </Animated.View>

        <View className="flex-1 items-center justify-center">
          <View style={{ width: FRAME_SIZE, height: FRAME_SIZE }}>
            <CornerBracket position="tl" />
            <CornerBracket position="tr" />
            <CornerBracket position="bl" />
            <CornerBracket position="br" />
            <Animated.View
              style={[
                scanLineStyle,
                {
                  shadowColor: colors.primary,
                  shadowOpacity: 0.9,
                  shadowRadius: 8,
                  shadowOffset: { width: 0, height: 0 },
                },
              ]}
              className="mx-4 h-0.5 rounded-full bg-primary"
            />
          </View>

          <Animated.View entering={FadeInDown.delay(150).duration(400)} className="mt-8 px-10">
            {scanMutation.isPending || mealMutation.isPending ? (
              <View className="flex-row items-center gap-2 rounded-full bg-black/50 px-5 py-2.5">
                <Ionicons name="sync" size={16} color="#fff" />
                <Text className="text-sm font-medium text-white">
                  {mealMutation.isPending ? 'Yemek kullaniliyor...' : 'Yoklama gonderiliyor...'}
                </Text>
              </View>
            ) : invalid ? (
              <View className="rounded-full bg-destructive/90 px-5 py-2.5">
                <Text className="text-center text-sm font-semibold text-white">
                  Gecersiz QR — ders veya yemekhane QR kodunu tara
                </Text>
              </View>
            ) : (
              <View className="rounded-full bg-black/50 px-5 py-2.5">
                <Text className="text-center text-sm text-white/90">
                  QR kodu cercevenin icine hizala
                </Text>
              </View>
            )}
          </Animated.View>
        </View>
      </SafeAreaView>

      {/* Sonuc modali */}
      <Modal visible={result !== null} transparent animationType="fade" onRequestClose={closeResult}>
        <View className="flex-1 items-center justify-center bg-black/70 px-8">
          {result?.type === 'success' && (
            <Animated.View
              entering={ZoomIn.springify().damping(14)}
              className="w-full items-center rounded-4xl bg-card p-8"
            >
              <Animated.View
                entering={ZoomIn.delay(120).springify().damping(10)}
                className="mb-5 h-24 w-24 items-center justify-center rounded-full bg-success"
              >
                <Ionicons name="checkmark" size={52} color={colors.background} />
              </Animated.View>
              <Text className="mb-1 text-2xl font-extrabold text-foreground">Yoklaman alindi</Text>
              <Text className="mb-4 text-center text-base text-muted-foreground">
                {result.data.course_name}
              </Text>
              <Text className="mb-6 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                {result.data.course_code} · {result.data.week_number}. hafta ·{' '}
                {SESSION_TYPE_LABEL[result.data.session_type]}
                {formatTime(result.data.marked_at) ? ` · ${formatTime(result.data.marked_at)}` : ''}
              </Text>
              <Button className="w-full" onPress={closeResult} accessibilityLabel="Kapat">
                <Text>Harika</Text>
              </Button>
            </Animated.View>
          )}

          {result?.type === 'meal' && (
            <Animated.View
              entering={ZoomIn.springify().damping(14)}
              className="w-full items-center rounded-4xl bg-card p-8"
            >
              <Animated.View
                entering={ZoomIn.delay(120).springify().damping(10)}
                className="mb-5 h-24 w-24 items-center justify-center rounded-full bg-success"
              >
                <Ionicons name="restaurant" size={44} color={colors.background} />
              </Animated.View>
              <Text className="mb-1 text-2xl font-extrabold text-foreground">Afiyet olsun</Text>
              <Text className="mb-4 text-center text-base text-muted-foreground">
                {result.data.cafeteria_name}
              </Text>
              <Text className="mb-6 font-mono text-xs uppercase tracking-widest text-muted-foreground">
                {MEAL_LABEL[result.data.meal_time] ?? result.data.meal_time} ·{' '}
                {MENU_LABEL[result.data.menu_type] ?? result.data.menu_type}
              </Text>
              <Button className="w-full" onPress={closeResult} accessibilityLabel="Kapat">
                <Text>Tamam</Text>
              </Button>
            </Animated.View>
          )}

          {result?.type === 'error' && (
            <Animated.View
              entering={ZoomIn.springify().damping(14)}
              className="w-full items-center rounded-4xl bg-card p-8"
            >
              <Animated.View
                entering={ZoomIn.delay(120).springify().damping(10)}
                className="mb-5 h-24 w-24 items-center justify-center rounded-full bg-destructive"
              >
                <Ionicons name="close" size={52} color="#fff" />
              </Animated.View>
              <Text className="mb-2 text-2xl font-extrabold text-foreground">
                {result.title}
              </Text>
              <Text className="mb-6 text-center text-base text-muted-foreground">
                {result.message}
              </Text>
              <Button className="w-full" onPress={closeResult} accessibilityLabel="Tekrar dene">
                <Text>Tekrar Dene</Text>
              </Button>
            </Animated.View>
          )}
        </View>
      </Modal>
    </View>
  );
}
