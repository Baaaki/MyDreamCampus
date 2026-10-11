import { useQuery } from "@tanstack/react-query"
import { AlertCircle, FileCheck, Loader2 } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { apiErrorMessage } from "@/lib/api-error"
import { getMyAttendance } from "@/lib/services/attendance-service"
import { semesterLabel } from "@/lib/services/enrollment-service"
import type {
  CourseAttendanceDetail,
  SessionTypeAttendance,
  WeeklyAttendanceRecord,
} from "@/lib/types"

const SESSION_LABELS: Record<WeeklyAttendanceRecord["session_type"], string> = {
  theory: "Teori",
  lab: "Lab",
}

const MARKED_VIA_LABELS: Record<WeeklyAttendanceRecord["marked_via"], string> =
  {
    qr_scan: "QR",
    manual: "Hoca",
    admin: "Yönetici",
  }

// Session dates are campus calendar days; parsing them as UTC would show
// the previous day west of Greenwich.
function formatSessionDate(date: string): string {
  const [y, m, d] = date.split("-").map(Number)
  if (!y || !m || !d) return date
  return new Date(y, m - 1, d).toLocaleDateString("tr-TR", {
    day: "numeric",
    month: "short",
  })
}

export default function StudentAttendancePage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["attendance", "my"],
    queryFn: () => getMyAttendance(),
  })

  const courses = data?.courses ?? []

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
          <FileCheck className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Yoklama
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            {data?.semester
              ? `${semesterLabel(data.semester)} dönemi yoklama durumunuz`
              : "Derslerinizin yoklama durumunu görüntüleyin"}
          </p>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 text-gray-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          Yoklama bilgileri yükleniyor...
        </div>
      ) : isError ? (
        <div className="flex items-center gap-2 text-destructive">
          <AlertCircle className="h-4 w-4" />
          {apiErrorMessage(error, "Yoklama bilgileri yüklenemedi")}
        </div>
      ) : courses.length === 0 ? (
        <Card>
          <CardContent>
            <p className="text-gray-500 dark:text-gray-400">
              Bu dönem kayıtlı olduğunuz bir ders bulunmuyor.
            </p>
          </CardContent>
        </Card>
      ) : (
        courses.map((course) => (
          <CourseAttendanceCard key={course.course_id} course={course} />
        ))
      )}
    </div>
  )
}

function CourseAttendanceCard({ course }: { course: CourseAttendanceDetail }) {
  const records = [...course.weekly_records].sort(
    (a, b) => a.week - b.week || a.session_type.localeCompare(b.session_type)
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {course.course_code} — {course.course_name}
        </CardTitle>
        {course.instructor && (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {course.instructor}
          </p>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {!course.theory && !course.lab ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Bu derste henüz yoklama alınmadı.
          </p>
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {course.theory && (
              <SessionTypeSummary label="Teori" stats={course.theory} />
            )}
            {course.lab && (
              <SessionTypeSummary label="Lab" stats={course.lab} />
            )}
          </div>
        )}

        {records.length > 0 && (
          <div>
            <p className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              Katıldığınız oturumlar
            </p>
            <ul className="divide-y rounded-md border text-sm dark:divide-gray-700 dark:border-gray-700">
              {records.map((r) => (
                <li
                  key={`${r.week}-${r.session_type}`}
                  className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2"
                >
                  <span className="font-medium">{r.week}. hafta</span>
                  <span>{SESSION_LABELS[r.session_type]}</span>
                  <span className="text-gray-500 dark:text-gray-400">
                    {formatSessionDate(r.date)}
                  </span>
                  <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
                    {MARKED_VIA_LABELS[r.marked_via] ?? r.marked_via}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function SessionTypeSummary({
  label,
  stats,
}: {
  label: string
  stats: SessionTypeAttendance
}) {
  return (
    <section
      aria-label={label}
      className="rounded-lg border p-3 dark:border-gray-700"
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="font-medium">{label}</span>
        {stats.passed ? (
          <Badge className="bg-green-600 hover:bg-green-600">Yeterli</Badge>
        ) : (
          <Badge variant="destructive">Sınırın altında</Badge>
        )}
      </div>
      <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
        <dt className="text-gray-500 dark:text-gray-400">Katıldığınız</dt>
        <dd>
          {stats.present_count} / {stats.total_sessions}
        </dd>
        <dt className="text-gray-500 dark:text-gray-400">Devamsızlık</dt>
        <dd>{stats.absent_count}</dd>
        <dt className="text-gray-500 dark:text-gray-400">Gereken en az</dt>
        <dd>{stats.min_required}</dd>
      </dl>
    </section>
  )
}
