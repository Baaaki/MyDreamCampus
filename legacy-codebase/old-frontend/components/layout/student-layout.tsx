'use client';

import { StudentSidebar } from './student-sidebar';
import { Header } from './header';

interface StudentLayoutProps {
  children: React.ReactNode;
}

export function StudentLayout({ children }: StudentLayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
      {/* Sidebar */}
      <StudentSidebar />

      {/* Header */}
      <Header />

      {/* Main Content */}
      <main className="ml-52 pt-16 min-h-screen">
        <div className="p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
