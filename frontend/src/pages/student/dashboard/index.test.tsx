import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type {
  CourseAttendanceDetail,
  EnrollmentProgramResponse,
} from "@/lib/types"
import { toLocalISODate } from "@/lib/services/meal-service"
import StudentDashboardPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const STUDENT = {
  id: "student-1",
  student_number: "20230001",
  first_name: "Zeynep",
  last_name: "Arslan",
  email: "zeynep@example.edu",
  faculty: "Mühendislik Fakültesi",
  department: "Bilgisayar Mühendisliği",
}

const GRADES = {
  student_id: "student-1",
  student_number: "20230001",
  active_courses: [],
  completed_courses: [{ course_code: "CENG102" }],
  cumulative_gpa: 3.256,
  total_credits: 34,
}

function course(
  code: string,
  theoryAbsent: number,
  labAbsent?: number
): CourseAttendanceDetail {
  const stats = (absent: number) => ({
    present_count: 6 - absent,
    absent_count: absent,
    total_sessions: 6,
    min_required: 5,
    passed: 6 - absent >= 5,
  })
  return {
    course_id: `id-${code}`,
    course_code: code,
    course_name: code,
    instructor: "",
    total_weeks: 14,
    theory: stats(theoryAbsent),
    lab: labAbsent === undefined ? undefined : stats(labAbsent),
    weekly_records: [],
  }
}

const ATTENDANCE = {
  student_id: "student-1",
  student_number: "20230001",
  semester: "2026-2027-Fall",
  courses: [course("CENG201", 1, 2), course("MATH201", 0)],
}

function reservation(id: string, date: string, status: string, used = false) {
  return {
    id,
    date,
    meal_time: "lunch",
    menu_type: "normal",
    cafeteria_name: `Yemekhane ${id}`,
    status,
    is_used: used,
    created_at: "2026-10-01T12:00:00Z",
  }
}

const MEALS = {
  reservations: [
    reservation("b", "2026-10-14", "confirmed"),
    reservation("a", "2026-10-13", "confirmed"),
    reservation("c", "2026-10-15", "cancelled"),
    reservation("d", "2026-10-16", "pending"),
  ],
  summary: { total: 4, confirmed: 2, pending: 1, used: 0, cancelled: 1 },
}

function program(semester: string, codes: string[]): EnrollmentProgramResponse {
  return {
    id: semester,
    student_id: "student-1",
    semester,
    status: "approved",
    created_at: "2026-09-01T00:00:00Z",
    courses: codes.map((code) => ({
      id: code,
      course_code: code,
      course_name: code,
      credits: 4,
    })),
  }
}

const PROGRAMS = [
  program("2026-2027-Spring", ["CENG202"]),
  program("2026-2027-Fall", ["CENG201", "MATH201", "PHYS101"]),
]

function stubBackend(overrides: Record<string, Response> = {}) {
  const asked: Record<string, string> = {}
  const responses: Record<string, () => Response> = {
    "/api/students/student-1": () => json(STUDENT),
    "/api/grades/my/grades": () => json(GRADES),
    "/api/attendance/my": () => json(ATTENDANCE),
    "/api/meals/reservations/my": () => json({ success: true, data: MEALS }),
    "/api/enrollment/my-enrollments": () => json({ programs: PROGRAMS }),
  }
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL((input as Request).url)
      asked[url.pathname] = url.search
      const override = overrides[url.pathname]
      if (override) return override.clone()
      return responses[url.pathname]?.() ?? json({}, 404)
    })
  )
  return asked
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <StudentDashboardPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const card = (title: string) =>
  within(screen.getByText(title).closest("[data-slot=card]") as HTMLElement)

beforeEach(() => {
  localStorage.setItem("user", JSON.stringify({ id: "student-1" }))
})

afterEach(() => {
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe("StudentDashboardPage", () => {
  it("summarises the student's own grades, attendance, meals and program", async () => {
    const asked = stubBackend()
    renderPage()

    expect(await screen.findByText("3.26")).toBeInTheDocument()
    expect(card("Akademik Durum").getByText("34")).toBeInTheDocument()

    const absences = card("Devamsızlık")
    expect(absences.getByText("CENG201 · Lab")).toBeInTheDocument()
    expect(
      absences.getByText("1 devamsızlık hakkınız kaldı")
    ).toBeInTheDocument()
    expect(absences.queryByText(/MATH201/)).not.toBeInTheDocument()

    const meals = card("Yaklaşan Yemekler")
    expect(
      meals.getAllByRole("listitem").map((li) => li.textContent),
      "confirmed meals only, soonest first"
    ).toEqual([
      expect.stringContaining("Yemekhane a"),
      expect.stringContaining("Yemekhane b"),
    ])
    expect(asked["/api/meals/reservations/my"]).toBe(
      `?from_date=${toLocalISODate(new Date())}`
    )

    const programCard = card("Ders Programı")
    expect(
      programCard.getByText("2026-2027 Güz onaylı programınız")
    ).toBeInTheDocument()
    expect(programCard.getByText("3 ders")).toBeInTheDocument()
    expect(asked["/api/enrollment/my-enrollments"]).toBe("?status=approved")

    for (const stale of [
      "Kurumsal Duyurular",
      "Bilgi İşlem Duyuruları",
      "Mesajlar",
      "Kablosuz Ağ Bağlantısı",
    ]) {
      expect(screen.queryByText(stale)).not.toBeInTheDocument()
    }
    expect(screen.getByText("Günün Sözü")).toBeInTheDocument()
  })

  it("says so when there is nothing to summarise yet", async () => {
    stubBackend({
      "/api/grades/my/grades": json({
        ...GRADES,
        completed_courses: null,
        cumulative_gpa: 0,
        total_credits: 0,
      }),
      "/api/attendance/my": json({ ...ATTENDANCE, courses: [] }),
      "/api/meals/reservations/my": json({
        success: true,
        data: { ...MEALS, reservations: [] },
      }),
      "/api/enrollment/my-enrollments": json({ programs: [] }),
    })
    renderPage()

    expect(
      await screen.findByText("Henüz not ortalamanız oluşmadı.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Devamsızlık sınırına yaklaşan dersiniz yok.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Yaklaşan yemek rezervasyonunuz yok.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Onaylı bir ders programınız yok.")
    ).toBeInTheDocument()
  })

  it("keeps the other cards when one source fails", async () => {
    stubBackend({
      "/api/meals/reservations/my": json(
        { error: "Yemek servisi şu anda yanıt vermiyor", code: "INTERNAL" },
        503
      ),
    })
    renderPage()

    expect(
      await screen.findByText("Yemek servisi şu anda yanıt vermiyor")
    ).toBeInTheDocument()
    expect(await screen.findByText("3.26")).toBeInTheDocument()
    expect(await screen.findByText("3 ders")).toBeInTheDocument()
  })
})
