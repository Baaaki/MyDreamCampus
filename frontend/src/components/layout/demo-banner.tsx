import { AlertTriangle } from "lucide-react"
import { SystemEditingBanner } from "@/components/layout/system-editing-banner"
import { useIsSystemEditing } from "@/lib/services/baseline-service"
import { useIsDemoMode } from "@/lib/services/demo-service"

export function DemoBanner() {
  const isDemo = useIsDemoMode()
  const isEditing = useIsSystemEditing()

  // Both notices use the strip the layouts leave room for; while the super
  // admin edits, that notice takes the demo notice's place.
  if (isEditing) return <SystemEditingBanner />
  if (!isDemo) return null

  return (
    <div
      role="region"
      aria-label="Demo bilgilendirmesi"
      className="fixed top-0 right-0 left-0 z-50 flex h-8 items-center justify-center gap-2 bg-amber-500 px-4 text-xs font-semibold text-amber-950 shadow-xs dark:bg-amber-600 dark:text-amber-50"
    >
      <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">
        Bu bir demo. Yaptığınız değişiklikler her gece 04:00&apos;te geri alınır. Gerçek kişisel bilgi girmeyin.
      </span>
    </div>
  )
}
