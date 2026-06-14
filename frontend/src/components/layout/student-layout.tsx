import { Outlet } from 'react-router';
import { StudentSidebar } from './student-sidebar';
import { Header } from './header';

export function StudentLayout() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
      <StudentSidebar />
      <Header />
      <main className="ml-52 pt-16 min-h-screen">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
