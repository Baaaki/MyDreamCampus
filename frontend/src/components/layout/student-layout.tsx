import { Outlet } from "react-router"
import { StudentSidebar } from "./student-sidebar"
import { Header } from "./header"

export function StudentLayout() {
  return (
    <div className="min-h-screen bg-gray-50 transition-colors dark:bg-gray-950">
      <StudentSidebar />
      <Header />
      <main className="ml-52 min-h-screen pt-16">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
