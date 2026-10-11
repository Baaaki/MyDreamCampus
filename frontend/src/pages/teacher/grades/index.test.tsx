import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter } from "react-router"
import type { TeacherCourseDTO } from "@/lib/types"
import GradesPage from "./index"

function stubCourses(courses: TeacherCourseDTO[], status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn<typeof fetch>(
      async () =>
        new Response(
          JSON.stringify(
            status === 200
              ? { instructor_id: "t-1", total_courses: courses.length, courses }
              : { error: "Sunucu hatası", code: "INTERNAL" }
          ),
          { status, headers: { "Content-Type": "application/json" } }
        )
    )
  )
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <GradesPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const course: TeacherCourseDTO = {
  id: "c-101",
  course_code: "CENG101",
  course_name: "Programlamaya Giriş",
  faculty: "Mühendislik Fakültesi",
  department: "Bilgisayar Mühendisliği",
  semester: "2026-2027-Fall",
  credits: 4,
  theoretical_hours: 3,
  lab_hours: 0,
  classroom_location: "B-201",
  max_capacity: 60,
  schedule: null,
}

describe("GradesPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("lists the teacher's courses with a grade entry button", async () => {
    stubCourses([course])
    renderPage()

    expect(await screen.findByText("Programlamaya Giriş")).toBeInTheDocument()
    expect(
      screen.getByText(
        "2026-2027-Fall döneminde verdiğiniz dersler için not girin"
      )
    ).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Not Gir" })).toBeInTheDocument()
  })

  it("reports a failed load", async () => {
    stubCourses([], 500)
    renderPage()

    expect(
      await screen.findByText("Dersler yüklenirken bir hata oluştu.")
    ).toBeInTheDocument()
  })
})
