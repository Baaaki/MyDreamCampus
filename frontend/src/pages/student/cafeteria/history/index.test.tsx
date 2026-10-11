import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import CafeteriaHistoryPage from "./index"

function json(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
  })
}

function confirmedLunch(id: string, date: string) {
  return {
    id,
    date,
    meal_time: "lunch",
    menu_type: "normal",
    cafeteria_name: "Merkez Kafeterya",
    status: "confirmed",
    is_used: false,
    created_at: "2026-10-05T12:00:00Z",
  }
}

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

  // Monday 12 Oct: this week's meals locked on Friday the 9th, next week's
  // stay cancellable until Friday the 16th.
  it("cancels a next-week reservation and reports the refund", async () => {
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date(2026, 9, 12, 10, 0))
    const deleted: string[] = []
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const request = input as Request
        const url = new URL(request.url)
        if (request.method === "DELETE") {
          deleted.push(url.pathname)
          return json({
            success: true,
            data: {
              reservation_id: "next-week",
              refund_amount: 15,
              currency: "TRY",
              refund_status: "completed",
            },
          })
        }
        const reservations = url.searchParams.has("from_date")
          ? [
              confirmedLunch("next-week", "2026-10-19"),
              confirmedLunch("this-week", "2026-10-14"),
            ]
          : []
        return json({ success: true, data: { reservations } })
      })
    )

    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <CafeteriaHistoryPage />
      </QueryClientProvider>
    )
    const user = userEvent.setup()

    await user.click(await screen.findByRole("button", { name: "İptal Et" }))
    expect(screen.getByText("İptal süresi doldu")).toBeInTheDocument()
    const dialog = await screen.findByRole("alertdialog")
    await user.click(within(dialog).getByRole("button", { name: "İptal Et" }))

    await waitFor(() =>
      expect(deleted).toEqual([
        expect.stringMatching(/\/reservations\/next-week$/),
      ])
    )
    expect(
      await screen.findByText("Rezervasyon iptal edildi, 15.00 ₺ iade edildi.")
    ).toBeInTheDocument()
  })
})
