import { StudentLayout } from '@/components/layout/student-layout';

export default function StudentGroupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <StudentLayout>{children}</StudentLayout>;
}
