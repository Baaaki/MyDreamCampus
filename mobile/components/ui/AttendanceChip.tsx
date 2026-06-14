/**
 * AttendanceChip — Devamsızlık yüzdesi gösterge bileşeni.
 *
 * Yüzdeye göre semantik renk (success/warning/danger) atar.
 * Courses ve Attendance ekranlarında kullanılır.
 *
 * Figma Specs:
 *   - Container: Chip compact flat, background attendanceColor light
 *   - Text: fontWeight 700, attendanceColor
 */
import { Chip } from 'react-native-paper';
import { semanticColors, getAttendanceColor } from '../../constants/tokens';

interface AttendanceChipProps {
  percentage: number;
}

export function AttendanceChip({ percentage }: AttendanceChipProps) {
  const color = getAttendanceColor(percentage);
  const bg =
    color === semanticColors.success
      ? semanticColors.successLight
      : color === semanticColors.warning
        ? semanticColors.warningLight
        : semanticColors.dangerLight;

  return (
    <Chip
      compact
      mode="flat"
      style={{ backgroundColor: bg }}
      textStyle={{ color, fontWeight: '700', fontSize: 12 }}
      accessibilityLabel={`Devam orani: yuzde ${percentage}`}
    >
      %{percentage}
    </Chip>
  );
}
