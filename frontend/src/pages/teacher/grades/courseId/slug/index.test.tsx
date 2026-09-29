import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import type {
  BulkSubmitScoresRequest,
  CourseStatusResponse,
  StudentGrades,
} from "@/lib/types"
import GradeEntryPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const STATUS: CourseStatusResponse = {
  course_id: "course-123",
  course_code: "CENG201",
  course_name: "Veri Yapıları",
  semester: "2026-2027-Fall",
  total_students: 3,
  is_finalized: false,
  assessments: [
    {
      slug: "vize",
      name: "Vize",
      weight: 40,
      graded_count: 2,
      pending_count: 1,
      is_complete: false,
    },
    {
      slug: "final",
      name: "Final",
      weight: 60,
      graded_count: 0,
      pending_count: 3,
      is_complete: false,
    },
  ],
}

function student(
  id: string,
  number: string,
  first: string,
  last: string,
  vize?: { score: number | null; is_absent?: boolean; is_locked?: boolean }
): StudentGrades {
  return {
    registration_id: `reg-${id}`,
    student_id: `stu-${id}`,
    student_number: number,
    first_name: first,
    last_name: last,
    current_average: null,
    scores: vize
      ? {
          vize: {
            score: vize.score,
            is_absent: vize.is_absent ?? false,
            is_locked: vize.is_locked ?? false,
          },
        }
      : {},
  }
}

const STUDENTS = [
  student("1", "2024001", "Ali", "Yılmaz", { score: 85 }),
  student("2", "2024002", "Ayşe", "Kaya"),
  student("3", "2024003", "Mehmet", "Demir", { score: 72, is_locked: true }),
]

/** Stands in for the grades service and records every bulk save. */
function stubBackend({
  status = STATUS,
  statusCode = 200,
  save = () => json({ slug: "vize", success_count: 1 }),
}: {
  status?: CourseStatusResponse
  statusCode?: number
  save?: () => Response
} = {}) {
  const saved: BulkSubmitScoresRequest[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      switch (`${request.method} ${new URL(request.url).pathname}`) {
        case "GET /api/grades/courses/course-123/status":
          return json(status, statusCode)
        case "GET /api/grades/courses/course-123/students":
          return json({
            course_id: "course-123",
            course_code: "CENG201",
            students: STUDENTS,
          })
        case "POST /api/grades/courses/course-123/scores/bulk":
          saved.push(await request.clone().json())
          return save()
      }
      return json({}, 404)
    })
  )
  return saved
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/teacher/grades/course-123/vize"]}>
      <Routes>
        <Route
          path="/teacher/grades/:courseId/:slug"
          element={<GradeEntryPage />}
        />
        <Route path="/teacher/grades" element={<div>grades-home</div>} />
      </Routes>
    </MemoryRouter>
  )
}

function row(name: string) {
  const tr = screen.getByText(name).closest("tr")
  if (!tr) throw new Error(`no table row for ${name}`)
  return within(tr)
}

const saveButton = () => screen.getByRole("button", { name: /^Kaydet/ })

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("GradeEntryPage", () => {
  it("shows the course, the assessment and how many students are graded", async () => {
    stubBackend()
    renderPage()

    expect(await screen.findByText("Veri Yapıları")).toBeInTheDocument()
    expect(screen.getByText("CENG201")).toBeInTheDocument()
    expect(screen.getByText("%40")).toBeInTheDocument()
    expect(screen.getByText("2/3 öğrenci notlandırıldı")).toBeInTheDocument()
    expect(row("Ali Yılmaz").getByRole("spinbutton")).toHaveValue(85)
    expect(saveButton()).toBeDisabled()
  })

  it("shows a locked score without an input", async () => {
    stubBackend()
    renderPage()

    await screen.findByText("Mehmet Demir")
    expect(row("Mehmet Demir").getByText("72")).toBeInTheDocument()
    expect(
      row("Mehmet Demir").queryByRole("spinbutton")
    ).not.toBeInTheDocument()
    expect(row("Mehmet Demir").queryByRole("checkbox")).not.toBeInTheDocument()
  })

  it("filters students by name or number", async () => {
    stubBackend()
    const user = userEvent.setup()
    renderPage()

    const search = await screen.findByPlaceholderText(/Öğrenci ara/)
    await user.type(search, "Ayşe")
    expect(screen.queryByText("Ali Yılmaz")).not.toBeInTheDocument()
    expect(screen.getByText("Ayşe Kaya")).toBeInTheDocument()

    await user.clear(search)
    await user.type(search, "2024003")
    expect(screen.getByText("Mehmet Demir")).toBeInTheDocument()
    expect(screen.queryByText("Ayşe Kaya")).not.toBeInTheDocument()
  })

  it("sends only the changed rows and reports the save", async () => {
    const saved = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ayşe Kaya")
    await user.type(row("Ayşe Kaya").getByRole("spinbutton"), "90")
    await user.click(
      screen.getByRole("button", { name: "Kaydet (1 değişiklik)" })
    )

    expect(
      await screen.findByText("1 öğrencinin notu başarıyla kaydedildi.")
    ).toBeInTheDocument()
    expect(saved).toEqual([
      {
        slug: "vize",
        scores: [{ registration_id: "reg-2", score: 90, is_absent: false }],
      },
    ])
    expect(saveButton()).toHaveTextContent(/^Kaydet$/)
  })

  it("keeps the decimals of a score", async () => {
    const saved = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ayşe Kaya")
    await user.type(row("Ayşe Kaya").getByRole("spinbutton"), "85.5")
    await user.click(saveButton())

    await screen.findByText("1 öğrencinin notu başarıyla kaydedildi.")
    expect(saved[0].scores).toEqual([
      { registration_id: "reg-2", score: 85.5, is_absent: false },
    ])
  })

  it("records an absence without a score", async () => {
    const saved = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ayşe Kaya")
    await user.click(row("Ayşe Kaya").getByRole("checkbox"))
    expect(row("Ayşe Kaya").getByRole("spinbutton")).toBeDisabled()
    await user.click(saveButton())

    await screen.findByText("1 öğrencinin notu başarıyla kaydedildi.")
    expect(saved[0].scores).toEqual([
      { registration_id: "reg-2", score: null, is_absent: true },
    ])
  })

  it("refuses a score outside 0-100 without calling the service", async () => {
    const saved = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ali Yılmaz")
    const input = row("Ali Yılmaz").getByRole("spinbutton")
    await user.clear(input)
    await user.type(input, "120")
    expect(input).toHaveAttribute("aria-invalid", "true")
    await user.click(saveButton())

    expect(
      await screen.findByText(
        "1 öğrencinin notu geçersiz. Notlar 0-100 arasında olmalıdır."
      )
    ).toBeInTheDocument()
    expect(saved).toEqual([])
  })

  it("says so when the save completed and finalized the course", async () => {
    stubBackend({
      save: () =>
        json({ slug: "vize", success_count: 1, auto_finalized: true }),
    })
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ayşe Kaya")
    await user.type(row("Ayşe Kaya").getByRole("spinbutton"), "70")
    await user.click(saveButton())

    expect(
      await screen.findByText(/ders notları otomatik olarak kesinleştirildi/)
    ).toBeInTheDocument()
  })

  it("keeps the unsaved change when the save fails", async () => {
    stubBackend({
      save: () =>
        json(
          { error: "Beklenmeyen bir hata oluştu", code: "INTERNAL_ERROR" },
          500
        ),
    })
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("Ayşe Kaya")
    await user.type(row("Ayşe Kaya").getByRole("spinbutton"), "70")
    await user.click(saveButton())

    expect(
      await screen.findByText("Notlar kaydedilirken bir hata oluştu.")
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Kaydet (1 değişiklik)" })
    ).toBeEnabled()
    expect(row("Ayşe Kaya").getByRole("spinbutton")).toHaveValue(70)
  })

  it("locks every input once the course is finalized", async () => {
    stubBackend({ status: { ...STATUS, is_finalized: true } })
    renderPage()

    expect(
      await screen.findByText(
        "Bu dersin notları kesinleştirilmiştir. Değişiklik yapılamaz."
      )
    ).toBeInTheDocument()
    for (const input of screen.getAllByRole("spinbutton")) {
      expect(input).toBeDisabled()
    }
    for (const box of screen.getAllByRole("checkbox")) {
      expect(box).toBeDisabled()
    }
  })

  it("offers the way back when the course cannot be loaded", async () => {
    stubBackend({ statusCode: 404 })
    renderPage()

    expect(
      await screen.findByText("Veriler yüklenirken bir hata oluştu.")
    ).toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: "Not Girme sayfasına dön" })
    ).toHaveAttribute("href", "/teacher/grades")
  })
})
