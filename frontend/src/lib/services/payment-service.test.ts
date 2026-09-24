import { afterEach, describe, expect, it, vi } from "vitest"
import type { Payment } from "@/lib/types"

const payment: Payment = {
  id: "0190a1b2-0000-7000-8000-00000000000b",
  status: "completed",
  amount: 45,
  currency: "TRY",
  card_brand: "Visa",
  card_last4: "4242",
  failure_reason: null,
  expires_at: "2026-09-24T12:15:00Z",
  completed_at: "2026-09-24T12:01:00Z",
}

// Request bodies are read at call time: ky consumes the request after fetch
// resolves, so reading it from the recorded call afterwards fails.
const sent: { url: string; method: string; body: string }[] = []

function stubFetch(body: unknown) {
  sent.length = 0
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      if (input instanceof Request) {
        sent.push({
          url: input.url,
          method: input.method,
          body: await input.clone().text(),
        })
      }
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    })
  )
}

describe("paymentService", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it("confirms through /api/payments/:id/confirm with the card body", async () => {
    stubFetch(payment)
    const { confirmPayment } = await import("./payment-service")
    const request = {
      card_number: "4242424242424242",
      exp_month: 12,
      exp_year: 2030,
      cvc: "123",
      cardholder_name: "Zeynep Şahin",
    }

    const result = await confirmPayment(payment.id, request)

    expect(result).toEqual(payment)
    expect(sent[0].method).toBe("POST")
    expect(new URL(sent[0].url).pathname).toBe(
      `/api/payments/${payment.id}/confirm`
    )
    expect(JSON.parse(sent[0].body)).toEqual(request)
  })

  it("reads a payment through /api/payments/:id", async () => {
    stubFetch(payment)
    const { getPayment } = await import("./payment-service")

    await getPayment(payment.id)

    expect(sent[0].method).toBe("GET")
    expect(new URL(sent[0].url).pathname).toBe(`/api/payments/${payment.id}`)
  })
})
