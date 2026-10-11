import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter, Route, Routes } from "react-router"
import type { TeacherCourseDTO } from "@/lib/types"
import AttendanceStartPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function course(overrides: Partial<TeacherCourseDTO> = {}): TeacherCourseDTO {
  return {
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
    schedule: [],
    ...overrides,
  }
}

function stubBackend(courses: TeacherCourseDTO[], listStatus = 200) {
  const started: unknown[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const { pathname } = new URL(request.url)
      switch (`${request.method} ${pathname}`) {
        case "GET /api/semesters/teacher/courses":
          return listStatus === 200
            ? json({
                instructor_id: "t-1",
                total_courses: courses.length,
                courses,
              })
            : json({ error: "Sunucu hatası", code: "INTERNAL" }, listStatus)
        case "POST /api/attendance/sessions":
          started.push(await request.json())
          return json({ session_id: "s-1" }, 201)
        default:
          return json({ error: "yok", code: "NOT_FOUND" }, 404)
      }
    })
  )
  return { started }
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/teacher/attendance/c-101"]}>
        <Routes>
          <Route
            path="/teacher/attendance/:courseId"
            element={<AttendanceStartPage />}
          />
          <Route
            path="/teacher/attendance/:courseId/session/:sessionId"
            element={<p>Oturum ekranı</p>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe("AttendanceStartPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("renders a course whose schedule arrives as null and still starts attendance", async () => {
    const { started } = stubBackend([course({ schedule: null })])
    const user = userEvent.setup()
    renderPage()

    expect(await screen.findByText("Programlamaya Giriş")).toBeInTheDocument()
    expect(screen.getByText("Ders programı tanımlanmamış")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /Yoklamayı Başlat/ }))
    await user.click(screen.getByRole("button", { name: "3" }))
    await user.click(screen.getByRole("button", { name: /^Başlat$/ }))

    expect(await screen.findByText("Oturum ekranı")).toBeInTheDocument()
    expect(started).toEqual([
      {
        course_id: "c-101",
        week_number: 3,
        duration_minutes: 30,
        session_type: "theory",
      },
    ])
  })

  it("lists the sessions of a scheduled course without the empty notice", async () => {
    stubBackend([
      course({
        schedule: [{ day: "Pazartesi", time: "09:00-10:30", room: "B-201" }],
      }),
    ])
    renderPage()

    expect(await screen.findByText("Pazartesi")).toBeInTheDocument()
    expect(screen.getByText("09:00-10:30")).toBeInTheDocument()
    expect(
      screen.queryByText("Ders programı tanımlanmamış")
    ).not.toBeInTheDocument()
  })

  it("says so when the course list cannot be loaded", async () => {
    stubBackend([], 500)
    renderPage()

    expect(
      await screen.findByText("Ders bilgileri yüklenirken bir hata oluştu.")
    ).toBeInTheDocument()
  })

  it("says so when the course is not among the teacher's courses", async () => {
    stubBackend([course({ id: "c-999" })])
    renderPage()

    expect(await screen.findByText("Ders bulunamadı.")).toBeInTheDocument()
  })
})
