import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import StaffPage from "./index"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

/** Serves an empty staff list and faculty list, and records the creates. */
function stubBackend() {
  const creates: unknown[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const path = new URL(request.url).pathname.replace(/\/$/, "")
      if (path === "/api/catalog/faculties") return json({ data: [] })
      if (path === "/api/staff" && request.method === "POST") {
        creates.push(await request.json())
        return json({ id: "0190a1b2-0000-7000-8000-000000000042" }, 201)
      }
      if (path === "/api/staff") {
        return json({
          data: [],
          pagination: { page: 1, limit: 10, total: 0, total_pages: 1 },
        })
      }
      return json({ error: "yok" }, 404)
    })
  )
  return creates
}

describe("StaffPage create form", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("creates a teacher, the only role the staff API accepts", async () => {
    const creates = stubBackend()
    const user = userEvent.setup()
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={client}>
        <StaffPage />
      </QueryClientProvider>
    )

    await user.click(screen.getByRole("button", { name: "Yeni Personel Ekle" }))
    const role = screen.getByLabelText("Rol")
    expect(role).toHaveValue("Öğretmen")
    expect(role).toBeDisabled()

    await user.type(screen.getByLabelText("E-posta *"), "ayse.kaya@uni.edu.tr")
    await user.type(screen.getByLabelText("Ad *"), "Ayşe")
    await user.type(screen.getByLabelText("Soyad *"), "Kaya")
    await user.click(screen.getByRole("button", { name: "Personel Oluştur" }))

    await vi.waitFor(() => expect(creates).toHaveLength(1))
    expect(creates[0]).toMatchObject({
      email: "ayse.kaya@uni.edu.tr",
      first_name: "Ayşe",
      last_name: "Kaya",
      role: "teacher",
    })
  })
})
