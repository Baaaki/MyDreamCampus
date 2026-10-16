import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { Semester, SemesterStatus } from "@/lib/types"
import OpenedCoursesPage from "./index"

vi.mock("@/components/course-hierarchy-view", () => ({
  default: ({ semesterName }: { semesterName: string }) => (
    <p>program: {semesterName}</p>
  ),
}))

function semester(name: string, status: SemesterStatus): Semester {
  return {
    id: name,
    name,
    status,
    hard_deadline: "2027-06-30T00:00:00Z",
    activated_at: null,
    completed_at: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  }
}

function renderWith(reply: () => Response) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => reply())
  )
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <OpenedCoursesPage />
    </QueryClientProvider>
  )
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

describe("OpenedCoursesPage", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("opens on the active semester", async () => {
    renderWith(() =>
      json([
        semester("2026-2027-Spring", "planned"),
        semester("2026-2027-Fall", "active"),
        semester("2025-2026-Spring", "completed"),
      ])
    )

    expect(
      await screen.findByText("program: 2026-2027-Fall")
    ).toBeInTheDocument()
  })

  it("falls back to the newest semester when none is active", async () => {
    renderWith(() =>
      json([
        semester("2025-2026-Fall", "completed"),
        semester("2025-2026-Spring", "completed"),
      ])
    )

    expect(
      await screen.findByText("program: 2025-2026-Spring")
    ).toBeInTheDocument()
  })

  it("says so when the semesters cannot be loaded", async () => {
    renderWith(() => json({ error: "INTERNAL" }, 500))

    expect(await screen.findByText("Dönemler yüklenemedi")).toBeInTheDocument()
    expect(screen.queryByText(/program:/)).toBeNull()
  })
})
