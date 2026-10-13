import { AppShell } from "./app-shell"
import { TeacherSidebar } from "./teacher-sidebar"

export function TeacherLayout() {
  return <AppShell sidebar={TeacherSidebar} />
}
