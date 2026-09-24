import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { DemoBanner } from "./demo-banner"

function renderBanner() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <DemoBanner />
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
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})

describe("DemoBanner", () => {
  it("renders when demo accounts exist", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse([
          {
            role: "admin",
            label: "Demo Yönetici",
            email: "demo.admin@mydreamcampus.com",
            password: "demo.admin@mydreamcampus.com",
          },
        ])
      )
    )

    renderBanner()

    expect(
      await screen.findByText(
        /Bu bir demo\. Yaptığınız değişiklikler her gece 04:00'te geri alınır\./i
      )
    ).toBeInTheDocument()
  })

  it("does not render when demo mode is disabled (404)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(JSON.stringify({ error: "DEMO_MODE_DISABLED" }), {
            status: 404,
            headers: { "Content-Type": "application/json" },
          })
      )
    )

    renderBanner()

    await new Promise((r) => setTimeout(r, 50))
    expect(
      screen.queryByText(/Bu bir demo\. Yaptığınız değişiklikler/i)
    ).not.toBeInTheDocument()
  })
})
