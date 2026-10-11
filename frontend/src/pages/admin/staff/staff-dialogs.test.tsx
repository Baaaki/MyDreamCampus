import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { Faculty, Staff } from "@/lib/types"
import { StaffDeleteDialog, StaffEditDialog } from "./staff-dialogs"

const STAFF_ID = "0190a1b2-0000-7000-8000-000000000042"

const teacher: Staff = {
  id: STAFF_ID,
  email: "ayse.kaya@uni.edu.tr",
  first_name: "Ayşe",
  last_name: "Kaya",
  role: "teacher",
  department: "Bilgisayar Mühendisliği",
  phone: "+90 232 000 0000",
  office_location: "B-101",
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}

const faculties: Faculty[] = [
  {
    id: "f-1",
    name: "Mühendislik Fakültesi",
    code: "MF",
    departments: [
      {
        id: "d-1",
        name: "Bilgisayar Mühendisliği",
        facultyId: "f-1",
        code: "BM",
      },
      {
        id: "d-2",
        name: "Elektrik Mühendisliği",
        facultyId: "f-1",
        code: "EM",
      },
    ],
  },
  {
    id: "f-2",
    name: "Fen Fakültesi",
    code: "FF",
    departments: [{ id: "d-3", name: "Fizik", facultyId: "f-2", code: "FZ" }],
  },
]

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

/** Serves the faculty list and answers the staff write with `answer`. */
function stubBackend(answer: () => Response) {
  const writes: { method: string; path: string; body: unknown }[] = []
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request
      const path = new URL(request.url).pathname
      if (path === "/api/catalog/faculties") return json({ data: faculties })
      if (path === `/api/staff/${STAFF_ID}`) {
        const text = await request.clone().text()
        writes.push({
          method: request.method,
          path,
          body: text ? JSON.parse(text) : null,
        })
        return answer()
      }
      return json({ error: "yok" }, 404)
    })
  )
  return writes
}

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const invalidate = vi.spyOn(client, "invalidateQueries")
  render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
  return { invalidate }
}

describe("StaffEditDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("sends only the fields the backend changes and closes", async () => {
    const writes = stubBackend(() => json({ ...teacher, phone: "+90 555" }))
    const onOpenChange = vi.fn()
    const user = userEvent.setup()
    const { invalidate } = renderWithClient(
      <StaffEditDialog staff={teacher} onOpenChange={onOpenChange} />
    )

    // The faculty is the one holding the account's department.
    expect(
      await screen.findByDisplayValue("Mühendislik Fakültesi")
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Bölüm")).toHaveValue(
      "Bilgisayar Mühendisliği"
    )
    expect(
      screen.getByText(/Ad, soyad ve e-posta değiştirilemez/)
    ).toBeInTheDocument()

    await user.selectOptions(
      screen.getByLabelText("Bölüm"),
      "Elektrik Mühendisliği"
    )
    await user.clear(screen.getByLabelText("Telefon"))
    await user.type(screen.getByLabelText("Telefon"), " +90 555 ")
    await user.click(screen.getByRole("button", { name: "Kaydet" }))

    await vi.waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
    expect(writes).toEqual([
      {
        method: "PUT",
        path: `/api/staff/${STAFF_ID}`,
        body: {
          department: "Elektrik Mühendisliği",
          phone: "+90 555",
          office_location: "B-101",
        },
      },
    ])
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["staff", "list"] })
  })

  it("clears the department when another faculty is picked", async () => {
    stubBackend(() => json(teacher))
    const user = userEvent.setup()
    renderWithClient(<StaffEditDialog staff={teacher} onOpenChange={vi.fn()} />)

    await screen.findByDisplayValue("Mühendislik Fakültesi")
    await user.selectOptions(screen.getByLabelText("Fakülte"), "Fen Fakültesi")

    expect(screen.getByLabelText("Bölüm")).toHaveValue("")
    expect(screen.getByRole("option", { name: "Fizik" })).toBeInTheDocument()
  })

  it("shows the backend's refusal inside the dialog", async () => {
    stubBackend(() =>
      json({ error: "Personel bulunamadı", code: "STAFF_NOT_FOUND" }, 404)
    )
    const onOpenChange = vi.fn()
    const user = userEvent.setup()
    renderWithClient(
      <StaffEditDialog staff={teacher} onOpenChange={onOpenChange} />
    )

    await screen.findByDisplayValue("Mühendislik Fakültesi")
    await user.click(screen.getByRole("button", { name: "Kaydet" }))

    expect(await screen.findByText("Personel bulunamadı")).toBeInTheDocument()
    expect(onOpenChange).not.toHaveBeenCalled()
  })
})

describe("StaffDeleteDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("deletes after the confirmation and closes", async () => {
    const writes = stubBackend(() => json({ message: "Personel silindi" }))
    const onOpenChange = vi.fn()
    const user = userEvent.setup()
    const { invalidate } = renderWithClient(
      <StaffDeleteDialog staff={teacher} onOpenChange={onOpenChange} />
    )

    expect(screen.getByText(/ayse.kaya@uni.edu.tr/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Sil" }))

    await vi.waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))
    expect(writes).toEqual([
      { method: "DELETE", path: `/api/staff/${STAFF_ID}`, body: null },
    ])
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["staff", "list"] })
  })

  it("keeps a protected account's 403 in the dialog", async () => {
    stubBackend(() =>
      json(
        {
          error: "Korunan sistem hesabı silinemez",
          code: "PROTECTED_ACCOUNT_DELETION_FORBIDDEN",
        },
        403
      )
    )
    const onOpenChange = vi.fn()
    const user = userEvent.setup()
    renderWithClient(
      <StaffDeleteDialog staff={teacher} onOpenChange={onOpenChange} />
    )

    await user.click(screen.getByRole("button", { name: "Sil" }))

    expect(
      await screen.findByText("Korunan sistem hesabı silinemez")
    ).toBeInTheDocument()
    expect(onOpenChange).not.toHaveBeenCalled()
  })

  it("does nothing when the admin backs out", async () => {
    const writes = stubBackend(() => json({}))
    const onOpenChange = vi.fn()
    const user = userEvent.setup()
    renderWithClient(
      <StaffDeleteDialog staff={teacher} onOpenChange={onOpenChange} />
    )

    await user.click(screen.getByRole("button", { name: "Vazgeç" }))

    expect(onOpenChange).toHaveBeenCalledWith(false)
    expect(writes).toHaveLength(0)
  })
})
