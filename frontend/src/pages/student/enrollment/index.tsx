import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, Loader2 } from "lucide-react"
import { apiErrorMessage } from "@/lib/api-error"
import {
  defaultSemester,
  enrollmentService,
  semesterLabel,
} from "@/lib/services/enrollment-service"
import type { AvailableCourse, CourseBasic, OpenSemester } from "@/lib/types"
import { Button } from "@/components/ui/button"
import CourseList from "@/components/enrollment/CourseList"
import WeeklyScheduleGrid from "@/components/enrollment/WeeklyScheduleGrid"
import Toast from "@/components/enrollment/Toast"

export default function StudentEnrollmentPage() {
  const semestersQuery = useQuery({
    queryKey: ["enrollment", "open-semesters"],
    queryFn: () => enrollmentService.getOpenSemesters(),
  })
  const [chosenSemester, setChosenSemester] = useState<string | null>(null)

  const semesters = semestersQuery.data ?? []
  const wanted = chosenSemester ?? defaultSemester(semesters)
  const current = semesters.find((s) => s.semester === wanted)

  return (
    <div className="container mx-auto p-4">
      <h1 className="mb-6 text-3xl font-bold text-gray-800 dark:text-white">
        Ders Kayıt Sistemi
      </h1>

      {semestersQuery.isLoading ? (
        <div className="flex items-center gap-2 text-gray-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          Kayıt dönemleri yükleniyor...
        </div>
      ) : semestersQuery.isError ? (
        <div className="flex items-center gap-2 text-destructive">
          <AlertCircle className="h-4 w-4" />
          {apiErrorMessage(semestersQuery.error, "Kayıt dönemleri yüklenemedi")}
        </div>
      ) : !current ? (
        <p className="text-gray-600 dark:text-gray-400">
          Şu anda ders kaydına açık bir dönem bulunmuyor.
        </p>
      ) : (
        <>
          <SemesterPicker
            semesters={semesters}
            value={current.semester}
            onChange={setChosenSemester}
          />
          {/* Keyed so a switch starts the next semester from a clean slate
              and a late reply for the previous one has nowhere to land. */}
          <EnrollmentWorkspace
            key={current.semester}
            semester={current.semester}
            enrollmentOpen={current.enrollment_open}
          />
        </>
      )}
    </div>
  )
}

function SemesterPicker({
  semesters,
  value,
  onChange,
}: {
  semesters: OpenSemester[]
  value: string
  onChange: (semester: string) => void
}) {
  return (
    <div
      role="group"
      aria-label="Dönem"
      className="mb-4 flex flex-wrap items-center gap-2"
    >
      {semesters.map((s) => (
        <Button
          key={s.semester}
          variant={s.semester === value ? "default" : "outline"}
          aria-pressed={s.semester === value}
          onClick={() => onChange(s.semester)}
        >
          {semesterLabel(s.semester)}
          <span className="text-xs opacity-75">
            {s.enrollment_open ? "Kayıt açık" : "Kayıt kapalı"}
          </span>
        </Button>
      ))}
    </div>
  )
}

type ToastType = "error" | "warning" | "success" | "info"

// A program's course in the shape the schedule grid draws.
function toAvailableCourse(course: CourseBasic): AvailableCourse {
  return {
    id: course.id,
    course_code: course.course_code,
    course_name: course.course_name,
    credits: course.credits,
    schedule_sessions: course.schedule_sessions ?? [],
    max_capacity: 0,
    current_enrollment: 0,
    available_seats: 0,
    instructor: course.instructor ?? "",
  }
}

function EnrollmentWorkspace({
  semester,
  enrollmentOpen,
}: {
  semester: string
  enrollmentOpen: boolean
}) {
  const queryClient = useQueryClient()
  const programKey = ["enrollment", "latest-program", semester]
  const rejectionKey = ["enrollment", "latest-rejection", semester]

  const coursesQuery = useQuery({
    queryKey: ["enrollment", "available-courses", semester],
    queryFn: () => enrollmentService.getAvailableCourses(semester),
    // A closed window only shows the program; the backend would refuse the
    // course list anyway.
    enabled: enrollmentOpen,
  })
  const programQuery = useQuery({
    queryKey: programKey,
    queryFn: () => enrollmentService.getLatestProgram(semester),
  })
  // A failed read must not block the page; it then offers a fresh program.
  const program = programQuery.data ?? null
  const enrollmentStatus = program?.status ?? null
  const rejectionQuery = useQuery({
    queryKey: rejectionKey,
    queryFn: () => enrollmentService.getLatestRejection(semester),
    enabled:
      !programQuery.isPending && (!program || program.status === "rejected"),
    // Only this page's own submit and withdraw change it, and they write it.
    staleTime: Infinity,
  })
  const rejectionInfo = rejectionQuery.data ?? null
  const availableCourses = coursesQuery.data ?? []

  // null until the student picks; until then a rejected program's courses
  // stay chosen so it can be adjusted and sent again.
  const [picked, setPicked] = useState<string[] | null>(null)
  const [courseColorMap, setCourseColorMap] = useState<Record<string, number>>(
    {}
  )
  const [toast, setToast] = useState<{
    message: string
    type: ToastType
    isVisible: boolean
  }>({
    message: "",
    type: "warning",
    isVisible: false,
  })
  const showToast = (message: string, type: ToastType) =>
    setToast({ message, type, isVisible: true })

  // ReadOnly is true for pending or approved status (not for rejected - user can re-enroll)
  const isReadOnly =
    !enrollmentOpen ||
    enrollmentStatus === "pending" ||
    enrollmentStatus === "approved"
  const showSubmitButton =
    enrollmentOpen && (!enrollmentStatus || enrollmentStatus === "rejected")
  const buttonText = "Ders Kaydını Tamamla"

  const programCourses = (program?.courses ?? []).map(toAvailableCourse)
  const programCourseIds = programCourses.map((c) => c.id)
  const selectedCourseIds = isReadOnly
    ? programCourseIds
    : (picked ?? (enrollmentStatus === "rejected" ? programCourseIds : []))
  const selectedCourses = isReadOnly
    ? programCourses
    : availableCourses.filter((course) => selectedCourseIds.includes(course.id))
  const colors = Object.fromEntries(
    selectedCourseIds.map((id, index) => [id, courseColorMap[id] ?? index])
  )

  // Check for conflicts
  const checkForConflicts = (courseIdToAdd: string) => {
    const newCourse = availableCourses.find((c) => c.id === courseIdToAdd)
    if (!newCourse) return []

    const existingCourses = availableCourses.filter((c) =>
      selectedCourseIds.includes(c.id)
    )

    const conflicts: string[] = []

    newCourse.schedule_sessions.forEach((newSession) => {
      existingCourses.forEach((existingCourse) => {
        existingCourse.schedule_sessions.forEach((existingSession) => {
          if (
            newSession.day_of_week.toLowerCase() ===
            existingSession.day_of_week.toLowerCase()
          ) {
            const commonSlots = newSession.slot_numbers.filter((slot) =>
              existingSession.slot_numbers.includes(slot)
            )

            if (commonSlots.length > 0) {
              const conflictStr = `${newCourse.course_code} & ${existingCourse.course_code}`
              if (!conflicts.includes(conflictStr)) {
                conflicts.push(conflictStr)
              }
            }
          }
        })
      })
    })

    return conflicts
  }

  // Toggle course selection
  const handleCourseToggle = (courseId: string) => {
    if (isReadOnly) return

    if (selectedCourseIds.includes(courseId)) {
      handleRemoveCourse(courseId)
      return
    }

    const conflicts = checkForConflicts(courseId)
    if (conflicts.length > 0) {
      showToast(`Çakışma var: ${conflicts.join(", ")}`, "error")
      return
    }

    setPicked([...selectedCourseIds, courseId])

    // A course keeps its color if it is dropped and picked again.
    if (courseColorMap[courseId] === undefined) {
      const nextColor = Math.max(-1, ...Object.values(colors)) + 1
      setCourseColorMap((prev) => ({ ...prev, [courseId]: nextColor }))
    }
  }

  const handleRemoveCourse = (courseId: string) => {
    if (isReadOnly) return
    setPicked(selectedCourseIds.filter((id) => id !== courseId))
  }

  const submit = useMutation({
    mutationFn: (courseIds: string[]) =>
      enrollmentService.submitProgram(semester, courseIds),
    // Apply the returned program directly — avoids a follow-up GET that can
    // miss the write on a read replica or expose a brief "no enrollment"
    // flicker.
    onSuccess: (created) => {
      queryClient.setQueryData(programKey, created)
      queryClient.setQueryData(rejectionKey, null)
      setPicked(null)
      showToast("Ders kaydı başarıyla oluşturuldu", "success")
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Kayıt işlemi başarısız"), "error"),
  })

  const handleEnrollment = () => {
    if (submit.isPending) return
    if (selectedCourseIds.length === 0) {
      showToast("Lütfen en az bir ders seçin", "warning")
      return
    }
    submit.mutate(selectedCourseIds)
  }

  const withdraw = useMutation({
    mutationFn: () => enrollmentService.withdrawProgram(semester),
    onSuccess: () => showToast("Yeni ders kaydı yapabilirsiniz", "success"),
    // Nothing left to withdraw still leaves the student free to start over.
    onError: () => showToast("Yeni ders kaydı yapabilirsiniz", "info"),
    onSettled: () => {
      queryClient.setQueryData(programKey, null)
      queryClient.setQueryData(rejectionKey, null)
      setPicked(null)
      setCourseColorMap({})
    },
  })

  // Cancel current enrollment and allow re-registration
  const handleCancelEnrollment = () => {
    if (withdraw.isPending) return
    if (enrollmentStatus === "pending" || rejectionInfo) withdraw.mutate()
  }

  const closeToast = () => {
    setToast((prev) => ({ ...prev, isVisible: false }))
  }

  return (
    <>
      {/* Pending Status Banner */}
      {enrollmentStatus === "pending" && (
        <div className="mb-4 flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-900/30">
          <div className="flex items-center gap-3">
            <svg
              className="h-6 w-6 text-amber-600 dark:text-amber-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <span className="font-medium text-amber-800 dark:text-amber-200">
              Ders kaydınız danışman hoca tarafından onaylanmayı bekliyor
            </span>
          </div>
          {enrollmentOpen && (
            <button
              onClick={handleCancelEnrollment}
              className="rounded-lg bg-amber-600 px-4 py-2 font-medium text-white transition-colors hover:bg-amber-700"
            >
              Yeniden Ders Kaydı Yap
            </button>
          )}
        </div>
      )}

      {/* Approved Status Banner */}
      {enrollmentStatus === "approved" && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-700 dark:bg-green-900/30">
          <svg
            className="h-6 w-6 text-green-600 dark:text-green-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span className="font-medium text-green-800 dark:text-green-200">
            Ders kaydınız danışman hoca tarafından onaylandı
          </span>
        </div>
      )}

      {/* Rejected Status Banner */}
      {(enrollmentStatus === "rejected" ||
        (!enrollmentStatus && rejectionInfo)) &&
        rejectionInfo && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-700 dark:bg-red-900/30">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex flex-1 items-start gap-3">
                <svg
                  className="mt-0.5 h-6 w-6 flex-shrink-0 text-red-600 dark:text-red-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <div className="flex-1">
                  <p className="mb-2 font-medium text-red-800 dark:text-red-200">
                    Ders kaydınız danışman hoca tarafından reddedildi
                  </p>
                  <div className="mb-2 rounded-lg bg-red-100 p-3 dark:bg-red-900/50">
                    <p className="mb-1 text-sm font-medium text-red-700 dark:text-red-300">
                      Ret Sebebi:
                    </p>
                    <p className="text-red-800 dark:text-red-200">
                      {rejectionInfo.rejection_reason}
                    </p>
                  </div>
                  <p className="text-sm text-red-600 dark:text-red-400">
                    Reddeden: {rejectionInfo.advisor_fullname} •{" "}
                    {new Date(rejectionInfo.rejected_at).toLocaleDateString(
                      "tr-TR"
                    )}
                  </p>
                </div>
              </div>
              {enrollmentOpen && (
                <button
                  onClick={handleCancelEnrollment}
                  className="flex-shrink-0 self-start rounded-lg bg-red-600 px-4 py-2 font-medium text-white transition-colors hover:bg-red-700"
                >
                  Yeniden Ders Kaydı Yap
                </button>
              )}
            </div>
          </div>
        )}

      {!isReadOnly && coursesQuery.isError && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-700 dark:bg-red-900/30 dark:text-red-300">
          <AlertCircle className="h-4 w-4" />
          {apiErrorMessage(coursesQuery.error, "Dersler yüklenemedi")}
        </div>
      )}

      {programQuery.isPending ? (
        <div className="flex items-center gap-2 text-gray-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          Ders kaydı yükleniyor...
        </div>
      ) : (
        <div className="flex flex-1 flex-col overflow-hidden rounded-lg bg-gray-100 lg:flex-row dark:bg-gray-900">
          {/* Toast Notification */}
          <Toast
            message={toast.message}
            type={toast.type}
            isVisible={toast.isVisible}
            onClose={closeToast}
            duration={5000}
          />

          {/* Left Panel - Course List (hidden when enrolled) */}
          {!isReadOnly && (
            <div className="h-[50vh] w-full flex-shrink-0 shadow-lg transition-all duration-300 lg:h-auto lg:w-80">
              <CourseList
                courses={availableCourses}
                selectedCourseIds={selectedCourseIds}
                onSelectCourse={handleCourseToggle}
              />
            </div>
          )}

          {/* Right Panel - Weekly Schedule Grid */}
          <div className="relative min-w-0 flex-1 shadow-lg">
            <WeeklyScheduleGrid
              sessions={selectedCourses}
              selectedCourseIds={selectedCourseIds}
              courseColorMap={colors}
              onRemoveCourse={handleRemoveCourse}
              onEnroll={handleEnrollment}
              readOnly={isReadOnly}
              submitButtonText={buttonText}
              showSubmitButton={showSubmitButton}
            />
          </div>
        </div>
      )}
    </>
  )
}
