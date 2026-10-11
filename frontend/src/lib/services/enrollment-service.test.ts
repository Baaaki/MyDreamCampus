import { describe, expect, it } from "vitest"
import type { EnrollmentProgramResponse } from "@/lib/types"
import { currentProgram } from "./enrollment-service"

function program(semester: string, status: string): EnrollmentProgramResponse {
  return {
    id: `${semester}-${status}`,
    student_id: "student-1",
    semester,
    status,
    courses: [],
    created_at: "2026-09-01T00:00:00Z",
  }
}

describe("currentProgram", () => {
  const programs = [
    program("2026-2027-Spring", "approved"),
    program("2026-2027-Fall", "approved"),
    program("2027-2028-Fall", "pending"),
  ]

  it("takes the active semester's approved program", () => {
    expect(currentProgram(programs, "2026-2027-Fall")?.semester).toBe(
      "2026-2027-Fall"
    )
  })

  it("falls back to the latest approved program", () => {
    expect(currentProgram(programs)?.semester).toBe("2026-2027-Spring")
  })

  it("has nothing to show without an approved program", () => {
    expect(currentProgram([program("2026-2027-Fall", "rejected")])).toBeNull()
  })
})
