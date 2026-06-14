import { TeacherLayout } from '@/components/layout/teacher-layout';

export default function TeacherGroupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <TeacherLayout>{children}</TeacherLayout>;
}
