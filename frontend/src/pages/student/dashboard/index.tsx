import { useEffect, useState, type ComponentType, type ReactNode } from "react"
import { Link } from "react-router"
import { useQuery, type UseQueryResult } from "@tanstack/react-query"
import { addDays, format } from "date-fns"
import { tr } from "date-fns/locale"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  AlertCircle,
  AlertTriangle,
  ArrowRight,
  BookOpen,
  CalendarDays,
  Clock,
  GraduationCap,
  ListChecks,
  Loader2,
  User,
  UtensilsCrossed,
} from "lucide-react"
import { studentApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import { TIME_SLOTS } from "@/lib/constants"
import { cn } from "@/lib/utils"
import type {
  CourseAttendanceDetail,
  EnrollmentProgramResponse,
  OpenSemester,
  Student,
} from "@/lib/types"
import {
  absenceWarnings,
  absencesLeft,
  getMyAttendance,
} from "@/lib/services/attendance-service"
import {
  currentProgram,
  enrollmentService,
  semesterLabel,
} from "@/lib/services/enrollment-service"
import { gradesService } from "@/lib/services/grades-service"
import {
  getMyReservations,
  toLocalISODate,
  type Reservation,
} from "@/lib/services/meal-service"
import {
  currentAndNext,
  dayBlocks,
  toMinutes,
  weekGrid,
  type ClassBlock,
  type SessionType,
  type UpcomingClass,
} from "./schedule"

const SESSION_LABELS: Record<SessionType, string> = {
  theory: "Teori",
  lab: "Lab",
}

const MEAL_TIME_LABELS: Record<Reservation["meal_time"], string> = {
  lunch: "Öğle",
  dinner: "Akşam",
}

// Where a meal sits among the day's classes; reservations carry no time.
const MEAL_SORT_MINUTES: Record<Reservation["meal_time"], number> = {
  lunch: toMinutes("12:10"),
  dinner: toMinutes("17:00"),
}

const UPCOMING_MEALS_SHOWN = 4

// Same palette as the enrollment page's weekly grid. Tailwind only ships
// classes it finds verbatim, so each colour is spelled out.
const COURSE_BORDER = [
  "border-blue-500",
  "border-green-500",
  "border-purple-500",
  "border-orange-500",
  "border-pink-500",
]
const COURSE_CELL = [
  "bg-blue-500 text-white",
  "bg-green-500 text-white",
  "bg-purple-500 text-white",
  "bg-orange-500 text-white",
  "bg-pink-500 text-white",
]

const WEEK_DAYS = ["Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma"]

// The class cards and the timeline follow the clock without a reload.
function useNow(intervalMs = 30_000): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), intervalMs)
    return () => clearInterval(id)
  }, [intervalMs])
  return now
}

function currentUserId(): string | null {
  try {
    const user = JSON.parse(localStorage.getItem("user") ?? "null")
    return typeof user?.id === "string" ? user.id : null
  } catch {
    return null
  }
}

function capitalize(text: string): string {
  return text.charAt(0).toLocaleUpperCase("tr-TR") + text.slice(1)
}

// Reservation dates are campus calendar days; parsing them as UTC would
// show the previous day west of Greenwich.
function parseDay(date: string): Date | null {
  const [y, m, d] = date.slice(0, 10).split("-").map(Number)
  return y && m && d ? new Date(y, m - 1, d) : null
}

function relativeDay(day: Date, now: Date): string {
  const key = toLocalISODate(day)
  if (key === toLocalISODate(now)) return "Bugün"
  if (key === toLocalISODate(addDays(now, 1))) return "Yarın"
  return capitalize(format(day, "EEEE", { locale: tr }))
}

// Meal days further out than tomorrow carry the date: "Pazartesi" alone
// cannot tell this week's from next week's.
function mealDay(day: Date, now: Date): string {
  const label = relativeDay(day, now)
  if (label === "Bugün" || label === "Yarın") return label
  return `${label} ${format(day, "d MMM", { locale: tr })}`
}

// The reservation list leaves cafeteria_name empty and nests the name.
function cafeteriaName(r: Reservation): string {
  return r.cafeteria_name || r.cafeteria?.name || ""
}

function greeting(now: Date): string {
  const hour = now.getHours()
  return hour < 12 ? "Günaydın" : hour < 18 ? "İyi Günler" : "İyi Akşamlar"
}

export default function StudentDashboardPage() {
  const now = useNow()
  const today = toLocalISODate(now)
  const userId = currentUserId()

  const student = useQuery({
    queryKey: ["students", userId],
    queryFn: () => studentApi.get(`${userId}`).json<Student>(),
    enabled: userId !== null,
  })
  const grades = useQuery({
    queryKey: ["grades", "my"],
    queryFn: () => gradesService.getMyGrades(),
  })
  const attendance = useQuery({
    queryKey: ["attendance", "my"],
    queryFn: () => getMyAttendance(),
  })
  const meals = useQuery({
    queryKey: ["meals", "reservations", "my", { from_date: today }],
    queryFn: () => getMyReservations({ from_date: today }),
  })
  const programs = useQuery({
    queryKey: ["enrollment", "programs", "approved"],
    queryFn: () => enrollmentService.getApprovedPrograms(),
  })
  const openSemesters = useQuery({
    queryKey: ["enrollment", "open-semesters"],
    queryFn: () => enrollmentService.getOpenSemesters(),
  })

  const program = currentProgram(programs.data ?? [], attendance.data?.semester)
  const todaysMeals = (meals.data?.reservations ?? []).filter(
    (r) => r.date.slice(0, 10) === today && r.status === "confirmed"
  )
  const upcomingMeals = (meals.data?.reservations ?? [])
    .filter(
      (r) =>
        r.status === "confirmed" && !r.is_used && r.date.slice(0, 10) > today
    )
    .sort(
      (a, b) =>
        a.date.localeCompare(b.date) ||
        MEAL_SORT_MINUTES[a.meal_time] - MEAL_SORT_MINUTES[b.meal_time]
    )
    .slice(0, UPCOMING_MEALS_SHOWN)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Öğrenci Portalı
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            {now.toLocaleDateString("tr-TR", {
              weekday: "long",
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <WelcomeCard query={student} now={now} />
        <ClassCard programs={programs} program={program} now={now} />

        <TodoCard
          attendance={attendance.data?.courses ?? []}
          openSemesters={openSemesters.data ?? []}
        />

        <SummaryCard
          title="Bugün"
          icon={CalendarDays}
          to="/student/attendance"
          linkLabel="Yoklama durumum"
          query={programs}
          errorMessage="Ders programı yüklenemedi"
        >
          <Timeline
            blocks={program ? dayBlocks(program.courses, now) : []}
            meals={todaysMeals}
            attendance={attendance.data?.courses ?? []}
            now={now}
          />
        </SummaryCard>

        <SummaryCard
          title="Devamsızlık"
          icon={AlertTriangle}
          to="/student/attendance"
          linkLabel="Yoklama durumum"
          query={attendance}
          errorMessage="Yoklama bilgileri yüklenemedi"
        >
          <AbsenceList courses={attendance.data?.courses ?? []} />
        </SummaryCard>

        <SummaryCard
          title="Akademik Durum"
          icon={GraduationCap}
          to="/student/grades"
          linkLabel="Notlarım"
          query={grades}
          errorMessage="Not bilgileri yüklenemedi"
        >
          {grades.data && grades.data.completed_courses?.length ? (
            <div className="flex gap-8">
              <Figure
                label="GNO"
                value={grades.data.cumulative_gpa.toFixed(2)}
              />
              <Figure label="Toplam kredi" value={grades.data.total_credits} />
              <Figure
                label="Bu dönem ders"
                value={grades.data.active_courses?.length ?? 0}
              />
            </div>
          ) : (
            <EmptyText>Henüz not ortalamanız oluşmadı.</EmptyText>
          )}
        </SummaryCard>

        <SummaryCard
          title="Yaklaşan Yemekler"
          icon={UtensilsCrossed}
          to="/student/cafeteria"
          linkLabel="Yemekhane"
          query={meals}
          errorMessage="Yemek rezervasyonları yüklenemedi"
        >
          {upcomingMeals.length > 0 ? (
            <ul className="space-y-2">
              {upcomingMeals.map((r) => {
                const day = parseDay(r.date)
                return (
                  <li
                    key={r.id}
                    className="flex items-center justify-between gap-3 rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-800"
                  >
                    <span className="font-medium text-gray-900 dark:text-white">
                      {day ? mealDay(day, now) : r.date} ·{" "}
                      {MEAL_TIME_LABELS[r.meal_time]}
                    </span>
                    <span className="text-gray-500 dark:text-gray-400">
                      {cafeteriaName(r)}
                      {r.menu_type === "vegan" && " · Vegan"}
                    </span>
                  </li>
                )
              })}
            </ul>
          ) : (
            <EmptyText>Yaklaşan yemek rezervasyonunuz yok.</EmptyText>
          )}
        </SummaryCard>
      </div>

      {program && program.courses.some((c) => c.schedule_sessions?.length) && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <BookOpen className="h-5 w-5 text-emerald-600" />
              Bu Hafta
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {semesterLabel(program.semester)} onaylı programınız ·{" "}
              {program.courses.length} ders
            </p>
            <WeekGrid program={program} now={now} />
            <CardLink to="/student/enrollment" label="Ders kaydı" />
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function WelcomeCard({
  query,
  now,
}: {
  query: UseQueryResult<Student>
  now: Date
}) {
  const s = query.data
  return (
    <Card className="border-0 bg-gradient-to-br from-emerald-500 to-emerald-600 text-white">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-white">
          <User className="h-5 w-5" />
          {greeting(now)}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="mb-2 text-2xl font-bold">
          {query.isLoading
            ? "Yükleniyor..."
            : s
              ? `Sayın ${s.first_name} ${s.last_name}`
              : "Öğrenci"}
        </p>
        {s && (
          <div className="space-y-1 text-sm text-emerald-100">
            <p>Öğrenci No: {s.student_number}</p>
            <p>
              {s.department}, {s.class_level}. sınıf
            </p>
            <p>{s.faculty}</p>
            {s.advisor_name && <p>Danışman: {s.advisor_name}</p>}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ClassCard({
  programs,
  program,
  now,
}: {
  programs: UseQueryResult<EnrollmentProgramResponse[]>
  program: EnrollmentProgramResponse | null
  now: Date
}) {
  const { current, next } = program
    ? currentAndNext(program.courses, now)
    : { current: null, next: null }

  return (
    <Card
      role="region"
      aria-label="Şu an ve sıradaki ders"
      className="border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white"
    >
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-white">
          <Clock className="h-5 w-5" />
          {current ? "Şu an derstesiniz" : "Sıradaki ders"}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {programs.isLoading ? (
          <p className="text-blue-100">Yükleniyor...</p>
        ) : programs.isError ? (
          <p className="text-blue-100">
            {apiErrorMessage(programs.error, "Ders programı yüklenemedi")}
          </p>
        ) : !program ? (
          <>
            <p className="text-lg font-medium">
              Onaylı bir ders programınız yok.
            </p>
            <Link
              to="/student/enrollment"
              className="inline-flex items-center gap-1 text-sm font-medium text-white hover:underline"
            >
              Ders kaydına git
              <ArrowRight className="h-4 w-4" />
            </Link>
          </>
        ) : current ? (
          <CurrentClass block={current} next={next} now={now} />
        ) : next ? (
          <NextClass next={next} now={now} />
        ) : (
          <p className="text-blue-100">Programınızda haftalık ders yok.</p>
        )}
      </CardContent>
    </Card>
  )
}

function CurrentClass({
  block,
  next,
  now,
}: {
  block: ClassBlock
  next: UpcomingClass | null
  now: Date
}) {
  const start = toMinutes(block.start)
  const end = toMinutes(block.end)
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const progress = Math.round(((nowMin - start) / (end - start)) * 100)
  return (
    <>
      <p className="text-2xl font-bold">{block.courseName}</p>
      <ClassMeta block={block} />
      <div
        className="h-2 overflow-hidden rounded-full bg-white/25"
        role="progressbar"
        aria-label="Dersin geçen kısmı"
        aria-valuenow={progress}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className="h-full rounded-full bg-white"
          style={{ width: `${progress}%` }}
        />
      </div>
      <div className="flex justify-between text-sm text-blue-100">
        <span>Başlangıç {block.start}</span>
        <span>Bitiş {block.end}</span>
      </div>
      {next && (
        <p className="border-t border-white/20 pt-3 text-sm text-blue-100">
          Sonraki:{" "}
          <span className="font-medium text-white">
            {next.block.courseName}
          </span>{" "}
          · {relativeDay(addDays(now, next.daysAhead), now)} {next.block.start}
        </p>
      )}
    </>
  )
}

function NextClass({ next, now }: { next: UpcomingClass; now: Date }) {
  return (
    <>
      <p className="text-2xl font-bold">{next.block.courseName}</p>
      <ClassMeta block={next.block} />
      <p className="text-lg font-medium">
        {relativeDay(addDays(now, next.daysAhead), now)} {next.block.start}
      </p>
    </>
  )
}

function ClassMeta({ block }: { block: ClassBlock }) {
  return (
    <p className="text-sm text-blue-100">
      {[block.courseCode, SESSION_LABELS[block.sessionType], block.instructor]
        .filter(Boolean)
        .join(" · ")}
    </p>
  )
}

interface Todo {
  key: string
  tone: "critical" | "warning" | "info"
  title: string
  detail?: string
  link: { to: string; label: string }
}

function TodoCard({
  attendance,
  openSemesters,
}: {
  attendance: CourseAttendanceDetail[]
  openSemesters: OpenSemester[]
}) {
  const names = new Map(attendance.map((c) => [c.course_id, c.course_name]))
  const todos: Todo[] = absenceWarnings(attendance).map((w) => ({
    key: `${w.course_id}-${w.session_type}`,
    tone: w.absences_left <= 0 ? "critical" : "warning",
    title: `${names.get(w.course_id) ?? w.course_code} · ${SESSION_LABELS[w.session_type]}`,
    detail:
      w.absences_left < 0
        ? "Devamsızlık sınırını aştınız."
        : w.absences_left === 0
          ? "Devamsızlık hakkınız bitti. Bir derse daha gelmezseniz sınırı aşarsınız."
          : "1 devamsızlık hakkınız kaldı.",
    link: { to: "/student/attendance", label: "Yoklamaya bak" },
  }))

  const open = openSemesters.find((s) => s.enrollment_open && !s.has_program)
  if (open) {
    todos.push({
      key: `enrollment-${open.semester}`,
      tone: "info",
      title: `${semesterLabel(open.semester)} ders kaydı açık.`,
      detail: `Son gün: ${format(new Date(open.period_end), "d MMMM HH:mm", { locale: tr })}.`,
      link: { to: "/student/enrollment", label: "Ders seç" },
    })
  }

  if (todos.length === 0) return null
  return (
    <Card className="md:col-span-2">
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
          <ListChecks className="h-5 w-5 text-emerald-600" />
          Yapmanız Gerekenler
        </CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-2">
          {todos.map((t) => (
            <li
              key={t.key}
              className={cn(
                "flex flex-wrap items-center justify-between gap-x-4 gap-y-1 rounded-lg p-3 text-sm",
                t.tone === "critical" && "bg-red-50 dark:bg-red-900/30",
                t.tone === "warning" && "bg-amber-50 dark:bg-amber-900/30",
                t.tone === "info" && "bg-emerald-50 dark:bg-emerald-900/30"
              )}
            >
              <span className="min-w-0">
                <span className="font-medium text-gray-900 dark:text-white">
                  {t.title}
                </span>{" "}
                {t.detail && (
                  <span
                    className={cn(
                      t.tone === "critical" && "text-red-700 dark:text-red-300",
                      t.tone === "warning" &&
                        "text-amber-700 dark:text-amber-300",
                      t.tone === "info" && "text-gray-600 dark:text-gray-300"
                    )}
                  >
                    {t.detail}
                  </span>
                )}
              </span>
              <CardLink to={t.link.to} label={t.link.label} />
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}

type TimelineItem =
  | { kind: "class"; block: ClassBlock; sortKey: number }
  | { kind: "meal"; meal: Reservation; sortKey: number }

function Timeline({
  blocks,
  meals,
  attendance,
  now,
}: {
  blocks: ClassBlock[]
  meals: Reservation[]
  attendance: CourseAttendanceDetail[]
  now: Date
}) {
  if (blocks.length === 0 && meals.length === 0) {
    return <EmptyText>Bugün dersiniz yok.</EmptyText>
  }

  const today = toLocalISODate(now)
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const byCode = new Map(attendance.map((c) => [c.course_code, c]))
  const items: TimelineItem[] = [
    ...blocks.map((block) => ({
      kind: "class" as const,
      block,
      sortKey: toMinutes(block.start),
    })),
    ...meals.map((meal) => ({
      kind: "meal" as const,
      meal,
      sortKey: MEAL_SORT_MINUTES[meal.meal_time],
    })),
  ].sort((a, b) => a.sortKey - b.sortKey)

  return (
    <ul className="space-y-2">
      {items.map((item) => {
        if (item.kind === "meal") {
          const { meal } = item
          return (
            <li
              key={meal.id}
              className={cn(
                "flex items-center justify-between gap-3 rounded-lg border-l-4 border-gray-300 bg-gray-50 p-3 text-sm dark:border-gray-600 dark:bg-gray-800",
                meal.is_used && "opacity-60"
              )}
            >
              <span className="font-medium text-gray-900 dark:text-white">
                {MEAL_TIME_LABELS[meal.meal_time]} yemeği
              </span>
              <span className="text-gray-500 dark:text-gray-400">
                {cafeteriaName(meal)}
                {meal.is_used && " · Kullanıldı"}
              </span>
            </li>
          )
        }

        const { block } = item
        const start = toMinutes(block.start)
        const end = toMinutes(block.end)
        const live = start <= nowMin && nowMin < end
        const past = end <= nowMin
        const record = byCode.get(block.courseCode)
        const attended = record?.weekly_records.some(
          (r) => r.date === today && r.session_type === block.sessionType
        )
        const missed = record?.missed_sessions.some(
          (r) => r.date === today && r.session_type === block.sessionType
        )
        return (
          <li
            key={`${block.courseCode}-${block.start}`}
            className={cn(
              "flex items-center justify-between gap-3 rounded-lg border-l-4 p-3 text-sm",
              COURSE_BORDER[block.colorIndex],
              live
                ? "bg-emerald-50 dark:bg-emerald-900/30"
                : "bg-gray-50 dark:bg-gray-800",
              past && "opacity-60"
            )}
          >
            <span className="min-w-0">
              <span className="block font-medium text-gray-900 dark:text-white">
                {block.courseName}
              </span>
              <span className="text-gray-500 dark:text-gray-400">
                {block.start}–{block.end} · {block.courseCode} ·{" "}
                {SESSION_LABELS[block.sessionType]}
              </span>
            </span>
            {attended ? (
              <span className="shrink-0 font-medium text-emerald-600">
                Yoklamaya katıldınız
              </span>
            ) : missed ? (
              <span className="shrink-0 font-medium text-destructive">
                Yoklamada yoktunuz
              </span>
            ) : live ? (
              <span className="shrink-0 font-medium text-emerald-600">
                Devam ediyor
              </span>
            ) : null}
          </li>
        )
      })}
    </ul>
  )
}

function AbsenceList({ courses }: { courses: CourseAttendanceDetail[] }) {
  const rows = courses.flatMap((course) =>
    (["theory", "lab"] as const).flatMap((type) => {
      const stats = course[type]
      if (!stats) return []
      const left = absencesLeft(type, stats, course.total_weeks)
      const allowed = absencesLeft(
        type,
        { ...stats, absent_count: 0 },
        course.total_weeks
      )
      return [{ course, type, used: stats.absent_count, left, allowed }]
    })
  )
  if (rows.length === 0) {
    return <EmptyText>Henüz yoklama alınmadı.</EmptyText>
  }
  rows.sort((a, b) => a.left - b.left)

  return (
    <ul className="space-y-3">
      {rows.map(({ course, type, used, left, allowed }) => {
        const tone = left <= 0 ? "critical" : left === 1 ? "warning" : "normal"
        return (
          <li key={`${course.course_id}-${type}`} className="space-y-1.5">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="font-medium text-gray-900 dark:text-white">
                {course.course_code} · {SESSION_LABELS[type]}
              </span>
              <span
                className={cn(
                  tone === "critical" && "font-medium text-destructive",
                  tone === "warning" &&
                    "font-medium text-amber-700 dark:text-amber-300",
                  tone === "normal" && "text-gray-500 dark:text-gray-400"
                )}
              >
                {left < 0
                  ? "Devamsızlık sınırı aşıldı"
                  : left === 0
                    ? "Devamsızlık hakkınız kalmadı"
                    : `${left} devamsızlık hakkınız kaldı`}
              </span>
            </div>
            {allowed > 0 && (
              <div
                className="grid gap-1"
                style={{ gridTemplateColumns: `repeat(${allowed}, 1fr)` }}
                role="img"
                aria-label={`${allowed} devamsızlık hakkından ${Math.min(used, allowed)} kullanıldı`}
              >
                {Array.from({ length: allowed }, (_, i) => (
                  <span
                    key={i}
                    className={cn(
                      "h-2 rounded-full",
                      i >= used
                        ? "bg-gray-200 dark:bg-gray-700"
                        : tone === "critical"
                          ? "bg-red-500"
                          : tone === "warning"
                            ? "bg-amber-500"
                            : "bg-emerald-500"
                    )}
                  />
                ))}
              </div>
            )}
          </li>
        )
      })}
    </ul>
  )
}

function WeekGrid({
  program,
  now,
}: {
  program: EnrollmentProgramResponse
  now: Date
}) {
  const grid = weekGrid(program.courses)
  const todayIndex = now.getDay()
  const slots = Object.entries(TIME_SLOTS).map(([slot, s]) => ({
    slot: Number(slot),
    ...s,
  }))
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[40rem] table-fixed border-collapse text-xs">
        <thead>
          <tr>
            <th className="w-24 border border-gray-200 bg-gray-50 p-2 text-left font-medium text-gray-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
              Saat
            </th>
            {WEEK_DAYS.map((d, i) => (
              <th
                key={d}
                className={cn(
                  "border border-gray-200 p-2 font-medium dark:border-gray-700",
                  i + 1 === todayIndex
                    ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                    : "bg-gray-50 text-gray-600 dark:bg-gray-800 dark:text-gray-300"
                )}
              >
                {d}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {slots.map(({ slot, label, time }) => (
            <tr key={slot}>
              <td className="border border-gray-200 p-2 text-gray-500 dark:border-gray-700 dark:text-gray-400">
                <span className="block font-medium text-gray-700 dark:text-gray-300">
                  {label}
                </span>
                {time}
              </td>
              {[1, 2, 3, 4, 5].map((day) => {
                const cell = grid.get(`${day}-${slot}`)
                return (
                  <td
                    key={day}
                    className={cn(
                      "border border-gray-200 p-2 text-center dark:border-gray-700",
                      cell
                        ? cn("font-medium", COURSE_CELL[cell.colorIndex])
                        : slot === 5 && "bg-gray-50 dark:bg-gray-800/50"
                    )}
                  >
                    {cell?.courseCode}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SummaryCard({
  title,
  icon: Icon,
  to,
  linkLabel,
  query,
  errorMessage,
  children,
}: {
  title: string
  icon: ComponentType<{ className?: string }>
  to: string
  linkLabel: string
  query: UseQueryResult<unknown>
  errorMessage: string
  children: ReactNode
}) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
          <Icon className="h-5 w-5 text-emerald-600" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Loader2 className="h-4 w-4 animate-spin" />
            Yükleniyor...
          </div>
        ) : query.isError ? (
          <div className="flex items-center gap-2 text-sm text-destructive">
            <AlertCircle className="h-4 w-4" />
            {apiErrorMessage(query.error, errorMessage)}
          </div>
        ) : (
          children
        )}
        <CardLink to={to} label={linkLabel} />
      </CardContent>
    </Card>
  )
}

function CardLink({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-1 text-sm font-medium text-emerald-600 hover:underline"
    >
      {label}
      <ArrowRight className="h-4 w-4" />
    </Link>
  )
}

function Figure({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
      <p className="text-3xl font-bold text-gray-900 dark:text-white">
        {value}
      </p>
    </div>
  )
}

function EmptyText({ children }: { children: ReactNode }) {
  return <p className="text-sm text-gray-500 dark:text-gray-400">{children}</p>
}
