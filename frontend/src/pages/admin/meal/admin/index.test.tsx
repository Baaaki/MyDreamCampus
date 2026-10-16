import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import MealAdminQRPage from "./index"

function cafeteria(id: string, name: string, isActive: boolean) {
  return {
    id,
    name,
    location: "Kampüs",
    has_vegan_menu: false,
    serves_dinner: true,
    is_active: isActive,
    created_at: "2026-09-01T00:00:00Z",
    updated_at: "2026-09-01T00:00:00Z",
  }
}

describe("MealAdminQRPage", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("lists the active cafeterias from the wrapped list response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              success: true,
              data: {
                cafeterias: [
                  cafeteria("c-1", "Merkez", true),
                  cafeteria("c-2", "Kapalı", false),
                ],
                meal_price: 40,
                currency: "TRY",
              },
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          )
      )
    )

    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <MealAdminQRPage />
      </QueryClientProvider>
    )

    expect(
      await screen.findByRole("option", { name: "Merkez - Kampüs" })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("option", { name: "Kapalı - Kampüs" })
    ).not.toBeInTheDocument()
    expect(screen.queryByText("Yemekhaneler yüklenemedi")).toBeNull()
  })
})
