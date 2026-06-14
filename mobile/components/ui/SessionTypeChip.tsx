/**
 * SessionTypeChip — Ders tipi (Teori/Lab) gösterge bileşeni.
 *
 * Home schedule kartları ve Attendance ekranında kullanılır.
 *
 * Figma Specs:
 *   - Teori: primaryContainer bg, primary text
 *   - Lab: warningLight bg, warningDark text
 */
import { Chip, useTheme } from 'react-native-paper';
import { semanticColors } from '../../constants/tokens';

interface SessionTypeChipProps {
  type: 'Teori' | 'Lab' | 'theory' | 'lab';
}

export function SessionTypeChip({ type }: SessionTypeChipProps) {
  const { colors } = useTheme();
  const isLab = type === 'Lab' || type === 'lab';

  return (
    <Chip
      compact
      mode="flat"
      style={{
        backgroundColor: isLab ? semanticColors.warningLight : colors.primaryContainer,
      }}
      textStyle={{
        color: isLab ? semanticColors.warningDark : colors.primary,
        fontSize: 10,
        fontWeight: '600',
      }}
    >
      {isLab ? 'Lab' : 'Teori'}
    </Chip>
  );
}
