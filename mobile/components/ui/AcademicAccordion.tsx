/**
 * AcademicAccordion — Yıl/Dönem accordion bileşeni.
 *
 * Courses ve Grades ekranlarındaki ortak yıl > dönem > içerik pattern'ini sağlar.
 * Reanimated ile açılma/kapanma animasyonu, haptic feedback ve accessibility desteği.
 *
 * Figma Specs:
 *   - YearHeader: yearColor tinted bg, graduation-cap icon, chevron toggle
 *   - SemesterHeader: dot indicator, chevron toggle
 *   - Container: Surface elevation 1, radius lg, border outlineVariant
 */
import { useState, useCallback, type ReactNode } from 'react';
import { StyleSheet, Pressable, View } from 'react-native';
import { Text, Surface, Chip, useTheme } from 'react-native-paper';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import Animated, { FadeIn, FadeOut, LinearTransition } from 'react-native-reanimated';
import { spacing, radius, yearColors, withOpacity } from '../../constants/tokens';
import { useHaptic } from '../../hooks/useHaptic';

// ─── Year Accordion ────────────────────────────────────────

interface YearAccordionProps {
  year: number;
  courseCount: number;
  defaultOpen?: boolean;
  children: ReactNode;
}

export function YearAccordion({ year, courseCount, defaultOpen = false, children }: YearAccordionProps) {
  const { colors } = useTheme();
  const haptic = useHaptic();
  const [open, setOpen] = useState(defaultOpen);
  const color = yearColors[(year - 1) % yearColors.length];

  const toggle = useCallback(() => {
    haptic.selection();
    setOpen((v) => !v);
  }, [haptic]);

  return (
    <Animated.View layout={LinearTransition.duration(200)}>
      <Surface style={[styles.yearBox, { borderColor: colors.outlineVariant }]} elevation={1}>
        <Pressable
          onPress={toggle}
          style={[styles.yearHeader, { backgroundColor: withOpacity(color, 0.07) }]}
          accessibilityRole="button"
          accessibilityLabel={`${year}. sinif, ${courseCount} ders`}
          accessibilityState={{ expanded: open }}
        >
          <FontAwesome name="graduation-cap" size={16} color={color} />
          <Text variant="titleSmall" style={[styles.yearTitle, { color }]}>
            {year}. Sinif
          </Text>
          <Chip
            compact
            mode="flat"
            style={{ backgroundColor: withOpacity(color, 0.12) }}
            textStyle={{ color, fontSize: 11 }}
          >
            {courseCount} ders
          </Chip>
          <FontAwesome name={open ? 'chevron-up' : 'chevron-down'} size={14} color={color} />
        </Pressable>

        {open && (
          <Animated.View entering={FadeIn.duration(200)} exiting={FadeOut.duration(150)}>
            {children}
          </Animated.View>
        )}
      </Surface>
    </Animated.View>
  );
}

// ─── Semester Accordion ────────────────────────────────────

interface SemesterAccordionProps {
  year: number;
  term: number;
  courseCount?: number;
  defaultOpen?: boolean;
  extra?: ReactNode;
  children: ReactNode;
}

export function SemesterAccordion({
  year,
  term,
  courseCount,
  defaultOpen = false,
  extra,
  children,
}: SemesterAccordionProps) {
  const { colors } = useTheme();
  const haptic = useHaptic();
  const [open, setOpen] = useState(defaultOpen);
  const color = yearColors[(year - 1) % yearColors.length];

  const toggle = useCallback(() => {
    haptic.selection();
    setOpen((v) => !v);
  }, [haptic]);

  const termLabel = term === 1 ? 'Guz' : 'Bahar';

  return (
    <View style={styles.semesterBox}>
      <Pressable
        onPress={toggle}
        style={styles.semesterHeader}
        accessibilityRole="button"
        accessibilityLabel={`${term}. donem ${termLabel}${courseCount !== undefined ? `, ${courseCount} ders` : ''}`}
        accessibilityState={{ expanded: open }}
      >
        <View style={[styles.semesterDot, { backgroundColor: color }]} />
        <Text variant="bodyMedium" style={[styles.semesterTitle, { color: colors.onSurface }]}>
          {term}. Donem ({termLabel})
        </Text>
        {courseCount !== undefined && (
          <Text variant="labelSmall" style={{ color: colors.onSurfaceVariant }}>
            {courseCount} ders
          </Text>
        )}
        {extra}
        <FontAwesome name={open ? 'chevron-up' : 'chevron-down'} size={12} color={colors.onSurfaceVariant} />
      </Pressable>

      {open && (
        <Animated.View entering={FadeIn.duration(200)} exiting={FadeOut.duration(150)}>
          {children}
        </Animated.View>
      )}
    </View>
  );
}

// ─── Empty State ───────────────────────────────────────────

interface EmptyStateProps {
  message: string;
  icon?: keyof typeof FontAwesome.glyphMap;
}

export function EmptyState({ message, icon = 'clock-o' }: EmptyStateProps) {
  const { colors } = useTheme();

  return (
    <View style={styles.emptyState} accessibilityRole="text" accessibilityLabel={message}>
      <FontAwesome name={icon} size={16} color={colors.onSurfaceVariant} />
      <Text variant="bodySmall" style={{ color: colors.onSurfaceVariant }}>
        {message}
      </Text>
    </View>
  );
}

// ─── Styles ────────────────────────────────────────────────

const styles = StyleSheet.create({
  yearBox: {
    borderRadius: radius.lg,
    marginBottom: spacing.md,
    borderWidth: 1,
    overflow: 'hidden',
  },
  yearHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: 14,
  },
  yearTitle: {
    fontWeight: '700',
    flex: 1,
  },
  semesterBox: {
    paddingHorizontal: spacing.md,
    paddingBottom: spacing.xs,
  },
  semesterHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.xs,
  },
  semesterDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  semesterTitle: {
    fontWeight: '600',
    flex: 1,
  },
  emptyState: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.sm,
    paddingVertical: spacing.md,
  },
});
