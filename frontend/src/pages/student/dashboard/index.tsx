import { useEffect, useState, type ComponentType, type ReactNode } from "react"
import { Link } from "react-router"
import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  User,
  Quote,
  AlertCircle,
  AlertTriangle,
  ArrowRight,
  BookOpen,
  GraduationCap,
  Loader2,
  UtensilsCrossed,
} from "lucide-react"
import { studentApi } from "@/lib/api-client"
import { apiErrorMessage } from "@/lib/api-error"
import {
  absenceWarnings,
  getMyAttendance,
  type AbsenceWarning,
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

interface StudentInfo {
  name: string
  studentId: string
  department: string
  faculty: string
  advisorName?: string
}

const quotes = [
  {
    text: "Başarı, her gün tekrarlanan küçük çabaların toplamıdır.",
    author: "Robert Collier",
  },
  {
    text: "Eğitim en güçlü silahtır. Dünyayı değiştirmek için kullanabilirsiniz.",
    author: "Nelson Mandela",
  },
  { text: "Öğrenmenin sınırı yoktur.", author: "Konfüçyüs" },
  { text: "Bugün yapabileceğini yarına bırakma.", author: "Benjamin Franklin" },
  { text: "Bilgi güçtür.", author: "Francis Bacon" },
  {
    text: "Başarısızlık, başarıya giden yolda sadece bir duraktır.",
    author: "Zig Ziglar",
  },
  {
    text: "Hayatta en hakiki mürşit ilimdir.",
    author: "Mustafa Kemal Atatürk",
  },
  { text: "Gelecek, bugünden başlar.", author: "Malcolm X" },
  { text: "Öğrenmek, bir hazineye sahip olmaktır.", author: "Çin Atasözü" },
  { text: "Azim ve kararlılık her şeyi başarır.", author: "Benjamin Disraeli" },
  { text: "Düşünceleriniz kaderinizi belirler.", author: "Lao Tzu" },
  { text: "Başlamak, bitirmenin yarısıdır.", author: "Aristoteles" },
]

const SESSION_LABELS: Record<AbsenceWarning["session_type"], string> = {
  theory: "Teori",
  lab: "Lab",
}

const MEAL_TIME_LABELS: Record<Reservation["meal_time"], string> = {
  lunch: "Öğle",
  dinner: "Akşam",
}

const UPCOMING_MEALS_SHOWN = 4

function absenceWarningText(left: number): string {
  if (left < 0) return "Devamsızlık sınırı aşıldı"
  if (left === 0) return "Devamsızlık hakkınız kalmadı"
  return `${left} devamsızlık hakkınız kaldı`
}

// Reservation dates are campus calendar days; parsing them as UTC would
// show the previous day west of Greenwich.
function formatMealDay(date: string): string {
  const [y, m, d] = date.slice(0, 10).split("-").map(Number)
  if (!y || !m || !d) return date
  return new Date(y, m - 1, d).toLocaleDateString("tr-TR", {
    weekday: "short",
    day: "numeric",
    month: "short",
  })
}

export default function StudentDashboardPage() {
  const [studentInfo, setStudentInfo] = useState<StudentInfo>({
    name: "",
    studentId: "",
    department: "",
    faculty: "",
  })
  const [loading, setLoading] = useState(true)
  const [currentQuote, setCurrentQuote] = useState(quotes[0])

  useEffect(() => {
    const fetchStudentInfo = async () => {
      try {
        // Get user from localStorage
        const userStr = localStorage.getItem("user")
        if (userStr) {
          const user = JSON.parse(userStr)
          // Fetch student details from API using user ID
          const response = await studentApi.get(`${user.id}`).json<{
            id: string
            student_number: string
            first_name: string
            last_name: string
            email: string
            faculty: string
            department: string
            advisor_name?: string
          }>()

          setStudentInfo({
            name: `${response.first_name} ${response.last_name}`,
            studentId: response.student_number,
            department: response.department,
            faculty: response.faculty,
            advisorName: response.advisor_name,
          })
        }
      } catch (error) {
        console.error("Failed to fetch student info:", error)
        // Fallback to localStorage user info
        const userStr = localStorage.getItem("user")
        if (userStr) {
          const user = JSON.parse(userStr)
          setStudentInfo({
            name: user.email?.split("@")[0] || "Öğrenci",
            studentId: "-",
            department: user.department || "-",
            faculty: "-",
          })
        }
      } finally {
        setLoading(false)
      }
    }

    fetchStudentInfo()
  }, [])

  // Rotate quotes every 5 minutes
  useEffect(() => {
    const getRandomQuote = () => {
      const randomIndex = Math.floor(Math.random() * quotes.length)
      setCurrentQuote(quotes[randomIndex])
    }

    // Set initial random quote
    getRandomQuote()

    // Change quote every 5 minutes (300000ms)
    const interval = setInterval(getRandomQuote, 5 * 60 * 1000)

    return () => clearInterval(interval)
  }, [])

  const today = toLocalISODate(new Date())
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

  const warnings = absenceWarnings(attendance.data?.courses ?? [])
  const upcomingMeals = (meals.data?.reservations ?? [])
    .filter((r) => r.status === "confirmed" && !r.is_used)
    .sort((a, b) => a.date.localeCompare(b.date))
    .slice(0, UPCOMING_MEALS_SHOWN)
  const program = currentProgram(programs.data ?? [], attendance.data?.semester)

  const currentHour = new Date().getHours()
  const greeting =
    currentHour < 12
      ? "Günaydın"
      : currentHour < 18
        ? "İyi Günler"
        : "İyi Akşamlar"

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Öğrenci Portalı
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            {new Date().toLocaleDateString("tr-TR", {
              weekday: "long",
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        </div>
      </div>

      {/* Grid Layout */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        {/* Merhaba - Welcome Card */}
        <Card className="border-0 bg-gradient-to-br from-emerald-500 to-emerald-600 text-white">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-white">
              <User className="h-5 w-5" />
              {greeting}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-2 text-2xl font-bold">
              {loading ? "Yükleniyor..." : `Sayın ${studentInfo.name}`}
            </p>
            <div className="space-y-1 text-sm text-emerald-100">
              <p>Öğrenci No: {studentInfo.studentId || "-"}</p>
              <p>{studentInfo.department || "-"}</p>
              <p>{studentInfo.faculty || "-"}</p>
              {studentInfo.advisorName && (
                <p>Danışman: {studentInfo.advisorName}</p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Günün Sözü - Quote of the Day */}
        <Card className="border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-white">
              <Quote className="h-5 w-5" />
              Günün Sözü
            </CardTitle>
          </CardHeader>
          <CardContent>
            <blockquote className="mb-3 text-lg italic">
              "{currentQuote.text}"
            </blockquote>
            <p className="text-right font-medium text-blue-100">
              — {currentQuote.author}
            </p>
          </CardContent>
        </Card>

        <SummaryCard
          title="Akademik Durum"
          icon={GraduationCap}
          to="/student/grades"
          linkLabel="Notlarım"
          isLoading={grades.isLoading}
          error={grades.error}
          errorMessage="Not bilgileri yüklenemedi"
        >
          {grades.data && grades.data.completed_courses.length > 0 ? (
            <div className="flex gap-8">
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">GNO</p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white">
                  {grades.data.cumulative_gpa.toFixed(2)}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Toplam kredi
                </p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white">
                  {grades.data.total_credits}
                </p>
              </div>
            </div>
          ) : (
            <EmptyText>Henüz not ortalamanız oluşmadı.</EmptyText>
          )}
        </SummaryCard>

        <SummaryCard
          title="Devamsızlık"
          icon={AlertTriangle}
          to="/student/attendance"
          linkLabel="Yoklama durumum"
          isLoading={attendance.isLoading}
          error={attendance.error}
          errorMessage="Yoklama bilgileri yüklenemedi"
        >
          {warnings.length > 0 ? (
            <ul className="space-y-2">
              {warnings.map((w) => (
                <li
                  key={`${w.course_id}-${w.session_type}`}
                  className="flex items-center justify-between gap-3 rounded-lg bg-amber-50 p-3 text-sm dark:bg-amber-900/30"
                >
                  <span className="font-medium text-gray-900 dark:text-white">
                    {w.course_code} · {SESSION_LABELS[w.session_type]}
                  </span>
                  <span
                    className={
                      w.absences_left < 0
                        ? "text-destructive"
                        : "text-amber-700 dark:text-amber-300"
                    }
                  >
                    {absenceWarningText(w.absences_left)}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyText>Devamsızlık sınırına yaklaşan dersiniz yok.</EmptyText>
          )}
        </SummaryCard>

        <SummaryCard
          title="Yaklaşan Yemekler"
          icon={UtensilsCrossed}
          to="/student/cafeteria"
          linkLabel="Yemekhane"
          isLoading={meals.isLoading}
          error={meals.error}
          errorMessage="Yemek rezervasyonları yüklenemedi"
        >
          {upcomingMeals.length > 0 ? (
            <ul className="space-y-2">
              {upcomingMeals.map((r) => (
                <li
                  key={r.id}
                  className="flex items-center justify-between gap-3 rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-800"
                >
                  <span className="font-medium text-gray-900 dark:text-white">
                    {formatMealDay(r.date)} · {MEAL_TIME_LABELS[r.meal_time]}
                  </span>
                  <span className="text-gray-500 dark:text-gray-400">
                    {r.cafeteria_name}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyText>Yaklaşan yemek rezervasyonunuz yok.</EmptyText>
          )}
        </SummaryCard>

        <SummaryCard
          title="Ders Programı"
          icon={BookOpen}
          to="/student/enrollment"
          linkLabel="Ders kaydı"
          isLoading={programs.isLoading || attendance.isLoading}
          error={programs.error}
          errorMessage="Ders programı yüklenemedi"
        >
          {program ? (
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {semesterLabel(program.semester)} onaylı programınız
              </p>
              <p className="text-3xl font-bold text-gray-900 dark:text-white">
                {program.courses.length} ders
              </p>
            </div>
          ) : (
            <EmptyText>Onaylı bir ders programınız yok.</EmptyText>
          )}
        </SummaryCard>
      </div>
    </div>
  )
}

function SummaryCard({
  title,
  icon: Icon,
  to,
  linkLabel,
  isLoading,
  error,
  errorMessage,
  children,
}: {
  title: string
  icon: ComponentType<{ className?: string }>
  to: string
  linkLabel: string
  isLoading: boolean
  error: unknown
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
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Loader2 className="h-4 w-4 animate-spin" />
            Yükleniyor...
          </div>
        ) : error ? (
          <div className="flex items-center gap-2 text-sm text-destructive">
            <AlertCircle className="h-4 w-4" />
            {apiErrorMessage(error, errorMessage)}
          </div>
        ) : (
          children
        )}
        <Link
          to={to}
          className="inline-flex items-center gap-1 text-sm font-medium text-emerald-600 hover:underline"
        >
          {linkLabel}
          <ArrowRight className="h-4 w-4" />
        </Link>
      </CardContent>
    </Card>
  )
}

function EmptyText({ children }: { children: ReactNode }) {
  return <p className="text-sm text-gray-500 dark:text-gray-400">{children}</p>
}
