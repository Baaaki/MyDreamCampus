import { describe, expect, it } from "vitest"
import { parseSemester } from "./semester"

describe("parseSemester", () => {
  it.each([
    ["2025-2026-Spring", "2025-2026 Bahar"],
    ["2026-2027-Fall", "2026-2027 Güz"],
    ["2025-2026-Summer", "2025-2026 Yaz"],
    ["2024_fall", "2024-2025 Güz"],
    ["2025_spring", "2024-2025 Bahar"],
  ])("labels %s as %s", (key, label) => {
    expect(parseSemester(key).label).toBe(label)
  })

  it("sorts spring after the fall of the same academic year", () => {
    const fall = parseSemester("2025-2026-Fall")
    const spring = parseSemester("2025-2026-Spring")
    const nextFall = parseSemester("2026-2027-Fall")

    expect(spring.sortKey).toBeGreaterThan(fall.sortKey)
    expect(nextFall.sortKey).toBeGreaterThan(spring.sortKey)
  })

  it("places both formats on one timeline", () => {
    expect(parseSemester("2025_spring").sortKey).toBe(
      parseSemester("2024-2025-Spring").sortKey
    )
    expect(parseSemester("2024_fall").sortKey).toBeLessThan(
      parseSemester("2025_spring").sortKey
    )
  })

  it.each(["", "2025", "2025-2027-Fall", "2025-2026-Winter", "foo_bar"])(
    "shows %j as it is and sorts it last",
    (key) => {
      const meta = parseSemester(key)
      expect(meta.label).toBe(key)
      expect(meta.sortKey).toBe(0)
    }
  )
})
