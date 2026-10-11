import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { AssessmentStatus, CourseResult } from "@/lib/types"
import { CourseResultsTable } from "./course-results"

const assessments: AssessmentStatus[] = [
  {
    slug: "midterm",
    name: "Vize",
    weight: 40,
    graded_count: 2,
    pending_count: 0,
    is_complete: true,
  },
  {
    slug: "final",
    name: "Final",
    weight: 60,
    graded_count: 2,
    pending_count: 0,
    is_complete: true,
  },
]

const results: CourseResult[] = [
  {
    student_id: "st-1",
    student_number: "2024001",
    first_name: "Ali",
    last_name: "Veli",
    scores: { midterm: 40, final: 55 },
    weighted_average: 49,
    grade_point: "DD",
    is_attendance_failed: false,
  },
  {
    student_id: "st-2",
    student_number: "2024002",
    first_name: "Ayşe",
    last_name: "Kaya",
    scores: { midterm: 10 },
    weighted_average: 4,
    grade_point: "FF",
    is_attendance_failed: true,
  },
]

function stubResults(status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = new URL((input as Request).url).pathname
      const body =
        path === "/api/grades/admin/courses/c-1/results" && status === 200
          ? { course_id: "c-1", results }
          : { error: "Sunucu hatası" }
      return new Response(JSON.stringify(body), {
        status: path.endsWith("/results") ? status : 404,
        headers: { "Content-Type": "application/json" },
      })
    })
  )
}

function renderTable(onAppeal = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <CourseResultsTable
        courseId="c-1"
        assessments={assessments}
        onAppeal={onAppeal}
      />
    </QueryClientProvider>
  )
  return onAppeal
}

describe("CourseResultsTable", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("lists each student's frozen scores and grade", async () => {
    stubResults()
    renderTable()

    const ali = (await screen.findByText("2024001")).closest("tr")!
    expect(within(ali).getByText("40")).toBeInTheDocument()
    expect(within(ali).getByText("55")).toBeInTheDocument()
    expect(within(ali).getByText("49.00")).toBeInTheDocument()
    expect(within(ali).getByText("DD")).toBeInTheDocument()

    const ayse = screen.getByText("2024002").closest("tr")!
    expect(within(ayse).getByText("-")).toBeInTheDocument()
    expect(within(ayse).getByText("(devamsız)")).toBeInTheDocument()
  })

  it("opens an appeal for the chosen student", async () => {
    stubResults()
    const onAppeal = renderTable()

    const ayse = (await screen.findByText("2024002")).closest("tr")!
    await userEvent.click(
      within(ayse).getByRole("button", { name: /Not İtirazı/ })
    )

    expect(onAppeal).toHaveBeenCalledWith(results[1])
  })

  it("says so when the results cannot be read", async () => {
    stubResults(500)
    renderTable()

    expect(
      await screen.findByText("Ders sonuçları alınamadı.")
    ).toBeInTheDocument()
  })
})
