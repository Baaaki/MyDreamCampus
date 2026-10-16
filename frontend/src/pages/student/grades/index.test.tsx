import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { CompletedCourse, MyGradesResponse } from "@/lib/types"
import StudentGradesPage from "./index"

function completed(
  semester: string,
  code: string,
  gradePoint: string
): CompletedCourse {
  return {
    course_code: code,
    course_name: `${code} Dersi`,
    semester,
    credits: 4,
    weighted_average: 86,
    grade_point: gradePoint,
  }
}

function renderWith(body: MyGradesResponse) {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        })
    )
  )
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <StudentGradesPage />
    </QueryClientProvider>
  )
}

describe("StudentGradesPage", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("opens on the latest semester, named for its real term", async () => {
    renderWith({
      student_id: "s-1",
      student_number: "20240001",
      active_courses: [],
      completed_courses: [
        completed("2025-2026-Fall", "BLM101", "4.00"),
        completed("2025-2026-Spring", "BLM102", "3.50"),
      ],
      cumulative_gpa: 3.75,
      total_credits: 8,
    })

    expect(await screen.findByText("BLM102")).toBeInTheDocument()
    expect(screen.queryByText("BLM101")).toBeNull()
    expect(screen.getAllByText("2025-2026 Bahar").length).toBeGreaterThan(0)
    expect(screen.queryByText("2025-2026 Güz")).toBeNull()
  })

  it("shows the grade point the backend stores, not a letter", async () => {
    renderWith({
      student_id: "s-1",
      student_number: "20240001",
      active_courses: [],
      completed_courses: [completed("2025-2026-Spring", "BLM102", "3.50")],
      cumulative_gpa: 3.5,
      total_credits: 4,
    })

    const row = (await screen.findByText("BLM102")).closest("tr")!
    expect(within(row).getByText("3.50")).toBeInTheDocument()
    expect(screen.getByText("Katsayı")).toBeInTheDocument()
    expect(screen.queryByText("Harf Notu")).toBeNull()
  })
})
