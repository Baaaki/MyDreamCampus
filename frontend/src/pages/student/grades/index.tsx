import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { gradesService } from "@/lib/services/grades-service"
import { apiErrorMessage } from "@/lib/api-error"
import { parseSemester, type SemesterMeta } from "@/lib/semester"
import type { ActiveCourse, CompletedCourse } from "@/lib/types"
import {
  GraduationCap,
  Trophy,
  BookOpen,
  Loader2,
  AlertCircle,
  Clock,
  CheckCircle2,
  TrendingUp,
  TrendingDown,
  CalendarRange,
  ChevronDown,
  Sigma,
} from "lucide-react"

// ============================================
// Notlandırma Sistemi
// Backend harf notu tutmaz: not, 15 kademeli bir katsayıdır (4.00-0.00).
// minScore, mutlak değerlendirmedeki eşiktir (grades-service
// calculateAbsoluteGradePoint); bağıl değerlendirmede katsayı z-skorundan
// gelir.
// ============================================
const GRADE_POINT_TABLE: Array<{ point: number; minScore: number }> = [
  { point: 4.0, minScore: 90 },
  { point: 3.75, minScore: 87.5 },
  { point: 3.5, minScore: 85 },
  { point: 3.25, minScore: 82.5 },
  { point: 3.0, minScore: 80 },
  { point: 2.75, minScore: 77.5 },
  { point: 2.5, minScore: 75 },
  { point: 2.25, minScore: 72.5 },
  { point: 2.0, minScore: 70 },
  { point: 1.75, minScore: 67.5 },
  { point: 1.5, minScore: 65 },
  { point: 1.25, minScore: 62.5 },
  { point: 1.0, minScore: 60 },
  { point: 0.5, minScore: 50 },
  { point: 0.0, minScore: 0 },
]

function formatGradePoint(gp: string): string {
  const value = parseFloat(gp)
  return Number.isNaN(value) ? gp : value.toFixed(2)
}

function getGradeStatus(gp: string): "pass" | "conditional" | "fail" {
  const value = parseFloat(gp)
  if (Number.isNaN(value)) return "fail"
  if (value >= 2.0) return "pass"
  if (value >= 1.0) return "conditional"
  return "fail"
}

function getGradePointClasses(gp: number): string {
  if (Number.isNaN(gp)) return "bg-gray-500 text-white"
  if (gp >= 3.75) return "bg-emerald-600 text-white"
  if (gp >= 3.25) return "bg-emerald-500 text-white"
  if (gp >= 2.75) return "bg-green-600 text-white"
  if (gp >= 2.25) return "bg-lime-600 text-white"
  if (gp >= 2.0) return "bg-yellow-600 text-white"
  if (gp >= 1.5) return "bg-orange-500 text-white"
  if (gp >= 1.0) return "bg-amber-700 text-white"
  if (gp >= 0.5) return "bg-red-500 text-white"
  return "bg-red-600 text-white"
}

function getAssessmentName(slug: string): string {
  const names: Record<string, string> = {
    midterm: "Vize",
    final: "Final",
    quiz: "Quiz",
    homework: "Ödev",
    project: "Proje",
    lab: "Lab",
    attendance: "Devam",
  }
  return names[slug] || slug.charAt(0).toUpperCase() + slug.slice(1)
}

// Bir grade point string'inin sayısal değerini güvenle döner
function gpFloat(gp: string): number {
  const v = parseFloat(gp)
  return Number.isNaN(v) ? 0 : v
}

export default function StudentGradesPage() {
  const [pickedSemester, setPickedSemester] = useState<string>("")
  const [showLegend, setShowLegend] = useState(false)

  const {
    data = null,
    isLoading: loading,
    error: queryError,
  } = useQuery({
    queryKey: ["grades", "my"],
    queryFn: () => gradesService.getMyGrades(),
  })
  const error = queryError
    ? apiErrorMessage(queryError, "Notlar yüklenemedi")
    : null

  // Tüm dönemleri (aktif + tamamlanan) topla, en yeniden eskiye sırala
  const semesterList = useMemo<SemesterMeta[]>(() => {
    if (!data) return []
    const set = new Map<string, SemesterMeta>()
    for (const c of data.active_courses) {
      if (c.semester && !set.has(c.semester))
        set.set(c.semester, parseSemester(c.semester))
    }
    for (const c of data.completed_courses) {
      if (c.semester && !set.has(c.semester))
        set.set(c.semester, parseSemester(c.semester))
    }
    return Array.from(set.values()).sort((a, b) => b.sortKey - a.sortKey)
  }, [data])

  // Default seçim: aktif dönem varsa ilk aktif, yoksa en yeni dönem
  const selectedSemester =
    pickedSemester ||
    data?.active_courses[0]?.semester ||
    semesterList[0]?.key ||
    ""

  // Seçilen dönemin dersleri
  const semesterView = useMemo(() => {
    if (!data || !selectedSemester) {
      return {
        active: [] as ActiveCourse[],
        completed: [] as CompletedCourse[],
      }
    }
    return {
      active: data.active_courses.filter(
        (c) => c.semester === selectedSemester
      ),
      completed: data.completed_courses.filter(
        (c) => c.semester === selectedSemester
      ),
    }
  }, [data, selectedSemester])

  // Seçilen dönemin GPA ve kredi istatistikleri
  const semesterStats = useMemo(() => {
    const completed = semesterView.completed
    const credits = completed.reduce((s, c) => s + c.credits, 0)
    const weightedSum = completed.reduce(
      (s, c) => s + gpFloat(c.grade_point) * c.credits,
      0
    )
    const gpa = credits > 0 ? weightedSum / credits : 0
    const passing = completed.filter(
      (c) => getGradeStatus(c.grade_point) === "pass"
    ).length
    const failing = completed.filter(
      (c) => getGradeStatus(c.grade_point) === "fail"
    ).length
    const conditional = completed.filter(
      (c) => getGradeStatus(c.grade_point) === "conditional"
    ).length
    return {
      credits,
      gpa,
      passing,
      failing,
      conditional,
      totalCourses: completed.length + semesterView.active.length,
    }
  }, [semesterView])

  if (loading) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="h-12 w-12 animate-spin text-emerald-600" />
          <p className="text-gray-500 dark:text-gray-400">
            Notlar yükleniyor...
          </p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
            <AlertCircle className="h-8 w-8 text-red-600 dark:text-red-400" />
          </div>
          <div>
            <p className="text-lg font-medium text-gray-900 dark:text-white">
              Notlar yüklenemedi
            </p>
            <p className="mt-1 text-gray-500 dark:text-gray-400">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  const isEmpty =
    !data ||
    (data.active_courses.length === 0 && data.completed_courses.length === 0)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
            <GraduationCap className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              Ders Notları
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              Dönem seçerek o döneme ait notlarınızı ve istatistiklerinizi
              görüntüleyin
            </p>
          </div>
        </div>

        {data && data.student_number && (
          <div className="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm dark:border-gray-700 dark:bg-gray-900">
            <span className="text-gray-500 dark:text-gray-400">
              Öğrenci No:
            </span>{" "}
            <span className="font-mono font-semibold text-gray-900 dark:text-white">
              {data.student_number}
            </span>
          </div>
        )}
      </div>

      {isEmpty ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <GraduationCap className="mb-4 h-16 w-16 text-gray-300 dark:text-gray-600" />
            <p className="text-center text-gray-500 dark:text-gray-400">
              Henüz kayıtlı ders veya not kaydınız bulunmuyor.
            </p>
          </CardContent>
        </Card>
      ) : (
        <>
          {/* Cumulative Summary (always visible) */}
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <Card className="border-0 bg-gradient-to-br from-emerald-500 to-emerald-700 text-white">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-emerald-100">
                      Genel Not Ortalaması
                    </p>
                    <p className="mt-1 text-4xl font-bold">
                      {data!.cumulative_gpa.toFixed(2)}
                    </p>
                    <p className="mt-1 text-xs text-emerald-100">
                      4.00 üzerinden (CGPA)
                    </p>
                  </div>
                  <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                    <Trophy className="h-7 w-7" />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-0 bg-gradient-to-br from-blue-500 to-blue-700 text-white">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-blue-100">
                      Toplam Kredi
                    </p>
                    <p className="mt-1 text-4xl font-bold">
                      {data!.total_credits}
                    </p>
                    <p className="mt-1 text-xs text-blue-100">
                      Tamamlanan toplam kredi
                    </p>
                  </div>
                  <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                    <Sigma className="h-7 w-7" />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-0 bg-gradient-to-br from-purple-500 to-purple-700 text-white">
              <CardContent className="pt-6">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-purple-100">
                      Toplam Ders
                    </p>
                    <p className="mt-1 text-4xl font-bold">
                      {data!.active_courses.length +
                        data!.completed_courses.length}
                    </p>
                    <p className="mt-1 text-xs text-purple-100">
                      {data!.active_courses.length} aktif ·{" "}
                      {data!.completed_courses.length} tamamlanmış
                    </p>
                  </div>
                  <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20">
                    <BookOpen className="h-7 w-7" />
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Semester Filter — primary UX */}
          <Card>
            <CardContent className="flex flex-col gap-4 py-4 md:flex-row md:items-center md:justify-between">
              <div className="flex items-center gap-3">
                <CalendarRange className="h-5 w-5 text-emerald-600" />
                <label
                  htmlFor="semester-select"
                  className="text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  Dönem Seç
                </label>
                <Select
                  value={selectedSemester}
                  onValueChange={setPickedSemester}
                >
                  <SelectTrigger id="semester-select" className="min-w-[260px]">
                    <SelectValue placeholder="Bir dönem seçin..." />
                  </SelectTrigger>
                  <SelectContent>
                    {semesterList.map((s) => {
                      const isActive = data!.active_courses.some(
                        (c) => c.semester === s.key
                      )
                      return (
                        <SelectItem key={s.key} value={s.key}>
                          <span className="flex items-center gap-2">
                            <span>{s.label}</span>
                            {isActive && (
                              <Badge
                                variant="outline"
                                className="border-orange-300 bg-orange-50 text-[10px] text-orange-700 dark:border-orange-800 dark:bg-orange-950/40 dark:text-orange-300"
                              >
                                Aktif
                              </Badge>
                            )}
                          </span>
                        </SelectItem>
                      )
                    })}
                  </SelectContent>
                </Select>
              </div>

              <button
                type="button"
                onClick={() => setShowLegend((s) => !s)}
                className="flex items-center gap-1.5 self-start text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              >
                <span>Notlandırma sistemi</span>
                <ChevronDown
                  className={`h-3.5 w-3.5 transition-transform ${showLegend ? "rotate-180" : ""}`}
                />
              </button>
            </CardContent>

            {showLegend && (
              <div className="border-t border-gray-200 px-6 py-4 dark:border-gray-700">
                <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
                  Katsayı (4.00 üzerinden) ve mutlak değerlendirmede gereken en
                  düşük ortalama:
                </p>
                <div className="grid grid-cols-3 gap-2 text-xs sm:grid-cols-4 md:grid-cols-5">
                  {GRADE_POINT_TABLE.map((g) => (
                    <div
                      key={g.point}
                      className="flex items-center justify-between rounded-md border border-gray-200 bg-gray-50 px-2.5 py-1.5 dark:border-gray-700 dark:bg-gray-800"
                    >
                      <span
                        className={`rounded px-1.5 py-0.5 font-mono text-[10px] font-bold ${getGradePointClasses(g.point)}`}
                      >
                        {g.point.toFixed(2)}
                      </span>
                      <span className="font-mono text-gray-700 dark:text-gray-300">
                        {g.minScore > 0 ? `≥ ${g.minScore}` : `< 50`}
                      </span>
                    </div>
                  ))}
                </div>
                <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                  Sınıf ortalaması 60&apos;ın altındaysa bağıl değerlendirme
                  yapılır: katsayı, notunuzun sınıf ortalamasından sapmasına
                  (z-skoru) göre belirlenir ve yukarıdaki eşikler uygulanmaz.
                </p>
                <div className="mt-3 flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400">
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-emerald-500" />{" "}
                    Geçti (≥ 2.00)
                  </span>
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-orange-500" />{" "}
                    Şartlı (1.00 - 1.75)
                  </span>
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-red-500" /> Kaldı
                    (0.50 - 0.00)
                  </span>
                </div>
              </div>
            )}
          </Card>

          {/* Selected Semester View */}
          {selectedSemester && (
            <SemesterView
              meta={parseSemester(selectedSemester)}
              activeCourses={semesterView.active}
              completedCourses={semesterView.completed}
              stats={semesterStats}
            />
          )}
        </>
      )}
    </div>
  )
}

// ============================================
// Semester View — selected semester detail
// ============================================
interface SemesterStats {
  credits: number
  gpa: number
  passing: number
  failing: number
  conditional: number
  totalCourses: number
}

function SemesterView({
  meta,
  activeCourses,
  completedCourses,
  stats,
}: {
  meta: SemesterMeta
  activeCourses: ActiveCourse[]
  completedCourses: CompletedCourse[]
  stats: SemesterStats
}) {
  const hasActive = activeCourses.length > 0
  const hasCompleted = completedCourses.length > 0

  return (
    <div className="space-y-6">
      {/* Semester header + stats */}
      <Card>
        <CardHeader className="border-b border-gray-200 pb-4 dark:border-gray-700">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <CalendarRange className="h-5 w-5 text-emerald-600" />
              {meta.label}
            </CardTitle>
            <div className="flex flex-wrap gap-2">
              {hasActive && (
                <Badge className="bg-orange-100 text-orange-700 hover:bg-orange-100 dark:bg-orange-900/40 dark:text-orange-300">
                  <Clock className="mr-1 h-3 w-3" />
                  Devam Ediyor
                </Badge>
              )}
              {hasCompleted && (
                <Badge className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-900/40 dark:text-emerald-300">
                  <CheckCircle2 className="mr-1 h-3 w-3" />
                  Tamamlandı
                </Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
            <StatBlock
              label="Dönem Ortalaması"
              value={hasCompleted ? stats.gpa.toFixed(2) : "—"}
              hint="GPA (4.00)"
              accent={
                hasCompleted
                  ? stats.gpa >= 2
                    ? "good"
                    : stats.gpa >= 1
                      ? "warn"
                      : "bad"
                  : "neutral"
              }
            />
            <StatBlock
              label="Dönem Kredisi"
              value={String(stats.credits)}
              hint="Tamamlanan kredi"
              accent="neutral"
            />
            <StatBlock
              label="Geçilen Ders"
              value={String(stats.passing)}
              hint="2.00 ve üzeri"
              accent="good"
            />
            <StatBlock
              label="Şartlı Geçilen"
              value={String(stats.conditional)}
              hint="1.00 - 1.75 arası"
              accent="warn"
            />
            <StatBlock
              label="Kalan Ders"
              value={String(stats.failing)}
              hint="0.50 / 0.00"
              accent="bad"
            />
          </div>
        </CardContent>
      </Card>

      {/* Active Courses (in-progress) */}
      {hasActive && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <Clock className="h-5 w-5 text-orange-500" />
              Devam Eden Dersler
              <Badge variant="outline" className="ml-2">
                {activeCourses.length} ders
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow className="bg-gray-50 dark:bg-gray-800/40">
                  <TableHead className="pl-6">Ders Kodu</TableHead>
                  <TableHead>Ders Adı</TableHead>
                  <TableHead className="text-center">Kredi</TableHead>
                  <TableHead>Girilen Notlar</TableHead>
                  <TableHead className="pr-6 text-right">Durum</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeCourses.map((c, idx) => {
                  const entries = Object.entries(c.scores)
                  const hasAnyScore = entries.length > 0
                  return (
                    <TableRow key={`${c.course_code}-${idx}`}>
                      <TableCell className="pl-6">
                        <span className="rounded-md bg-blue-100 px-2 py-1 font-mono text-xs font-semibold text-blue-700 dark:bg-blue-900/40 dark:text-blue-300">
                          {c.course_code}
                        </span>
                      </TableCell>
                      <TableCell className="font-medium text-gray-900 dark:text-white">
                        {c.course_name}
                      </TableCell>
                      <TableCell className="text-center text-gray-700 dark:text-gray-300">
                        {c.credits}
                      </TableCell>
                      <TableCell>
                        {hasAnyScore ? (
                          <div className="flex flex-wrap gap-1.5">
                            {entries.map(([slug, detail]) => (
                              <span
                                key={slug}
                                className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-xs font-medium ${getScoreClasses(detail.score, detail.is_absent)}`}
                              >
                                <span className="opacity-70">
                                  {getAssessmentName(slug)}:
                                </span>
                                <span className="font-semibold">
                                  {detail.is_absent
                                    ? "Girmedi"
                                    : detail.score !== null
                                      ? detail.score
                                      : "—"}
                                </span>
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className="text-xs text-gray-400 italic dark:text-gray-500">
                            Henüz not girilmedi
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <Badge
                          variant="outline"
                          className="border-orange-300 bg-orange-50 text-orange-700 dark:border-orange-800 dark:bg-orange-950/30 dark:text-orange-300"
                        >
                          Devam Ediyor
                        </Badge>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Completed Courses (final grades) */}
      {hasCompleted && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-gray-900 dark:text-white">
              <CheckCircle2 className="h-5 w-5 text-emerald-600" />
              Tamamlanan Dersler
              <Badge variant="outline" className="ml-2">
                {completedCourses.length} ders
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow className="bg-gray-50 dark:bg-gray-800/40">
                  <TableHead className="pl-6">Ders Kodu</TableHead>
                  <TableHead>Ders Adı</TableHead>
                  <TableHead className="text-center">Kredi</TableHead>
                  <TableHead>Değerlendirme Notları</TableHead>
                  <TableHead className="text-right">Ortalama</TableHead>
                  <TableHead className="pr-6 text-right">Katsayı</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {completedCourses.map((c, idx) => {
                  const status = getGradeStatus(c.grade_point)
                  const scoreEntries = c.assessment_scores
                    ? Object.entries(c.assessment_scores)
                    : []
                  return (
                    <TableRow
                      key={`${c.course_code}-${idx}`}
                      className={
                        status === "fail"
                          ? "bg-red-50/40 dark:bg-red-950/10"
                          : ""
                      }
                    >
                      <TableCell className="pl-6">
                        <span className="rounded-md bg-gray-100 px-2 py-1 font-mono text-xs font-semibold text-gray-700 dark:bg-gray-800 dark:text-gray-300">
                          {c.course_code}
                        </span>
                      </TableCell>
                      <TableCell className="font-medium text-gray-900 dark:text-white">
                        {c.course_name}
                        {status === "fail" && (
                          <span className="ml-2 inline-flex items-center gap-1 text-[11px] text-red-600 dark:text-red-400">
                            <TrendingDown className="h-3 w-3" />
                            Tekrar gerekli
                          </span>
                        )}
                        {status === "pass" && gpFloat(c.grade_point) >= 3.5 && (
                          <span className="ml-2 inline-flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400">
                            <TrendingUp className="h-3 w-3" />
                            Yüksek başarı
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-center text-gray-700 dark:text-gray-300">
                        {c.credits}
                      </TableCell>
                      <TableCell>
                        {scoreEntries.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {scoreEntries.map(([slug, score]) => (
                              <span
                                key={slug}
                                className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] font-medium ${getScoreClasses(score, false)}`}
                              >
                                <span className="opacity-70">
                                  {getAssessmentName(slug)}:
                                </span>
                                <span className="font-semibold">{score}</span>
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className="text-xs text-gray-400 dark:text-gray-500">
                            —
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                        {c.weighted_average.toFixed(1)}
                      </TableCell>
                      <TableCell className="pr-6 text-right">
                        <span
                          className={`inline-flex min-w-[2.5rem] items-center justify-center rounded px-2 py-0.5 font-mono text-xs font-bold ${getGradePointClasses(parseFloat(c.grade_point))}`}
                          title="4.00 üzerinden"
                        >
                          {formatGradePoint(c.grade_point)}
                        </span>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Edge case: semester selected but has no courses (shouldn't normally happen) */}
      {!hasActive && !hasCompleted && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <BookOpen className="mb-3 h-12 w-12 text-gray-300 dark:text-gray-600" />
            <p className="text-gray-500 dark:text-gray-400">
              Bu döneme ait ders kaydınız bulunmuyor.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ============================================
// Helpers — UI primitives
// ============================================
function StatBlock({
  label,
  value,
  hint,
  accent,
}: {
  label: string
  value: string
  hint: string
  accent: "good" | "warn" | "bad" | "neutral"
}) {
  const accentMap = {
    good: "text-emerald-600 dark:text-emerald-400",
    warn: "text-orange-600 dark:text-orange-400",
    bad: "text-red-600 dark:text-red-400",
    neutral: "text-gray-900 dark:text-white",
  }
  return (
    <div className="rounded-lg border border-gray-100 bg-gray-50 px-4 py-3 dark:border-gray-800 dark:bg-gray-800/40">
      <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
        {label}
      </p>
      <p className={`mt-1 text-2xl font-bold ${accentMap[accent]}`}>{value}</p>
      <p className="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500">
        {hint}
      </p>
    </div>
  )
}

function getScoreClasses(score: number | null, isAbsent: boolean): string {
  if (isAbsent)
    return "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300"
  if (score === null)
    return "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"
  if (score >= 85)
    return "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300"
  if (score >= 70)
    return "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300"
  if (score >= 50)
    return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300"
  return "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300"
}
