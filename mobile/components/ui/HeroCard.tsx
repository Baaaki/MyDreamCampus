/**
 * HeroCard — Ekran üstü özet kartı.
 *
 * Primary renkli arka plan üzerinde 2-3 istatistik gösteren hero bileşen.
 * Courses (Ders/Kredi/Sınıf) ve Grades (GPA/Kredi) ekranlarında kullanılır.
 *
 * Figma Specs:
 *   - Container: Surface elevation 2, primary bg, radius xl, padding lg
 *   - Values: headlineSmall, white, bold
 *   - Labels: labelSmall, primaryLight
 *   - Divider: 1px, primaryLight
 */
import { StyleSheet, View } from 'react-native';
import { Surface, Text, useTheme } from 'react-native-paper';
import { spacing, radius, withOpacity } from '../../constants/tokens';

interface HeroItem {
  value: string | number;
  label: string;
}

interface HeroCardProps {
  items: HeroItem[];
}

export function HeroCard({ items }: HeroCardProps) {
  const { colors } = useTheme();

  return (
    <Surface
      style={[styles.card, { backgroundColor: colors.primary }]}
      elevation={2}
      accessibilityRole="summary"
      accessibilityLabel={items.map((item) => `${item.label}: ${item.value}`).join(', ')}
    >
      {items.map((item, i) => (
        <View key={item.label} style={styles.row}>
          {i > 0 && (
            <View style={[styles.divider, { backgroundColor: withOpacity('#ffffff', 0.25) }]} />
          )}
          <View style={styles.item}>
            <Text variant="headlineSmall" style={styles.value}>
              {item.value}
            </Text>
            <Text variant="labelSmall" style={styles.label}>
              {item.label}
            </Text>
          </View>
        </View>
      ))}
    </Surface>
  );
}

const styles = StyleSheet.create({
  card: {
    flexDirection: 'row',
    borderRadius: radius.xl,
    padding: spacing.lg,
    marginBottom: spacing.lg,
    alignItems: 'center',
    justifyContent: 'space-around',
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  item: {
    flex: 1,
    alignItems: 'center',
  },
  value: {
    fontWeight: 'bold',
    color: '#ffffff',
  },
  label: {
    color: 'rgba(255,255,255,0.7)',
    marginTop: spacing.xs,
  },
  divider: {
    width: 1,
    height: 32,
  },
});
