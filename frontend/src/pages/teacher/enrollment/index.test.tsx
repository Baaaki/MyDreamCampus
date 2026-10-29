import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import EnrollmentPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

// Kaan has no advisor: only an admin's pending list holds his program.
const PROGRAM = {
  id: "prog-kaan",
  student_id: "student-kaan",
  student_number: "20260101",
  student_name: "Kaan Aydın",
  department: "Bilgisayar Bilimleri",
  class_level: 1,
  semester: "2026-2027-Spring",
  status: "pending",
  created_at: "2026-10-20T09:00:00Z",
  courses: [
    { id: "c-1", course_code: "BB101", course_name: "Algoritma", credits: 4 },
  ],
}

function stubBackend() {
  const approved: string[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const { pathname } = new URL(request.url)
      if (request.method === "GET" && pathname.endsWith("/pending-programs")) {
        return json({ programs: [PROGRAM] })
      }
      if (request.method === "POST" && pathname.endsWith("/approve")) {
        approved.push(pathname)
        return json({ ...PROGRAM, status: "approved" })
      }
      return json({}, 404)
    })
  )
  return approved
}

afterEach(() => vi.unstubAllGlobals())

describe("EnrollmentPage", () => {
  it("lets an admin approve the program of a student without an advisor", async () => {
    const approved = stubBackend()
    const user = userEvent.setup()
    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <MemoryRouter>
          <EnrollmentPage description="Danışmanı olmayan öğrencilerin ders kayıt taleplerini buradan inceleyebilirsiniz." />
        </MemoryRouter>
      </QueryClientProvider>
    )

    expect(
      await screen.findByText(
        "Danışmanı olmayan öğrencilerin ders kayıt taleplerini buradan inceleyebilirsiniz."
      )
    ).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "İncele" }))
    await user.click(await screen.findByRole("button", { name: "Onayla" }))

    expect(
      await screen.findByText("Bekleyen Talep Yok"),
      "an approved program leaves the list"
    ).toBeInTheDocument()
    expect(approved).toEqual([
      "/api/enrollment/advisor/programs/prog-kaan/approve",
    ])
  })
})
