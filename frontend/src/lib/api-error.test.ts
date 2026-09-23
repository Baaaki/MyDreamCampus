import { describe, expect, it } from "vitest"
import ky, { HTTPError } from "ky"
import { apiErrorMessage } from "./api-error"

// ky fills HTTPError.data while handling the response, so the error has to
// come out of a real request rather than the constructor.
async function httpError(body: string, status = 400): Promise<HTTPError> {
  const fetch = async () =>
    new Response(body, {
      status,
      headers: { "Content-Type": "application/json" },
    })
  try {
    await ky.post("http://localhost/api/auth/login", { fetch, retry: 0 })
  } catch (err) {
    if (err instanceof HTTPError) return err
    throw err
  }
  throw new Error("expected an HTTPError")
}

describe("apiErrorMessage", () => {
  it("prefers the message field (auth shape)", async () => {
    const err = await httpError(
      JSON.stringify({
        error: "INVALID_CREDENTIALS",
        message: "Geçersiz e-posta veya şifre",
      })
    )
    expect(await apiErrorMessage(err, "x")).toBe("Geçersiz e-posta veya şifre")
  })

  it("uses the error field when it is prose (service shape)", async () => {
    const err = await httpError(
      JSON.stringify({ error: "Ders bulunamadı", code: "COURSE_NOT_FOUND" })
    )
    expect(await apiErrorMessage(err, "x")).toBe("Ders bulunamadı")
  })

  it("never shows a bare machine code", async () => {
    const err = await httpError(JSON.stringify({ error: "CSRF_ERROR" }), 403)
    expect(await apiErrorMessage(err, "İşlem başarısız")).toBe(
      "İşlem başarısız"
    )
  })

  it("falls back on a non-JSON body and on non-HTTP errors", async () => {
    expect(await apiErrorMessage(await httpError("<html>"), "Hata")).toBe(
      "Hata"
    )
    expect(await apiErrorMessage(new TypeError("offline"), "Hata")).toBe("Hata")
  })
})
