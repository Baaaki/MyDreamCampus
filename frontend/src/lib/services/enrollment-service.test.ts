import { afterEach, describe, expect, it, vi } from "vitest"
import { HTTPError } from "ky"
import type { EnrollmentProgramResponse } from "@/lib/types"
import { currentProgram, enrollmentService } from "./enrollment-service"

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

describe("withdrawProgram", () => {
  afterEach(() => vi.unstubAllGlobals())

  function answer(status: number, body?: unknown) {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(body === undefined ? null : JSON.stringify(body), {
            status,
            headers: { "Content-Type": "application/json" },
          })
      )
    )
  }

  it("takes a program that is already gone as withdrawn", async () => {
    answer(404, {
      error: "Bu dönem için ders kaydı bulunamadı",
      code: "NOT_FOUND",
    })
    await expect(
      enrollmentService.withdrawProgram("2026-2027-Fall")
    ).resolves.toBeUndefined()
  })

  it("fails when the program is still there", async () => {
    answer(403, {
      error: "Bu dönem için ders kayıt süresi sona erdi",
      code: "ENROLLMENT_PERIOD_ENDED",
    })
    await expect(
      enrollmentService.withdrawProgram("2026-2027-Fall")
    ).rejects.toBeInstanceOf(HTTPError)
  })
})
