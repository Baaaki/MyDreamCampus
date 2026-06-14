/**
 * StatusChip — Durum gösterge bileşeni.
 *
 * "Geçti", "Risk", "QR", "Manuel" gibi durum badge'leri için.
 * Attendance ekranında kullanılır.
 *
 * Figma Specs:
 *   - Container: Chip compact flat, semantic background
 *   - Text: fontSize 11, fontWeight 600
 */
import { Chip } from 'react-native-paper';
import { semanticColors } from '../../constants/tokens';

type StatusVariant = 'success' | 'danger' | 'warning' | 'info';

interface StatusChipProps {
  label: string;
  variant: StatusVariant;
}

const variantMap: Record<StatusVariant, { bg: string; color: string }> = {
  success: { bg: semanticColors.successLight, color: semanticColors.success },
  danger: { bg: semanticColors.dangerLight, color: semanticColors.dangerDark },
  warning: { bg: semanticColors.warningLight, color: semanticColors.warningDark },
  info: { bg: semanticColors.infoLight, color: semanticColors.info },
};

export function StatusChip({ label, variant }: StatusChipProps) {
  const { bg, color } = variantMap[variant];

  return (
    <Chip
      compact
      mode="flat"
      style={{ backgroundColor: bg }}
      textStyle={{ color, fontSize: 11, fontWeight: '600' }}
      accessibilityLabel={`Durum: ${label}`}
    >
      {label}
    </Chip>
  );
}
