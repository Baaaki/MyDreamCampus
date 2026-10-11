import { afterEach, describe, expect, it, vi } from "vitest"
import {
  canCancelReservation,
  checkoutOutcome,
  getMonthlyMenu,
  menuForDay,
  nextWeekWeekdays,
  reservationCancelDeadline,
  toLocalISODate,
  type MonthlyMenuData,
  type Reservation,
} from "./meal-service"

function reservation(id: string, status: Reservation["status"]): Reservation {
  return {
    id,
    date: "2026-09-28",
    meal_time: "lunch",
    menu_type: "normal",
    cafeteria_name: "Merkez Yemekhane",
    status,
    is_used: false,
    created_at: "2026-09-24T12:00:00Z",
  }
}

describe("checkoutOutcome", () => {
  const ids = ["a", "b"]

  it("is confirmed once every reservation of the checkout is", () => {
    expect(
      checkoutOutcome(ids, [
        reservation("a", "confirmed"),
        reservation("b", "confirmed"),
        reservation("other", "pending"),
      ])
    ).toBe("confirmed")
  })

  it("stays pending while meal has not heard from payment", () => {
    expect(
      checkoutOutcome(ids, [
        reservation("a", "confirmed"),
        reservation("b", "pending"),
      ])
    ).toBe("pending")
  })

  it("is cancelled when a declined card dropped the reservations", () => {
    expect(
      checkoutOutcome(ids, [
        reservation("a", "cancelled"),
        reservation("b", "cancelled"),
      ])
    ).toBe("cancelled")
  })

  it("stays pending while a reservation is missing from the list", () => {
    expect(checkoutOutcome(ids, [reservation("a", "confirmed")])).toBe(
      "pending"
    )
  })
})

describe("nextWeekWeekdays", () => {
  const nextWeek = [
    "2026-10-12",
    "2026-10-13",
    "2026-10-14",
    "2026-10-15",
    "2026-10-16",
  ]

  // 5 Oct 2026 is a Monday, 11 Oct the Sunday of the same week.
  it.each([5, 6, 7, 8, 9, 10, 11])(
    "offers Monday to Friday of the following week on 2026-10-%i",
    (day) => {
      const now = new Date(2026, 9, day, 23, 30)
      const days = nextWeekWeekdays(now)

      expect(days.map(toLocalISODate)).toEqual(nextWeek)
      for (const d of days) {
        expect(d.getTime(), "never a past day").toBeGreaterThan(now.getTime())
      }
    }
  )

  it("moves on to the week after once Monday begins", () => {
    expect(
      nextWeekWeekdays(new Date(2026, 9, 12, 0, 0)).map(toLocalISODate)
    ).toEqual([
      "2026-10-19",
      "2026-10-20",
      "2026-10-21",
      "2026-10-22",
      "2026-10-23",
    ])
  })

  it("runs across the end of a month", () => {
    expect(
      nextWeekWeekdays(new Date(2026, 9, 28, 12, 0)).map(toLocalISODate)
    ).toEqual([
      "2026-11-02",
      "2026-11-03",
      "2026-11-04",
      "2026-11-05",
      "2026-11-06",
    ])
  })
})

describe("canCancelReservation", () => {
  // Thursday 15 Oct 2026's week starts on Monday the 12th, so its cancel
  // window closes on Friday the 9th at 23:59:59 Istanbul (20:59:59 UTC).
  const thursday = { ...reservation("r", "confirmed"), date: "2026-10-15" }

  it("allows a cancel until the Friday before the reservation's week", () => {
    expect(
      canCancelReservation(thursday, new Date("2026-10-09T20:59:59Z"))
    ).toBe(true)
  })

  it("refuses a cancel once that Friday has ended", () => {
    expect(
      canCancelReservation(thursday, new Date("2026-10-09T21:00:00Z"))
    ).toBe(false)
  })

  it("puts a Monday reservation's deadline three days before it", () => {
    expect(reservationCancelDeadline("2026-10-12")).toEqual(
      new Date("2026-10-09T20:59:59Z")
    )
  })

  it("refuses a used or cancelled reservation", () => {
    const early = new Date("2026-10-01T09:00:00Z")
    expect(canCancelReservation({ ...thursday, is_used: true }, early)).toBe(
      false
    )
    expect(
      canCancelReservation(
        { ...reservation("r", "cancelled"), date: "2026-10-15" },
        early
      )
    ).toBe(false)
  })
})

describe("toLocalISODate", () => {
  it("keeps the local day at midnight", () => {
    expect(toLocalISODate(new Date(2026, 9, 12, 0, 0))).toBe("2026-10-12")
  })
})

describe("menuForDay", () => {
  const day = (name: string) => ({ items: [name], calories: 500 })
  // 1 Oct 2026 is a Thursday, so week 0 runs 28 Sep - 4 Oct and 12 Oct
  // opens week 2.
  const menu: MonthlyMenuData = {
    normalMenus: [{}, {}, { monday: day("Etli Nohut") }],
    veganMenus: [{}, {}, { monday: day("Nohut Yemeği") }],
  }

  it("finds a day by its Monday-to-Sunday week of the month", () => {
    expect(menuForDay(menu, new Date(2026, 9, 12), "normal")?.items).toEqual([
      "Etli Nohut",
    ])
    expect(menuForDay(menu, new Date(2026, 9, 12), "vegan")?.items).toEqual([
      "Nohut Yemeği",
    ])
  })

  it("has nothing for a weekend or a day the menu leaves out", () => {
    expect(menuForDay(menu, new Date(2026, 9, 11), "normal")).toBeNull()
    expect(menuForDay(menu, new Date(2026, 9, 13), "normal")).toBeNull()
    expect(menuForDay(menu, new Date(2026, 9, 26), "normal")).toBeNull()
    expect(menuForDay(null, new Date(2026, 9, 12), "normal")).toBeNull()
  })
})

describe("getMonthlyMenu", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("reads an unpublished month as no menu rather than an error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ error: "Kayıt bulunamadı", code: "NOT_FOUND" }),
            { status: 404, headers: { "Content-Type": "application/json" } }
          )
      )
    )

    await expect(getMonthlyMenu(2026, 11)).resolves.toBeNull()
  })
})
