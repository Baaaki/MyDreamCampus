import { AppShell } from "./app-shell"
import { Sidebar } from "./sidebar"

export function AdminLayout() {
  return <AppShell sidebar={Sidebar} />
}
