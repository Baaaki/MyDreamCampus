import type { ConfirmPaymentRequest } from "@/lib/types"

/** The card form as typed: number grouped by four, expiry as AA/YY. */
export interface CardForm {
  number: string
  expiry: string
  cvc: string
  holder: string
}

export type CardFormErrors = Partial<Record<keyof CardForm, string>>

export const EMPTY_CARD_FORM: CardForm = {
  number: "",
  expiry: "",
  cvc: "",
  holder: "",
}

/** The payment service's published test numbers. */
export const TEST_CARDS = [
  { number: "4242 4242 4242 4242", result: "Başarılı" },
  { number: "4000 0000 0000 0002", result: "Reddedilir: Kart reddedildi" },
  { number: "4000 0000 0000 9995", result: "Reddedilir: Yetersiz bakiye" },
] as const

const MAX_CARD_DIGITS = 19

function digitsOnly(value: string): string {
  return value.replace(/\D/g, "")
}

/** Keeps the digits and groups them by four as the student types. */
export function formatCardNumber(value: string): string {
  const digits = digitsOnly(value).slice(0, MAX_CARD_DIGITS)
  return digits.replace(/(\d{4})(?=\d)/g, "$1 ")
}

/** Keeps up to four digits and inserts the slash after the month. */
export function formatExpiry(value: string): string {
  const digits = digitsOnly(value).slice(0, 4)
  return digits.length > 2 ? `${digits.slice(0, 2)}/${digits.slice(2)}` : digits
}

export function luhnValid(digits: string): boolean {
  if (!/^\d+$/.test(digits)) return false
  let sum = 0
  let double = false
  for (let i = digits.length - 1; i >= 0; i--) {
    let d = Number(digits[i])
    if (double) {
      d *= 2
      if (d > 9) d -= 9
    }
    sum += d
    double = !double
  }
  return sum % 10 === 0
}

/** Reads AA/YY; null when it is not a real month. */
export function parseExpiry(
  value: string
): { month: number; year: number } | null {
  const match = /^(\d{2})\/(\d{2})$/.exec(value)
  if (!match) return null
  const month = Number(match[1])
  if (month < 1 || month > 12) return null
  return { month, year: 2000 + Number(match[2]) }
}

/**
 * Checks the form the way the payment service will, so most mistakes are
 * caught before a request. A card is valid through the last day of its
 * expiry month. The service decides on its own clock, which the time
 * machine may have moved; its answer wins.
 */
export function validateCardForm(
  form: CardForm,
  now: Date = new Date()
): CardFormErrors {
  const errors: CardFormErrors = {}

  const digits = digitsOnly(form.number)
  if (digits.length < 12 || !luhnValid(digits)) {
    errors.number = "Kart numarası geçersiz"
  }

  const expiry = parseExpiry(form.expiry)
  if (!expiry) {
    errors.expiry = "Son kullanma tarihini AA/YY biçiminde girin"
  } else {
    // Month index `month` is the month after expiry (Date months are 0-based).
    const firstInvalidDay = new Date(expiry.year, expiry.month, 1)
    if (now >= firstInvalidDay) {
      errors.expiry = "Kartın son kullanma tarihi geçmiş"
    }
  }

  if (!/^\d{3,4}$/.test(form.cvc)) {
    errors.cvc = "CVC 3 veya 4 haneli olmalı"
  }

  if (form.holder.trim() === "") {
    errors.holder = "Kart üzerindeki adı girin"
  }

  return errors
}

export function toConfirmRequest(form: CardForm): ConfirmPaymentRequest {
  const expiry = parseExpiry(form.expiry)
  return {
    card_number: digitsOnly(form.number),
    exp_month: expiry?.month ?? 0,
    exp_year: expiry?.year ?? 0,
    cvc: form.cvc,
    cardholder_name: form.holder.trim(),
  }
}
