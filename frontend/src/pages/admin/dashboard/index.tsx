import { Link } from "react-router"
import { useQuery } from "@tanstack/react-query"
import {
  Users,
  GraduationCap,
  BookOpen,
  FileCheck,
  BarChart3,
  UtensilsCrossed,
  UserCheck,
  Calendar,
  Loader2,
} from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"
import { getAdminDashboardStats } from "@/lib/services/admin-dashboard-service"

const statCards = [
  {
    key: "students",
    title: "Toplam Öğrenci",
    icon: GraduationCap,
    color: "bg-blue-500",
  },
  {
    key: "staff",
    title: "Toplam Personel",
    icon: Users,
    color: "bg-green-500",
  },
  {
    key: "openedCourses",
    title: "Aktif Dönemde Açılan Ders",
    icon: BookOpen,
    color: "bg-purple-500",
  },
  {
    key: "todaySessions",
    title: "Bugünkü Yoklama Oturumu",
    icon: UserCheck,
    color: "bg-orange-500",
  },
] as const

function formatCount(value: number | null | undefined): string {
  return value == null ? "—" : value.toLocaleString("tr-TR")
}

const quickLinks = [
  {
    title: "Personel Yönetimi",
    description: "Personel ekle, düzenle ve yönet",
    href: "/staff",
    icon: Users,
    color: "from-green-500 to-emerald-600",
  },
  {
    title: "Öğrenci Yönetimi",
    description: "Öğrenci kayıtları ve danışman atamaları",
    href: "/students",
    icon: GraduationCap,
    color: "from-blue-500 to-cyan-600",
  },
  {
    title: "Ders Kataloğu",
    description: "Ders ekle ve müfredatı yönet",
    href: "/catalog",
    icon: BookOpen,
    color: "from-purple-500 to-pink-600",
  },
  {
    title: "Yoklama Sistemi",
    description: "Yoklama takibi ve raporları",
    href: "/attendance",
    icon: FileCheck,
    color: "from-red-500 to-orange-600",
  },
  {
    title: "Not Yönetimi",
    description: "Not girişi ve transkript",
    href: "/grades",
    icon: BarChart3,
    color: "from-indigo-500 to-blue-600",
  },
  {
    title: "Yemekhane",
    description: "Yemekhane rezervasyon yönetimi",
    href: "/meal/admin",
    icon: UtensilsCrossed,
    color: "from-orange-500 to-amber-600",
  },
  {
    title: "Oturumlar",
    description: "Aktif oturum yönetimi",
    href: "/auth/sessions",
    icon: Calendar,
    color: "from-gray-500 to-slate-600",
  },
]

export default function DashboardPage() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ["admin", "dashboard", "stats"],
    queryFn: () => getAdminDashboardStats(),
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Dashboard
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Hoş geldiniz! Kampüs yönetim sistemine genel bakış.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        {statCards.map((stat) => {
          const Icon = stat.icon
          return (
            <Card
              key={stat.title}
              className="dark:border-gray-800 dark:bg-gray-900"
            >
              <CardContent className="p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {stat.title}
                    </p>
                    <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">
                      {isLoading ? (
                        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
                      ) : (
                        formatCount(stats?.[stat.key])
                      )}
                    </p>
                  </div>
                  <div className={`rounded-lg p-3 ${stat.color}`}>
                    <Icon className="h-6 w-6 text-white" />
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* Quick Links */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
          Hızlı Erişim
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          {quickLinks.map((link) => {
            const Icon = link.icon
            return (
              <Link key={link.href} to={link.href} className="group block">
                <Card className="h-full transition-all hover:-translate-y-1 hover:shadow-lg dark:border-gray-800 dark:bg-gray-900 dark:hover:border-gray-700">
                  <CardContent className="p-6">
                    <div
                      className={`inline-flex rounded-lg bg-gradient-to-br p-3 ${link.color} mb-4`}
                    >
                      <Icon className="h-6 w-6 text-white" />
                    </div>
                    <h3 className="font-semibold text-gray-900 transition-colors group-hover:text-indigo-600 dark:text-white dark:group-hover:text-indigo-400">
                      {link.title}
                    </h3>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                      {link.description}
                    </p>
                  </CardContent>
                </Card>
              </Link>
            )
          })}
        </div>
      </div>
    </div>
  )
}
