import { Suspense, lazy } from 'react';
import { Routes, Route, Navigate } from 'react-router';
import { AdminLayout } from '@/components/layout/admin-layout';
import { StudentLayout } from '@/components/layout/student-layout';
import { TeacherLayout } from '@/components/layout/teacher-layout';
import { AuthGuard } from '@/components/auth-guard';

// Auth pages stay eager: login is the cold-start landing for unauthenticated
// users, and the auth bundle is small enough that splitting it would only
// add a network round trip before the first render.
import LoginPage from '@/pages/auth/login';
import NotFoundPage from '@/pages/not-found';
import ChangePasswordPage from '@/pages/auth/change-password';
import SessionsPage from '@/pages/auth/sessions';

// Admin pages — lazy loaded; users only land here after auth.
const DashboardPage = lazy(() => import('@/pages/admin/dashboard'));
const SettingsPage = lazy(() => import('@/pages/admin/settings'));
const AdminEnrollmentPage = lazy(() => import('@/pages/admin/enrollment'));
const StaffPage = lazy(() => import('@/pages/admin/staff'));
const StaffDetailsPage = lazy(() => import('@/pages/admin/staff/personel-details'));
const StudentsPage = lazy(() => import('@/pages/admin/students'));
const StudentPage = lazy(() => import('@/pages/admin/students/student'));
const StudentProfilePage = lazy(() => import('@/pages/admin/students/student/profile'));
const AdvisorsPage = lazy(() => import('@/pages/admin/students/advisors'));
const CatalogPage = lazy(() => import('@/pages/admin/catalog'));
const CatalogAddPage = lazy(() => import('@/pages/admin/catalog/add'));
const CatalogEditPage = lazy(() => import('@/pages/admin/catalog/edit'));
const CatalogSchedulePage = lazy(() => import('@/pages/admin/catalog/schedule'));
const SemesterCoursesPage = lazy(() => import('@/pages/admin/semester-courses'));
const SemesterCoursesListPage = lazy(() => import('@/pages/admin/semester-courses/list'));
const SemesterReviewPage = lazy(() => import('@/pages/admin/semester-courses/review'));
const CafeteriasPage = lazy(() => import('@/pages/admin/meal/cafeterias'));
const MealAdminPage = lazy(() => import('@/pages/admin/meal/admin'));
const MenusPage = lazy(() => import('@/pages/admin/meal/menus'));
const MealStudentPage = lazy(() => import('@/pages/admin/meal/student'));
const TimeSettingsPage = lazy(() => import('@/pages/admin/system/time'));
const SemestersPage = lazy(() => import('@/pages/admin/system/semesters'));
const SemesterWizardPage = lazy(() => import('@/pages/admin/system/semesters/new'));
const AuditPage = lazy(() => import('@/pages/admin/system/audit'));
const AdminAttendancePage = lazy(() => import('@/pages/admin/attendance'));
const AdminAttendanceSessionPage = lazy(() => import('@/pages/admin/attendance/sessionId'));
const AdminGradesPage = lazy(() => import('@/pages/admin/grades'));

// Teacher pages
const TeacherAttendancePage = lazy(() => import('@/pages/teacher/attendance'));
const TeacherAttendanceCoursePage = lazy(() => import('@/pages/teacher/attendance/courseId'));
const TeacherAttendanceSessionPage = lazy(() => import('@/pages/teacher/attendance/courseId/session/sessionId'));
const TeacherEnrollmentPage = lazy(() => import('@/pages/teacher/enrollment'));
const TeacherGradesPage = lazy(() => import('@/pages/teacher/grades'));
const TeacherGradesCourseSlugPage = lazy(() => import('@/pages/teacher/grades/courseId/slug'));

// Student pages
const StudentDashboardPage = lazy(() => import('@/pages/student/dashboard'));
const StudentAttendancePage = lazy(() => import('@/pages/student/attendance'));
const StudentEnrollmentPage = lazy(() => import('@/pages/student/enrollment'));
const StudentEnrollmentRejectionsPage = lazy(() => import('@/pages/student/enrollment/rejections'));
const StudentGradesPage = lazy(() => import('@/pages/student/grades'));
const StudentCafeteriaPage = lazy(() => import('@/pages/student/cafeteria'));
const StudentCafeteriaMenuPage = lazy(() => import('@/pages/student/cafeteria/menu'));
const StudentCafeteriaHistoryPage = lazy(() => import('@/pages/student/cafeteria/history'));

function LoadingFallback() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="flex flex-col items-center gap-3">
        <div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
        <p className="text-sm text-gray-600">Yükleniyor…</p>
      </div>
    </div>
  );
}

export function AppRoutes() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <Routes>
        <Route path="/" element={<Navigate to="/auth/login" replace />} />

        {/* Auth (public) */}
        <Route path="/auth/login" element={<LoginPage />} />
        <Route path="/auth/change-password" element={<ChangePasswordPage />} />
        <Route path="/auth/sessions" element={<SessionsPage />} />

        {/* Admin */}
        <Route element={<AuthGuard allowedRoles={['admin']} />}>
          <Route element={<AdminLayout />}>
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/enrollment" element={<AdminEnrollmentPage />} />
            <Route path="/staff" element={<StaffPage />} />
            <Route path="/staff/personel-details" element={<StaffDetailsPage />} />
            <Route path="/students" element={<StudentsPage />} />
            <Route path="/students/student" element={<StudentPage />} />
            <Route path="/students/student/profile" element={<StudentProfilePage />} />
            <Route path="/students/advisors" element={<AdvisorsPage />} />
            <Route path="/catalog" element={<CatalogPage />} />
            <Route path="/catalog/add" element={<CatalogAddPage />} />
            <Route path="/catalog/edit" element={<CatalogEditPage />} />
            <Route path="/catalog/schedule" element={<CatalogSchedulePage />} />
            <Route path="/semester-courses" element={<SemesterCoursesPage />} />
            <Route path="/semester-courses/list" element={<SemesterCoursesListPage />} />
            <Route path="/semester-courses/review" element={<SemesterReviewPage />} />
            <Route path="/meal/cafeterias" element={<CafeteriasPage />} />
            <Route path="/meal/admin" element={<MealAdminPage />} />
            <Route path="/meal/menus" element={<MenusPage />} />
            <Route path="/meal/student" element={<MealStudentPage />} />
            <Route path="/attendance" element={<AdminAttendancePage />} />
            <Route path="/attendance/:sessionId" element={<AdminAttendanceSessionPage />} />
            <Route path="/grades" element={<AdminGradesPage />} />
            <Route path="/system/time" element={<TimeSettingsPage />} />
            <Route path="/system/semesters" element={<SemestersPage />} />
            <Route path="/system/semesters/new" element={<SemesterWizardPage />} />
            <Route path="/system/audit" element={<AuditPage />} />
          </Route>
        </Route>

        {/* Teacher */}
        <Route element={<AuthGuard allowedRoles={['teacher']} />}>
          <Route element={<TeacherLayout />}>
            <Route path="/teacher/attendance" element={<TeacherAttendancePage />} />
            <Route path="/teacher/attendance/:courseId" element={<TeacherAttendanceCoursePage />} />
            <Route path="/teacher/attendance/:courseId/session/:sessionId" element={<TeacherAttendanceSessionPage />} />
            <Route path="/teacher/enrollment" element={<TeacherEnrollmentPage />} />
            <Route path="/teacher/grades" element={<TeacherGradesPage />} />
            <Route path="/teacher/grades/:courseId/:slug" element={<TeacherGradesCourseSlugPage />} />
          </Route>
        </Route>

        {/* Student */}
        <Route element={<AuthGuard allowedRoles={['student']} />}>
          <Route element={<StudentLayout />}>
            <Route path="/student/dashboard" element={<StudentDashboardPage />} />
            <Route path="/student/attendance" element={<StudentAttendancePage />} />
            <Route path="/student/enrollment" element={<StudentEnrollmentPage />} />
            <Route path="/student/enrollment/rejections" element={<StudentEnrollmentRejectionsPage />} />
            <Route path="/student/grades" element={<StudentGradesPage />} />
            <Route path="/student/cafeteria" element={<StudentCafeteriaPage />} />
            <Route path="/student/cafeteria/menu" element={<StudentCafeteriaMenuPage />} />
            <Route path="/student/cafeteria/history" element={<StudentCafeteriaHistoryPage />} />
          </Route>
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  );
}
