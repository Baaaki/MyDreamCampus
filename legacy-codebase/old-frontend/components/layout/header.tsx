'use client';

import { useState, useEffect } from 'react';
import { useTheme } from '@/components/providers/theme-provider';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Moon, Sun, LogOut, User, Settings, Bell } from 'lucide-react';
import { useRouter } from 'next/navigation';

const roleTitles: Record<string, string> = {
  admin: 'Admin Panel',
  teacher: 'Akademisyen Paneli',
  student: 'Ogrenci Paneli',
};

const roleLabels: Record<string, string> = {
  admin: 'Admin',
  teacher: 'Akademisyen',
  student: 'Ogrenci',
};

export function Header() {
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();
  const [user, setUser] = useState<{ email: string; role: string } | null>(null);

  useEffect(() => {
    const stored = localStorage.getItem('user');
    if (stored) {
      try {
        setUser(JSON.parse(stored));
      } catch {
        // ignore parse errors
      }
    }
  }, []);

  const handleLogout = () => {
    // Clear localStorage
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');

    // Clear cookies
    document.cookie = 'access_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    document.cookie = 'user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';

    // Redirect to login
    router.push('/auth/login');
  };

  const panelTitle = roleTitles[user?.role ?? ''] ?? 'Panel';
  const roleLabel = roleLabels[user?.role ?? ''] ?? 'Kullanici';

  return (
    <header className="fixed top-0 right-0 left-52 z-30 h-16 border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900 transition-colors">
      <div className="flex h-full items-center justify-between px-6">
        {/* Left side - Page title or breadcrumb can go here */}
        <div className="flex items-center gap-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            {panelTitle}
          </h2>
        </div>

        {/* Right side - Actions */}
        <div className="flex items-center gap-3">
          {/* Notifications */}
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="h-5 w-5 text-gray-600 dark:text-gray-400" />
            <span className="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-red-500 text-[10px] font-medium text-white flex items-center justify-center">
              3
            </span>
          </Button>

          {/* Theme Toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="text-gray-600 dark:text-gray-400"
          >
            {theme === 'light' ? (
              <Moon className="h-5 w-5" />
            ) : (
              <Sun className="h-5 w-5" />
            )}
          </Button>

          {/* User Menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="flex items-center gap-2">
                <div className="h-8 w-8 rounded-full bg-indigo-100 dark:bg-indigo-900 flex items-center justify-center">
                  <User className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                </div>
                <div className="text-left hidden sm:block">
                  <p className="text-sm font-medium text-gray-900 dark:text-white">{roleLabel}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{user?.email ?? ''}</p>
                </div>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuItem onClick={() => router.push('/staff/profile')}>
                <User className="mr-2 h-4 w-4" />
                Profil
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => router.push('/auth/change-password')}>
                <Settings className="mr-2 h-4 w-4" />
                Şifre Değiştir
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout} className="text-red-600 dark:text-red-400">
                <LogOut className="mr-2 h-4 w-4" />
                Çıkış Yap
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
