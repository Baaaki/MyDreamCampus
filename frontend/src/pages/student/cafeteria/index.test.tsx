import { afterEach, beforeAll, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { MonthlyMenuData } from "@/lib/services/meal-service"
import StudentCafeteriaPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

const CAFETERIAS = {
  cafeterias: [
    {
      id: "caf-main",
      name: "Merkez Yemekhane",
      location: "Kampüs",
      has_vegan_menu: true,
      serves_dinner: true,
      is_active: true,
    },
    {
      id: "caf-eng",
      name: "Mühendislik Yemekhanesi",
      location: "Mühendislik",
      has_vegan_menu: false,
      serves_dinner: false,
      is_active: true,
    },
  ],
  meal_price: 15,
  currency: "TRY",
}

const day = (dishes: string[]) => ({ items: dishes, calories: 800 })

/**
 * Stands in for meal: the cafeteria list and the published months' menus,
 * keyed "year-month". A month left out answers 404, as an unpublished one
 * does.
 */
function stubMeal(menus: Record<string, MonthlyMenuData>) {
  const monthsAsked: string[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL((input as Request).url)
      if (url.pathname === "/api/meals/cafeterias") {
        return json({ success: true, data: CAFETERIAS })
      }
      if (url.pathname === "/api/meals/menu/monthly") {
        const key = `${url.searchParams.get("year")}-${url.searchParams.get("month")}`
        monthsAsked.push(key)
        const menu = menus[key]
        return menu
          ? json({ success: true, data: { menu_data: menu } })
          : json({ error: "Kayıt bulunamadı", code: "NOT_FOUND" }, 404)
      }
      return json({}, 404)
    })
  )
  return monthsAsked
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <StudentCafeteriaPage />
    </QueryClientProvider>
  )
}

async function pickCafeteria(name: string) {
  const user = userEvent.setup()
  await user.click(await screen.findByRole("combobox"))
  await user.click(
    await screen.findByRole("option", { name: new RegExp(name) })
  )
  return user
}

const dayColumn = (index: number) => within(screen.getAllByRole("cell")[index])

beforeAll(() => {
  // Radix Select leans on pointer capture and scrolling jsdom lacks.
  Element.prototype.hasPointerCapture = () => false
  Element.prototype.releasePointerCapture = () => {}
  Element.prototype.scrollIntoView = () => {}
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe("StudentCafeteriaPage", () => {
  it("shows the published menu and prices meals as meal does", async () => {
    // Sunday 11 Oct 2026: next week is 12-16 Oct, week 2 of October.
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date(2026, 9, 11, 18, 0))
    stubMeal({
      "2026-10": {
        normalMenus: [
          {},
          {},
          { monday: day(["Mercimek Çorbası", "Etli Nohut", ""]) },
        ],
        veganMenus: [
          {},
          {},
          { monday: day(["Mercimek Çorbası", "Nohut Yemeği"]) },
        ],
      },
    })
    renderPage()

    const user = await pickCafeteria("Merkez Yemekhane")

    const monday = dayColumn(0)
    expect(await monday.findByText("Etli Nohut")).toBeInTheDocument()
    expect(monday.getAllByRole("listitem")).toHaveLength(2)
    expect(
      dayColumn(1).getByText("Menü henüz yayınlanmadı")
    ).toBeInTheDocument()

    await user.click(monday.getByRole("button", { name: "Vegan" }))
    expect(monday.getByText("Nohut Yemeği")).toBeInTheDocument()
    await user.click(dayColumn(1).getByRole("button", { name: "Normal" }))

    expect(screen.getByText("30.00 ₺")).toBeInTheDocument()
    expect(screen.getByText("Öğün başı 15.00 ₺")).toBeInTheDocument()
  })

  it("offers no vegan meal where the cafeteria serves none", async () => {
    stubMeal({})
    renderPage()

    await pickCafeteria("Mühendislik Yemekhanesi")

    expect(await screen.findAllByText("Menü henüz yayınlanmadı")).toHaveLength(
      5
    )
    expect(
      screen.queryByRole("button", { name: "Vegan" })
    ).not.toBeInTheDocument()
    expect(screen.queryByText("Vegan Menü")).not.toBeInTheDocument()
  })

  it("reads a week that runs into the next month from both menus", async () => {
    // Sunday 29 Nov 2026: next week is Mon 30 Nov - Fri 4 Dec.
    vi.useFakeTimers({ toFake: ["Date"] })
    vi.setSystemTime(new Date(2026, 10, 29, 12, 0))
    const monthsAsked = stubMeal({
      // 1 Nov 2026 is a Sunday, so 30 Nov falls in week 5.
      "2026-11": {
        normalMenus: [{}, {}, {}, {}, {}, { monday: day(["Kasım Pilavı"]) }],
        veganMenus: [],
      },
      // 1 Dec 2026 is a Tuesday, in week 0.
      "2026-12": {
        normalMenus: [{ tuesday: day(["Aralık Çorbası"]) }],
        veganMenus: [],
      },
    })
    renderPage()

    await pickCafeteria("Merkez Yemekhane")

    expect(await dayColumn(0).findByText("Kasım Pilavı")).toBeInTheDocument()
    expect(dayColumn(1).getByText("Aralık Çorbası")).toBeInTheDocument()
    expect([...new Set(monthsAsked)].sort()).toEqual(["2026-11", "2026-12"])
  })
})
