export interface SemesterMeta {
  key: string
  /** Year the academic year starts in; 0 when the name is not recognised. */
  startYear: number
  season: string
  label: string
  /** Chronological: a later semester sorts higher. */
  sortKey: number
}

const SEASON_LABELS: Record<string, string> = {
  fall: "Güz",
  spring: "Bahar",
  summer: "Yaz",
}

const SEASON_ORDER: Record<string, number> = { fall: 1, spring: 2, summer: 3 }

/**
 * Reads a semester name in either format the backend has used: the current
 * "2025-2026-Spring" and the legacy "2024_fall". A name in neither format is
 * shown as it is and sorts last.
 */
export function parseSemester(key: string): SemesterMeta {
  let startYear = 0
  let season = ""

  const current = /^(\d{4})-(\d{4})-([A-Za-z]+)$/.exec(key)
  const legacy = /^(\d{4})_([A-Za-z]+)$/.exec(key)
  if (current && Number(current[2]) === Number(current[1]) + 1) {
    startYear = Number(current[1])
    season = current[3].toLowerCase()
  } else if (legacy) {
    season = legacy[2].toLowerCase()
    // The legacy name carries the calendar year: spring 2025 belongs to the
    // academic year that began in 2024.
    startYear = Number(legacy[1]) - (season === "fall" ? 0 : 1)
  }

  if (!startYear || !SEASON_ORDER[season]) {
    return { key, startYear: 0, season: "", label: key, sortKey: 0 }
  }
  return {
    key,
    startYear,
    season,
    label: `${startYear}-${startYear + 1} ${SEASON_LABELS[season]}`,
    sortKey: startYear * 10 + SEASON_ORDER[season],
  }
}
