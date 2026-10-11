import { describe, expect, it } from "vitest"
import {
  toCreateStudentPayload,
  type CreateStudentForm,
} from "./admin-student-service"

const form: CreateStudentForm = {
  student_number: "20260001",
  first_name: "Ayşe",
  last_name: "Yılmaz",
  email: "ayse@example.com",
  faculty: "Mühendislik Fakültesi",
  department: "Bilgisayar Mühendisliği",
  enrollment_year: 2026,
  class_level: 1,
  advisor_id: "",
}

describe("toCreateStudentPayload", () => {
  it("leaves advisor_id out when no advisor is picked", () => {
    expect(toCreateStudentPayload(form)).not.toHaveProperty("advisor_id")
  })

  it("sends the picked advisor", () => {
    const advisorId = "5b0f6c3e-8d0a-4c58-9a59-2f3a0f9a1b11"
    expect(
      toCreateStudentPayload({ ...form, advisor_id: advisorId })
    ).toMatchObject({ advisor_id: advisorId, student_number: "20260001" })
  })
})
