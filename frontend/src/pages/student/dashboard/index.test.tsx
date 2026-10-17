import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type {
  CourseAttendanceDetail,
  EnrollmentProgramResponse,
  OpenSemester,
} from "@/lib/types"
import StudentDashboardPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

// Wednesday 21 October 2026, 13:20 local: CENG304 is in progress.
const NOW = new Date(2026, 9, 21, 13, 20)

const STUDENT = {
  id: "student-1",
  student_number: "20230001",
  first_name: "Zeynep",
  last_name: "Arslan",
  email: "zeynep@example.edu",
  faculty: "Mühendislik Fakültesi",
  department: "Bilgisayar Mühendisliği",
  class_level: 2,
}

const GRADES = {
  student_id: "student-1",
  student_number: "20230001",
  active_courses: [{ course_code: "CENG302" }, { course_code: "CENG304" }],
  completed_courses: [{ course_code: "CENG102" }],
  cumulative_gpa: 3.256,
  total_credits: 34,
}

function course(
  code: string,
  name: string,
  theoryAbsent: number,
  attendedToday = false
): CourseAttendanceDetail {
  return {
    course_id: `id-${code}`,
    course_code: code,
    course_name: name,
    instructor: "",
    total_weeks: 14,
    theory: {
      present_count: 5 - theoryAbsent,
      absent_count: theoryAbsent,
      total_sessions: 5,
      min_required: 4,
      passed: true,
    },
    weekly_records: attendedToday
      ? [
          {
            week: 5,
            session_type: "theory",
            date: "2026-10-21",
            marked_via: "qr_scan",
          },
        ]
      : [],
    missed_sessions: [],
  }
}

const ATTENDANCE = {
  student_id: "student-1",
  student_number: "20230001",
  semester: "2026-2027-Fall",
  courses: [
    course("CENG302", "Bilgisayar Ağları", 0, true),
    course("CENG304", "Veritabanı Sistemleri", 3),
    course("CENG402", "Dağıtık Sistemler", 4),
  ],
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
    reservation("today", "2026-10-21", "confirmed", true),
    reservation("tomorrow", "2026-10-22", "confirmed"),
    reservation("cancelled", "2026-10-23", "cancelled"),
  ],
  summary: { total: 3, confirmed: 2, pending: 0, used: 1, cancelled: 1 },
}

const theory = (day: string, slots: number[]) => ({
  day_of_week: day,
  slot_numbers: slots,
  session_type: "theory" as const,
})

function program(
  semester: string,
  courses: [string, string, ReturnType<typeof theory>[]][]
): EnrollmentProgramResponse {
  return {
    id: semester,
    student_id: "student-1",
    semester,
    status: "approved",
    created_at: "2026-09-01T00:00:00Z",
    courses: courses.map(([code, name, sessions]) => ({
      id: code,
      course_code: code,
      course_name: name,
      credits: 3,
      instructor: "Ahmet Yılmaz",
      schedule_sessions: sessions,
    })),
  }
}

const PROGRAMS = [
  program("2026-2027-Spring", [["CENG499", "Bitirme", []]]),
  program("2026-2027-Fall", [
    ["CENG302", "Bilgisayar Ağları", [theory("wednesday", [1, 2, 3])]],
    ["CENG304", "Veritabanı Sistemleri", [theory("wednesday", [6, 7, 8])]],
    ["CENG402", "Dağıtık Sistemler", [theory("thursday", [1, 2, 3])]],
  ]),
]

const OPEN_SEMESTERS: OpenSemester[] = [
  {
    semester: "2026-2027-Fall",
    enrollment_open: false,
    period_start: "2026-09-01T06:00:00Z",
    period_end: "2026-09-14T14:00:00Z",
    has_program: true,
  },
  {
    semester: "2026-2027-Spring",
    enrollment_open: true,
    period_start: "2026-10-15T06:00:00Z",
    period_end: "2026-11-04T14:00:00Z",
    has_program: false,
  },
]

function stubBackend(overrides: Record<string, Response> = {}) {
  const asked: Record<string, string> = {}
  const responses: Record<string, () => Response> = {
    "/api/students/student-1": () => json(STUDENT),
    "/api/grades/my/grades": () => json(GRADES),
    "/api/attendance/my": () => json(ATTENDANCE),
    "/api/meals/reservations/my": () => json({ success: true, data: MEALS }),
    "/api/enrollment/my-enrollments": () => json({ programs: PROGRAMS }),
    "/api/enrollment/open-semesters": () => json(OPEN_SEMESTERS),
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

const section = (title: string) =>
  within(
    screen
      .getByRole("heading", { name: title })
      .closest("section") as HTMLElement
  )

beforeEach(() => {
  // Only Date is faked: react-query and findBy* still need real timers.
  vi.useFakeTimers({ toFake: ["Date"] })
  vi.setSystemTime(NOW)
  localStorage.setItem("user", JSON.stringify({ id: "student-1" }))
})

afterEach(() => {
  vi.useRealTimers()
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe("StudentDashboardPage", () => {
  it("leads with the class in progress and what comes next", async () => {
    const asked = stubBackend()
    renderPage()

    expect(
      await screen.findByRole("heading", { name: "Zeynep Arslan" })
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        "20230001 · Bilgisayar Mühendisliği, 2. sınıf · Çarşamba, 21 Ekim"
      )
    ).toBeInTheDocument()

    const now = within(
      await screen.findByRole("region", { name: "Şu an ve sıradaki ders" })
    )
    expect(now.getByText("Şu an derstesiniz")).toBeInTheDocument()
    expect(now.getByText("Veritabanı Sistemleri")).toBeInTheDocument()
    expect(now.getByText("Bitiş 15:35")).toBeInTheDocument()
    expect(now.getByText("Dağıtık Sistemler")).toBeInTheDocument()
    expect(now.getByText("Yarın 08:30")).toBeInTheDocument()
    expect(asked["/api/enrollment/my-enrollments"]).toBe("?status=approved")
  })

  it("lists what the student has to act on", async () => {
    stubBackend()
    renderPage()

    const todo = await screen.findByRole("heading", {
      name: "Yapmanız gerekenler",
    })
    const list = within(todo.closest("section") as HTMLElement)
    expect(
      await list.findByText("2026-2027 Bahar ders kaydı açık.")
    ).toBeInTheDocument()
    expect(list.getByText("Dağıtık Sistemler · Teori")).toBeInTheDocument()
    expect(list.getByText(/Devamsızlık hakkınız bitti/)).toBeInTheDocument()
    expect(list.getByText("Veritabanı Sistemleri · Teori")).toBeInTheDocument()
    expect(list.queryByText(/Bilgisayar Ağları/)).not.toBeInTheDocument()
  })

  it("shows today's classes and meals, the record and upcoming meals", async () => {
    const asked = stubBackend()
    renderPage()

    const today = section("Bugün")
    expect(
      await today.findByText("Yoklamaya katıldınız", { exact: false })
    ).toBeInTheDocument()
    expect(today.getByText("Öğle yemeği")).toBeInTheDocument()
    expect(
      today.getByText("Devam ediyor", { exact: false })
    ).toBeInTheDocument()

    expect(await screen.findByText("3.26")).toBeInTheDocument()
    expect(section("Notlar").getByText("34")).toBeInTheDocument()

    const absences = section("Devamsızlık")
    expect(absences.getByText("Hak kalmadı")).toBeInTheDocument()
    expect(absences.getByText("1 hak")).toBeInTheDocument()
    expect(absences.getByText("4 hak")).toBeInTheDocument()

    const meals = section("Yemek")
    expect(
      (await meals.findAllByRole("listitem")).map((li) => li.textContent),
      "confirmed meals after today only"
    ).toEqual([expect.stringContaining("Yemekhane tomorrow")])
    expect(meals.getByText("Yarın öğle")).toBeInTheDocument()
    expect(asked["/api/meals/reservations/my"]).toBe("?from_date=2026-10-21")

    expect(section("Bu hafta").getAllByText("CENG304")).toHaveLength(1)
    expect(screen.queryByText("Günün Sözü")).not.toBeInTheDocument()
  })

  it("says so when there is nothing to show yet", async () => {
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
      "/api/enrollment/open-semesters": json([]),
    })
    renderPage()

    expect(
      await screen.findByText("Onaylı bir ders programınız yok.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Henüz not ortalamanız oluşmadı.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Henüz yoklama alınmadı.")
    ).toBeInTheDocument()
    expect(
      await screen.findByText("Yaklaşan yemek rezervasyonunuz yok.")
    ).toBeInTheDocument()
    expect(screen.getByText("Bugün dersiniz yok.")).toBeInTheDocument()
    expect(screen.queryByText("Yapmanız gerekenler")).not.toBeInTheDocument()
    expect(screen.queryByText("Bu hafta")).not.toBeInTheDocument()
  })

  it("keeps the other sections when one source fails", async () => {
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
    expect(await screen.findByText("Şu an derstesiniz")).toBeInTheDocument()
  })
})
