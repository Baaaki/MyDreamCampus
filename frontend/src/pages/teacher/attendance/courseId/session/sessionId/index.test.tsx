import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import AttendanceSessionPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const SESSION = {
  session_id: "sess-123",
  course_id: "course-123",
  course_code: "CENG101",
  course_name: "Bilgisayar Mühendisliğine Giriş",
  week_number: 5,
  session_type: "theory",
  session_date: "2026-09-28",
  semester: "2026-2027-Fall",
  is_active: true,
  qr_rotation_interval: 15,
  started_at: "2026-09-28T10:00:00Z",
  expires_at: new Date(Date.now() + 10 * 60_000).toISOString(),
  enrolled_student_count: 2,
  present_count: 1,
  absent_count: 1,
}

const RECORDS = {
  session_id: "sess-123",
  week_number: 5,
  total_count: 1,
  present_count: 1,
  records: [
    {
      id: "rec-1",
      student_id: "stu-1",
      student_number: "2024001",
      student_name: "Ali Yılmaz",
      is_present: true,
      marked_via: "qr",
      marked_at: "2026-09-28T10:05:00Z",
    },
  ],
}

const STUDENTS = [
  {
    student_id: "stu-1",
    student_number: "2024001",
    first_name: "Ali",
    last_name: "Yılmaz",
    email: "ali@uni.edu.tr",
    is_marked: true,
  },
  {
    student_id: "stu-2",
    student_number: "2024002",
    first_name: "Ayşe",
    last_name: "Kaya",
    email: "ayse@uni.edu.tr",
    is_marked: false,
  },
]

const MANUAL_OK = () =>
  json({
    id: "rec-2",
    session_id: "sess-123",
    student_id: "stu-2",
    student_number: "2024002",
    student_name: "Ayşe Kaya",
    is_present: true,
    marked_via: "manual",
    marked_at: "2026-09-28T10:15:00Z",
  })

/** Stands in for the attendance service and records what the page sent. */
function stubBackend({
  session = json(SESSION),
  students = STUDENTS,
  records = RECORDS,
  manual = MANUAL_OK,
  close = () => json({ message: "Yoklama kapatıldı" }),
}: {
  session?: Response
  students?: typeof STUDENTS
  records?: typeof RECORDS
  manual?: () => Response
  close?: () => Response
} = {}) {
  const sent = { manual: [] as unknown[], closed: 0, qrFetches: 0 }
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const base = "/api/attendance/sessions/sess-123"
      switch (`${request.method} ${new URL(request.url).pathname}`) {
        case `GET ${base}`:
          return session.clone()
        case `GET ${base}/qr`:
          sent.qrFetches++
          return json({
            session_id: "sess-123",
            qr_payload: {
              token: "qr-token",
              refresh_interval: 15,
              expires_at: "2026-09-28T10:30:00Z",
            },
            valid_until: "2026-09-28T10:30:00Z",
            rotation_interval: 15,
          })
        case `GET ${base}/records`:
          return json(records)
        case `GET ${base}/students`:
          return json({
            session_id: "sess-123",
            course_id: "course-123",
            total_enrolled: students.length,
            marked_count: students.filter((s) => s.is_marked).length,
            students,
          })
        case `POST ${base}/manual`:
          sent.manual.push(await request.clone().json())
          return manual()
        case `POST ${base}/close`:
          sent.closed++
          return close()
      }
      return json({}, 404)
    })
  )
  return sent
}

function renderPage() {
  return render(
    <MemoryRouter
      initialEntries={["/teacher/attendance/course-123/session/sess-123"]}
    >
      <Routes>
        <Route
          path="/teacher/attendance/:courseId/session/:sessionId"
          element={<AttendanceSessionPage />}
        />
        <Route
          path="/teacher/attendance/:courseId"
          element={<div>course-attendance-page</div>}
        />
      </Routes>
    </MemoryRouter>
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("AttendanceSessionPage", () => {
  it("shows the session, its QR code and who is present", async () => {
    stubBackend()
    renderPage()

    expect(await screen.findByText("CENG101")).toBeInTheDocument()
    expect(
      screen.getByText("Bilgisayar Mühendisliğine Giriş")
    ).toBeInTheDocument()
    expect(screen.getByText("Hafta 5")).toBeInTheDocument()
    expect(screen.getByText("Teorik")).toBeInTheDocument()
    expect(
      screen.getByText(/Her 15 saniyede otomatik yenilenir/)
    ).toBeInTheDocument()
    expect(screen.getByText("Ali Yılmaz")).toBeInTheDocument()
    expect(screen.getByText("Katıldı")).toBeInTheDocument()
    expect(screen.getByText("Ayşe Kaya")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Ekle/ })).toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: /Ders Sayfasına Dön/ })
    ).toHaveAttribute("href", "/teacher/attendance/course-123")
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
    await user.type(search, "2024001")
    expect(screen.getByText("Ali Yılmaz")).toBeInTheDocument()
    expect(screen.queryByText("Ayşe Kaya")).not.toBeInTheDocument()
  })

  it("marks a student present by hand", async () => {
    const sent = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole("button", { name: /Ekle/ }))

    await waitFor(() => expect(screen.getAllByText("Katıldı")).toHaveLength(2))
    expect(sent.manual).toEqual([
      { student_id: "stu-2", is_present: true, note: "Manuel olarak eklendi" },
    ])
    expect(
      screen.queryByRole("button", { name: /Ekle/ })
    ).not.toBeInTheDocument()
  })

  it("says why a manual mark failed and lets the teacher retry", async () => {
    let attempts = 0
    const sent = stubBackend({
      manual: () =>
        ++attempts === 1
          ? json({ error: "Oturum kapalı", code: "SESSION_CLOSED" }, 409)
          : MANUAL_OK(),
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole("button", { name: /Ekle/ }))

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Ayşe Kaya: Oturum kapalı"
    )
    expect(screen.getByRole("button", { name: /Ekle/ })).toBeEnabled()
    expect(screen.getAllByText("Katıldı")).toHaveLength(1)

    await user.click(screen.getByRole("button", { name: /Ekle/ }))

    await waitFor(() => expect(screen.getAllByText("Katıldı")).toHaveLength(2))
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
    expect(sent.manual).toHaveLength(2)
  })

  it("fetches a fresh QR code on demand", async () => {
    const sent = stubBackend()
    const user = userEvent.setup()
    renderPage()

    const refresh = await screen.findByRole("button", { name: /Şimdi Yenile/ })
    expect(sent.qrFetches).toBe(1)
    await user.click(refresh)

    await waitFor(() => expect(sent.qrFetches).toBe(2))
  })

  it("closes the session only after confirmation", async () => {
    const sent = stubBackend()
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: /Yoklamayı Bitir/ })
    )
    const dialog = await screen.findByRole("dialog")
    expect(
      within(dialog).getByText(/katılmayan öğrenciler otomatik olarak devamsız/)
    ).toBeInTheDocument()
    expect(
      within(dialog).getByText("Devamsız sayılacak:").nextSibling
    ).toHaveTextContent("1")
    await user.click(within(dialog).getByRole("button", { name: "İptal" }))
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    )
    expect(sent.closed).toBe(0)

    await user.click(screen.getByRole("button", { name: /Yoklamayı Bitir/ }))
    await user.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: /Yoklamayı Bitir/,
      })
    )

    expect(
      await screen.findByText("course-attendance-page")
    ).toBeInTheDocument()
    expect(sent.closed).toBe(1)
  })

  it("says why closing failed and stays on the session", async () => {
    const sent = stubBackend({
      close: () => new Response(null, { status: 500 }),
    })
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: /Yoklamayı Bitir/ })
    )
    const dialog = await screen.findByRole("dialog")
    await user.click(
      within(dialog).getByRole("button", { name: /Yoklamayı Bitir/ })
    )

    expect(await within(dialog).findByRole("alert")).toHaveTextContent(
      "Yoklama bitirilemedi. Lütfen tekrar deneyin."
    )
    expect(sent.closed).toBe(1)
    expect(screen.queryByText("course-attendance-page")).not.toBeInTheDocument()

    await user.click(within(dialog).getByRole("button", { name: "İptal" }))
    await user.click(screen.getByRole("button", { name: /Yoklamayı Bitir/ }))
    expect(
      within(await screen.findByRole("dialog")).queryByRole("alert"),
      "a reopened dialog starts clean"
    ).not.toBeInTheDocument()
  })

  it("shows that the session's time is up", async () => {
    stubBackend({
      session: json({
        ...SESSION,
        expires_at: new Date(Date.now() - 60_000).toISOString(),
      }),
    })
    renderPage()

    expect(await screen.findByText("Süre doldu")).toBeInTheDocument()
  })

  it("explains a session that cannot be opened and links back", async () => {
    stubBackend({
      session: json(
        { error: "Oturum bulunamadı veya yetkisiz erişim", code: "NOT_FOUND" },
        404
      ),
    })
    renderPage()

    expect(
      await screen.findByText("Oturum bulunamadı veya yetkisiz erişim")
    ).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /Geri Dön/ })).toHaveAttribute(
      "href",
      "/teacher/attendance/course-123"
    )
  })

  it("says so when the course has no students", async () => {
    stubBackend({
      session: json({ ...SESSION, enrolled_student_count: 0 }),
      students: [],
      records: { ...RECORDS, records: [] },
    })
    renderPage()

    expect(
      await screen.findByText("Bu derse kayıtlı öğrenci bulunamadı.")
    ).toBeInTheDocument()
  })
})
