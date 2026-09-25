import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import ForgotPasswordPage from "./index"

function renderAt(initial = "/auth/forgot-password") {
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
          <Route
            path="/auth/forgot-password"
            element={<ForgotPasswordPage />}
          />
          <Route path="/auth/login" element={<div>login-page</div>} />
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

describe("ForgotPasswordPage", () => {
  it("renders demo mode note when demo accounts are available", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url =
          typeof input === "string"
            ? input
            : input instanceof Request
              ? input.url
              : input.toString()
        if (url.includes("/api/auth/demo-accounts")) {
          return jsonResponse([
            {
              role: "admin",
              label: "Demo Yönetici",
              email: "demo.admin@mydreamcampus.com",
              password: "demo.admin@mydreamcampus.com",
            },
          ])
        }
        return jsonResponse({})
      })
    )

    renderAt()

    expect(
      await screen.findByText("Demo ortamında e-posta gönderilmez")
    ).toBeInTheDocument()
  })

  it("does not render demo mode note when demo mode is disabled", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url =
          typeof input === "string"
            ? input
            : input instanceof Request
              ? input.url
              : input.toString()
        if (url.includes("/api/auth/demo-accounts")) {
          return new Response(JSON.stringify({ error: "DEMO_MODE_DISABLED" }), {
            status: 404,
            headers: { "Content-Type": "application/json" },
          })
        }
        return jsonResponse({})
      })
    )

    renderAt()

    await new Promise((r) => setTimeout(r, 50))
    expect(
      screen.queryByText("Demo ortamında e-posta gönderilmez")
    ).not.toBeInTheDocument()
  })

  it("submits password reset request successfully", async () => {
    let resetRequested = false
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url =
          typeof input === "string"
            ? input
            : input instanceof Request
              ? input.url
              : input.toString()
        if (url.includes("/api/auth/request-password-reset")) {
          resetRequested = true
          return jsonResponse({ message: "sent" })
        }
        return jsonResponse({})
      })
    )

    renderAt()

    await userEvent.type(
      screen.getByPlaceholderText("E-posta adresi"),
      "user@university.edu.tr"
    )
    await userEvent.click(
      screen.getByRole("button", { name: /bağlantı gönder/i })
    )

    expect(
      await screen.findByText(/şifre sıfırlama bağlantısı gönderildi/i)
    ).toBeInTheDocument()
    expect(resetRequested).toBe(true)
  })
})
