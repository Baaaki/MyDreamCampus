import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CheckoutDialog } from "./checkout-dialog"

const PAYMENT_ID = "0190a1b2-0000-7000-8000-00000000000b"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function reservation(status: string) {
  return {
    id: "r-1",
    date: "2026-09-28",
    meal_time: "lunch",
    menu_type: "normal",
    cafeteria_name: "Merkez Yemekhane",
    status,
    is_used: false,
    created_at: "2026-09-24T12:00:00Z",
  }
}

/**
 * Stands in for meal and payment: the reservation is created pending, the
 * card settles the payment, and the next reservations read shows what the
 * payment event did to it.
 */
function stubBackend(outcome: "completed" | "failed") {
  const confirmBodies: unknown[] = []
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const request = input as Request
    const path = new URL(request.url).pathname
    if (path === "/api/meals/reservations/batch") {
      return json({
        success: true,
        data: {
          batch_id: "b-1",
          payment_id: PAYMENT_ID,
          total_amount: 50,
          currency: "TRY",
          expires_at: "2026-09-24T12:15:00Z",
          reservations: [reservation("pending")],
        },
      })
    }
    if (path === `/api/payments/${PAYMENT_ID}/confirm`) {
      confirmBodies.push(await request.clone().json())
      return json({
        id: PAYMENT_ID,
        status: outcome,
        amount: 50,
        currency: "TRY",
        card_brand: "Visa",
        card_last4: outcome === "completed" ? "4242" : "0002",
        failure_reason: outcome === "failed" ? "Kart reddedildi" : null,
        expires_at: "2026-09-24T12:15:00Z",
        completed_at: null,
      })
    }
    if (path === "/api/meals/reservations/my") {
      return json({
        success: true,
        data: {
          reservations: [
            reservation(outcome === "completed" ? "confirmed" : "cancelled"),
          ],
          summary: {
            total: 1,
            confirmed: 0,
            pending: 0,
            used: 0,
            cancelled: 0,
          },
        },
      })
    }
    return json({ error: "unexpected" }, 500)
  })
  vi.stubGlobal("fetch", fetchSpy)
  return { fetchSpy, confirmBodies }
}

function renderDialog(onPaid = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <CheckoutDialog
        open
        onOpenChange={() => {}}
        items={[{ key: "monday", label: "Pazartesi", menuType: "normal" }]}
        estimatedTotal={50}
        buildRequest={() => [
          {
            cafeteria_id: "c-1",
            date: "2026-09-28",
            meal_time: "lunch",
            menu_type: "normal",
          },
        ]}
        onPaid={onPaid}
      />
    </QueryClientProvider>
  )
  return onPaid
}

async function reachCardForm() {
  await userEvent.click(screen.getByRole("button", { name: /ödemeye geç/i }))
  expect(await screen.findByText("Kart Bilgileri")).toBeInTheDocument()
  expect(
    screen.getByText("Demo ödeme: gerçek para çekilmez.")
  ).toBeInTheDocument()
}

describe("CheckoutDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("reserves, pays with a test card and reports the confirmed reservation", async () => {
    const { confirmBodies } = stubBackend("completed")
    const onPaid = renderDialog()

    await reachCardForm()
    await userEvent.click(screen.getAllByRole("button", { name: "Kullan" })[0])
    await userEvent.type(
      screen.getByLabelText("Kart Üzerindeki Ad"),
      "Zeynep Şahin"
    )
    await userEvent.click(screen.getByRole("button", { name: /50\.00 ₺ öde/i }))

    expect(await screen.findByText("Rezervasyon Onaylandı")).toBeInTheDocument()
    expect(confirmBodies[0]).toEqual({
      card_number: "4242424242424242",
      exp_month: 12,
      exp_year: 2030,
      cvc: "123",
      cardholder_name: "Zeynep Şahin",
    })

    await userEvent.click(screen.getByRole("button", { name: "Tamam" }))
    expect(onPaid).toHaveBeenCalledOnce()
  })

  it("shows the decline reason once the reservation is cancelled", async () => {
    stubBackend("failed")
    const onPaid = renderDialog()

    await reachCardForm()
    await userEvent.click(screen.getAllByRole("button", { name: "Kullan" })[1])
    await userEvent.type(screen.getByLabelText("Kart Üzerindeki Ad"), "Ali")
    await userEvent.click(screen.getByRole("button", { name: /öde$/i }))

    expect(await screen.findByText("Ödeme Başarısız")).toBeInTheDocument()
    expect(screen.getByText(/Kart reddedildi/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole("button", { name: "Tekrar Dene" }))
    expect(screen.getByText("Ödeme Özeti")).toBeInTheDocument()
    expect(onPaid).not.toHaveBeenCalled()
  })

  it("keeps an invalid card on the client", async () => {
    const { confirmBodies } = stubBackend("completed")
    renderDialog()

    await reachCardForm()
    await userEvent.type(
      screen.getByLabelText("Kart Numarası"),
      "4242424242424241"
    )
    await userEvent.click(screen.getByRole("button", { name: /öde$/i }))

    expect(screen.getByText("Kart numarası geçersiz")).toBeInTheDocument()
    expect(screen.getByText("Kart üzerindeki adı girin")).toBeInTheDocument()
    expect(confirmBodies).toHaveLength(0)
  })
})
