import { Outlet } from 'react-router';
import { TeacherSidebar } from './teacher-sidebar';
import { Header } from './header';

export function TeacherLayout() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
      <TeacherSidebar />
      <Header />
      <main className="ml-64 pt-16 min-h-screen">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
