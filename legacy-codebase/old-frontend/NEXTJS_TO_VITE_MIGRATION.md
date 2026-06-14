# Next.js -> React + Vite + shadcn Migration

Bu prompt, `old-frontend/` dizinindeki Next.js kodlarini `frontend/` (Vite) projesine migrate eder.

> **KURAL**: Paket yoneticisi olarak `bun` kullan. `npm`/`npx` KULLANMA. `npx shadcn` yerine `bunx --bun shadcn` kullan.
> **KURAL**: Hicbir dosyayi silme, yeni seyler ekleme veya refactor yapma. Sadece bu planda yazanlari birebir uygula.
> **KURAL**: Her adimi tamamladiktan sonra bir sonrakine gec. Adim atlama.

---

## Proje Bilgileri

- Calisma dizini: `/home/nautilus/Desktop/Playground/mydreamcampus/`
- Eski frontend (kaynak): `old-frontend/` (Next.js 16.1.1 — dosyalar buradan okunacak)
- Yeni frontend (hedef): `frontend/` (Vite + React — dosyalar buraya yazilacak)
- Her sey client-side SPA, SSR/SSG/Server Components/API Routes YOK.

---

## ON KOSUL (kullanici tarafindan zaten yapildi, ATLAMA)

Asagidaki 3 adim kullanici tarafindan interaktif terminalde elle yapildi. Bu adimlari YAPMA, direkt ADIM 1'den basla.

1. Eski `frontend/` dizini `old-frontend/` olarak yeniden adlandirildi
2. Yeni `frontend/` dizini olusturuldu: `bunx --bun shadcn@latest init -t vite` (Vite + shadcn birlikte kuruldu)
3. `bun add react-router @tanstack/react-query @base-ui/react cmdk date-fns ky qrcode.react tw-animate-css && bun add -d @types/qrcode.react` (ek bagimliliklar kuruldu)

---

## ADIM 1: Dosyalari kopyala

Asagidaki bash komutlarini calistir:

```bash
SRC=/home/nautilus/Desktop/Playground/mydreamcampus/old-frontend
DEST=/home/nautilus/Desktop/Playground/mydreamcampus/frontend/src

# --- UI componentleri (old-frontend'den tum shadcn + custom componentleri kopyala) ---
# shadcn init bos bir proje olusturdugu icin, mevcut componentleri oldugu gibi tasiyoruz
cp -r "$SRC/components/ui/"* "$DEST/components/ui/"

# --- Feature componentleri ---
cp -r "$SRC/components/enrollment" "$DEST/components/" 2>/dev/null || true
cp -r "$SRC/components/grades" "$DEST/components/" 2>/dev/null || true
cp -r "$SRC/components/meal" "$DEST/components/" 2>/dev/null || true

# --- Layout componentleri ---
cp -r "$SRC/components/layout" "$DEST/components/"

# --- Providers ---
cp -r "$SRC/components/providers" "$DEST/components/"

# --- Lib (api-client, types, constants, services) ---
# NOT: lib/utils.ts'i KOPYALAMA, shadcn zaten olusturdu
cp "$SRC/lib/api-client.ts" "$DEST/lib/"
cp "$SRC/lib/types.ts" "$DEST/lib/"
[ -f "$SRC/lib/constants.ts" ] && cp "$SRC/lib/constants.ts" "$DEST/lib/"
cp -r "$SRC/lib/services" "$DEST/lib/" 2>/dev/null || true

# --- Mock data ---
cp -r "$SRC/mock_data" "$DEST/" 2>/dev/null || true

# --- Env ---
[ -f "$SRC/.env.local" ] && cp "$SRC/.env.local" "$DEST/../.env.local"
[ -f "$SRC/.env.example" ] && cp "$SRC/.env.example" "$DEST/../.env.example"

# --- Sayfa dosyalari ---
# Auth
mkdir -p "$DEST/pages/auth/login" "$DEST/pages/auth/change-password" "$DEST/pages/auth/sessions"
cp "$SRC/app/auth/login/page.tsx" "$DEST/pages/auth/login/index.tsx"
cp "$SRC/app/auth/change-password/page.tsx" "$DEST/pages/auth/change-password/index.tsx"
cp "$SRC/app/auth/sessions/page.tsx" "$DEST/pages/auth/sessions/index.tsx"

# Admin
mkdir -p "$DEST/pages/admin/dashboard"
mkdir -p "$DEST/pages/admin/settings"
mkdir -p "$DEST/pages/admin/enrollment"
mkdir -p "$DEST/pages/admin/staff/personel-details"
mkdir -p "$DEST/pages/admin/students/student/profile"
mkdir -p "$DEST/pages/admin/students/advisors"
mkdir -p "$DEST/pages/admin/catalog/add"
mkdir -p "$DEST/pages/admin/catalog/edit"
mkdir -p "$DEST/pages/admin/catalog/schedule"
mkdir -p "$DEST/pages/admin/semester-courses/list"
mkdir -p "$DEST/pages/admin/meal/cafeterias"
mkdir -p "$DEST/pages/admin/meal/admin"
mkdir -p "$DEST/pages/admin/meal/menus"
mkdir -p "$DEST/pages/admin/meal/student"
mkdir -p "$DEST/pages/admin/system/time"
mkdir -p "$DEST/pages/admin/system/periods"
mkdir -p "$DEST/pages/admin/system/semesters"
mkdir -p "$DEST/pages/admin/system/audit"

cp "$SRC/app/(admin)/dashboard/page.tsx" "$DEST/pages/admin/dashboard/index.tsx"
cp "$SRC/app/(admin)/settings/page.tsx" "$DEST/pages/admin/settings/index.tsx"
cp "$SRC/app/(admin)/enrollment/page.tsx" "$DEST/pages/admin/enrollment/index.tsx"
cp "$SRC/app/(admin)/staff/page.tsx" "$DEST/pages/admin/staff/index.tsx"
cp "$SRC/app/(admin)/staff/personel-details/page.tsx" "$DEST/pages/admin/staff/personel-details/index.tsx"
cp "$SRC/app/(admin)/students/page.tsx" "$DEST/pages/admin/students/index.tsx"
cp "$SRC/app/(admin)/students/student/page.tsx" "$DEST/pages/admin/students/student/index.tsx"
cp "$SRC/app/(admin)/students/student/profile/page.tsx" "$DEST/pages/admin/students/student/profile/index.tsx"
cp "$SRC/app/(admin)/students/advisors/page.tsx" "$DEST/pages/admin/students/advisors/index.tsx"
cp "$SRC/app/(admin)/catalog/page.tsx" "$DEST/pages/admin/catalog/index.tsx"
cp "$SRC/app/(admin)/catalog/add/page.tsx" "$DEST/pages/admin/catalog/add/index.tsx"
cp "$SRC/app/(admin)/catalog/edit/page.tsx" "$DEST/pages/admin/catalog/edit/index.tsx"
cp "$SRC/app/(admin)/catalog/schedule/page.tsx" "$DEST/pages/admin/catalog/schedule/index.tsx"
cp "$SRC/app/(admin)/semester-courses/page.tsx" "$DEST/pages/admin/semester-courses/index.tsx"
cp "$SRC/app/(admin)/semester-courses/list/page.tsx" "$DEST/pages/admin/semester-courses/list/index.tsx"
cp "$SRC/app/(admin)/meal/cafeterias/page.tsx" "$DEST/pages/admin/meal/cafeterias/index.tsx"
cp "$SRC/app/(admin)/meal/admin/page.tsx" "$DEST/pages/admin/meal/admin/index.tsx"
cp "$SRC/app/(admin)/meal/menus/page.tsx" "$DEST/pages/admin/meal/menus/index.tsx"
cp "$SRC/app/(admin)/meal/student/page.tsx" "$DEST/pages/admin/meal/student/index.tsx"
cp "$SRC/app/(admin)/system/time/page.tsx" "$DEST/pages/admin/system/time/index.tsx"
cp "$SRC/app/(admin)/system/periods/page.tsx" "$DEST/pages/admin/system/periods/index.tsx"
cp "$SRC/app/(admin)/system/semesters/page.tsx" "$DEST/pages/admin/system/semesters/index.tsx"
cp "$SRC/app/(admin)/system/audit/page.tsx" "$DEST/pages/admin/system/audit/index.tsx"

# Teacher
mkdir -p "$DEST/pages/teacher/attendance/courseId/session/sessionId"
mkdir -p "$DEST/pages/teacher/enrollment"
mkdir -p "$DEST/pages/teacher/grades/courseId/slug"

cp "$SRC/app/(teacher)/teacher/attendance/page.tsx" "$DEST/pages/teacher/attendance/index.tsx"
cp "$SRC/app/(teacher)/teacher/attendance/[courseId]/page.tsx" "$DEST/pages/teacher/attendance/courseId/index.tsx"
cp "$SRC/app/(teacher)/teacher/attendance/[courseId]/session/[sessionId]/page.tsx" "$DEST/pages/teacher/attendance/courseId/session/sessionId/index.tsx"
cp "$SRC/app/(teacher)/teacher/enrollment/page.tsx" "$DEST/pages/teacher/enrollment/index.tsx"
cp "$SRC/app/(teacher)/teacher/grades/page.tsx" "$DEST/pages/teacher/grades/index.tsx"
cp "$SRC/app/(teacher)/teacher/grades/[courseId]/[slug]/page.tsx" "$DEST/pages/teacher/grades/courseId/slug/index.tsx"

# Student
mkdir -p "$DEST/pages/student/dashboard"
mkdir -p "$DEST/pages/student/attendance"
mkdir -p "$DEST/pages/student/enrollment/rejections"
mkdir -p "$DEST/pages/student/grades"
mkdir -p "$DEST/pages/student/cafeteria/menu"
mkdir -p "$DEST/pages/student/cafeteria/history"

cp "$SRC/app/(student)/student/dashboard/page.tsx" "$DEST/pages/student/dashboard/index.tsx"
cp "$SRC/app/(student)/student/attendance/page.tsx" "$DEST/pages/student/attendance/index.tsx"
cp "$SRC/app/(student)/student/enrollment/page.tsx" "$DEST/pages/student/enrollment/index.tsx"
cp "$SRC/app/(student)/student/enrollment/rejections/page.tsx" "$DEST/pages/student/enrollment/rejections/index.tsx"
cp "$SRC/app/(student)/student/grades/page.tsx" "$DEST/pages/student/grades/index.tsx"
cp "$SRC/app/(student)/student/cafeteria/page.tsx" "$DEST/pages/student/cafeteria/index.tsx"
cp "$SRC/app/(student)/student/cafeteria/menu/page.tsx" "$DEST/pages/student/cafeteria/menu/index.tsx"
cp "$SRC/app/(student)/student/cafeteria/history/page.tsx" "$DEST/pages/student/cafeteria/history/index.tsx"
```

---

## ADIM 2: CSS theme'i aktar

`old-frontend/app/globals.css` icindeki `:root { ... }` ve `.dark { ... }` bloklarini (satir 50-117) `frontend/src/index.css` dosyasina kopyala. Mevcut `:root` ve `.dark` bloklari varsa ONLARI SIL ve asagidakileri yapistir.

Ayrica `frontend/src/index.css` dosyasinin en ustune (diger importlarin yanina) su satiri ekle:
```css
@import "tw-animate-css";
```

---

## ADIM 3: Toplu find-replace islemleri

`frontend/src/` altindaki TUM `.ts` ve `.tsx` dosyalarinda asagidaki degisiklikleri uygula. SIRASI ONEMLI, yukaridan asagiya uygula.

### 3.1: `'use client'` ve `"use client"` satirlarini sil
Her dosyanin en ustundeki `'use client';` veya `"use client";` satirini (ve altindaki bos satiri) tamamen sil.

### 3.2: `next/link` importunu degistir
Asagidaki dosyalarda bu degisikligi yap:
- `src/components/layout/sidebar.tsx`
- `src/components/layout/student-sidebar.tsx`
- `src/components/layout/teacher-sidebar.tsx`
- `src/pages/admin/dashboard/index.tsx`
- `src/pages/admin/students/index.tsx`
- `src/pages/admin/students/advisors/index.tsx`
- `src/pages/teacher/attendance/index.tsx`
- `src/pages/teacher/attendance/courseId/index.tsx`
- `src/pages/teacher/attendance/courseId/session/sessionId/index.tsx`
- `src/pages/teacher/grades/courseId/slug/index.tsx`

```
ESKI: import Link from 'next/link';
YENI: import { Link } from 'react-router';
```

Ayrica bu dosyalardaki TUM `<Link href=` kullanumlarini `<Link to=` olarak degistir.
Ve TUM `href={` kullanumlarini (Link componenti icindeki) `to={` olarak degistir.

### 3.3: `useRouter` -> `useNavigate`
Asagidaki dosyalarda (login HARIC — login 3.5'te ayrica ele alinacak):
- `src/components/layout/header.tsx`
- `src/components/grades/assessment-select-dialog.tsx`
- `src/pages/auth/change-password/index.tsx`
- `src/pages/auth/sessions/index.tsx`
- `src/pages/admin/catalog/index.tsx`
- `src/pages/admin/catalog/add/index.tsx`
- `src/pages/admin/catalog/edit/index.tsx`
- `src/pages/admin/semester-courses/index.tsx`
- `src/pages/admin/settings/index.tsx`
- `src/pages/teacher/attendance/courseId/index.tsx`
- `src/pages/teacher/attendance/courseId/session/sessionId/index.tsx`

Degisiklikler:
```
ESKI: import { useRouter } from 'next/navigation';
YENI: import { useNavigate } from 'react-router';

ESKI: import { useRouter } from "next/navigation";
YENI: import { useNavigate } from 'react-router';

ESKI: const router = useRouter();
YENI: const navigate = useNavigate();

ESKI: router.push('/...
YENI: navigate('/...

ESKI: router.push("/...
YENI: navigate("/...

ESKI: router.back();
YENI: navigate(-1);

ESKI: router.replace('/...
YENI: navigate('/...', { replace: true });
```

NOT: Bazi dosyalarda `useRouter` baska hook'larla birlikte import ediliyor. Ornegin:
```
ESKI: import { useParams, useRouter } from 'next/navigation';
YENI: import { useParams, useNavigate } from 'react-router';
```
Bu durumda sadece `useRouter`'i `useNavigate` ile degistir, diger hook'lari birak.

### 3.4: `usePathname` -> `useLocation`
Asagidaki dosyalarda:
- `src/components/layout/sidebar.tsx`
- `src/components/layout/student-sidebar.tsx`
- `src/components/layout/teacher-sidebar.tsx`

```
ESKI: import { usePathname } from 'next/navigation';
YENI: import { useLocation } from 'react-router';

ESKI: const pathname = usePathname();
YENI: const { pathname } = useLocation();
```

### 3.5: `useRouter` + `useSearchParams` (sadece login sayfasi)
Sadece `src/pages/auth/login/index.tsx` dosyasinda. Bu dosya hem `useRouter` hem `useSearchParams` kullaniyor, hepsini tek seferde degistir:

```
ESKI: import { useRouter, useSearchParams } from "next/navigation";
YENI: import { useNavigate, useSearchParams } from 'react-router';

ESKI: const router = useRouter();
YENI: const navigate = useNavigate();

ESKI: const searchParams = useSearchParams();
YENI: const [searchParams] = useSearchParams();

ESKI: router.push(
YENI: navigate(
```
(Dosyadaki TUM `router.push(` cagrilarini `navigate(` olarak degistir.)

### 3.6: `useParams`
Asagidaki dosyalarda:
- `src/pages/teacher/attendance/courseId/index.tsx`
- `src/pages/teacher/attendance/courseId/session/sessionId/index.tsx`
- `src/pages/teacher/grades/courseId/slug/index.tsx`

`useParams` ayni isimle kalir, sadece import kaynagi degisir:
```
ESKI: import { useParams, useRouter } from 'next/navigation';
YENI: import { useParams, useNavigate } from 'react-router';
```
(Bu dosyalarda `useRouter` da var, ikisi birlikte degisecek.)

### 3.7: Environment variable
`src/lib/api-client.ts` dosyasinda:
```
ESKI: const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost';
YENI: const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost';
```

Ayrica `frontend/.env.local` dosyasinda (varsa):
```
ESKI: NEXT_PUBLIC_API_BASE_URL=...
YENI: VITE_API_BASE_URL=...
```

---

## ADIM 4: Layout componentlerini guncelle

### 4.1: `src/components/layout/admin-layout.tsx`
Dosyanin TAMAMINI su sekilde degistir:
```tsx
import { Outlet } from 'react-router';
import { Sidebar } from './sidebar';
import { Header } from './header';

export function AdminLayout() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 transition-colors">
      <Sidebar />
      <Header />
      <main className="ml-64 pt-16 min-h-screen">
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
```

### 4.2: `src/components/layout/student-layout.tsx`
Dosyanin TAMAMINI su sekilde degistir:
```tsx
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
```

### 4.3: `src/components/layout/teacher-layout.tsx`
Dosyanin TAMAMINI su sekilde degistir:
```tsx
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
```

---

## ADIM 5: Yeni dosyalar olustur

### 5.1: `src/components/auth-guard.tsx`
```tsx
import { Navigate, Outlet } from 'react-router';

interface AuthGuardProps {
  allowedRoles: string[];
}

export function AuthGuard({ allowedRoles }: AuthGuardProps) {
  const token = localStorage.getItem('access_token');
  const userStr = localStorage.getItem('user');

  if (!token || !userStr) {
    return <Navigate to="/auth/login" replace />;
  }

  try {
    const user = JSON.parse(userStr);
    if (!allowedRoles.includes(user.role)) {
      if (user.role === 'admin') return <Navigate to="/dashboard" replace />;
      if (user.role === 'teacher') return <Navigate to="/teacher/attendance" replace />;
      if (user.role === 'student') return <Navigate to="/student/dashboard" replace />;
      return <Navigate to="/auth/login" replace />;
    }
  } catch {
    return <Navigate to="/auth/login" replace />;
  }

  return <Outlet />;
}
```

### 5.2: `src/routes.tsx`
```tsx
import { Routes, Route, Navigate } from 'react-router';
import { AdminLayout } from '@/components/layout/admin-layout';
import { StudentLayout } from '@/components/layout/student-layout';
import { TeacherLayout } from '@/components/layout/teacher-layout';
import { AuthGuard } from '@/components/auth-guard';

// Auth
import LoginPage from '@/pages/auth/login';
import ChangePasswordPage from '@/pages/auth/change-password';
import SessionsPage from '@/pages/auth/sessions';

// Admin
import DashboardPage from '@/pages/admin/dashboard';
import SettingsPage from '@/pages/admin/settings';
import AdminEnrollmentPage from '@/pages/admin/enrollment';
import StaffPage from '@/pages/admin/staff';
import StaffDetailsPage from '@/pages/admin/staff/personel-details';
import StudentsPage from '@/pages/admin/students';
import StudentPage from '@/pages/admin/students/student';
import StudentProfilePage from '@/pages/admin/students/student/profile';
import AdvisorsPage from '@/pages/admin/students/advisors';
import CatalogPage from '@/pages/admin/catalog';
import CatalogAddPage from '@/pages/admin/catalog/add';
import CatalogEditPage from '@/pages/admin/catalog/edit';
import CatalogSchedulePage from '@/pages/admin/catalog/schedule';
import SemesterCoursesPage from '@/pages/admin/semester-courses';
import SemesterCoursesListPage from '@/pages/admin/semester-courses/list';
import CafeteriasPage from '@/pages/admin/meal/cafeterias';
import MealAdminPage from '@/pages/admin/meal/admin';
import MenusPage from '@/pages/admin/meal/menus';
import MealStudentPage from '@/pages/admin/meal/student';
import TimeSettingsPage from '@/pages/admin/system/time';
import PeriodsPage from '@/pages/admin/system/periods';
import SemestersPage from '@/pages/admin/system/semesters';
import AuditPage from '@/pages/admin/system/audit';

// Teacher
import TeacherAttendancePage from '@/pages/teacher/attendance';
import TeacherAttendanceCoursePage from '@/pages/teacher/attendance/courseId';
import TeacherAttendanceSessionPage from '@/pages/teacher/attendance/courseId/session/sessionId';
import TeacherEnrollmentPage from '@/pages/teacher/enrollment';
import TeacherGradesPage from '@/pages/teacher/grades';
import TeacherGradesCourseSlugPage from '@/pages/teacher/grades/courseId/slug';

// Student
import StudentDashboardPage from '@/pages/student/dashboard';
import StudentAttendancePage from '@/pages/student/attendance';
import StudentEnrollmentPage from '@/pages/student/enrollment';
import StudentEnrollmentRejectionsPage from '@/pages/student/enrollment/rejections';
import StudentGradesPage from '@/pages/student/grades';
import StudentCafeteriaPage from '@/pages/student/cafeteria';
import StudentCafeteriaMenuPage from '@/pages/student/cafeteria/menu';
import StudentCafeteriaHistoryPage from '@/pages/student/cafeteria/history';

export function AppRoutes() {
  return (
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
          <Route path="/meal/cafeterias" element={<CafeteriasPage />} />
          <Route path="/meal/admin" element={<MealAdminPage />} />
          <Route path="/meal/menus" element={<MenusPage />} />
          <Route path="/meal/student" element={<MealStudentPage />} />
          <Route path="/system/time" element={<TimeSettingsPage />} />
          <Route path="/system/periods" element={<PeriodsPage />} />
          <Route path="/system/semesters" element={<SemestersPage />} />
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

      <Route path="*" element={<Navigate to="/auth/login" replace />} />
    </Routes>
  );
}
```

### 5.3: `src/main.tsx` dosyasini guncelle
shadcn'in olusturdugu `src/main.tsx` dosyasinin TAMAMINI su sekilde degistir:
```tsx
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import { QueryProvider } from '@/components/providers/query-provider';
import { ThemeProvider } from '@/components/providers/theme-provider';
import { AppRoutes } from './routes';
import './index.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <QueryProvider>
        <ThemeProvider>
          <AppRoutes />
        </ThemeProvider>
      </QueryProvider>
    </BrowserRouter>
  </StrictMode>,
);
```

---

## ADIM 6: `index.html` guncelle

`frontend/index.html` dosyasinin TAMAMINI su sekilde degistir:
```html
<!DOCTYPE html>
<html lang="tr">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>MyDreamCampus - Kampus Yonetim Sistemi</title>
    <meta name="description" content="Dokuz Eylul Universitesi Kampus Yonetim Sistemi" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet" />
  </head>
  <body class="antialiased">
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

---

## ADIM 7: `vite.config.ts` guncelle

shadcn'in olusturdugu `vite.config.ts` dosyasina `server` blogu ekle (eger yoksa):

```ts
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost',
      changeOrigin: true,
    },
  },
},
```

---

## ADIM 8: TypeScript kontrolu

```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/frontend
bun tsc --noEmit 2>&1 | head -50
```

Hatalari duzelt. En yaygin sorunlar:
- Kalan `next/navigation` veya `next/link` importlari -> Adim 3'e don
- `@/` alias cozumleme hatasi -> tsconfig.json ve vite.config.ts'deki alias'i kontrol et
- `import.meta.env` tipi -> `src/vite-env.d.ts` dosyasinin varligini kontrol et

---

## ADIM 9: Calistir ve test et

```bash
cd /home/nautilus/Desktop/Playground/mydreamcampus/frontend
bun run dev
```

Tarayicida `http://localhost:3000` ac. Kontrol listesi:
- Login sayfasi aciliyor mu?
- Login sonrasi dogru sayfaya yonlendiriyor mu?
- Sidebar calisiyor mu?
- API cagrilari calisiyor mu?
- Dark mode calisiyor mu?

---

## ADIM 10: skills.md olustur

`frontend/skills.md` dosyasini olustur:

```markdown
# Frontend - React + Vite

Bu proje React + Vite ile yazilmistir. Next.js DEGILDIR.

## Kurallar

- Paket yoneticisi: `bun` (npm/npx KULLANMA, `bunx --bun` kullan)
- `'use client'` KULLANMA — Vite'da boyle bir sey yok
- `next/navigation`, `next/link`, `next/image`, `next/font` gibi Next.js modulleri KULLANMA

## Teknoloji

- **Build**: Vite
- **Routing**: react-router v7 (`import { Link, useNavigate, useLocation, useParams, useSearchParams } from 'react-router'`)
- **UI**: shadcn/ui (radix-vega style) + Tailwind CSS 4
- **State/Data**: @tanstack/react-query
- **HTTP**: ky
- **Icons**: lucide-react

## Routing Kurallari

- Link componenti: `<Link to="/path">` (`href` DEGIL, `to` kullan)
- Programmatic navigation: `const navigate = useNavigate()` sonra `navigate('/path')`
- Geri gitme: `navigate(-1)`
- Aktif route: `const { pathname } = useLocation()`
- URL parametreleri: `const { id } = useParams()`
- Query parametreleri: `const [searchParams] = useSearchParams()`

## Dosya Yapisi

- `src/pages/` — Sayfa componentleri
- `src/components/ui/` — shadcn componentleri
- `src/components/layout/` — Layout componentleri (Outlet kullanir)
- `src/components/providers/` — Context provider'lar
- `src/lib/api-client.ts` — ky HTTP client (tum API instance'lari)
- `src/lib/services/` — API servis fonksiyonlari
- `src/lib/types.ts` — TypeScript tipleri
- `src/lib/utils.ts` — Utility fonksiyonlari (cn helper)
- `src/routes.tsx` — Tum route tanimlari
- `src/main.tsx` — Entry point

## Alias

- `@/` -> `src/` (ornek: `import { Button } from '@/components/ui/button'`)

## Environment Variables

- Prefix: `VITE_` (ornek: `VITE_API_BASE_URL`)
- Erisim: `import.meta.env.VITE_API_BASE_URL` (`process.env` DEGIL)
```
