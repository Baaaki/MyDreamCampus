import {
  LayoutDashboard,
  FileCheck,
  BarChart3,
  UtensilsCrossed,
  BookOpen,
} from "lucide-react"
import {
  AppSidebar,
  type SidebarDrawerProps,
  type SidebarNavItem,
} from "./app-sidebar"

const navItems: SidebarNavItem[] = [
  {
    label: "Ana Sayfa",
    href: "/student/dashboard",
    icon: LayoutDashboard,
  },
  {
    label: "Ders Kayıt",
    icon: BookOpen,
    children: [
      { label: "Ders Kaydı", href: "/student/enrollment" },
      { label: "Reddedilmeler", href: "/student/enrollment/rejections" },
    ],
  },
  {
    label: "Yoklama",
    href: "/student/attendance",
    icon: FileCheck,
  },
  {
    label: "Ders Notları",
    href: "/student/grades",
    icon: BarChart3,
  },
  {
    label: "Yemekhane",
    icon: UtensilsCrossed,
    children: [
      { label: "Randevu Al", href: "/student/cafeteria" },
      { label: "Geçmiş Randevularım", href: "/student/cafeteria/history" },
      { label: "Menü", href: "/student/cafeteria/menu" },
    ],
  },
]

export function StudentSidebar(props: SidebarDrawerProps) {
  return <AppSidebar items={navItems} {...props} />
}
