/**
 * StatCard — İstatistik kartı bileşeni.
 *
 * Figma'daki "Stat Card" component'ine karşılık gelir.
 * Home, Grades, Attendance ekranlarında kullanılır.
 *
 * Figma Specs:
 *   - Container: Surface elevation 1, radius md, padding md
 *   - Icon: 24x24, secondary color
 *   - Value: fontSize xl, fontWeight bold
 *   - Label: fontSize sm, onSurfaceVariant color
 */
import { StyleSheet, View } from 'react-native';
import { Surface, Text, useTheme } from 'react-native-paper';
import { FontAwesome } from '@expo/vector-icons';
import { spacing, radius, fontSize } from '../../constants/tokens';

interface StatCardProps {
  icon: keyof typeof FontAwesome.glyphMap;
  value: string | number;
  label: string;
  color?: string;
}

export function StatCard({ icon, value, label, color }: StatCardProps) {
  const theme = useTheme();
  const iconColor = color ?? theme.colors.secondary;

  return (
    <Surface style={styles.container} elevation={1} accessibilityRole="text" accessibilityLabel={`${label}: ${value}`}>
      <FontAwesome name={icon} size={24} color={iconColor} />
      <Text style={[styles.value, { color: theme.colors.onSurface }]}>
        {value}
      </Text>
      <Text style={[styles.label, { color: theme.colors.onSurfaceVariant }]}>
        {label}
      </Text>
    </Surface>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    padding: spacing.md,
    borderRadius: radius.md,
    gap: spacing.xs,
  },
  value: {
    fontSize: fontSize.xl,
    fontWeight: '700',
  },
  label: {
    fontSize: fontSize.sm,
    textAlign: 'center',
  },
});
