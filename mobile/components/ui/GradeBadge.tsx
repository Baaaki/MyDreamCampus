/**
 * GradeBadge — Harf notu gösterge bileşeni.
 *
 * Not harfini semantik renkle Chip içinde gösterir.
 * Grades, Courses ve Attendance ekranlarında kullanılır.
 *
 * Figma Specs:
 *   - Container: Chip compact flat, background gradeColor %20 opacity
 *   - Text: fontSize 12, fontWeight 700, gradeColor
 */
import { Chip, useTheme } from 'react-native-paper';
import { getGradeColor, withOpacity } from '../../constants/tokens';

interface GradeBadgeProps {
  grade: string;
}

export function GradeBadge({ grade }: GradeBadgeProps) {
  const { colors } = useTheme();
  const color = getGradeColor(grade, colors.onSurfaceVariant);

  return (
    <Chip
      compact
      mode="flat"
      style={{ backgroundColor: withOpacity(color, 0.12) }}
      textStyle={{ color, fontWeight: '700', fontSize: 12 }}
      accessibilityLabel={`Not: ${grade}`}
    >
      {grade}
    </Chip>
  );
}
