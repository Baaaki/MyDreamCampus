import { Outlet } from "react-router"
import { TeacherSidebar } from "./teacher-sidebar"
import { Header } from "./header"

export function TeacherLayout() {
  return (
    <div className="min-h-screen bg-gray-50 transition-colors dark:bg-gray-950">
      <TeacherSidebar />
      <Header />
      <main className="ml-64 min-h-screen pt-16">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
