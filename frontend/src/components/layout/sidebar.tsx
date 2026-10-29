import {
  GraduationCap,
  BookOpen,
  FileCheck,
  BarChart3,
  UtensilsCrossed,
  Settings,
  LayoutDashboard,
  UserCog,
  ServerCog,
  CalendarPlus,
  History,
} from "lucide-react"
import {
  AppSidebar,
  type SidebarDrawerProps,
  type SidebarNavItem,
} from "./app-sidebar"

const navItems: SidebarNavItem[] = [
  {
    label: "Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    label: "Personel",
    icon: UserCog,
    children: [
      { label: "Personel Listesi", href: "/staff" },
      { label: "Personel Detay", href: "/staff/personel-details" },
    ],
  },
  {
    label: "Öğrenci",
    icon: GraduationCap,
    children: [
      { label: "Öğrenci Listesi", href: "/students" },
      { label: "Danışman Atamaları", href: "/students/advisors" },
      { label: "Ders Kaydı Onayları", href: "/students/enrollment" },
      { label: "Öğrenci Profili", href: "/students/student" },
    ],
  },
  {
    label: "Ders Kataloğu",
    icon: BookOpen,
    children: [
      { label: "Katalog", href: "/catalog" },
      { label: "Ders Ekle", href: "/catalog/add" },
      { label: "Ders Güncelle", href: "/catalog/edit" },
    ],
  },
  {
    label: "Dönem Dersleri",
    icon: CalendarPlus,
    children: [
      { label: "Ders Aç", href: "/semester-courses" },
      { label: "Açılan Dersler", href: "/semester-courses/list" },
    ],
  },
  {
    label: "Yoklama",
    href: "/attendance",
    icon: FileCheck,
  },
  {
    label: "Notlar",
    href: "/grades",
    icon: BarChart3,
  },
  {
    label: "Yemekhane",
    icon: UtensilsCrossed,
    children: [
      { label: "Yemekhaneler", href: "/meal/cafeterias" },
      { label: "Menü Ekle", href: "/meal/menus" },
    ],
  },
  {
    label: "Zaman Makinesi",
    href: "/system/time",
    icon: History,
  },
  {
    label: "Sistem",
    icon: ServerCog,
    children: [
      { label: "Dönem Yönetimi", href: "/system/semesters" },
      { label: "Audit Log", href: "/system/audit" },
    ],
  },
  {
    label: "Ayarlar",
    href: "/settings",
    icon: Settings,
  },
]

export function Sidebar(props: SidebarDrawerProps) {
  const userStr =
    typeof window !== "undefined" ? localStorage.getItem("user") : null
  let isSuperAdmin = false
  if (userStr) {
    try {
      const u = JSON.parse(userStr)
      isSuperAdmin = !!u.is_superadmin
    } catch {
      // ignore invalid json in storage
    }
  }

  const currentNavItems = navItems.map((item) => {
    if (item.label === "Sistem") {
      const children = [...(item.children || [])]
      if (
        isSuperAdmin &&
        !children.some((c) => c.href === "/system/baseline")
      ) {
        children.push({ label: "Kalıcı Veri", href: "/system/baseline" })
      }
      return { ...item, children }
    }
    return item
  })

  return (
    <AppSidebar
      brandInitial="M"
      brandTitle="MyDreamCampus"
      accent="indigo"
      items={currentNavItems}
      {...props}
    />
  )
}
