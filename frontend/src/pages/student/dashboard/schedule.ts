import { addDays } from "date-fns"
import { TIME_SLOTS } from "@/lib/constants"
import type { CourseBasic } from "@/lib/types"

export type SessionType = "theory" | "lab"

export interface ClassBlock {
  courseCode: string
  courseName: string
  instructor?: string
  sessionType: SessionType
  start: string // HH:MM
  end: string // HH:MM
  colorIndex: number
}

export interface UpcomingClass {
  block: ClassBlock
  daysAhead: number
}

export const COURSE_COLOR_COUNT = 5

const DAY_NAMES = [
  "sunday",
  "monday",
  "tuesday",
  "wednesday",
  "thursday",
  "friday",
  "saturday",
]

export function toMinutes(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return h * 60 + m
}

function slotRange(slot: number): [string, string] | null {
  const entry = TIME_SLOTS[slot as keyof typeof TIME_SLOTS]
  if (!entry) return null
  const [start, end] = entry.time.split("-")
  return [start, end]
}

// A course's colour follows its place in the program so the same course
// keeps it on every day and in the week grid.
export function courseColors(courses: CourseBasic[]): Map<string, number> {
  return new Map(
    courses.map((c, i) => [c.course_code, i % COURSE_COLOR_COUNT] as const)
  )
}

// Back-to-back slots of one session are one sitting; a gap (lunch, a free
// hour) starts a new block.
export function dayBlocks(courses: CourseBasic[], day: Date): ClassBlock[] {
  const dayName = DAY_NAMES[day.getDay()]
  const colors = courseColors(courses)
  const blocks: ClassBlock[] = []

  for (const course of courses) {
    for (const session of course.schedule_sessions ?? []) {
      if (session.day_of_week.toLowerCase() !== dayName) continue
      const slots = [...session.slot_numbers].sort((a, b) => a - b)
      let runStart = 0
      for (let i = 1; i <= slots.length; i++) {
        if (i < slots.length && slots[i] === slots[i - 1] + 1) continue
        const first = slotRange(slots[runStart])
        const last = slotRange(slots[i - 1])
        if (first && last) {
          blocks.push({
            courseCode: course.course_code,
            courseName: course.course_name,
            instructor: course.instructor,
            sessionType: session.session_type,
            start: first[0],
            end: last[1],
            colorIndex: colors.get(course.course_code) ?? 0,
          })
        }
        runStart = i
      }
    }
  }

  return blocks.sort((a, b) => toMinutes(a.start) - toMinutes(b.start))
}

export function currentAndNext(
  courses: CourseBasic[],
  now: Date
): { current: ClassBlock | null; next: UpcomingClass | null } {
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const today = dayBlocks(courses, now)
  const current =
    today.find(
      (b) => toMinutes(b.start) <= nowMin && nowMin < toMinutes(b.end)
    ) ?? null

  const laterToday = today.find((b) => toMinutes(b.start) > nowMin)
  if (laterToday) return { current, next: { block: laterToday, daysAhead: 0 } }

  for (let d = 1; d <= 7; d++) {
    const first = dayBlocks(courses, addDays(now, d))[0]
    if (first) return { current, next: { block: first, daysAhead: d } }
  }
  return { current, next: null }
}

export interface WeekCell {
  courseCode: string
  colorIndex: number
  // Only the first slot of a sitting carries the label.
  isFirst: boolean
}

// Monday..Friday (1..5) x slot -> the course sitting there.
export function weekGrid(courses: CourseBasic[]): Map<string, WeekCell> {
  const colors = courseColors(courses)
  const grid = new Map<string, WeekCell>()
  for (const course of courses) {
    for (const session of course.schedule_sessions ?? []) {
      const day = DAY_NAMES.indexOf(session.day_of_week.toLowerCase())
      if (day < 1 || day > 5) continue
      for (const slot of session.slot_numbers) {
        grid.set(`${day}-${slot}`, {
          courseCode: course.course_code,
          colorIndex: colors.get(course.course_code) ?? 0,
          isFirst: !session.slot_numbers.includes(slot - 1),
        })
      }
    }
  }
  return grid
}
