import { useEffect, useState, type ReactNode } from "react"
import { Link } from "react-router"
import { useQuery, type UseQueryResult } from "@tanstack/react-query"
import { addDays, format } from "date-fns"
import { tr } from "date-fns/locale"
import { AlertCircle, Loader2 } from "lucide-react"
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
  lunch: "öğle",
  dinner: "akşam",
}

// Where a meal sits among the day's classes; reservations carry no time.
const MEAL_SORT_MINUTES: Record<Reservation["meal_time"], number> = {
  lunch: toMinutes("12:10"),
  dinner: toMinutes("17:00"),
}

const UPCOMING_MEALS_SHOWN = 3

// Tailwind only ships classes it finds verbatim, so each course colour is
// spelled out instead of interpolated.
const COURSE_BORDER = [
  "border-course-1",
  "border-course-2",
  "border-course-3",
  "border-course-4",
  "border-course-5",
]
const COURSE_CELL = [
  "bg-course-1/15 text-course-1",
  "bg-course-2/15 text-course-2",
  "bg-course-3/15 text-course-3",
  "bg-course-4/15 text-course-4",
  "bg-course-5/15 text-course-5",
]

const WEEK_DAYS = ["Pzt", "Sal", "Çar", "Per", "Cum"]

// The "now" card and the timeline follow the clock without a reload.
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
  const confirmedMeals = (meals.data?.reservations ?? [])
    .filter((r) => r.status === "confirmed" && !r.is_used)
    .sort(
      (a, b) =>
        a.date.localeCompare(b.date) ||
        MEAL_SORT_MINUTES[a.meal_time] - MEAL_SORT_MINUTES[b.meal_time]
    )
  const todaysMeals = (meals.data?.reservations ?? []).filter(
    (r) => r.date.slice(0, 10) === today && r.status === "confirmed"
  )
  const upcomingMeals = confirmedMeals
    .filter((r) => r.date.slice(0, 10) > today)
    .slice(0, UPCOMING_MEALS_SHOWN)

  return (
    <div className="flex max-w-6xl flex-col gap-8">
      <Header student={student.data} now={now} />

      <NowSection programs={programs} program={program} now={now} />

      <TodoSection
        attendance={attendance.data?.courses ?? []}
        openSemesters={openSemesters.data ?? []}
      />

      <div className="grid items-start gap-8 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
        <Section title="Bugün">
          <QueryState query={programs} errorMessage="Ders programı yüklenemedi">
            <Timeline
              blocks={program ? dayBlocks(program.courses, now) : []}
              meals={todaysMeals}
              attendance={attendance.data?.courses ?? []}
              now={now}
            />
          </QueryState>
        </Section>

        <div className="flex min-w-0 flex-col gap-8">
          <Section
            title="Devamsızlık"
            link={{ to: "/student/attendance", label: "Tümü" }}
          >
            <QueryState
              query={attendance}
              errorMessage="Yoklama bilgileri yüklenemedi"
            >
              <AbsenceList courses={attendance.data?.courses ?? []} />
            </QueryState>
          </Section>

          <Section
            title="Notlar"
            link={{ to: "/student/grades", label: "Notlarım" }}
          >
            <QueryState query={grades} errorMessage="Not bilgileri yüklenemedi">
              {grades.data && grades.data.completed_courses?.length ? (
                <div className="flex flex-wrap items-end gap-7 rounded-xl bg-card p-5">
                  <div>
                    <p className="text-5xl leading-none font-extrabold tracking-tighter">
                      {grades.data.cumulative_gpa.toFixed(2)}
                    </p>
                    <p className="mt-1.5 text-sm text-muted-foreground">
                      Genel not ortalaması
                    </p>
                  </div>
                  <div className="flex gap-6">
                    <Figure
                      value={grades.data.total_credits}
                      label="Toplam kredi"
                    />
                    <Figure
                      value={grades.data.active_courses?.length ?? 0}
                      label="Bu dönem ders"
                    />
                  </div>
                </div>
              ) : (
                <Empty>Henüz not ortalamanız oluşmadı.</Empty>
              )}
            </QueryState>
          </Section>

          <Section
            title="Yemek"
            link={{ to: "/student/cafeteria", label: "Rezervasyon yap" }}
          >
            <QueryState
              query={meals}
              errorMessage="Yemek rezervasyonları yüklenemedi"
            >
              {upcomingMeals.length > 0 ? (
                <ul className="divide-y rounded-xl bg-card py-1.5">
                  {upcomingMeals.map((r) => {
                    const day = parseDay(r.date)
                    return (
                      <li
                        key={r.id}
                        className="grid grid-cols-[9rem_minmax(0,1fr)] gap-3.5 px-4.5 py-2.5 text-sm"
                      >
                        <span className="font-bold">
                          {day ? mealDay(day, now) : r.date}{" "}
                          {MEAL_TIME_LABELS[r.meal_time]}
                        </span>
                        <span>
                          {cafeteriaName(r)}
                          {r.menu_type === "vegan" && (
                            <span className="text-muted-foreground">
                              {" "}
                              · Vegan
                            </span>
                          )}
                        </span>
                      </li>
                    )
                  })}
                </ul>
              ) : (
                <Empty>Yaklaşan yemek rezervasyonunuz yok.</Empty>
              )}
            </QueryState>
          </Section>
        </div>
      </div>

      {program && program.courses.some((c) => c.schedule_sessions?.length) && (
        <Section
          title="Bu hafta"
          link={{ to: "/student/enrollment", label: "Ders kaydı" }}
        >
          <WeekGrid program={program} now={now} />
        </Section>
      )}
    </div>
  )
}

function Header({ student, now }: { student?: Student; now: Date }) {
  const details = student
    ? [
        student.student_number,
        `${student.department}, ${student.class_level}. sınıf`,
      ]
    : []
  return (
    <header className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1">
      <h1 className="text-2xl font-bold tracking-tight">
        {student ? `${student.first_name} ${student.last_name}` : "Özet"}
      </h1>
      <p className="text-sm text-muted-foreground">
        {[
          ...details,
          capitalize(format(now, "EEEE, d MMMM", { locale: tr })),
        ].join(" · ")}
      </p>
    </header>
  )
}

function NowSection({
  programs,
  program,
  now,
}: {
  programs: UseQueryResult<EnrollmentProgramResponse[]>
  program: EnrollmentProgramResponse | null
  now: Date
}) {
  if (programs.isLoading || programs.isError) return null
  if (!program) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-card p-6">
        <p className="font-semibold">Onaylı bir ders programınız yok.</p>
        <Link
          to="/student/enrollment"
          className="text-sm font-semibold text-primary hover:underline"
        >
          Ders kaydına git
        </Link>
      </div>
    )
  }

  const { current, next } = currentAndNext(program.courses, now)
  if (!current && !next) return null

  return (
    <section
      aria-label="Şu an ve sıradaki ders"
      className={cn(
        "grid gap-3.5",
        current && next && "md:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]"
      )}
    >
      {current ? (
        <CurrentClass block={current} now={now} />
      ) : (
        next && <NextClass next={next} now={now} filled />
      )}
      {current && next && <NextClass next={next} now={now} />}
    </section>
  )
}

function CurrentClass({ block, now }: { block: ClassBlock; now: Date }) {
  const start = toMinutes(block.start)
  const end = toMinutes(block.end)
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const progress = Math.round(((nowMin - start) / (end - start)) * 100)
  return (
    <div className="flex flex-col gap-3.5 rounded-2xl bg-primary p-6 text-primary-foreground">
      <p className="text-sm font-semibold opacity-80">Şu an derstesiniz</p>
      <h2 className="text-3xl leading-tight font-extrabold tracking-tight text-balance md:text-4xl">
        {block.courseName}
      </h2>
      <ClassMeta block={block} className="opacity-90" />
      <div
        className="h-1.5 overflow-hidden rounded-full bg-primary-foreground/25"
        role="progressbar"
        aria-label="Dersin geçen kısmı"
        aria-valuenow={progress}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className="h-full rounded-full bg-primary-foreground"
          style={{ width: `${progress}%` }}
        />
      </div>
      <div className="flex justify-between text-sm opacity-85">
        <span>Başlangıç {block.start}</span>
        <span>Bitiş {block.end}</span>
      </div>
    </div>
  )
}

function NextClass({
  next,
  now,
  filled = false,
}: {
  next: UpcomingClass
  now: Date
  filled?: boolean
}) {
  const day = relativeDay(addDays(now, next.daysAhead), now)
  return (
    <div
      className={cn(
        "flex flex-col gap-1.5 rounded-2xl p-6",
        filled ? "bg-primary text-primary-foreground" : "bg-card"
      )}
    >
      <p
        className={cn(
          "text-sm font-semibold",
          filled ? "opacity-80" : "text-muted-foreground"
        )}
      >
        Sıradaki ders
      </p>
      <h2
        className={cn(
          "leading-tight font-bold tracking-tight text-balance",
          filled ? "text-3xl font-extrabold md:text-4xl" : "text-xl"
        )}
      >
        {next.block.courseName}
      </h2>
      <ClassMeta
        block={next.block}
        className={filled ? "opacity-90" : "text-muted-foreground"}
      />
      <p className="mt-auto pt-3 font-bold">
        {day} {next.block.start}
      </p>
    </div>
  )
}

function ClassMeta({
  block,
  className,
}: {
  block: ClassBlock
  className?: string
}) {
  return (
    <p className={cn("flex flex-wrap gap-x-4 gap-y-1 text-sm", className)}>
      <span>{block.courseCode}</span>
      <span>{SESSION_LABELS[block.sessionType]}</span>
      {block.instructor && <span>{block.instructor}</span>}
    </p>
  )
}

interface Todo {
  key: string
  tone: "critical" | "info"
  title: string
  detail?: string
  link: { to: string; label: string }
}

function TodoSection({
  attendance,
  openSemesters,
}: {
  attendance: CourseAttendanceDetail[]
  openSemesters: OpenSemester[]
}) {
  const names = new Map(attendance.map((c) => [c.course_id, c.course_name]))
  const todos: Todo[] = absenceWarnings(attendance).map((w) => ({
    key: `${w.course_id}-${w.session_type}`,
    tone: w.absences_left <= 0 ? "critical" : "info",
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
    <Section title="Yapmanız gerekenler">
      <ul className="flex flex-col gap-2">
        {todos.map((t) => (
          <li
            key={t.key}
            className="grid grid-cols-[0.625rem_minmax(0,1fr)] items-center gap-x-3.5 gap-y-1 rounded-xl bg-card px-4.5 py-3.5 sm:grid-cols-[0.625rem_minmax(0,1fr)_auto]"
          >
            <span
              aria-hidden
              className={cn(
                "size-2.5 rounded-full",
                t.tone === "critical" ? "bg-destructive" : "bg-primary"
              )}
            />
            <p className="text-sm">
              <strong className="font-bold">{t.title}</strong>{" "}
              {t.detail && (
                <span className="text-muted-foreground">{t.detail}</span>
              )}
            </p>
            <Link
              to={t.link.to}
              className="col-start-2 text-sm font-semibold text-primary hover:underline sm:col-start-3"
            >
              {t.link.label}
            </Link>
          </li>
        ))}
      </ul>
    </Section>
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
    return <Empty>Bugün dersiniz yok.</Empty>
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
    <ol className="rounded-xl bg-card py-2">
      {items.map((item) => {
        if (item.kind === "meal") {
          const { meal } = item
          return (
            <li
              key={meal.id}
              className={cn(
                "grid grid-cols-[4rem_minmax(0,1fr)] gap-3.5 px-4.5 py-2",
                meal.is_used && "opacity-60"
              )}
            >
              <span className="pt-0.5 text-sm font-semibold">
                {capitalize(MEAL_TIME_LABELS[meal.meal_time])}
              </span>
              <span className="flex flex-col border-l-[3px] border-border pl-3.5 text-sm">
                <strong className="font-bold">
                  {meal.meal_time === "lunch" ? "Öğle yemeği" : "Akşam yemeği"}
                </strong>
                <span className="text-muted-foreground">
                  {cafeteriaName(meal)}
                  {meal.is_used && " · Kullanıldı"}
                </span>
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
              "grid grid-cols-[4rem_minmax(0,1fr)] gap-3.5 px-4.5 py-2",
              live && "bg-primary/10"
            )}
          >
            <span className="pt-0.5 text-sm text-muted-foreground">
              <b className="block font-semibold text-foreground">
                {block.start}
              </b>
              {block.end}
            </span>
            <span
              className={cn(
                "flex flex-col border-l-[3px] pl-3.5 text-sm",
                COURSE_BORDER[block.colorIndex],
                past && "opacity-60"
              )}
            >
              <strong className="text-[0.95rem] font-bold">
                {block.courseName}
              </strong>
              <span className="text-muted-foreground">
                {block.courseCode} · {SESSION_LABELS[block.sessionType]}
                {attended && (
                  <span className="font-semibold text-primary">
                    {" "}
                    · Yoklamaya katıldınız
                  </span>
                )}
                {missed && (
                  <span className="font-semibold text-destructive">
                    {" "}
                    · Yoklamada yoktunuz
                  </span>
                )}
                {live && !attended && (
                  <span className="font-semibold text-primary">
                    {" "}
                    · Devam ediyor
                  </span>
                )}
              </span>
            </span>
          </li>
        )
      })}
    </ol>
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
    return <Empty>Henüz yoklama alınmadı.</Empty>
  }
  rows.sort((a, b) => a.left - b.left)

  return (
    <ul className="divide-y rounded-xl bg-card py-1.5">
      {rows.map(({ course, type, used, left, allowed }) => {
        const tone = left <= 0 ? "critical" : left === 1 ? "warning" : "normal"
        return (
          <li
            key={`${course.course_id}-${type}`}
            className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3.5 gap-y-1.5 px-4.5 py-3"
          >
            <span className="min-w-0 text-sm font-semibold">
              {course.course_name}{" "}
              <span className="font-normal text-muted-foreground">
                {SESSION_LABELS[type].toLocaleLowerCase("tr-TR")}
              </span>
            </span>
            <span
              className={cn(
                "text-sm font-bold",
                tone === "critical" && "text-destructive",
                tone === "warning" && "text-warning",
                tone === "normal" && "text-muted-foreground"
              )}
            >
              {left < 0
                ? "Sınır aşıldı"
                : left === 0
                  ? "Hak kalmadı"
                  : `${left} hak`}
            </span>
            {allowed > 0 && (
              <span
                className="col-span-2 grid gap-0.5"
                style={{ gridTemplateColumns: `repeat(${allowed}, 1fr)` }}
                role="img"
                aria-label={`${allowed} devamsızlık hakkından ${Math.min(used, allowed)} kullanıldı`}
              >
                {Array.from({ length: allowed }, (_, i) => (
                  <i
                    key={i}
                    className={cn(
                      "h-1.5 rounded-full",
                      i >= used
                        ? "bg-muted"
                        : tone === "critical"
                          ? "bg-destructive"
                          : tone === "warning"
                            ? "bg-warning"
                            : "bg-muted-foreground"
                    )}
                  />
                ))}
              </span>
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
    start: s.time.split("-")[0],
  }))
  return (
    <div className="overflow-x-auto">
      <div className="grid min-w-[36rem] grid-cols-[3.25rem_repeat(5,minmax(6rem,1fr))] gap-0.5 text-xs">
        <span />
        {WEEK_DAYS.map((d, i) => (
          <span
            key={d}
            className={cn(
              "px-2 pb-1 font-bold",
              i + 1 === todayIndex ? "text-primary" : "text-muted-foreground"
            )}
          >
            {d}
          </span>
        ))}
        {slots.map(({ slot, start }) => (
          <WeekRow key={slot} slot={slot} start={start} grid={grid} />
        ))}
      </div>
    </div>
  )
}

function WeekRow({
  slot,
  start,
  grid,
}: {
  slot: number
  start: string
  grid: ReturnType<typeof weekGrid>
}) {
  // Slot 5 is the lunch break: no class is scheduled there.
  const lunch = slot === 5
  return (
    <>
      <span className="pt-1.5 text-muted-foreground">{start}</span>
      {[1, 2, 3, 4, 5].map((day) => {
        const cell = grid.get(`${day}-${slot}`)
        return (
          <span
            key={day}
            className={cn(
              "min-h-7 rounded-md px-2 py-1.5",
              cell
                ? cn("font-bold", COURSE_CELL[cell.colorIndex])
                : !lunch && "bg-card"
            )}
          >
            {cell?.isFirst && cell.courseCode}
          </span>
        )
      })}
    </>
  )
}

function Section({
  title,
  link,
  children,
}: {
  title: string
  link?: { to: string; label: string }
  children: ReactNode
}) {
  return (
    <section className="min-w-0">
      <div className="mb-3 flex items-baseline justify-between gap-3">
        <h2 className="text-[0.95rem] font-bold tracking-tight">{title}</h2>
        {link && (
          <Link
            to={link.to}
            className="text-sm font-semibold text-primary hover:underline"
          >
            {link.label}
          </Link>
        )}
      </div>
      {children}
    </section>
  )
}

function QueryState({
  query,
  errorMessage,
  children,
}: {
  query: UseQueryResult<unknown>
  errorMessage: string
  children: ReactNode
}) {
  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 rounded-xl bg-card px-4.5 py-4 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Yükleniyor...
      </div>
    )
  }
  if (query.isError) {
    return (
      <div className="flex items-center gap-2 rounded-xl bg-card px-4.5 py-4 text-sm text-destructive">
        <AlertCircle className="size-4" />
        {apiErrorMessage(query.error, errorMessage)}
      </div>
    )
  }
  return children
}

function Figure({ value, label }: { value: number; label: string }) {
  return (
    <div>
      <p className="text-xl font-bold tracking-tight">{value}</p>
      <p className="text-sm text-muted-foreground">{label}</p>
    </div>
  )
}

function Empty({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-xl bg-card px-4.5 py-4 text-sm text-muted-foreground">
      {children}
    </p>
  )
}
