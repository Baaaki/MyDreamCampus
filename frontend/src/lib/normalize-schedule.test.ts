import { describe, expect, it } from "vitest"
import {
  normalizeAvailableCourse,
  normalizeSemesterCourse,
  normalizeTeacherCourse,
} from "./normalize-schedule"
import type {
  AvailableCourseDTO,
  SemesterCourseDTO,
  TeacherCourseDTO,
} from "@/lib/types"

describe("schedule normalization", () => {
  it("turns a null teacher schedule into an empty list", () => {
    const wire = { id: "c-1", schedule: null } as TeacherCourseDTO

    expect(normalizeTeacherCourse(wire)).toEqual({ id: "c-1", schedule: [] })
  })

  it("turns null semester course sessions into an empty list", () => {
    const wire = { id: "c-1", schedule_sessions: null } as SemesterCourseDTO

    expect(normalizeSemesterCourse(wire)).toEqual({
      id: "c-1",
      schedule_sessions: [],
    })
  })

  it("turns null available course sessions into an empty list", () => {
    const wire = { id: "c-1", schedule_sessions: null } as AvailableCourseDTO

    expect(normalizeAvailableCourse(wire)).toEqual({
      id: "c-1",
      schedule_sessions: [],
    })
  })

  it("keeps the sessions of a scheduled course", () => {
    const sessions = [{ day_of_week: "monday", slot_numbers: [1, 2] }]
    const wire = {
      id: "c-1",
      schedule_sessions: sessions,
    } as AvailableCourseDTO

    expect(normalizeAvailableCourse(wire).schedule_sessions).toBe(sessions)
  })
})
