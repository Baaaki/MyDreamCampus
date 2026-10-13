import { AppShell } from "./app-shell"
import { StudentSidebar } from "./student-sidebar"

export function StudentLayout() {
  return <AppShell sidebar={StudentSidebar} />
}
