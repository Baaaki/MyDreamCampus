/**
 * SectionHeader — Bölüm başlığı bileşeni.
 *
 * Tüm ekranlarda tekrarlanan "Hızlı Erişim", "Bugünün Dersleri" gibi
 * section başlıkları için tek kaynak.
 *
 * Figma Specs:
 *   - Text: titleMedium variant, fontWeight 600
 *   - Spacing: marginBottom md
 */
import { StyleSheet } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { spacing } from '../../constants/tokens';

interface SectionHeaderProps {
  title: string;
}

export function SectionHeader({ title }: SectionHeaderProps) {
  const { colors } = useTheme();

  return (
    <Text variant="titleMedium" style={[styles.title, { color: colors.onBackground }]} accessibilityRole="header">
      {title}
    </Text>
  );
}

const styles = StyleSheet.create({
  title: {
    fontWeight: '600',
    marginBottom: spacing.md,
  },
});
