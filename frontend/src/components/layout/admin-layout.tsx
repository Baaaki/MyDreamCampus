import { Outlet } from "react-router"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { cn } from "@/lib/utils"
import { useIsDemoMode } from "@/lib/services/demo-service"

export function AdminLayout() {
  const isDemo = useIsDemoMode()
  return (
    <div className="min-h-screen bg-gray-50 transition-colors dark:bg-gray-950">
      <Sidebar />
      <Header />
      <main className={cn("ml-64 min-h-screen", isDemo ? "pt-24" : "pt-16")}>
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
