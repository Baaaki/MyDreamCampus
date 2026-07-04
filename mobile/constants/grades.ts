// Not sistemi yardimcilari — harf notu renkleri ve degerlendirme etiketleri.

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

// Harf notuna gore renk (mavi/beyaz temaya notr, sadece anlam tasir).
export function gradePointColor(gp: string): 'success' | 'warning' | 'destructive' | 'secondary' {
  const g = gp.toUpperCase();
  if (['AA', 'BA', 'BB'].includes(g)) return 'success';
  if (['CB', 'CC', 'DC'].includes(g)) return 'warning';
  if (['DD', 'FD', 'FF'].includes(g)) return 'destructive';
  return 'secondary';
}
