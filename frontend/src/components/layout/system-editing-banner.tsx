import { AlertCircle } from "lucide-react"
import { useIsSystemEditing } from "@/lib/services/baseline-service"

export function SystemEditingBanner() {
  const isEditing = useIsSystemEditing()

  if (!isEditing) return null

  return (
    <div
      role="alert"
      aria-label="Sistem güncelleme bilgilendirmesi"
      className="text-destructive-foreground fixed top-0 right-0 left-0 z-50 flex h-8 animate-in items-center justify-center gap-2 bg-destructive px-4 text-xs font-semibold shadow-xs fade-in"
    >
      <AlertCircle className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">
        Sistem güncelleniyor, şu an değişiklik yapılamaz.
      </span>
    </div>
  )
}
