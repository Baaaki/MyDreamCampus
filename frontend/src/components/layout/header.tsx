import { useState } from "react"
import { useTheme } from "@/components/providers/theme-context"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Moon, Sun, LogOut, User, Settings, Bell, Menu, X } from "lucide-react"
import { useNavigate } from "react-router"
import { useQueryClient } from "@tanstack/react-query"
import { authApi } from "@/lib/api-client"
import { clearUserSession } from "@/lib/session"
import { cn } from "@/lib/utils"
import { useIsDemoMode } from "@/lib/services/demo-service"

const roleLabels: Record<string, string> = {
  admin: "Admin",
  teacher: "Akademisyen",
  student: "Öğrenci",
}

// The roles have no shared profile page, so each lands on the page that
// holds its own account details or day-to-day view.
const profilePaths: Record<string, string> = {
  admin: "/settings",
  teacher: "/teacher/attendance",
  student: "/student/dashboard",
}

interface HeaderProps {
  menuOpen: boolean
  onMenuClick: () => void
}

export function Header({ menuOpen, onMenuClick }: HeaderProps) {
  const { theme, toggleTheme } = useTheme()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [user] = useState<{ email: string; role: string } | null>(() => {
    const stored = localStorage.getItem("user")
    if (!stored) return null
    try {
      return JSON.parse(stored)
    } catch {
      return null
    }
  })

  const handleLogout = async () => {
    try {
      // Call backend logout to clear httpOnly cookies and invalidate session
      await authApi.post("logout")
    } catch {
      // Even if the API call fails, clear local state
    }

    clearUserSession(queryClient)

    // Redirect to login
    navigate("/auth/login")
  }

  const isDemo = useIsDemoMode()
  const profilePath = profilePaths[user?.role ?? ""]
  const roleLabel = roleLabels[user?.role ?? ""] ?? "Kullanıcı"

  return (
    <header
      className={cn(
        "fixed right-0 left-0 z-30 h-16 border-b border-border bg-background/90 backdrop-blur transition-colors",
        isDemo ? "top-8" : "top-0"
      )}
    >
      <div className="flex h-full items-center justify-between gap-2 px-4 md:px-6">
        <div className="flex min-w-0 items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={onMenuClick}
            aria-label={menuOpen ? "Menüyü kapat" : "Menüyü aç"}
            aria-expanded={menuOpen}
            className="-ml-2 shrink-0 text-muted-foreground lg:hidden"
          >
            {menuOpen ? (
              <X className="h-5 w-5" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </Button>
          <p className="flex min-w-0 items-baseline gap-2 truncate">
            <span className="text-[1.05rem] font-extrabold tracking-tight">
              MyDreamCampus
            </span>
            <span className="hidden text-sm text-muted-foreground sm:inline">
              {roleLabel}
            </span>
          </p>
        </div>

        {/* Right side - Actions */}
        <div className="flex shrink-0 items-center gap-1 sm:gap-3">
          {/* Notifications */}
          <Button variant="ghost" size="icon">
            <Bell className="h-5 w-5 text-muted-foreground" />
          </Button>

          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label={
              theme === "light" ? "Koyu temaya geç" : "Açık temaya geç"
            }
            className="text-muted-foreground"
          >
            {theme === "light" ? (
              <Moon className="h-5 w-5" />
            ) : (
              <Sun className="h-5 w-5" />
            )}
          </Button>

          {/* User Menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
                  <User className="h-4 w-4 text-primary" />
                </div>
                <div className="hidden text-left sm:block">
                  <p className="text-sm font-medium">{user?.email ?? ""}</p>
                </div>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              {profilePath && (
                <DropdownMenuItem onClick={() => navigate(profilePath)}>
                  <User className="mr-2 h-4 w-4" />
                  Profil
                </DropdownMenuItem>
              )}
              <DropdownMenuItem
                onClick={() => navigate("/auth/change-password")}
              >
                <Settings className="mr-2 h-4 w-4" />
                Şifre Değiştir
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={handleLogout}
                className="text-destructive"
              >
                <LogOut className="mr-2 h-4 w-4" />
                Çıkış Yap
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}
