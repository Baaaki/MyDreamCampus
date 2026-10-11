import { describe, expect, it } from "vitest"
import type { CourseAttendanceDetail, SessionTypeAttendance } from "@/lib/types"
import { absenceWarnings, absencesLeft } from "./attendance-service"

function stats(absent: number): SessionTypeAttendance {
  return {
    present_count: 6 - absent,
    absent_count: absent,
    total_sessions: 6,
    min_required: 5,
    passed: true,
  }
}

function course(
  code: string,
  theory?: SessionTypeAttendance,
  lab?: SessionTypeAttendance
): CourseAttendanceDetail {
  return {
    course_id: `id-${code}`,
    course_code: code,
    course_name: code,
    instructor: "",
    total_weeks: 14,
    theory,
    lab,
    weekly_records: [],
  }
}

describe("absencesLeft", () => {
  it("allows four theory and three lab absences over fourteen weeks", () => {
    expect(absencesLeft("theory", stats(0), 14)).toBe(4)
    expect(absencesLeft("lab", stats(0), 14)).toBe(3)
  })

  it("goes negative once the limit is past", () => {
    expect(absencesLeft("lab", stats(4), 14)).toBe(-1)
  })

  it("falls back to a standard semester when the course has no length", () => {
    expect(absencesLeft("theory", stats(1), 0)).toBe(3)
  })
})

describe("absenceWarnings", () => {
  it("warns about session types with at most one absence to spare, tightest first", () => {
    const warnings = absenceWarnings([
      course("CENG201", stats(3), stats(1)),
      course("MATH201", stats(2)),
      course("PHYS101", stats(5)),
    ])

    expect(warnings).toEqual([
      {
        course_id: "id-PHYS101",
        course_code: "PHYS101",
        session_type: "theory",
        absences_left: -1,
      },
      {
        course_id: "id-CENG201",
        course_code: "CENG201",
        session_type: "theory",
        absences_left: 1,
      },
    ])
  })

  it("does not flag a single early absence", () => {
    const early = {
      present_count: 1,
      absent_count: 1,
      total_sessions: 2,
      min_required: 2,
      passed: false,
    }
    expect(absenceWarnings([course("CENG201", early)])).toEqual([])
  })
})
