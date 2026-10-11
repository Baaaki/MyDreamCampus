import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter, Route, Routes } from "react-router"
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

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

/** Serves the session list and answers logout-all with `logoutAll`. */
function stubBackend(logoutAll: () => Response) {
  const logoutAllCalls: string[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const path = new URL(request.url).pathname
      if (path === "/api/auth/sessions") {
        return json({ sessions: [session()] })
      }
      if (request.method === "POST" && path === "/api/auth/logout-all") {
        logoutAllCalls.push(path)
        return logoutAll()
      }
      return json({ error: "NOT_FOUND" }, 404)
    })
  )
  return logoutAllCalls
}

function signIn(user: Record<string, unknown>) {
  localStorage.setItem(
    "user",
    JSON.stringify({
      id: "u-1",
      email: "ali@uni.edu.tr",
      role: "student",
      ...user,
    })
  )
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/auth/sessions"]}>
        <Routes>
          <Route path="/auth/sessions" element={<SessionsPage />} />
          <Route path="/auth/login" element={<p>Giriş sayfası</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe("SessionsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
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

  it("signs out of every device after a confirmation and goes to login", async () => {
    signIn({})
    const calls = stubBackend(() =>
      json({ message: "Tüm cihazlardan çıkış yapıldı" })
    )
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Tüm Cihazlardan Çık" })
    )
    expect(screen.getByText("Tüm cihazlardan çıkılsın mı?")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Çıkış Yap" }))

    expect(await screen.findByText("Giriş sayfası")).toBeInTheDocument()
    expect(calls).toHaveLength(1)
    expect(localStorage.getItem("user")).toBeNull()
  })

  it("keeps the user on the page when they back out", async () => {
    signIn({})
    const calls = stubBackend(() => json({}))
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Tüm Cihazlardan Çık" })
    )
    await user.click(screen.getByRole("button", { name: "Vazgeç" }))

    expect(calls).toHaveLength(0)
    expect(screen.getByText("Aktif Oturumlar")).toBeInTheDocument()
  })

  it("disables the button for a shared demo account", async () => {
    signIn({ is_demo: true })
    stubBackend(() => json({}))
    renderPage()

    expect(
      await screen.findByRole("button", { name: "Tüm Cihazlardan Çık" })
    ).toBeDisabled()
    expect(
      screen.getByText("Demo hesabı ortak kullanıldığı için bu işlem kapalı.")
    ).toBeInTheDocument()
  })

  it("shows auth's refusal and stays signed in", async () => {
    signIn({})
    stubBackend(() =>
      json(
        {
          error: "DEMO_ACCOUNT_ACTION_FORBIDDEN",
          message: "Demo hesabında bu işlem kapalı",
        },
        403
      )
    )
    const user = userEvent.setup()
    renderPage()

    await user.click(
      await screen.findByRole("button", { name: "Tüm Cihazlardan Çık" })
    )
    await user.click(screen.getByRole("button", { name: "Çıkış Yap" }))

    expect(
      await screen.findByText("Demo hesabında bu işlem kapalı")
    ).toBeInTheDocument()
    expect(localStorage.getItem("user")).not.toBeNull()
  })
})
