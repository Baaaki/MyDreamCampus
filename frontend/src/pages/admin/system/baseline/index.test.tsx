import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import ky, { HTTPError } from "ky"
import BaselinePage from "./index"
import * as baselineService from "@/lib/services/baseline-service"

vi.mock("@/lib/services/baseline-service", () => ({
  getBaselineStatus: vi.fn(),
  beginEdit: vi.fn(),
  saveBaseline: vi.fn(),
  cancelEdit: vi.fn(),
  restoreNow: vi.fn(),
  restoreVersion: vi.fn(),
}))

function renderComponent() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <BaselinePage />
    </QueryClientProvider>
  )
}

// ky fills HTTPError.data while handling the response, so the error has to
// come out of a real request rather than the constructor.
async function httpError(body: string, status: number): Promise<HTTPError> {
  const fetch = async () =>
    new Response(body, {
      status,
      headers: { "Content-Type": "application/json" },
    })
  try {
    await ky.get("http://localhost/api/catalog/admin/ops/status", {
      fetch,
      retry: 0,
    })
  } catch (err) {
    if (err instanceof HTTPError) return err
    throw err
  }
  throw new Error("expected an HTTPError")
}

describe("BaselinePage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("points to Cloudflare Access when the status request never reaches the backend", async () => {
    // What fetch throws when Access redirects it to its login page.
    vi.mocked(baselineService.getBaselineStatus).mockRejectedValue(
      new TypeError("Failed to fetch")
    )

    renderComponent()

    const link = await screen.findByRole("link", {
      name: "doğrulama sayfasını yeni sekmede aç",
    })
    expect(link).toHaveAttribute("href", "/api/catalog/admin/ops/status")
    expect(screen.getByText("Sistem durumu okunamadı")).toBeInTheDocument()
  })

  it("shows the backend's message when the status request is refused", async () => {
    vi.mocked(baselineService.getBaselineStatus).mockRejectedValue(
      await httpError(
        JSON.stringify({
          error: "Bu işlem için yetkiniz yok",
          code: "FORBIDDEN",
        }),
        403
      )
    )

    renderComponent()

    expect(
      await screen.findByText("Bu işlem için yetkiniz yok")
    ).toBeInTheDocument()
    expect(screen.queryByRole("link")).not.toBeInTheDocument()
  })

  it("renders normal status correctly", async () => {
    vi.mocked(baselineService.getBaselineStatus).mockResolvedValue({
      mode: "normal",
      current: "20260925-100000",
      versions: ["20260925-100000", "20260924-040000"],
      edit_deadline: null,
      last_action: "save",
      last_error: null,
      updated_at: "2026-09-25T10:00:00Z",
    })

    renderComponent()

    await waitFor(() => {
      expect(
        screen.getByText("Kalıcı Veri ve Canlı Demo Yönetimi")
      ).toBeInTheDocument()
    })

    expect(screen.getByText("Normal (Korumalı)")).toBeInTheDocument()
    expect(screen.getByText("Düzenlemeye Başla")).toBeInTheDocument()
    expect(
      screen.getByText("Bugünkü Değişiklikleri Şimdi Geri Al")
    ).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText("Aktif Sürüm")).toBeInTheDocument()
    })
  })

  it("renders editing status correctly", async () => {
    vi.mocked(baselineService.getBaselineStatus).mockResolvedValue({
      mode: "editing",
      current: "20260925-100000",
      versions: ["20260925-100000"],
      edit_deadline: new Date(Date.now() + 60000).toISOString(),
      last_action: "begin_edit",
      last_error: null,
      updated_at: "2026-09-25T10:00:00Z",
    })

    renderComponent()

    await waitFor(() => {
      expect(screen.getByText("Düzenleme Modu Aktif")).toBeInTheDocument()
    })

    expect(screen.getByText("Kaydet ve Yayına Al")).toBeInTheDocument()
    expect(screen.getByText("Vazgeç ve Sıfırla")).toBeInTheDocument()
  })

  it("opens confirmation dialog when clicking Düzenlemeye Başla", async () => {
    const user = userEvent.setup()
    vi.mocked(baselineService.getBaselineStatus).mockResolvedValue({
      mode: "normal",
      current: "20260925-100000",
      versions: ["20260925-100000"],
      edit_deadline: null,
      last_action: "save",
      last_error: null,
      updated_at: "2026-09-25T10:00:00Z",
    })

    renderComponent()

    await waitFor(() => {
      expect(screen.getByText("Düzenlemeye Başla")).toBeInTheDocument()
    })

    await user.click(screen.getByText("Düzenlemeye Başla"))

    expect(screen.getByText("Düzenleme Moduna Başla")).toBeInTheDocument()
    expect(screen.getByText("Başla")).toBeInTheDocument()
    expect(screen.getByText("İptal")).toBeInTheDocument()
  })

  it("keeps polling until demo-ops has handled the queued command", async () => {
    const user = userEvent.setup()
    vi.mocked(baselineService.getBaselineStatus).mockResolvedValue({
      mode: "normal",
      current: "20260925-100000",
      versions: ["20260925-100000"],
      edit_deadline: null,
      last_action: "save",
      last_command_id: "cmd-0",
      last_error: null,
      updated_at: "2026-09-25T10:00:00Z",
    })
    vi.mocked(baselineService.beginEdit).mockResolvedValue("cmd-1")

    renderComponent()
    await user.click(await screen.findByText("Düzenlemeye Başla"))
    await user.click(screen.getByText("Başla"))

    // The status still predates the command, so the page must not settle.
    expect(await screen.findByText("İşlem Yapılıyor...")).toBeInTheDocument()

    vi.mocked(baselineService.getBaselineStatus).mockResolvedValue({
      mode: "editing",
      current: "20260925-100000",
      versions: ["20260925-100000"],
      edit_deadline: new Date(Date.now() + 60000).toISOString(),
      last_action: "begin_edit",
      last_command_id: "cmd-1",
      last_error: null,
      updated_at: "2026-09-25T10:01:00Z",
    })

    await waitFor(
      () => {
        expect(screen.getByText("Düzenleme Modu Aktif")).toBeInTheDocument()
      },
      { timeout: 4000 }
    )
  })
})
