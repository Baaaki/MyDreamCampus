const TERM_LABELS: Record<string, string> = {
  Fall: 'Güz',
  Spring: 'Bahar',
  Summer: 'Yaz',
};

// "2026-2027-Fall" -> "2026-2027 Güz", as the web labels it. A name in any
// other shape is shown as it is.
export function semesterLabel(semester: string): string {
  const match = /^(\d{4}-\d{4})-(\w+)$/.exec(semester);
  if (!match) return semester;
  return `${match[1]} ${TERM_LABELS[match[2]] ?? match[2]}`;
}
