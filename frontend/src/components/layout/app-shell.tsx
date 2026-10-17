import { useState, type ComponentType } from "react"
import { Outlet } from "react-router"
import { Header } from "./header"
import type { SidebarDrawerProps } from "./app-sidebar"
import { cn } from "@/lib/utils"
import { useIsDemoMode } from "@/lib/services/demo-service"

interface AppShellProps {
  sidebar: ComponentType<SidebarDrawerProps>
}

export function AppShell({ sidebar: Sidebar }: AppShellProps) {
  const isDemo = useIsDemoMode()
  const [drawerOpen, setDrawerOpen] = useState(false)

  return (
    <div className="min-h-screen bg-background transition-colors">
      <Sidebar
        mobileOpen={drawerOpen}
        onMobileClose={() => setDrawerOpen(false)}
      />
      <Header
        menuOpen={drawerOpen}
        onMenuClick={() => setDrawerOpen((open) => !open)}
      />
      {/* The rail reserves only its collapsed width; when it widens on hover
          it overlays the page instead of pushing it. */}
      <main className={cn("min-h-screen lg:ml-16", isDemo ? "pt-24" : "pt-16")}>
        <div className="p-4 md:p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
