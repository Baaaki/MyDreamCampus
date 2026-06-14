
import { Link } from 'react-router';
import { useLocation } from 'react-router';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  FileCheck,
  BarChart3,
  UtensilsCrossed,
  BookOpen,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { useState } from 'react';

interface NavItem {
  label: string;
  href?: string;
  icon: React.ElementType;
  children?: { label: string; href: string }[];
}

const navItems: NavItem[] = [
  {
    label: 'Ana Sayfa',
    href: '/student/dashboard',
    icon: LayoutDashboard,
  },
  {
    label: 'Ders Kayıt',
    icon: BookOpen,
    children: [
      { label: 'Ders Kaydı', href: '/student/enrollment' },
      { label: 'Reddedilmeler', href: '/student/enrollment/rejections' },
    ],
  },
  {
    label: 'Yoklama',
    href: '/student/attendance',
    icon: FileCheck,
  },
  {
    label: 'Ders Notları',
    href: '/student/grades',
    icon: BarChart3,
  },
  {
    label: 'Yemekhane',
    icon: UtensilsCrossed,
    children: [
      { label: 'Randevu Al', href: '/student/cafeteria' },
      { label: 'Geçmiş Randevularım', href: '/student/cafeteria/history' },
      { label: 'Menü', href: '/student/cafeteria/menu' },
    ],
  },
];

export function StudentSidebar() {
  const { pathname } = useLocation();
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  const toggleExpand = (label: string) => {
    setExpandedItems(prev =>
      prev.includes(label)
        ? prev.filter(item => item !== label)
        : [...prev, label]
    );
  };

  const isActive = (href: string) => {
    return pathname === href || pathname.startsWith(href + '/');
  };

  const isParentActive = (item: NavItem) => {
    if (item.href) return isActive(item.href);
    return item.children?.some(child => isActive(child.href)) || false;
  };

  return (
    <aside className="fixed left-0 top-0 z-40 h-screen w-52 border-r border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900 transition-colors">
      {/* Logo */}
      <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-4 dark:border-gray-800">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-600 text-white text-sm font-bold shrink-0">
          Ö
        </div>
        <span className="text-base font-bold text-gray-900 dark:text-white truncate">
          Öğrenci Portalı
        </span>
      </div>

      {/* Navigation */}
      <nav className="h-[calc(100vh-4rem)] overflow-y-auto p-4">
        <ul className="space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const hasChildren = item.children && item.children.length > 0;
            const isExpanded = expandedItems.includes(item.label);
            const parentActive = isParentActive(item);

            return (
              <li key={item.label}>
                {hasChildren ? (
                  <>
                    <button
                      onClick={() => toggleExpand(item.label)}
                      className={cn(
                        'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                        parentActive
                          ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300'
                          : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                      )}
                    >
                      <span className="flex items-center gap-3">
                        <Icon className="h-5 w-5" />
                        {item.label}
                      </span>
                      {isExpanded ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                    </button>
                    {isExpanded && item.children && (
                      <ul className="ml-4 mt-1 space-y-1 border-l-2 border-gray-200 pl-4 dark:border-gray-700">
                        {item.children.map((child) => (
                          <li key={child.href}>
                            <Link
                              to={child.href}
                              className={cn(
                                'block rounded-lg px-3 py-2 text-sm transition-colors',
                                isActive(child.href)
                                  ? 'bg-emerald-100 text-emerald-700 font-medium dark:bg-emerald-900/50 dark:text-emerald-300'
                                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200'
                              )}
                            >
                              {child.label}
                            </Link>
                          </li>
                        ))}
                      </ul>
                    )}
                  </>
                ) : (
                  <Link
                    to={item.href!}
                    className={cn(
                      'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                      isActive(item.href!)
                        ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300'
                        : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
                    )}
                  >
                    <Icon className="h-5 w-5" />
                    {item.label}
                  </Link>
                )}
              </li>
            );
          })}
        </ul>
      </nav>
    </aside>
  );
}
