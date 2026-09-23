import { HTTPError } from "ky"

// Backend error bodies come in two shapes: auth answers
// {error: CODE, message: "Türkçe metin"}, the other services
// {error: "Türkçe metin", code: CODE}. Whichever field holds prose is the
// one to show; an UPPER_SNAKE value is a machine code, not a message.
type ErrorBody = { error?: unknown; message?: unknown }

const MACHINE_CODE = /^[A-Z][A-Z0-9_]+$/

/** The backend's user-facing message for a failed request, or `fallback`. */
export function apiErrorMessage(err: unknown, fallback: string): string {
  if (!(err instanceof HTTPError)) return fallback
  // ky has already consumed the body into `data`; a non-JSON body arrives as
  // a string and has no message fields to offer.
  const body = err.data
  if (typeof body !== "object" || body === null) return fallback
  const { message, error } = body as ErrorBody
  for (const candidate of [message, error]) {
    if (
      typeof candidate === "string" &&
      candidate !== "" &&
      !MACHINE_CODE.test(candidate)
    ) {
      return candidate
    }
  }
  return fallback
}
