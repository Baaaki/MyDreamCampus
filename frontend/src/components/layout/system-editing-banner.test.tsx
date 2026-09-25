import { describe, it, expect, beforeEach } from "vitest"
import { render, screen, act } from "@testing-library/react"
import { SystemEditingBanner } from "./system-editing-banner"
import { setSystemEditing } from "@/lib/services/baseline-service"

describe("SystemEditingBanner", () => {
  beforeEach(() => {
    act(() => {
      setSystemEditing(false)
    })
  })

  it("renders nothing when system is not editing", () => {
    render(<SystemEditingBanner />)
    expect(
      screen.queryByText("Sistem güncelleniyor, şu an değişiklik yapılamaz.")
    ).not.toBeInTheDocument()
  })

  it("renders warning banner when system editing is true", () => {
    render(<SystemEditingBanner />)
    act(() => {
      setSystemEditing(true)
    })
    expect(
      screen.getByText("Sistem güncelleniyor, şu an değişiklik yapılamaz.")
    ).toBeInTheDocument()
  })
})
