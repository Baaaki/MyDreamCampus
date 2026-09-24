import { describe, expect, it } from "vitest"
import {
  EMPTY_CARD_FORM,
  formatCardNumber,
  formatExpiry,
  luhnValid,
  parseExpiry,
  toConfirmRequest,
  validateCardForm,
  type CardForm,
} from "./payment-card"

const now = new Date(2026, 8, 24, 12, 0) // 24 Sep 2026, local time

const validForm: CardForm = {
  number: "4242 4242 4242 4242",
  expiry: "12/30",
  cvc: "123",
  holder: "Zeynep Şahin",
}

describe("luhnValid", () => {
  it.each([
    "4242424242424242",
    "4000000000000002",
    "4000000000009995",
    "378282246310005",
  ])("accepts %s", (number) => {
    expect(luhnValid(number)).toBe(true)
  })

  it.each(["4242424242424241", "1234567812345678", "4242 4242", ""])(
    "rejects %j",
    (number) => {
      expect(luhnValid(number)).toBe(false)
    }
  )
})

describe("formatting", () => {
  it("groups the card number by four and drops everything but digits", () => {
    expect(formatCardNumber("4242-4242 4242x4242")).toBe("4242 4242 4242 4242")
  })

  it("caps the card number at 19 digits", () => {
    expect(formatCardNumber("1".repeat(25)).replace(/ /g, "")).toHaveLength(19)
  })

  it("inserts the expiry slash after the month", () => {
    expect(formatExpiry("1")).toBe("1")
    expect(formatExpiry("12")).toBe("12")
    expect(formatExpiry("1230")).toBe("12/30")
    expect(formatExpiry("12/305")).toBe("12/30")
  })
})

describe("parseExpiry", () => {
  it("reads AA/YY into a four-digit year", () => {
    expect(parseExpiry("09/27")).toEqual({ month: 9, year: 2027 })
  })

  it.each(["13/27", "00/27", "9/27", "0927"])("rejects %s", (value) => {
    expect(parseExpiry(value)).toBeNull()
  })
})

describe("validateCardForm", () => {
  it("passes a complete, valid form", () => {
    expect(validateCardForm(validForm, now)).toEqual({})
  })

  it("flags every empty field", () => {
    expect(Object.keys(validateCardForm(EMPTY_CARD_FORM, now)).sort()).toEqual([
      "cvc",
      "expiry",
      "holder",
      "number",
    ])
  })

  it("rejects a number that fails Luhn", () => {
    const errors = validateCardForm(
      { ...validForm, number: "4242 4242 4242 4241" },
      now
    )
    expect(errors.number).toBe("Kart numarası geçersiz")
  })

  it("keeps a card valid through the last day of its expiry month", () => {
    const card = { ...validForm, expiry: "09/26" }
    expect(validateCardForm(card, new Date(2026, 8, 30, 23, 59))).toEqual({})
    expect(validateCardForm(card, new Date(2026, 9, 1)).expiry).toBe(
      "Kartın son kullanma tarihi geçmiş"
    )
  })

  it("rejects a CVC that is not three or four digits", () => {
    expect(validateCardForm({ ...validForm, cvc: "12" }, now).cvc).toBeDefined()
    expect(
      validateCardForm({ ...validForm, cvc: "12a" }, now).cvc
    ).toBeDefined()
    expect(
      validateCardForm({ ...validForm, cvc: "1234" }, now).cvc
    ).toBeUndefined()
  })
})

describe("toConfirmRequest", () => {
  it("sends bare digits and a four-digit year", () => {
    expect(
      toConfirmRequest({ ...validForm, holder: "  Zeynep Şahin " })
    ).toEqual({
      card_number: "4242424242424242",
      exp_month: 12,
      exp_year: 2030,
      cvc: "123",
      cardholder_name: "Zeynep Şahin",
    })
  })
})
