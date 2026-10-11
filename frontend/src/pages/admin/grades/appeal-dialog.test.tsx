import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { AssessmentStatus, CourseResult } from "@/lib/types"
import { GradeAppealDialog } from "./appeal-dialog"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const student: CourseResult = {
  student_id: "st-1",
  student_number: "2024001",
  first_name: "Ali",
  last_name: "Veli",
  scores: { midterm: 40, final: 55 },
  weighted_average: 49,
  grade_point: "DD",
  is_attendance_failed: false,
}

const assessments: AssessmentStatus[] = [
  {
    slug: "midterm",
    name: "Vize",
    weight: 40,
    graded_count: 1,
    pending_count: 0,
    is_complete: true,
  },
  {
    slug: "final",
    name: "Final",
    weight: 60,
    graded_count: 1,
    pending_count: 0,
    is_complete: true,
  },
]

function stubBackend(answer: () => Response) {
  const bodies: unknown[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      if (
        request.method === "POST" &&
        new URL(request.url).pathname === "/api/grades/admin/appeals"
      ) {
        bodies.push(await request.clone().json())
        return answer()
      }
      return json({ error: "yok" }, 404)
    })
  )
  return bodies
}

function renderDialog() {
  const onOpenChange = vi.fn()
  render(
    <QueryClientProvider client={new QueryClient()}>
      <GradeAppealDialog
        courseId="c-1"
        student={student}
        assessments={assessments}
        onOpenChange={onOpenChange}
      />
    </QueryClientProvider>
  )
  return onOpenChange
}

async function fillForm(score: string, reason: string) {
  const user = userEvent.setup()
  await user.selectOptions(screen.getByLabelText("Değerlendirme"), "midterm")
  await user.type(screen.getByLabelText("Yeni Not (0-100)"), score)
  await user.type(screen.getByLabelText("Gerekçe"), reason)
  await user.click(screen.getByRole("button", { name: "İtirazı İşle" }))
}

describe("GradeAppealDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("keeps an invalid appeal on the client", async () => {
    const bodies = stubBackend(() => json({}))
    renderDialog()

    await fillForm("120", "kısa")

    expect(
      screen.getByText("Not 0 ile 100 arasında olmalı")
    ).toBeInTheDocument()
    expect(
      screen.getByText("Gerekçe en az 10 karakter olmalı")
    ).toBeInTheDocument()
    expect(bodies).toHaveLength(0)
  })

  it("sends the appeal and shows the regraded result", async () => {
    const bodies = stubBackend(() =>
      json({
        student_id: "st-1",
        course_code: "CENG101",
        slug: "midterm",
        old_score: 40,
        new_score: 60,
        old_weighted_average: 49,
        new_weighted_average: 57,
        old_grade_point: "DD",
        new_grade_point: "CC",
        grading_type: "relative",
        frozen_class_mean: 55,
      })
    )
    renderDialog()

    expect(screen.queryByText(/Mevcut not/)).not.toBeInTheDocument()
    await fillForm("60", "Toplama hatası düzeltildi")

    expect(await screen.findByText(/İtiraz işlendi/)).toBeInTheDocument()
    expect(bodies).toEqual([
      {
        student_id: "st-1",
        course_id: "c-1",
        slug: "midterm",
        new_score: 60,
        reason: "Toplama hatası düzeltildi",
      },
    ])
    expect(screen.getByText("Vize")).toBeInTheDocument()
    expect(screen.getByText("49.00")).toBeInTheDocument()
    expect(screen.getByText("57.00")).toBeInTheDocument()
    expect(screen.getByText("DD")).toBeInTheDocument()
    expect(screen.getByText("CC")).toBeInTheDocument()
  })

  it("shows the current score of the picked assessment", async () => {
    stubBackend(() => json({}))
    renderDialog()

    await userEvent.selectOptions(
      screen.getByLabelText("Değerlendirme"),
      "final"
    )

    expect(screen.getByText("Mevcut not: 55")).toBeInTheDocument()
  })

  it("shows the backend's refusal", async () => {
    stubBackend(() =>
      json(
        {
          error: "Not düzenleme süresi sona erdi",
          code: "GRADING_PERIOD_ENDED",
        },
        403
      )
    )
    renderDialog()

    await fillForm("60", "Toplama hatası düzeltildi")

    expect(
      await screen.findByText("Not düzenleme süresi sona erdi")
    ).toBeInTheDocument()
  })
})
