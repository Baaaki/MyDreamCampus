
import { Link } from 'react-router';
import { useLocation } from 'react-router';
import { cn } from '@/lib/utils';
import {
  FileCheck,
  BarChart3,
  ClipboardCheck,
} from 'lucide-react';

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
}

const navItems: NavItem[] = [
  {
    label: 'Yoklama',
    href: '/teacher/attendance',
    icon: FileCheck,
  },
  {
    label: 'Not Girme',
    href: '/teacher/grades',
    icon: BarChart3,
  },
  {
    label: 'Ders Kaydı Onaylama',
    href: '/teacher/enrollment',
    icon: ClipboardCheck,
  },
];

export function TeacherSidebar() {
  const { pathname } = useLocation();

  const isActive = (href: string) => {
    return pathname === href || pathname.startsWith(href + '/');
  };

  return (
    <aside className="fixed left-0 top-0 z-40 h-screen w-64 border-r border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900 transition-colors">
      {/* Logo */}
      <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6 dark:border-gray-800">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-600 text-white font-bold">
          Ö
        </div>
        <span className="text-lg font-bold text-gray-900 dark:text-white">
          Öğretmen Portalı
        </span>
      </div>

      {/* Navigation */}
      <nav className="h-[calc(100vh-4rem)] overflow-y-auto p-4">
        <ul className="space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;

            return (
              <li key={item.label}>
                <Link
                  to={item.href}
                  className={cn(
                    'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                    isActive(item.href)
                      ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                      : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                  )}
                >
                  <Icon className="h-5 w-5" />
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </aside>
  );
}
