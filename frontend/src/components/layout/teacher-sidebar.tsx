import { FileCheck, BarChart3, ClipboardCheck } from "lucide-react"
import {
  AppSidebar,
  type SidebarDrawerProps,
  type SidebarNavItem,
} from "./app-sidebar"

const navItems: SidebarNavItem[] = [
  {
    label: "Yoklama",
    href: "/teacher/attendance",
    icon: FileCheck,
  },
  {
    label: "Not Girme",
    href: "/teacher/grades",
    icon: BarChart3,
  },
  {
    label: "Ders Kaydı Onaylama",
    href: "/teacher/enrollment",
    icon: ClipboardCheck,
  },
]

export function TeacherSidebar(props: SidebarDrawerProps) {
  return <AppSidebar items={navItems} {...props} />
}
