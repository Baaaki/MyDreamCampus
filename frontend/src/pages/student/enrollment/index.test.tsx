import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type {
  AvailableCourse,
  EnrollmentProgramResponse,
  OpenSemester,
} from "@/lib/types"
import StudentEnrollmentPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function course(
  id: string,
  code: string,
  day: string,
  slots: number[]
): AvailableCourse {
  return {
    id,
    course_code: code,
    course_name: `${code} dersi`,
    credits: 4,
    schedule_sessions: [
      { day_of_week: day, slot_numbers: slots, session_type: "theory" },
    ],
    max_capacity: 50,
    current_enrollment: 20,
    available_seats: 30,
    instructor: "Dr. Ahmet Yılmaz",
  }
}

const CENG101 = course("c-101", "CENG101", "monday", [1, 2])
// Shares Monday's second slot with CENG101.
const CENG103 = course("c-103", "CENG103", "monday", [2, 3])
const MATH101 = course("c-m101", "MATH101", "tuesday", [1, 2])
// Offered in spring; its prerequisite CENG102 is the backend's to check.
const CENG201 = course("c-201", "CENG201", "wednesday", [3, 4])

const FALL = "2026-2027-Fall"
const SPRING = "2026-2027-Spring"

function openSemester(
  semester: string,
  enrollmentOpen: boolean,
  hasProgram = false
): OpenSemester {
  return {
    semester,
    enrollment_open: enrollmentOpen,
    period_start: "2026-09-01T00:00:00Z",
    period_end: "2026-09-30T23:59:59Z",
    has_program: hasProgram,
  }
}

function program(
  status: string,
  courses: AvailableCourse[],
  semester = FALL
): EnrollmentProgramResponse {
  return {
    id: "prog-1",
    student_id: "student-1",
    semester,
    status,
    created_at: "2026-09-28T09:00:00Z",
    courses: courses.map((c) => ({
      id: c.id,
      course_code: c.course_code,
      course_name: c.course_name,
      credits: c.credits,
      schedule_sessions: c.schedule_sessions.map((s) => ({
        ...s,
        session_type: s.session_type ?? "theory",
      })),
    })),
  }
}

const REJECTION = {
  id: "rej-1",
  advisor_fullname: "Prof. Dr. Veli Demir",
  rejection_reason: "Kredi sınırı aşıldı, lütfen bir ders çıkarın.",
  rejected_at: "2026-09-27T09:00:00Z",
}

/**
 * Stands in for the enrollment service and records what the page sent to
 * it. Programs are served for the semester they belong to.
 */
function stubBackend({
  semesters = [openSemester(FALL, true)] as OpenSemester[] | Response,
  courses = json({
    student_id: "student-1",
    semester: FALL,
    available_courses: [CENG101, CENG103, MATH101],
  }),
  programs = [] as EnrollmentProgramResponse[],
  rejection = null as typeof REJECTION | null,
  submit = (body: { course_ids: string[] }) =>
    json(
      program(
        "pending",
        [CENG101, CENG103, MATH101].filter((c) =>
          body.course_ids.includes(c.id)
        )
      ),
      201
    ),
} = {}) {
  const sent = {
    submitted: [] as unknown[],
    deleted: [] as string[],
    coursesAskedFor: [] as (string | null)[],
  }
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const url = new URL(request.url)
      const semester = url.searchParams.get("semester")
      switch (`${request.method} ${url.pathname}`) {
        case "GET /api/enrollment/open-semesters":
          return semesters instanceof Response
            ? semesters.clone()
            : json(semesters)
        case "GET /api/enrollment/available-courses":
          sent.coursesAskedFor.push(semester)
          return courses.clone()
        case "GET /api/enrollment/my-enrollments":
          return json({
            programs: programs.filter((p) => p.semester === semester),
          })
        case "GET /api/enrollment/latest-rejection":
          return json(
            rejection
              ? { has_rejection: true, latest_rejection: rejection }
              : { has_rejection: false }
          )
        case "POST /api/enrollment/programs": {
          const body = await request.clone().json()
          sent.submitted.push(body)
          return submit(body)
        }
        case "DELETE /api/enrollment/programs":
          sent.deleted.push(url.search)
          return new Response(null, { status: 204 })
      }
      return json({}, 404)
    })
  )
  return sent
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <StudentEnrollmentPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const semesterButton = (name: RegExp) => screen.getByRole("button", { name })
const submitButton = (count: number) =>
  screen.getByRole("button", {
    name: `Ders Kaydını Tamamla (${count} Ders)`,
  })

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("StudentEnrollmentPage", () => {
  it("lists the offered courses with nothing selected", async () => {
    stubBackend()
    renderPage()

    expect(await screen.findByText("3 ders • 0 seçili")).toBeInTheDocument()
    for (const code of ["CENG101", "CENG103", "MATH101"]) {
      expect(screen.getByText(code)).toBeInTheDocument()
    }
    expect(submitButton(0)).toBeEnabled()
  })

  it("submits the chosen courses and shows the program as awaiting approval", async () => {
    const sent = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByText("CENG101"))
    await user.click(screen.getByText("MATH101"))
    expect(screen.getByText("3 ders • 2 seçili")).toBeInTheDocument()
    await user.click(submitButton(2))

    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).toBeInTheDocument()
    expect(sent.submitted).toEqual([
      {
        semester: FALL,
        course_ids: ["c-101", "c-m101"],
      },
    ])
    expect(
      screen.getByText("Ders kaydı başarıyla oluşturuldu")
    ).toBeInTheDocument()
    expect(
      screen.queryByText("Açılan Dersler"),
      "a submitted program is read-only"
    ).not.toBeInTheDocument()
  })

  it("refuses a course that clashes with one already chosen", async () => {
    stubBackend()
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByText("CENG101"))
    await user.click(screen.getByText("CENG103"))

    expect(
      await screen.findByText("Çakışma var: CENG103 & CENG101")
    ).toBeInTheDocument()
    expect(screen.getByText("3 ders • 1 seçili")).toBeInTheDocument()
    expect(submitButton(1)).toBeEnabled()
  })

  it("asks for a course before sending an empty program", async () => {
    const sent = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("3 ders • 0 seçili")
    await user.click(submitButton(0))

    expect(
      await screen.findByText("Lütfen en az bir ders seçin")
    ).toBeInTheDocument()
    expect(sent.submitted).toEqual([])
  })

  it("keeps the selection and explains a refused submission", async () => {
    stubBackend({
      submit: () =>
        json(
          {
            error: "CENG101 dersinin kontenjanı dolu",
            code: "COURSE_CAPACITY_FULL",
          },
          409
        ),
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByText("CENG101"))
    await user.click(submitButton(1))

    expect(
      await screen.findByText("CENG101 dersinin kontenjanı dolu")
    ).toBeInTheDocument()
    expect(submitButton(1)).toBeEnabled()
    expect(
      screen.queryByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).not.toBeInTheDocument()
  })

  it("explains why the course list could not be loaded", async () => {
    stubBackend({
      courses: json(
        {
          error: "Ders kayıt dönemi şu anda kapalı",
          code: "ENROLLMENT_PERIOD_CLOSED",
        },
        403
      ),
    })
    renderPage()

    expect(
      await screen.findByText("Ders kayıt dönemi şu anda kapalı")
    ).toBeInTheDocument()
    expect(screen.getByText("0 ders • 0 seçili")).toBeInTheDocument()
  })

  it("lets a student withdraw a pending program and start over", async () => {
    const sent = stubBackend({ programs: [program("pending", [CENG101])] })
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Yeniden Ders Kaydı Yap" })
    )

    expect(
      await screen.findByText("Yeni ders kaydı yapabilirsiniz")
    ).toBeInTheDocument()
    expect(sent.deleted).toEqual([`?semester=${FALL}`])
    expect(
      screen.queryByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).not.toBeInTheDocument()
    expect(screen.getByText("3 ders • 0 seçili")).toBeInTheDocument()
  })

  it("shows an approved program as final", async () => {
    stubBackend({ programs: [program("approved", [CENG101])] })
    renderPage()

    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylandı"
      )
    ).toBeInTheDocument()
    expect(screen.queryByText("Açılan Dersler")).not.toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: /Ders Kaydını Tamamla/ })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Yeniden Ders Kaydı Yap" })
    ).not.toBeInTheDocument()
  })

  it("shows the advisor's reason for a rejection and lets the student re-enroll", async () => {
    const sent = stubBackend({
      programs: [program("rejected", [])],
      rejection: REJECTION,
    })
    const user = userEvent.setup()
    renderPage()

    expect(
      await screen.findByText(REJECTION.rejection_reason)
    ).toBeInTheDocument()
    expect(screen.getByText(/Prof\. Dr\. Veli Demir/)).toBeInTheDocument()

    await user.click(
      screen.getByRole("button", { name: "Yeniden Ders Kaydı Yap" })
    )

    await waitFor(() =>
      expect(
        screen.queryByText(REJECTION.rejection_reason)
      ).not.toBeInTheDocument()
    )
    expect(sent.deleted).toHaveLength(1)
    expect(submitButton(0)).toBeEnabled()
  })

  it("keeps a rejected program's courses chosen so it can be adjusted and sent again", async () => {
    const sent = stubBackend({
      programs: [program("rejected", [CENG101])],
      rejection: REJECTION,
    })
    const user = userEvent.setup()
    renderPage()

    expect(
      await screen.findByText(REJECTION.rejection_reason)
    ).toBeInTheDocument()
    expect(screen.getByText("3 ders • 1 seçili")).toBeInTheDocument()

    await user.click(screen.getByText("MATH101"))
    await user.click(submitButton(2))

    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).toBeInTheDocument()
    expect(sent.submitted).toEqual([
      { semester: FALL, course_ids: ["c-101", "c-m101"] },
    ])
  })

  it("does not bring back an older rejection once a pending program is withdrawn", async () => {
    stubBackend({
      programs: [program("pending", [CENG101])],
      rejection: REJECTION,
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Yeniden Ders Kaydı Yap" })
    )

    expect(
      await screen.findByText("Yeni ders kaydı yapabilirsiniz")
    ).toBeInTheDocument()
    expect(screen.getByText("3 ders • 0 seçili")).toBeInTheDocument()
    expect(
      screen.queryByText(REJECTION.rejection_reason)
    ).not.toBeInTheDocument()
  })

  it("opens the semester whose window is open and still shows the current program", async () => {
    const sent = stubBackend({
      semesters: [openSemester(FALL, false, true), openSemester(SPRING, true)],
      courses: json({
        student_id: "student-1",
        semester: SPRING,
        available_courses: [CENG201],
      }),
      programs: [program("approved", [CENG101], FALL)],
      submit: (body) =>
        json(
          program(
            "pending",
            [CENG201].filter((c) => body.course_ids.includes(c.id)),
            SPRING
          ),
          201
        ),
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByText("CENG201"))
    expect(semesterButton(/2026-2027 Bahar/)).toHaveAttribute(
      "aria-pressed",
      "true"
    )
    await user.click(submitButton(1))
    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).toBeInTheDocument()
    expect(sent.submitted).toEqual([
      { semester: SPRING, course_ids: ["c-201"] },
    ])

    await user.click(semesterButton(/2026-2027 Güz/))

    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylandı"
      )
    ).toBeInTheDocument()
    expect(screen.getAllByText("CENG101").length).toBeGreaterThan(0)
    expect(screen.queryByText("CENG201")).not.toBeInTheDocument()
    expect(sent.coursesAskedFor, "a closed window offers no courses").toEqual([
      SPRING,
    ])
  })

  it("keeps a closed semester's pending program out of reach", async () => {
    stubBackend({
      semesters: [openSemester(FALL, false, true)],
      programs: [program("pending", [CENG101])],
    })
    renderPage()

    expect(
      await screen.findByText(
        "Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor"
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Yeniden Ders Kaydı Yap" })
    ).not.toBeInTheDocument()
  })

  it("says so when no semester is open for enrollment", async () => {
    const sent = stubBackend({ semesters: [] })
    renderPage()

    expect(
      await screen.findByText("Şu anda ders kaydına açık bir dönem bulunmuyor.")
    ).toBeInTheDocument()
    expect(sent.coursesAskedFor).toEqual([])
  })

  it("explains why the semesters could not be loaded", async () => {
    stubBackend({
      semesters: json({ error: "Sunucu hatası", code: "INTERNAL_ERROR" }, 500),
    })
    renderPage()

    expect(await screen.findByText("Sunucu hatası")).toBeInTheDocument()
  })
})
