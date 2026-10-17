import { useState } from "react"
import { useNavigate } from "react-router"
import { useQuery } from "@tanstack/react-query"
import type { CourseCatalog, Department, Faculty } from "@/lib/types"
import { catalogService, useFaculties } from "@/lib/services/catalog-service"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Book,
  Clock,
  GraduationCap,
  Calendar,
  FileText,
  Target,
  List,
  ChevronDown,
  ChevronRight,
  ArrowLeft,
  BookOpen,
  Building2,
  Plus,
  Loader2,
  AlertCircle,
} from "lucide-react"

// Semester info helper
const getSemesterInfo = (semester: number) => {
  const year = Math.ceil(semester / 2)
  const term = semester % 2 === 1 ? "Güz" : "Bahar"
  return { year, term, label: `${year}. Yıl - ${term} Dönemi` }
}

export default function CourseCatalogPage() {
  const navigate = useNavigate()
  const [expandedFaculties, setExpandedFaculties] = useState<string[]>([])
  const [selectedDepartment, setSelectedDepartment] = useState<{
    dept: Department
    faculty: Faculty
  } | null>(null)
  const [selectedCourse, setSelectedCourse] = useState<CourseCatalog | null>(
    null
  )
  const [isDetailOpen, setIsDetailOpen] = useState(false)

  const {
    data: faculties = [],
    isLoading: isLoadingFaculties,
    isError: isErrorFaculties,
  } = useFaculties()

  const selectedDepartmentName = selectedDepartment?.dept.name
  const {
    data: departmentCourses = [],
    isLoading,
    isError,
    refetch: refetchDepartmentCourses,
  } = useQuery({
    queryKey: ["catalog", "department-courses", selectedDepartmentName],
    queryFn: () =>
      catalogService.getCoursesByDepartment(selectedDepartmentName!),
    enabled: !!selectedDepartmentName,
  })
  const error = isError ? "Dersler yüklenirken bir hata oluştu." : null

  // Course count per department, for the accordion badges
  const { data: courseCounts = {} } = useQuery({
    queryKey: ["catalog", "course-counts"],
    queryFn: async () => {
      const { courses } = await catalogService.listCourses({ limit: 100 }) // Backend max is 100
      const counts: Record<string, number> = {}
      courses.forEach((course) => {
        counts[course.department] = (counts[course.department] || 0) + 1
      })
      return counts
    },
  })

  // Toggle faculty accordion
  const toggleFaculty = (facultyId: string) => {
    setExpandedFaculties((prev) =>
      prev.includes(facultyId)
        ? prev.filter((id) => id !== facultyId)
        : [...prev, facultyId]
    )
  }

  // Group courses by semester
  const groupCoursesBySemester = (courses: CourseCatalog[]) => {
    const grouped: {
      [key: number]: { mandatory: CourseCatalog[]; elective: CourseCatalog[] }
    } = {}

    for (let i = 1; i <= 8; i++) {
      grouped[i] = { mandatory: [], elective: [] }
    }

    courses.forEach((course) => {
      const semester = course.semester || 1
      if (course.course_type === "mandatory") {
        grouped[semester].mandatory.push(course)
      } else {
        grouped[semester].elective.push(course)
      }
    })

    return grouped
  }

  const handleCourseClick = async (course: CourseCatalog) => {
    setSelectedCourse(course) // Show basic info immediately
    setIsDetailOpen(true)

    // Fetch full course details for complete information
    try {
      const fullCourse = await catalogService.getCourseByCode(
        course.course_code
      )
      setSelectedCourse(fullCourse)
    } catch (err) {
      console.error("Failed to fetch course details:", err)
      // Keep showing basic info if detailed fetch fails
    }
  }

  const handleDepartmentClick = (dept: Department, faculty: Faculty) => {
    setSelectedDepartment({ dept, faculty })
  }

  // Faculty & Department List View (Accordion)
  if (!selectedDepartment) {
    return (
      <div className="min-h-screen bg-background py-8">
        <div className="mx-auto max-w-4xl px-4">
          <div className="rounded-lg bg-card p-4 shadow-md sm:p-6">
            <div className="mb-8 flex flex-col items-center gap-4 sm:flex-row sm:justify-between">
              <div className="flex-1 text-center">
                <h1 className="mb-2 text-2xl font-bold text-foreground sm:text-3xl">
                  Ders Kataloğu
                </h1>
                <p className="text-muted-foreground">
                  Fakülte ve bölüm seçerek ders programını görüntüleyebilirsiniz
                </p>
              </div>
              <Button
                onClick={() => navigate("/catalog/add")}
                className="bg-indigo-600 hover:bg-indigo-700"
              >
                <Plus className="mr-2 h-4 w-4" />
                Yeni Ders Ekle
              </Button>
            </div>

            {/* Faculty Accordion */}
            {isLoadingFaculties && (
              <div className="flex items-center justify-center p-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                <span className="ml-2 text-sm text-muted-foreground">
                  Fakülteler yükleniyor...
                </span>
              </div>
            )}
            {isErrorFaculties && (
              <div className="flex items-center justify-center gap-2 p-8 text-sm text-destructive">
                <AlertCircle className="h-5 w-5" />
                <span>Fakülteler yüklenirken bir hata oluştu</span>
              </div>
            )}
            {!isLoadingFaculties && !isErrorFaculties && (
              <div className="space-y-2">
                {faculties.map((faculty) => {
                  const isExpanded = expandedFaculties.includes(faculty.id)
                  return (
                    <div
                      key={faculty.id}
                      className="overflow-hidden rounded-lg border"
                    >
                      {/* Faculty Header */}
                      <button
                        onClick={() => toggleFaculty(faculty.id)}
                        className="flex w-full items-center justify-between bg-muted/50 p-4 text-left transition-colors hover:bg-muted"
                      >
                        <div className="flex items-center gap-3">
                          <Building2 className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                          <span className="font-semibold text-foreground">
                            {faculty.name}
                          </span>
                          <Badge variant="outline" className="text-xs">
                            {faculty.departments.length} bölüm
                          </Badge>
                        </div>
                        {isExpanded ? (
                          <ChevronDown className="h-5 w-5 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-5 w-5 text-muted-foreground" />
                        )}
                      </button>

                      {/* Departments List */}
                      {isExpanded && (
                        <div className="border-t bg-card">
                          {faculty.departments.map((dept, index) => {
                            const courseCount = courseCounts[dept.name] || 0
                            return (
                              <button
                                key={dept.id}
                                onClick={() =>
                                  handleDepartmentClick(dept, faculty)
                                }
                                className={`flex w-full items-center justify-between p-3 pl-12 text-left transition-colors hover:bg-indigo-50 dark:hover:bg-indigo-950/40 ${
                                  index !== faculty.departments.length - 1
                                    ? "border-b border-border"
                                    : ""
                                }`}
                              >
                                <div className="flex items-center gap-3">
                                  <GraduationCap className="h-4 w-4 text-muted-foreground/70" />
                                  <span className="text-foreground/80">
                                    {dept.name}
                                  </span>
                                </div>
                                <div className="flex items-center gap-2">
                                  {courseCount > 0 && (
                                    <span className="text-xs text-muted-foreground">
                                      {courseCount} ders
                                    </span>
                                  )}
                                  <ChevronRight className="h-4 w-4 text-muted-foreground/70" />
                                </div>
                              </button>
                            )
                          })}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}

            {/* Stats */}
            <div className="mt-8 grid grid-cols-2 gap-4 border-t pt-6 text-center">
              <div className="rounded-lg bg-indigo-50 p-4 dark:bg-indigo-950/30">
                <div className="text-2xl font-bold text-indigo-600 dark:text-indigo-400">
                  {faculties.length}
                </div>
                <div className="text-sm text-muted-foreground">Fakülte</div>
              </div>
              <div className="rounded-lg bg-green-50 p-4 dark:bg-green-950/30">
                <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                  {faculties.reduce((sum, f) => sum + f.departments.length, 0)}
                </div>
                <div className="text-sm text-muted-foreground">Bölüm</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Department Detail View (Semester-based course listing)
  const groupedCourses = groupCoursesBySemester(departmentCourses)

  // Loading state
  if (isLoading) {
    return (
      <div className="min-h-screen bg-background py-8">
        <div className="mx-auto max-w-7xl px-4">
          <div className="mb-6 rounded-lg bg-card p-4 shadow-md sm:p-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="mb-4 flex items-center gap-2 text-indigo-600 transition-colors hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="flex items-center justify-center py-12">
              <div className="h-12 w-12 animate-spin rounded-full border-b-2 border-indigo-600"></div>
              <span className="ml-4 text-muted-foreground">
                Dersler yükleniyor...
              </span>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="min-h-screen bg-background py-8">
        <div className="mx-auto max-w-7xl px-4">
          <div className="mb-6 rounded-lg bg-card p-4 shadow-md sm:p-6">
            <button
              onClick={() => setSelectedDepartment(null)}
              className="mb-4 flex items-center gap-2 text-indigo-600 transition-colors hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              <ArrowLeft className="h-5 w-5" />
              <span>Tüm Fakülteler</span>
            </button>
            <div className="py-12 text-center">
              <div className="mb-4 text-red-500 dark:text-red-400">
                <BookOpen className="mx-auto h-16 w-16 text-red-300" />
              </div>
              <h2 className="mb-2 text-xl font-semibold text-foreground/80">
                Hata Oluştu
              </h2>
              <p className="mb-4 text-muted-foreground">{error}</p>
              <Button
                onClick={() => refetchDepartmentCourses()}
                variant="outline"
              >
                Tekrar Dene
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background py-8">
      <div className="mx-auto max-w-7xl px-4">
        {/* Header */}
        <div className="mb-6 rounded-lg bg-card p-4 shadow-md sm:p-6">
          <button
            onClick={() => setSelectedDepartment(null)}
            className="mb-4 flex items-center gap-2 text-indigo-600 transition-colors hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-300"
          >
            <ArrowLeft className="h-5 w-5" />
            <span>Tüm Fakülteler</span>
          </button>

          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-xl bg-indigo-100 dark:bg-indigo-900/40">
              <GraduationCap className="h-8 w-8 text-indigo-600 dark:text-indigo-400" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-foreground">
                {selectedDepartment.dept.name}
              </h1>
              <p className="text-muted-foreground">
                {selectedDepartment.faculty.name}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {departmentCourses.length} ders
              </p>
            </div>
          </div>
        </div>

        {/* Course content check */}
        {departmentCourses.length === 0 ? (
          <div className="rounded-lg bg-card p-8 text-center shadow-md">
            <BookOpen className="mx-auto mb-4 h-16 w-16 text-muted-foreground/40" />
            <h2 className="mb-2 text-xl font-semibold text-foreground/80">
              Ders Bilgisi Bulunamadı
            </h2>
            <p className="text-muted-foreground">
              Bu bölüm için henüz ders bilgisi eklenmemiş.
            </p>
          </div>
        ) : (
          /* Semester Grid - 4 years x 2 semesters */
          <div className="space-y-8">
            {[1, 2, 3, 4].map((year) => (
              <div
                key={year}
                className="overflow-hidden rounded-lg bg-card shadow-md"
              >
                <div className="bg-gradient-to-r from-indigo-600 to-indigo-700 px-6 py-4">
                  <h2 className="text-xl font-bold text-white">{year}. Yıl</h2>
                </div>

                <div className="grid grid-cols-1 divide-y divide-border lg:grid-cols-2 lg:divide-x lg:divide-y-0">
                  {[year * 2 - 1, year * 2].map((semester) => {
                    const semesterInfo = getSemesterInfo(semester)
                    const semesterCourses = groupedCourses[semester]
                    const totalEcts = [
                      ...semesterCourses.mandatory,
                      ...semesterCourses.elective,
                    ].reduce((sum, c) => sum + (c.ects || c.credits), 0)

                    return (
                      <div key={semester} className="p-6">
                        <div className="mb-4 flex items-center justify-between">
                          <h3 className="text-lg font-semibold text-foreground">
                            {semester}. Dönem ({semesterInfo.term})
                          </h3>
                          <Badge variant="outline" className="text-sm">
                            {totalEcts} AKTS
                          </Badge>
                        </div>

                        {/* Zorunlu Dersler */}
                        {semesterCourses.mandatory.length > 0 && (
                          <div className="mb-6">
                            <h4 className="mb-3 flex items-center gap-2 text-sm font-semibold text-red-700 dark:text-red-300">
                              <div className="h-2 w-2 rounded-full bg-red-500"></div>
                              ZORUNLU DERSLER
                            </h4>
                            <div className="relative overflow-x-auto">
                              <table className="w-full text-sm">
                                <thead>
                                  <tr className="bg-red-50 text-left dark:bg-red-950/30">
                                    <th className="px-3 py-2 font-medium text-foreground/80">
                                      Ders Kodu
                                    </th>
                                    <th className="px-3 py-2 font-medium text-foreground/80">
                                      Ders Adı
                                    </th>
                                    <th className="px-3 py-2 text-center font-medium text-foreground/80">
                                      D
                                    </th>
                                    <th className="px-3 py-2 text-center font-medium text-foreground/80">
                                      AKTS
                                    </th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.mandatory.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="cursor-pointer border-b border-border transition-colors hover:bg-red-50 dark:hover:bg-red-950/40"
                                      onClick={() => handleCourseClick(course)}
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600 dark:text-indigo-400">
                                        {course.course_code}
                                      </td>
                                      <td className="px-3 py-2 text-foreground">
                                        {course.name}
                                      </td>
                                      <td className="px-3 py-2 text-center text-muted-foreground">
                                        {course.theoretical_hours}
                                      </td>
                                      <td className="px-3 py-2 text-center font-medium text-foreground">
                                        {course.ects || course.credits}
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        )}

                        {/* Seçmeli Dersler */}
                        {semesterCourses.elective.length > 0 && (
                          <div>
                            <h4 className="mb-3 flex items-center gap-2 text-sm font-semibold text-green-700 dark:text-green-300">
                              <div className="h-2 w-2 rounded-full bg-green-500"></div>
                              SEÇMELİ DERSLER
                            </h4>
                            <div className="relative overflow-x-auto">
                              <table className="w-full text-sm">
                                <thead>
                                  <tr className="bg-green-50 text-left dark:bg-green-950/30">
                                    <th className="px-3 py-2 font-medium text-foreground/80">
                                      Ders Kodu
                                    </th>
                                    <th className="px-3 py-2 font-medium text-foreground/80">
                                      Ders Adı
                                    </th>
                                    <th className="px-3 py-2 text-center font-medium text-foreground/80">
                                      D
                                    </th>
                                    <th className="px-3 py-2 text-center font-medium text-foreground/80">
                                      U
                                    </th>
                                    <th className="px-3 py-2 text-center font-medium text-foreground/80">
                                      AKTS
                                    </th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {semesterCourses.elective.map((course) => (
                                    <tr
                                      key={course.id}
                                      className="cursor-pointer border-b border-border transition-colors hover:bg-green-50 dark:hover:bg-green-950/40"
                                      onClick={() => handleCourseClick(course)}
                                    >
                                      <td className="px-3 py-2 font-medium text-indigo-600 dark:text-indigo-400">
                                        {course.course_code}
                                      </td>
                                      <td className="px-3 py-2 text-foreground">
                                        {course.name}
                                      </td>
                                      <td className="px-3 py-2 text-center text-muted-foreground">
                                        {course.theoretical_hours}
                                      </td>
                                      <td className="px-3 py-2 text-center text-muted-foreground">
                                        {course.lab_hours}
                                      </td>
                                      <td className="px-3 py-2 text-center font-medium text-foreground">
                                        {course.ects || course.credits}
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        )}

                        {/* Empty state for semester */}
                        {semesterCourses.mandatory.length === 0 &&
                          semesterCourses.elective.length === 0 && (
                            <p className="text-sm text-muted-foreground/70 italic">
                              Bu dönem için ders bilgisi yok
                            </p>
                          )}
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Course Detail Dialog */}
      <Dialog open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl md:max-w-4xl lg:max-w-5xl">
          <DialogHeader>
            <DialogTitle className="text-xl">
              {selectedCourse?.course_code} - {selectedCourse?.name}
            </DialogTitle>
          </DialogHeader>

          {selectedCourse && (
            <div className="flex-1 space-y-6">
              {/* Badges */}
              <div className="flex flex-wrap gap-2">
                <Badge
                  variant={
                    selectedCourse.course_type === "mandatory"
                      ? "default"
                      : "secondary"
                  }
                >
                  {selectedCourse.course_type === "mandatory"
                    ? "Zorunlu"
                    : "Seçmeli"}
                </Badge>
                <Badge variant="outline">
                  {selectedCourse.class_level}. Sınıf
                </Badge>
                {selectedCourse.semester && (
                  <Badge variant="outline">
                    {selectedCourse.semester}. Dönem
                  </Badge>
                )}
                {selectedCourse.education_level && (
                  <Badge variant="outline">
                    {selectedCourse.education_level}
                  </Badge>
                )}
                {selectedCourse.language && (
                  <Badge variant="outline">{selectedCourse.language}</Badge>
                )}
              </div>

              {/* Basic Info Grid */}
              <div className="grid grid-cols-1 gap-4 text-sm md:grid-cols-2">
                <div className="flex items-center gap-2 rounded-lg bg-muted/50 p-3">
                  <Book className="h-5 w-5 flex-shrink-0 text-indigo-500 dark:text-indigo-400" />
                  <span className="text-muted-foreground">Fakülte:</span>
                  <span className="font-medium text-foreground">
                    {selectedCourse.faculty}
                  </span>
                </div>
                <div className="flex items-center gap-2 rounded-lg bg-muted/50 p-3">
                  <GraduationCap className="h-5 w-5 flex-shrink-0 text-indigo-500 dark:text-indigo-400" />
                  <span className="text-muted-foreground">Bölüm:</span>
                  <span className="font-medium text-foreground">
                    {selectedCourse.department}
                  </span>
                </div>
                {selectedCourse.offering_unit && (
                  <div className="flex items-center gap-2 rounded-lg bg-muted/50 p-3 md:col-span-2">
                    <BookOpen className="h-5 w-5 flex-shrink-0 text-indigo-500 dark:text-indigo-400" />
                    <span className="text-muted-foreground">
                      Dersi Veren Birim:
                    </span>
                    <span className="font-medium text-foreground">
                      {selectedCourse.offering_unit}
                    </span>
                  </div>
                )}
                {selectedCourse.teaching_type && (
                  <div className="flex items-center gap-2 rounded-lg bg-muted/50 p-3">
                    <FileText className="h-5 w-5 flex-shrink-0 text-indigo-500 dark:text-indigo-400" />
                    <span className="text-muted-foreground">Öğretim Türü:</span>
                    <span className="font-medium text-foreground">
                      {selectedCourse.teaching_type}
                    </span>
                  </div>
                )}
              </div>

              {/* Credits & Hours */}
              <div className="rounded-xl bg-gradient-to-r from-indigo-50 to-blue-50 p-5 dark:from-indigo-950/40 dark:to-blue-950/40">
                <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                  <Clock className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                  Kredi ve Saat Bilgileri
                </h4>
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
                  <div className="rounded-lg border border-indigo-100 bg-card p-3 text-center dark:border-indigo-900">
                    <div className="text-2xl font-bold text-indigo-600 dark:text-indigo-400">
                      {selectedCourse.theoretical_hours}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      Teorik (D)
                    </div>
                  </div>
                  <div className="rounded-lg border border-blue-100 bg-card p-3 text-center dark:border-blue-900">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                      {selectedCourse.lab_hours}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      Uygulama (U)
                    </div>
                  </div>
                  <div className="rounded-lg border border-cyan-100 bg-card p-3 text-center dark:border-cyan-900">
                    <div className="text-2xl font-bold text-cyan-600 dark:text-cyan-400">
                      {selectedCourse.lab_hours || 0}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      Lab (L)
                    </div>
                  </div>
                  <div className="rounded-lg border border-purple-100 bg-card p-3 text-center dark:border-purple-900">
                    <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">
                      {selectedCourse.credits}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      Kredi
                    </div>
                  </div>
                  <div className="rounded-lg border border-green-100 bg-card p-3 text-center dark:border-green-900">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                      {selectedCourse.ects || selectedCourse.credits}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      AKTS
                    </div>
                  </div>
                </div>
              </div>

              {/* Coordinator */}
              {selectedCourse.coordinator && (
                <div className="rounded-xl border border-blue-200 bg-blue-50 p-5 dark:border-blue-900 dark:bg-blue-950/30">
                  <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                    <GraduationCap className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    Ders Koordinatörü
                  </h4>
                  <div className="space-y-2 text-sm">
                    <p className="font-semibold text-foreground">
                      {selectedCourse.coordinator.title}{" "}
                      {selectedCourse.coordinator.name}
                    </p>
                    {selectedCourse.coordinator.email && (
                      <p className="text-muted-foreground">
                        <span className="font-medium">E-posta:</span>{" "}
                        {selectedCourse.coordinator.email}
                      </p>
                    )}
                    {selectedCourse.coordinator.phone && (
                      <p className="text-muted-foreground">
                        <span className="font-medium">Telefon:</span>{" "}
                        {selectedCourse.coordinator.phone}
                      </p>
                    )}
                    {selectedCourse.coordinator.office && (
                      <p className="text-muted-foreground">
                        <span className="font-medium">Ofis:</span>{" "}
                        {selectedCourse.coordinator.office}
                      </p>
                    )}
                  </div>
                </div>
              )}

              {/* Purpose */}
              {selectedCourse.purpose && (
                <div className="rounded-xl border bg-card p-5">
                  <h4 className="mb-2 flex items-center gap-2 text-base font-bold text-foreground">
                    <Target className="h-5 w-5 text-purple-500 dark:text-purple-400" />
                    Dersin Amacı
                  </h4>
                  <p className="text-sm leading-relaxed text-foreground/80">
                    {selectedCourse.purpose}
                  </p>
                </div>
              )}

              {/* Learning Outcomes List */}
              {selectedCourse.learning_outcomes_list &&
                selectedCourse.learning_outcomes_list.length > 0 && (
                  <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-5 dark:border-emerald-900 dark:bg-emerald-950/30">
                    <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                      <Target className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                      Öğrenme Kazanımları
                    </h4>
                    <ol className="space-y-2 text-sm">
                      {selectedCourse.learning_outcomes_list.map(
                        (outcome, index) => (
                          <li
                            key={index}
                            className="flex gap-3 text-foreground/80"
                          >
                            <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-emerald-500 text-xs font-bold text-white">
                              {index + 1}
                            </span>
                            <span className="leading-relaxed">{outcome}</span>
                          </li>
                        )
                      )}
                    </ol>
                  </div>
                )}

              {/* Weekly Topics */}
              {selectedCourse.weekly_topics &&
                selectedCourse.weekly_topics.length > 0 && (
                  <div className="rounded-xl border bg-card p-5">
                    <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                      <Calendar className="h-5 w-5 text-cyan-500 dark:text-cyan-400" />
                      Ders İçeriği (Haftalık)
                    </h4>
                    <div className="relative overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="bg-muted/50">
                            <th className="w-20 px-3 py-2 text-left font-medium text-foreground/80">
                              Hafta
                            </th>
                            <th className="px-3 py-2 text-left font-medium text-foreground/80">
                              Konu
                            </th>
                          </tr>
                        </thead>
                        <tbody>
                          {selectedCourse.weekly_topics.map((topic) => (
                            <tr
                              key={topic.week}
                              className="border-b border-border"
                            >
                              <td className="px-3 py-2 font-medium text-indigo-600 dark:text-indigo-400">
                                {topic.week}
                              </td>
                              <td className="px-3 py-2 text-foreground/80">
                                {topic.topic}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

              {/* Recommended Sources */}
              {selectedCourse.recommended_sources &&
                selectedCourse.recommended_sources.length > 0 && (
                  <div className="rounded-xl border border-amber-200 bg-amber-50 p-5 dark:border-amber-900 dark:bg-amber-950/30">
                    <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                      <BookOpen className="h-5 w-5 text-amber-600 dark:text-amber-400" />
                      Önerilen Kaynaklar
                    </h4>
                    <ul className="space-y-2 text-sm">
                      {selectedCourse.recommended_sources.map(
                        (source, index) => (
                          <li
                            key={index}
                            className="flex gap-2 text-foreground/80"
                          >
                            <span className="text-amber-500 dark:text-amber-400">
                              •
                            </span>
                            <span>{source}</span>
                          </li>
                        )
                      )}
                    </ul>
                  </div>
                )}

              {/* Prerequisites */}
              {selectedCourse.prerequisites &&
                selectedCourse.prerequisites.length > 0 && (
                  <div className="rounded-xl border border-orange-200 bg-orange-50 p-5 dark:border-orange-900 dark:bg-orange-950/30">
                    <h4 className="mb-3 flex items-center gap-2 text-base font-bold text-foreground">
                      <List className="h-5 w-5 text-orange-500 dark:text-orange-400" />
                      Ön Koşullar
                    </h4>
                    <div className="space-y-2">
                      {selectedCourse.prerequisites.map((prereq) => (
                        <div
                          key={prereq.id}
                          className="flex items-center gap-3 rounded-lg border border-orange-200 bg-card p-3 text-sm dark:border-orange-900"
                        >
                          <Badge
                            variant="outline"
                            className="bg-orange-100 dark:bg-orange-900/40"
                          >
                            {prereq.course_code}
                          </Badge>
                          <span className="font-medium text-foreground">
                            {prereq.course_name}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              {/* No Prerequisites */}
              {(!selectedCourse.prerequisites ||
                selectedCourse.prerequisites.length === 0) && (
                <div className="rounded-xl border bg-muted/50 p-5">
                  <h4 className="mb-2 flex items-center gap-2 text-base font-bold text-foreground">
                    <List className="h-5 w-5 text-muted-foreground" />
                    Ön Koşullar
                  </h4>
                  <p className="text-sm text-muted-foreground">Yok</p>
                </div>
              )}

              {/* Dates */}
              <div className="flex justify-between border-t pt-4 text-xs text-muted-foreground/70">
                <span>
                  Oluşturulma:{" "}
                  {new Date(selectedCourse.created_at).toLocaleDateString(
                    "tr-TR"
                  )}
                </span>
                <span>
                  Güncelleme:{" "}
                  {new Date(selectedCourse.updated_at).toLocaleDateString(
                    "tr-TR"
                  )}
                </span>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
