/**
 * ActionCard — Tıklanabilir aksiyon kartı.
 *
 * Home ekranındaki "QR Yoklama", "Notlarım" gibi hızlı erişim kartları.
 *
 * Figma Specs:
 *   - Container: Surface elevation 1, radius md, padding md
 *   - Icon area: 48x48, primaryContainer background, radius sm
 *   - Title: fontSize md, fontWeight semibold
 *   - Subtitle: fontSize xs, onSurfaceVariant
 */
import { StyleSheet, View, Pressable } from 'react-native';
import { Surface, Text, useTheme } from 'react-native-paper';
import { FontAwesome } from '@expo/vector-icons';
import { useRouter, type Href } from 'expo-router';
import { spacing, radius, fontSize } from '../../constants/tokens';
import { useHaptic } from '../../hooks/useHaptic';

interface ActionCardProps {
  icon: keyof typeof FontAwesome.glyphMap;
  title: string;
  subtitle?: string;
  href: Href;
  color?: string;
}

export function ActionCard({ icon, title, subtitle, href, color }: ActionCardProps) {
  const theme = useTheme();
  const router = useRouter();
  const haptic = useHaptic();
  const iconColor = color ?? theme.colors.primary;

  return (
    <Pressable
      onPress={() => { haptic.light(); router.push(href); }}
      style={styles.pressable}
      accessibilityRole="button"
      accessibilityLabel={title}
    >
      <Surface style={styles.container} elevation={1}>
        <View style={[styles.iconWrap, { backgroundColor: theme.colors.primaryContainer }]}>
          <FontAwesome name={icon} size={22} color={iconColor} />
        </View>
        <Text style={[styles.title, { color: theme.colors.onSurface }]}>
          {title}
        </Text>
        {subtitle && (
          <Text style={[styles.subtitle, { color: theme.colors.onSurfaceVariant }]}>
            {subtitle}
          </Text>
        )}
      </Surface>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  pressable: {
    flex: 1,
  },
  container: {
    alignItems: 'center',
    padding: spacing.md,
    borderRadius: radius.md,
    gap: spacing.sm,
  },
  iconWrap: {
    width: 48,
    height: 48,
    borderRadius: radius.sm,
    alignItems: 'center',
    justifyContent: 'center',
  },
  title: {
    fontSize: fontSize.md,
    fontWeight: '600',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: fontSize.xs,
    textAlign: 'center',
  },
});
