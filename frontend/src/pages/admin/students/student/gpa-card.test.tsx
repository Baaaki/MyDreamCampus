import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { GpaCard } from "./gpa-card"

function renderWith(reply: () => Response) {
  const fetchSpy = vi.fn(async (_input: RequestInfo | URL) => reply())
  vi.stubGlobal("fetch", fetchSpy)
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <GpaCard studentId="st-1" />
    </QueryClientProvider>
  )
  return fetchSpy
}

function transcript(totalCredits: number, gpa: number) {
  return new Response(
    JSON.stringify({
      student: {
        student_number: "2024001",
        first_name: "Ali",
        last_name: "Veli",
        department: "Bilgisayar",
        enrollment_year: 2024,
      },
      semesters: [],
      summary: { total_credits: totalCredits, cumulative_gpa: gpa },
      generated_at: "2026-10-17T10:00:00Z",
    }),
    { status: 200, headers: { "Content-Type": "application/json" } }
  )
}

describe("GpaCard", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("shows the transcript's GPA with two decimals", async () => {
    const fetchSpy = renderWith(() => transcript(32, 3.4167))

    expect(await screen.findByText("3.42")).toBeInTheDocument()
    const request = fetchSpy.mock.calls[0][0] as Request
    expect(new URL(request.url).pathname).toBe(
      "/api/grades/students/st-1/transcript"
    )
  })

  it("says there is no data before any credit is completed", async () => {
    renderWith(() => transcript(0, 0))

    expect(await screen.findByText("Veri yok")).toBeInTheDocument()
  })

  it("says there is no data when the transcript cannot be read", async () => {
    renderWith(
      () =>
        new Response('{"code":"STUDENT_NOT_FOUND"}', {
          status: 404,
          headers: { "Content-Type": "application/json" },
        })
    )

    expect(screen.getByText("—")).toBeInTheDocument()
    expect(await screen.findByText("Veri yok")).toBeInTheDocument()
  })
})
