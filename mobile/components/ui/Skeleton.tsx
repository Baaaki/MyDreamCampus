/**
 * Skeleton — Shimmer loading placeholder.
 *
 * Veri yüklenirken içerik alanlarında kullanılır.
 * Reanimated ile pulse animasyonu uygular.
 *
 * Figma Specs:
 *   - Background: surfaceVariant
 *   - Animation: opacity 0.3 → 1.0 pulse, 1s cycle
 *   - Radius: parent component ile eşleşir
 */
import { useEffect } from 'react';
import { StyleSheet, View, type ViewStyle } from 'react-native';
import { useTheme } from 'react-native-paper';
import Animated, {
  useSharedValue,
  useAnimatedStyle,
  withRepeat,
  withTiming,
  Easing,
} from 'react-native-reanimated';
import { radius, spacing } from '../../constants/tokens';

interface SkeletonProps {
  width?: number | `${number}%`;
  height?: number;
  borderRadius?: number;
  style?: ViewStyle;
}

export function Skeleton({
  width = '100%',
  height = 16,
  borderRadius = radius.sm,
  style,
}: SkeletonProps) {
  const { colors } = useTheme();
  const opacity = useSharedValue(0.3);

  useEffect(() => {
    opacity.value = withRepeat(
      withTiming(1, { duration: 800, easing: Easing.inOut(Easing.ease) }),
      -1,
      true,
    );
  }, [opacity]);

  const animatedStyle = useAnimatedStyle(() => ({
    opacity: opacity.value,
  }));

  return (
    <Animated.View
      accessibilityRole="progressbar"
      accessibilityLabel="Yukleniyor"
      style={[
        {
          width,
          height,
          borderRadius,
          backgroundColor: colors.surfaceVariant,
        },
        animatedStyle,
        style,
      ]}
    />
  );
}

/**
 * SkeletonCard — Kart şeklinde skeleton placeholder.
 */
export function SkeletonCard({ lines = 3 }: { lines?: number }) {
  const { colors } = useTheme();

  return (
    <View
      style={[styles.card, { backgroundColor: colors.surface }]}
      accessibilityRole="progressbar"
      accessibilityLabel="Icerik yukleniyor"
    >
      <Skeleton width="40%" height={12} />
      <View style={styles.gap} />
      {Array.from({ length: lines }).map((_, i) => (
        <View key={i} style={styles.lineRow}>
          <Skeleton
            width={i === lines - 1 ? '60%' : '100%'}
            height={14}
          />
        </View>
      ))}
    </View>
  );
}

/**
 * SkeletonList — Birden fazla SkeletonCard listesi.
 */
export function SkeletonList({ count = 3, lines = 3 }: { count?: number; lines?: number }) {
  return (
    <View accessibilityRole="progressbar" accessibilityLabel="Liste yukleniyor">
      {Array.from({ length: count }).map((_, i) => (
        <SkeletonCard key={i} lines={lines} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: radius.md,
    padding: spacing.md,
    marginBottom: spacing.sm,
  },
  gap: {
    height: spacing.sm,
  },
  lineRow: {
    marginTop: spacing.xs,
  },
});
