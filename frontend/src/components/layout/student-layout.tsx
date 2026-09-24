import { Outlet } from "react-router"
import { StudentSidebar } from "./student-sidebar"
import { Header } from "./header"
import { cn } from "@/lib/utils"
import { useIsDemoMode } from "@/lib/services/demo-service"

export function StudentLayout() {
  const isDemo = useIsDemoMode()
  return (
    <div className="min-h-screen bg-gray-50 transition-colors dark:bg-gray-950">
      <StudentSidebar />
      <Header />
      <main className={cn("ml-52 min-h-screen", isDemo ? "pt-24" : "pt-16")}>
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
