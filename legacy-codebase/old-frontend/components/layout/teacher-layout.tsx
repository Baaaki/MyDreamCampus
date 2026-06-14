'use client';

import { TeacherSidebar } from './teacher-sidebar';
import { Header } from './header';

interface TeacherLayoutProps {
  children: React.ReactNode;
}

export function TeacherLayout({ children }: TeacherLayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
      {/* Sidebar */}
      <TeacherSidebar />

      {/* Header */}
      <Header />

      {/* Main Content */}
      <main className="ml-64 pt-16 min-h-screen">
        <div className="p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
