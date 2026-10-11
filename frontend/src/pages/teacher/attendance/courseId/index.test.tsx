import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter, Route, Routes } from "react-router"
import type {
  AttendanceFinalization,
  AttendanceFinalizationStatus,
  TeacherCourseDTO,
} from "@/lib/types"
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

const finalization: AttendanceFinalization = {
  course_id: "c-101",
  course_code: "CENG101",
  semester: "2026-2027-Fall",
  total_students: 3,
  total_weeks: 14,
  thresholds: { theory_min_required: 10, lab_min_required: 11 },
  finalization_summary: { passing_count: 2, failing_count: 1 },
  failed_students: [
    {
      student_id: "st-2",
      student_number: "2024002",
      student_name: "Ayşe Yılmaz",
      failed_type: "theory",
      theory: {
        present_count: 9,
        absent_count: 5,
        total_sessions: 14,
        min_required: 10,
        passed: false,
      },
    },
  ],
  events_published: 1,
  finalized_at: "2027-01-20T10:00:00Z",
}

interface Backend {
  listStatus?: number
  status?: AttendanceFinalizationStatus
  finalize?: () => Response
}

function stubBackend(
  courses: TeacherCourseDTO[],
  { listStatus = 200, status = { finalized: false }, finalize }: Backend = {}
) {
  const started: unknown[] = []
  const finalizeCalls: string[] = []
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
        case "GET /api/attendance/courses/c-101/finalization":
          return json(status)
        case "POST /api/attendance/courses/c-101/finalize":
          finalizeCalls.push(request.url)
          return finalize ? finalize() : json(finalization)
        default:
          return json({ error: "yok", code: "NOT_FOUND" }, 404)
      }
    })
  )
  return { started, finalizeCalls }
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
    stubBackend([], { listStatus: 500 })
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

  it("finalizes the course after a confirmation and shows the outcome", async () => {
    const { finalizeCalls } = stubBackend([course()])
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Yoklamayı Kesinleştir" })
    )
    expect(screen.getByText("Yoklama kesinleştirilsin mi?")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Kesinleştir" }))

    expect(
      await screen.findByText("Yoklama kesinleştirildi")
    ).toBeInTheDocument()
    expect(finalizeCalls).toHaveLength(1)
    // The course id names its semester; none is sent.
    expect(new URL(finalizeCalls[0]!).search).toBe("")
    expect(screen.getByText("Ayşe Yılmaz")).toBeInTheDocument()
    expect(screen.getByText("9 / 14")).toBeInTheDocument()
    expect(screen.getByText(/Teorik eşik: en az 10 oturum/)).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Yoklamayı Kesinleştir" })
    ).not.toBeInTheDocument()
  })

  it("does nothing when the teacher backs out of the confirmation", async () => {
    const { finalizeCalls } = stubBackend([course()])
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Yoklamayı Kesinleştir" })
    )
    await user.click(screen.getByRole("button", { name: "Vazgeç" }))

    expect(finalizeCalls).toHaveLength(0)
    expect(
      screen.getByRole("button", { name: "Yoklamayı Kesinleştir" })
    ).toBeInTheDocument()
  })

  it("shows the saved outcome of an already finalized course", async () => {
    const { finalizeCalls } = stubBackend([course({ lab_hours: 2 })], {
      status: { finalized: true, result: finalization },
    })
    renderPage()

    expect(
      await screen.findByText("Yoklama kesinleştirildi")
    ).toBeInTheDocument()
    expect(screen.getByText("Devamsızlıktan Kalan")).toBeInTheDocument()
    expect(
      screen.getByText(/Uygulama eşiği: en az 11 oturum/)
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Yoklamayı Kesinleştir" })
    ).not.toBeInTheDocument()
    expect(finalizeCalls).toHaveLength(0)
  })

  it("shows why the backend refused to finalize", async () => {
    stubBackend([course()], {
      finalize: () =>
        json(
          {
            error:
              "Açık yoklama oturumu var; kesinleştirmeden önce oturumu kapatın",
            code: "OPEN_SESSION_EXISTS",
          },
          409
        ),
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Yoklamayı Kesinleştir" })
    )
    await user.click(screen.getByRole("button", { name: "Kesinleştir" }))

    expect(
      await screen.findByText(/Açık yoklama oturumu var/)
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Yoklamayı Kesinleştir" })
    ).toBeInTheDocument()
  })
})
