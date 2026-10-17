import { describe, expect, it } from "vitest"
import type { CourseBasic } from "@/lib/types"
import { currentAndNext, dayBlocks, weekGrid } from "./schedule"

const COURSES: CourseBasic[] = [
  {
    id: "1",
    course_code: "CENG302",
    course_name: "Bilgisayar Ağları",
    credits: 3,
    schedule_sessions: [
      {
        day_of_week: "wednesday",
        slot_numbers: [3, 1, 2],
        session_type: "theory",
      },
    ],
  },
  {
    id: "2",
    course_code: "CENG304",
    course_name: "Veritabanı Sistemleri",
    credits: 3,
    schedule_sessions: [
      // Slot 5 is lunch, so 4 and 6 are two sittings.
      {
        day_of_week: "Wednesday",
        slot_numbers: [4, 6, 7],
        session_type: "lab",
      },
    ],
  },
  {
    id: "3",
    course_code: "CENG402",
    course_name: "Dağıtık Sistemler",
    credits: 3,
    schedule_sessions: [
      {
        day_of_week: "thursday",
        slot_numbers: [1, 2, 3],
        session_type: "theory",
      },
    ],
  },
  { id: "4", course_code: "CENG499", course_name: "Bitirme", credits: 3 },
]

// 21 Oct 2026 is a Wednesday.
const at = (day: number, h: number, m = 0) => new Date(2026, 9, day, h, m)

describe("dayBlocks", () => {
  it("joins back-to-back slots and splits across a gap", () => {
    expect(
      dayBlocks(COURSES, at(21, 0)).map((b) => [b.courseCode, b.start, b.end])
    ).toEqual([
      ["CENG302", "08:30", "11:05"],
      ["CENG304", "11:15", "12:00"],
      ["CENG304", "13:00", "14:40"],
    ])
  })

  it("has nothing on a day without sessions", () => {
    expect(dayBlocks(COURSES, at(24, 10))).toEqual([])
  })
})

describe("currentAndNext", () => {
  it("finds the sitting in progress and the one after it today", () => {
    const { current, next } = currentAndNext(COURSES, at(21, 9, 40))
    expect(current?.courseCode).toBe("CENG302")
    expect(next).toMatchObject({ daysAhead: 0, block: { start: "11:15" } })
  })

  it("looks ahead to the next school day once today is over", () => {
    const { current, next } = currentAndNext(COURSES, at(21, 15))
    expect(current).toBeNull()
    expect(next).toMatchObject({
      daysAhead: 1,
      block: { courseCode: "CENG402" },
    })
  })

  it("wraps over the weekend", () => {
    const { next } = currentAndNext(COURSES, at(23, 10))
    expect(next).toMatchObject({
      daysAhead: 5,
      block: { courseCode: "CENG302" },
    })
  })

  it("has no next class for a program without a schedule", () => {
    expect(currentAndNext([COURSES[3]], at(21, 10))).toEqual({
      current: null,
      next: null,
    })
  })
})

describe("weekGrid", () => {
  it("labels only the first slot of each sitting", () => {
    const grid = weekGrid(COURSES)
    expect(grid.get("3-1")).toMatchObject({
      courseCode: "CENG302",
      isFirst: true,
    })
    expect(grid.get("3-2")?.isFirst).toBe(false)
    expect(grid.get("3-6")?.isFirst).toBe(true)
    expect(grid.get("4-1")?.colorIndex).toBe(2)
  })
})
