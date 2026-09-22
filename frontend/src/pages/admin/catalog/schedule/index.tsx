import { useEffect, useState } from "react"
import { catalogApi } from "@/lib/api-client"
import type { CourseOffering } from "@/lib/types"
import { TIME_SLOTS, DAYS_OF_WEEK } from "@/lib/constants"

export default function SchedulePage() {
  const [courses, setCourses] = useState<CourseOffering[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  useEffect(() => {
    fetchSchedule()
  }, [])

  const fetchSchedule = async () => {
    try {
      const data = await catalogApi.get("schedule/my").json<CourseOffering[]>()
      setCourses(data)
    } catch (err: any) {
      setError(err.message || "Ders programı yüklenemedi")
    } finally {
      setLoading(false)
    }
  }

  // Create a schedule grid
  const createScheduleGrid = () => {
    const grid: { [key: string]: CourseOffering | null } = {}

    // Initialize grid
    for (let day = 1; day <= 5; day++) {
      for (let slot = 1; slot <= 9; slot++) {
        grid[`${day}-${slot}`] = null
      }
    }

    // Fill grid with courses
    courses.forEach((course) => {
      course.schedule.forEach((s) => {
        grid[`${s.day}-${s.slot}`] = course
      })
    })

    return grid
  }

  const scheduleGrid = createScheduleGrid()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-gray-600">Yükleniyor...</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="mx-auto max-w-7xl px-4">
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h1 className="mb-6 text-2xl font-bold text-gray-900">
            Ders Programım
          </h1>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {courses.length === 0 ? (
            <p className="text-gray-600">Kayıtlı ders bulunamadı.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full border-collapse border border-gray-300">
                <thead>
                  <tr className="bg-gray-50">
                    <th className="border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700">
                      Saat
                    </th>
                    {[1, 2, 3, 4, 5].map((day) => (
                      <th
                        key={day}
                        className="border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700"
                      >
                        {DAYS_OF_WEEK[day as keyof typeof DAYS_OF_WEEK]}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((slot) => (
                    <tr key={slot}>
                      <td className="border border-gray-300 bg-gray-50 px-4 py-2 text-sm text-gray-700">
                        {TIME_SLOTS[slot as keyof typeof TIME_SLOTS].label}
                      </td>
                      {[1, 2, 3, 4, 5].map((day) => {
                        const course = scheduleGrid[`${day}-${slot}`]
                        return (
                          <td
                            key={`${day}-${slot}`}
                            className={`border border-gray-300 px-2 py-2 text-xs ${
                              course ? "bg-indigo-50" : ""
                            }`}
                          >
                            {course && (
                              <div>
                                <div className="font-semibold text-indigo-900">
                                  {course.course_code}
                                </div>
                                <div className="text-xs text-gray-600">
                                  {course.instructor}
                                </div>
                              </div>
                            )}
                          </td>
                        )
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="mt-6">
            <button
              onClick={() => window.history.back()}
              className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            >
              Geri Dön
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
