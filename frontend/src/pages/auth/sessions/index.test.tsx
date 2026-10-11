import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter } from "react-router"
import type { Session } from "@/lib/types"
import SessionsPage from "./index"

function session(overrides: Partial<Session> = {}): Session {
  return {
    id: "s-1",
    device_info: "Firefox / Linux",
    ip_address: "192.168.1.20",
    created_at: "2026-10-11T09:00:00Z",
    last_used_at: "2026-10-11T09:30:00Z",
    expires_at: "2026-10-18T09:00:00Z",
    is_current: true,
    ...overrides,
  }
}

function stubSessions(body: unknown) {
  const fetchSpy = vi.fn<typeof fetch>(
    async () =>
      new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
  )
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <SessionsPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe("SessionsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("lists the sessions wrapped in the response envelope", async () => {
    const fetchSpy = stubSessions({
      sessions: [
        session(),
        session({ id: "s-2", device_info: "Safari / iOS", is_current: false }),
      ],
    })
    renderPage()

    expect(await screen.findByText("Firefox / Linux")).toBeInTheDocument()
    expect(screen.getByText("Safari / iOS")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "Sonlandır" })).toHaveLength(2)
    const request = fetchSpy.mock.calls[0][0] as Request
    expect(new URL(request.url).pathname).toBe("/api/auth/sessions")
  })

  it("shows the empty notice when the envelope carries no sessions", async () => {
    stubSessions({ sessions: null })
    renderPage()

    expect(
      await screen.findByText("Aktif oturum bulunamadı.")
    ).toBeInTheDocument()
  })
})
