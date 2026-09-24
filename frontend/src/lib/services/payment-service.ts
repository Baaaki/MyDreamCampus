import { paymentApi } from "@/lib/api-client"
import type { ConfirmPaymentRequest, Payment } from "@/lib/types"

/**
 * Charges a pending payment with a test card. A declined card is not an
 * error: the payment comes back with status "failed" and a reason.
 */
export async function confirmPayment(
  paymentId: string,
  request: ConfirmPaymentRequest
): Promise<Payment> {
  return paymentApi
    .post(`${paymentId}/confirm`, { json: request })
    .json<Payment>()
}

export async function getPayment(paymentId: string): Promise<Payment> {
  return paymentApi.get(paymentId).json<Payment>()
}
