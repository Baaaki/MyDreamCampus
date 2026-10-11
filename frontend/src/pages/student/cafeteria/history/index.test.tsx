import { afterEach, describe, expect, it, vi } from "vitest"
import { render, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import CafeteriaHistoryPage from "./index"

function stubReservations() {
  const asked: Record<string, string>[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL((input as Request).url)
      asked.push(Object.fromEntries(url.searchParams))
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            reservations: [],
            summary: {
              total: 0,
              confirmed: 0,
              pending: 0,
              used: 0,
              cancelled: 0,
            },
          },
        }),
        { headers: { "Content-Type": "application/json" } }
      )
    })
  )
  return asked
}

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe("CafeteriaHistoryPage", () => {
  // Local midnight lies on the previous UTC day east of Greenwich; the split
  // between this week and the past must still fall on the student's Monday.
  it("splits current and past reservations on the local Monday", async () => {
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date(2026, 9, 12, 0, 5))
    const asked = stubReservations()

    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <CafeteriaHistoryPage />
      </QueryClientProvider>
    )

    await waitFor(() => expect(asked).toHaveLength(2))
    expect(asked).toEqual(
      expect.arrayContaining([
        { from_date: "2026-10-12" },
        { to_date: "2026-10-11", page: "1", limit: "10" },
      ])
    )
  })
})
