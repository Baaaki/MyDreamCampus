// Not sistemi yardimcilari — basari katsayisi renkleri ve degerlendirme etiketleri.

export const ASSESSMENT_LABELS: Record<string, string> = {
  midterm: 'Vize',
  midterm1: '1. Vize',
  midterm2: '2. Vize',
  final: 'Final',
  quiz: 'Quiz',
  homework: 'Ödev',
  project: 'Proje',
  lab: 'Lab',
  attendance: 'Devam',
};

export function assessmentLabel(slug: string): string {
  return ASSESSMENT_LABELS[slug] ?? slug.charAt(0).toUpperCase() + slug.slice(1);
}

// grade_point harf notu degil, 4'luk katsayinin metnidir ("2.75"); web'deki
// gibi: 2.00 ve ustu gecer, 1.00-1.99 sartli gecer, altinda kalir.
export function gradePointColor(gp: string): 'success' | 'warning' | 'destructive' | 'secondary' {
  const value = parseFloat(gp);
  if (Number.isNaN(value)) return 'secondary';
  if (value >= 2) return 'success';
  if (value >= 1) return 'warning';
  return 'destructive';
}

export function formatGradePoint(gp: string): string {
  const value = parseFloat(gp);
  return Number.isNaN(value) ? gp : value.toFixed(2);
}
