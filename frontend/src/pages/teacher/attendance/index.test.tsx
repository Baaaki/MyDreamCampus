import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter } from "react-router"
import type { TeacherCourseDTO } from "@/lib/types"
import AttendancePage from "./index"

function stubCourses(courses: TeacherCourseDTO[] | null, status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn<typeof fetch>(
      async () =>
        new Response(
          JSON.stringify(
            status === 200
              ? { instructor_id: "t-1", total_courses: 0, courses }
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
        <AttendancePage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const unscheduled: TeacherCourseDTO = {
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

describe("AttendancePage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("links to a course that has no weekly schedule", async () => {
    stubCourses([unscheduled])
    renderPage()

    const link = await screen.findByRole("link", {
      name: /Programlamaya Giriş/,
    })
    expect(link).toHaveAttribute("href", "/teacher/attendance/c-101")
  })

  it("shows the empty notice when the teacher has no courses", async () => {
    stubCourses(null)
    renderPage()

    expect(
      await screen.findByText("Bu dönemde verdiğiniz ders bulunmamaktadır.")
    ).toBeInTheDocument()
  })

  it("reports a failed load", async () => {
    stubCourses(null, 500)
    renderPage()

    expect(
      await screen.findByText("Dersler yüklenirken bir hata oluştu.")
    ).toBeInTheDocument()
  })
})
