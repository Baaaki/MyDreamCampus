import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
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

describe("BaselinePage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
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
})
