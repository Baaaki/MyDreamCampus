import { Outlet } from "react-router"
import { Sidebar } from "./sidebar"
import { Header } from "./header"

export function AdminLayout() {
  return (
    <div className="min-h-screen bg-gray-50 transition-colors dark:bg-gray-950">
      <Sidebar />
      <Header />
      <main className="ml-64 min-h-screen pt-16">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
