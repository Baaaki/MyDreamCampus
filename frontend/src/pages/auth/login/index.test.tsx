import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import LoginPage from "./index"

function renderAt(initial = "/auth/login") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/auth/login" element={<LoginPage />} />
          <Route
            path="/auth/change-password"
            element={<div>change-password-page</div>}
          />
          <Route path="/dashboard" element={<div>admin-home</div>} />
          <Route path="/teacher/attendance" element={<div>teacher-home</div>} />
          <Route path="/student/dashboard" element={<div>student-home</div>} />
          <Route
            path="/grades/transcripts"
            element={<div>safe-redirect-target</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

beforeEach(() => {
  localStorage.clear()
  document.cookie = "csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/"
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})

describe("LoginPage - happy paths", () => {
  it("redirects admin to /dashboard and stores user", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", email: "a@x.tr", role: "admin" },
        })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("admin-home")).toBeInTheDocument()
    expect(JSON.parse(localStorage.getItem("user")!)).toMatchObject({
      id: "u-1",
      role: "admin",
    })
  })

  it("redirects teacher to /teacher/attendance", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-2", role: "teacher" },
        })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "t@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("teacher-home")).toBeInTheDocument()
  })

  it("redirects student to /student/dashboard", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-3", role: "student" },
        })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "s@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("student-home")).toBeInTheDocument()
  })
})

describe("LoginPage - force_password_change", () => {
  it("redirects to /auth/change-password regardless of role", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
          force_password_change: true,
        })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("change-password-page")).toBeInTheDocument()
  })
})

describe("LoginPage - safe redirect param", () => {
  it("honors a safe relative redirect", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    )

    renderAt("/auth/login?redirect=/grades/transcripts")
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("safe-redirect-target")).toBeInTheDocument()
  })

  it("ignores protocol-relative redirect (//evil.com)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    )

    renderAt("/auth/login?redirect=//evil.com/x")
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("admin-home")).toBeInTheDocument()
  })

  it("ignores absolute URL redirect (https://evil.com)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          access_token: "at",
          user: { id: "u-1", role: "admin" },
        })
      )
    )

    renderAt("/auth/login?redirect=https://evil.com")
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "Password1")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("admin-home")).toBeInTheDocument()
  })
})

describe("LoginPage - error handling", () => {
  it("shows error message on 401 and stays on page", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(JSON.stringify({ message: "invalid credentials" }), {
            status: 401,
            headers: { "Content-Type": "application/json" },
          })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "wrong")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(
      await screen.findByText(/giriş başarısız|invalid|request failed/i)
    ).toBeInTheDocument()
    expect(screen.queryByText("admin-home")).not.toBeInTheDocument()
  })

  it("does not store user on failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(JSON.stringify({ message: "locked" }), { status: 423 })
      )
    )

    renderAt()
    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "a@x.tr"
    )
    await userEvent.type(screen.getByPlaceholderText("Şifre"), "x")
    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    await screen.findByText(/giriş başarısız|locked|request failed/i)
    expect(localStorage.getItem("user")).toBeNull()
  })
})

describe("LoginPage - demo account buttons", () => {
  const demoAccounts = [
    {
      role: "admin",
      label: "Demo Yönetici",
      email: "demo.admin@mydreamcampus.com",
      password: "demo.admin@mydreamcampus.com",
    },
    {
      role: "teacher",
      label: "Demo Öğretmen",
      email: "ahmet.yilmaz@uni.edu.tr",
      password: "ahmet.yilmaz@uni.edu.tr",
    },
    {
      role: "student",
      label: "Demo Öğrenci",
      email: "zeynep.sahin@uni.edu.tr",
      password: "zeynep.sahin@uni.edu.tr",
    },
  ]

  function requestUrl(input: RequestInfo | URL) {
    if (typeof input === "string") return input
    return input instanceof Request ? input.url : input.toString()
  }

  function stubDemoFetch(accounts: unknown = demoAccounts, status = 200) {
    const loginBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = requestUrl(input)
      if (url.includes("/api/auth/demo-accounts")) {
        return jsonResponse(accounts, status)
      }
      if (url.includes("/api/auth/login")) {
        loginBodies.push(await (input as Request).clone().json())
        return jsonResponse({
          access_token: "at-demo",
          user: { id: "u-teacher", role: "teacher" },
        })
      }
      return jsonResponse({})
    })
    vi.stubGlobal("fetch", fetchMock)
    return loginBodies
  }

  it("shows the three role buttons in order with the one-hour notice", async () => {
    stubDemoFetch()

    renderAt()

    const buttons = await screen.findAllByRole("button", {
      name: /olarak giriş/i,
    })
    expect(buttons.map((b) => b.textContent)).toEqual([
      "Öğrenci Olarak Giriş",
      "Öğretmen Olarak Giriş",
      "Admin Olarak Giriş",
    ])
    expect(screen.getByRole("note")).toHaveTextContent(
      "Yaptığınız tüm değişiklikler 1 saat sonra geri döndürülecektir."
    )
  })

  it("fills the form without signing in, then signs in with Giriş Yap", async () => {
    const loginBodies = stubDemoFetch()

    renderAt()
    await userEvent.click(
      await screen.findByRole("button", { name: "Öğretmen Olarak Giriş" })
    )

    expect(screen.getByPlaceholderText("E-posta adresi")).toHaveValue(
      "ahmet.yilmaz@uni.edu.tr"
    )
    expect(screen.getByPlaceholderText("Şifre")).toHaveValue(
      "ahmet.yilmaz@uni.edu.tr"
    )
    expect(
      screen.getByRole("button", { name: "Öğretmen Olarak Giriş" })
    ).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByRole("button", { name: /giriş yap/i })).toHaveFocus()
    expect(loginBodies).toHaveLength(0)

    await userEvent.click(screen.getByRole("button", { name: /giriş yap/i }))

    expect(await screen.findByText("teacher-home")).toBeInTheDocument()
    expect(loginBodies).toEqual([
      {
        email: "ahmet.yilmaz@uni.edu.tr",
        password: "ahmet.yilmaz@uni.edu.tr",
      },
    ])
  })

  it("leaves out a role the backend does not list", async () => {
    stubDemoFetch(demoAccounts.filter((a) => a.role !== "admin"))

    renderAt()

    expect(
      await screen.findByRole("button", { name: "Öğrenci Olarak Giriş" })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Admin Olarak Giriş" })
    ).not.toBeInTheDocument()
  })

  it("hides the buttons and the notice when demo mode is off (404)", async () => {
    stubDemoFetch({ error: "DEMO_MODE_DISABLED" }, 404)

    renderAt()

    await new Promise((r) => setTimeout(r, 50))
    expect(
      screen.queryByRole("button", { name: /olarak giriş/i })
    ).not.toBeInTheDocument()
    expect(screen.queryByRole("note")).not.toBeInTheDocument()
  })
})
