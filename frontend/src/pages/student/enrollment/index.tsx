import { useState, useEffect, useCallback } from "react"
import { enrollmentApi } from "@/lib/api-client"
import type {
  AvailableCourse,
  MyEnrollmentsResponse,
  EnrollmentProgramResponse,
  LatestRejectionResponse,
  RejectionDetail,
} from "@/lib/types"
import CourseList from "@/components/enrollment/CourseList"
import WeeklyScheduleGrid from "@/components/enrollment/WeeklyScheduleGrid"
import Toast from "@/components/enrollment/Toast"

// Determine current semester based on date:
// Sep-Jan → Fall, Feb-Jun → Spring, Jul-Aug → Summer
function getCurrentSemester(): string {
  const now = new Date()
  const month = now.getMonth() + 1 // 1-12
  const year = now.getFullYear()

  if (month >= 9) {
    // Fall semester: Sep-Jan → "2025-2026-Fall"
    return `${year}-${year + 1}-Fall`
  } else if (month <= 1) {
    // Still Fall semester from previous year: Jan → "2024-2025-Fall"
    return `${year - 1}-${year}-Fall`
  } else if (month >= 2 && month <= 6) {
    // Spring semester: Feb-Jun → "2025-2026-Spring"
    return `${year - 1}-${year}-Spring`
  } else {
    // Summer semester: Jul-Aug → "2025-2026-Summer"
    return `${year - 1}-${year}-Summer`
  }
}

interface AvailableCoursesResponse {
  student_id: string
  department: string
  class_level: number
  semester: string
  available_courses: AvailableCourse[]
}

export default function StudentEnrollmentPage() {
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([])
  const [courseColorMap, setCourseColorMap] = useState<Record<string, number>>(
    {}
  )
  const [nextColorIndex, setNextColorIndex] = useState(0)
  const [availableCourses, setAvailableCourses] = useState<AvailableCourse[]>(
    []
  )
  const [, setStudentId] = useState<string>("")
  const [enrollmentStatus, setEnrollmentStatus] = useState<string | null>(null)
  const [rejectionInfo, setRejectionInfo] = useState<RejectionDetail | null>(
    null
  )
  const [, setLoading] = useState(true)
  const [toast, setToast] = useState<{
    message: string
    type: "error" | "warning" | "success" | "info"
    isVisible: boolean
  }>({
    message: "",
    type: "warning",
    isVisible: false,
  })

  const [enrolledCourses, setEnrolledCourses] = useState<AvailableCourse[]>([])

  const fetchLatestRejection = useCallback(async () => {
    try {
      const semesterParam = getCurrentSemester()

      const response = await enrollmentApi
        .get(`latest-rejection?semester=${semesterParam}`)
        .json<LatestRejectionResponse>()

      if (response.has_rejection && response.latest_rejection) {
        setRejectionInfo(response.latest_rejection)
      }
    } catch (err) {
      console.error("Failed to fetch rejection info", err)
    }
  }, [])

  const fetchCourses = useCallback(async () => {
    try {
      const semesterParam = getCurrentSemester()

      const response = await enrollmentApi
        .get(`available-courses?semester=${semesterParam}`)
        .json<AvailableCoursesResponse>()

      setAvailableCourses(response.available_courses || [])
      setStudentId(response.student_id)
    } catch (err: unknown) {
      setAvailableCourses([])
      setToast({
        message: err instanceof Error ? err.message : "Dersler yüklenemedi",
        type: "error",
        isVisible: true,
      })
    }
  }, [])

  const fetchEnrollmentStatus = useCallback(async () => {
    try {
      const semesterParam = getCurrentSemester()

      const response = await enrollmentApi
        .get(`my-enrollments?semester=${semesterParam}`)
        .json<MyEnrollmentsResponse>()

      if (response.programs && response.programs.length > 0) {
        // Assume the most recent one is relevant (or the only one for this semester)
        // Sort by created_at desc just in case
        const program = response.programs.sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )[0]

        setEnrollmentStatus(program.status)

        // If rejected, fetch rejection details
        if (program.status === "rejected") {
          await fetchLatestRejection()
        }

        // Populate selected courses
        const courseIds = program.courses.map((c) => c.id)
        setSelectedCourseIds(courseIds)

        // Convert enrolled CourseBasic to AvailableCourse format for grid display
        const enrolledAsAvailable: AvailableCourse[] = program.courses.map(
          (c) => ({
            id: c.id,
            course_code: c.course_code,
            course_name: c.course_name,
            credits: c.credits,
            schedule_sessions: c.schedule_sessions || [],
            max_capacity: 0,
            current_enrollment: 0,
            available_seats: 0,
            instructor: c.instructor || "",
          })
        )
        setEnrolledCourses(enrolledAsAvailable)

        // Assign colors to these courses
        const newColorMap: Record<string, number> = {}
        courseIds.forEach((id, index) => {
          newColorMap[id] = index
        })
        setCourseColorMap(newColorMap)
        setNextColorIndex(courseIds.length)
      } else {
        // No active enrollment, check for rejections
        await fetchLatestRejection()
      }
    } catch (err) {
      console.error("Failed to fetch enrollments", err)
      // Don't block the page if this fails, just assume no enrollment
      // Still try to check for rejections
      await fetchLatestRejection()
    }
  }, [fetchLatestRejection])

  useEffect(() => {
    const init = async () => {
      setLoading(true)
      await fetchCourses()
      // Fetch enrollment status after courses (though they can be parallel, doing sequential for simplicity)
      await fetchEnrollmentStatus()
      setLoading(false)
    }
    init()
  }, [fetchCourses, fetchEnrollmentStatus])

  // Use enrolled courses for the grid if read-only, otherwise use filtered available courses
  const selectedCourses = enrollmentStatus
    ? enrolledCourses
    : availableCourses.filter((course) => selectedCourseIds.includes(course.id))

  // Check for conflicts
  const checkForConflicts = useCallback(
    (courseIdToAdd: string) => {
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
    },
    [selectedCourseIds, availableCourses]
  )

  // Toggle course selection
  const handleCourseToggle = (courseId: string) => {
    // Read-only for pending or approved status
    if (isReadOnly) return

    // Check if already selected
    if (selectedCourseIds.includes(courseId)) {
      handleRemoveCourse(courseId)
      return
    }

    // Check for conflicts before adding
    const conflicts = checkForConflicts(courseId)
    if (conflicts.length > 0) {
      setToast({
        message: `Çakışma var: ${conflicts.join(", ")}`,
        type: "error",
        isVisible: true,
      })
      return
    }

    setSelectedCourseIds((prev) => [...prev, courseId])

    // Assign a color if not already assigned
    if (!courseColorMap[courseId]) {
      setCourseColorMap((prev) => ({
        ...prev,
        [courseId]: nextColorIndex,
      }))
      setNextColorIndex((prev) => prev + 1)
    }
  }

  // Remove course
  const handleRemoveCourse = (courseId: string) => {
    // Read-only for pending or approved status
    if (isReadOnly) return
    setSelectedCourseIds((prev) => prev.filter((id) => id !== courseId))
  }

  // Submit enrollment
  const handleEnrollment = async () => {
    if (selectedCourseIds.length === 0) {
      setToast({
        message: "Lütfen en az bir ders seçin",
        type: "warning",
        isVisible: true,
      })
      return
    }

    try {
      const semesterParam = getCurrentSemester()

      // Apply the returned program directly — avoids a follow-up GET that can miss the
      // write on a read replica or expose a brief "no enrollment" flicker.
      const program = await enrollmentApi
        .post("programs", {
          json: {
            semester: semesterParam,
            course_ids: selectedCourseIds,
          },
        })
        .json<EnrollmentProgramResponse>()

      setEnrollmentStatus(program.status)
      setRejectionInfo(null)

      const enrolledAsAvailable: AvailableCourse[] = program.courses.map(
        (c) => ({
          id: c.id,
          course_code: c.course_code,
          course_name: c.course_name,
          credits: c.credits,
          schedule_sessions: c.schedule_sessions || [],
          max_capacity: 0,
          current_enrollment: 0,
          available_seats: 0,
          instructor: c.instructor || "",
        })
      )
      setEnrolledCourses(enrolledAsAvailable)

      const newCourseIds = program.courses.map((c) => c.id)
      setSelectedCourseIds(newCourseIds)
      const newColorMap: Record<string, number> = {}
      newCourseIds.forEach((id, index) => {
        newColorMap[id] = index
      })
      setCourseColorMap(newColorMap)
      setNextColorIndex(newCourseIds.length)

      setToast({
        message: "Ders kaydı başarıyla oluşturuldu",
        type: "success",
        isVisible: true,
      })
    } catch (err: unknown) {
      setToast({
        message: err instanceof Error ? err.message : "Kayıt işlemi başarısız",
        type: "error",
        isVisible: true,
      })
    }
  }

  // ReadOnly is true for pending or approved status (not for rejected - user can re-enroll)
  const isReadOnly =
    enrollmentStatus === "pending" || enrollmentStatus === "approved"
  const showSubmitButton = !enrollmentStatus || enrollmentStatus === "rejected"
  const buttonText = "Ders Kaydını Tamamla"

  // Cancel current enrollment and allow re-registration
  const handleCancelEnrollment = async () => {
    // Call DELETE API for both pending enrollment and rejected state
    if (enrollmentStatus === "pending" || rejectionInfo) {
      try {
        const semesterParam = getCurrentSemester()

        await enrollmentApi.delete(`programs?semester=${semesterParam}`)

        setEnrollmentStatus(null)
        setEnrolledCourses([])
        setSelectedCourseIds([])
        setCourseColorMap({})
        setNextColorIndex(0)
        setRejectionInfo(null)

        setToast({
          message: "Yeni ders kaydı yapabilirsiniz",
          type: "success",
          isVisible: true,
        })
      } catch {
        // If DELETE fails (e.g., no enrollment to delete), just clear UI state
        setEnrollmentStatus(null)
        setEnrolledCourses([])
        setSelectedCourseIds([])
        setCourseColorMap({})
        setNextColorIndex(0)
        setRejectionInfo(null)

        setToast({
          message: "Yeni ders kaydı yapabilirsiniz",
          type: "info",
          isVisible: true,
        })
      }
    }
  }

  const closeToast = () => {
    setToast((prev) => ({ ...prev, isVisible: false }))
  }

  return (
    <div className="container mx-auto p-4">
      <h1 className="mb-6 text-3xl font-bold text-gray-800 dark:text-white">
        Ders Kayıt Sistemi
      </h1>

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
          <button
            onClick={handleCancelEnrollment}
            className="rounded-lg bg-amber-600 px-4 py-2 font-medium text-white transition-colors hover:bg-amber-700"
          >
            Yeniden Ders Kaydı Yap
          </button>
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
            <div className="flex items-start justify-between gap-4">
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
              <button
                onClick={handleCancelEnrollment}
                className="flex-shrink-0 rounded-lg bg-red-600 px-4 py-2 font-medium text-white transition-colors hover:bg-red-700"
              >
                Yeniden Ders Kaydı Yap
              </button>
            </div>
          </div>
        )}

      <div className="flex flex-1 overflow-hidden rounded-lg bg-gray-100 dark:bg-gray-900">
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
          <div className="w-80 flex-shrink-0 shadow-lg transition-all duration-300">
            <CourseList
              courses={availableCourses}
              selectedCourseIds={selectedCourseIds}
              onSelectCourse={handleCourseToggle}
            />
          </div>
        )}

        {/* Right Panel - Weekly Schedule Grid */}
        <div className="relative flex-1 shadow-lg">
          <WeeklyScheduleGrid
            sessions={selectedCourses}
            selectedCourseIds={selectedCourseIds}
            courseColorMap={courseColorMap}
            onRemoveCourse={handleRemoveCourse}
            onEnroll={handleEnrollment}
            readOnly={isReadOnly}
            submitButtonText={buttonText}
            showSubmitButton={showSubmitButton}
          />
        </div>
      </div>
    </div>
  )
}
