import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { CourseAttendanceDetail, MyAttendanceResponse } from "@/lib/types"
import StudentAttendancePage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const CENG201: CourseAttendanceDetail = {
  course_id: "c-201",
  course_code: "CENG201",
  course_name: "Veri Yapıları",
  instructor: "Dr. Ahmet Yılmaz",
  total_weeks: 14,
  theory: {
    present_count: 3,
    absent_count: 1,
    total_sessions: 4,
    min_required: 3,
    passed: true,
  },
  lab: {
    present_count: 1,
    absent_count: 2,
    total_sessions: 3,
    min_required: 3,
    passed: false,
  },
  weekly_records: [
    { week: 2, session_type: "lab", date: "2026-10-08", marked_via: "manual" },
    {
      week: 1,
      session_type: "theory",
      date: "2026-09-29",
      marked_via: "qr_scan",
    },
  ],
  missed_sessions: [{ week: 2, session_type: "theory", date: "2026-10-06" }],
}

const MATH201: CourseAttendanceDetail = {
  course_id: "c-m201",
  course_code: "MATH201",
  course_name: "Lineer Cebir",
  instructor: "Dr. Ayşe Kaya",
  total_weeks: 14,
  theory: {
    present_count: 4,
    absent_count: 0,
    total_sessions: 4,
    min_required: 3,
    passed: true,
  },
  weekly_records: [],
  missed_sessions: [],
}

function stubAttendance(response: Response) {
  const asked: string[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL((input as Request).url)
      if (url.pathname === "/api/attendance/my") {
        asked.push(url.search)
        return response.clone()
      }
      return json({}, 404)
    })
  )
  return asked
}

function attendance(courses: CourseAttendanceDetail[]): MyAttendanceResponse {
  return {
    student_id: "student-1",
    student_number: "20230001",
    semester: "2026-2027-Fall",
    courses,
  }
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <StudentAttendancePage />
    </QueryClientProvider>
  )
}

const courseCard = (code: RegExp) =>
  screen.getByText(code).closest("[data-slot=card]") as HTMLElement

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("StudentAttendancePage", () => {
  it("shows theory and lab standing per course for the active semester", async () => {
    const asked = stubAttendance(json(attendance([CENG201, MATH201])))
    renderPage()

    expect(
      await screen.findByText("2026-2027 Güz dönemi yoklama durumunuz")
    ).toBeInTheDocument()
    expect(asked, "the backend picks the active semester").toEqual([""])

    const ceng = within(courseCard(/CENG201/))
    const theory = within(ceng.getByRole("region", { name: "Teori" }))
    expect(theory.getByText("3 / 4")).toBeInTheDocument()
    expect(theory.getByText("Yeterli")).toBeInTheDocument()
    const lab = within(ceng.getByRole("region", { name: "Lab" }))
    expect(lab.getByText("1 / 3")).toBeInTheDocument()
    expect(lab.getByText("2")).toBeInTheDocument()
    expect(lab.getByText("Sınırın altında")).toBeInTheDocument()

    const weeks = ceng.getAllByRole("listitem").map((li) => li.textContent)
    expect(weeks, "oldest week first, missed ones among them").toEqual([
      "1. haftaTeori29 EylQR",
      "2. haftaLab8 EkiHoca",
      "2. haftaTeori6 EkiKatılmadınız",
    ])
  })

  it("leaves lab out of a course that has none", async () => {
    stubAttendance(json(attendance([MATH201])))
    renderPage()

    await screen.findByText(/MATH201/)
    const math = within(courseCard(/MATH201/))
    expect(math.getByRole("region", { name: "Teori" })).toBeInTheDocument()
    expect(math.queryByRole("region", { name: "Lab" })).not.toBeInTheDocument()
    expect(math.queryByText("Oturumlar")).not.toBeInTheDocument()
  })

  it("copes with an attendance service that predates missed sessions", async () => {
    const { missed_sessions: _missed, ...older } = CENG201
    stubAttendance(json({ ...attendance([]), courses: [older] }))
    renderPage()

    await screen.findByText(/CENG201/)
    expect(
      within(courseCard(/CENG201/))
        .getAllByRole("listitem")
        .map((li) => li.textContent)
    ).toEqual(["1. haftaTeori29 EylQR", "2. haftaLab8 EkiHoca"])
  })

  it("says so when the student has no courses this semester", async () => {
    stubAttendance(json(attendance([])))
    renderPage()

    expect(
      await screen.findByText("Bu dönem kayıtlı olduğunuz bir ders bulunmuyor.")
    ).toBeInTheDocument()
  })

  it("explains why attendance could not be loaded", async () => {
    stubAttendance(
      json({ error: "Öğrenci bulunamadı", code: "STUDENT_NOT_FOUND" }, 404)
    )
    renderPage()

    expect(await screen.findByText("Öğrenci bulunamadı")).toBeInTheDocument()
  })
})
