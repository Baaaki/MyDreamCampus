import { useState } from "react"
import { CalendarPlus } from "lucide-react"
import CourseHierarchyView from "@/components/course-hierarchy-view"

export default function OpenedCoursesPage() {
  const [currentSemester] = useState("2024-2025-Fall")

  return (
    <div className="min-h-screen bg-gray-50 py-8 dark:bg-gray-900">
      <div className="mx-auto max-w-[1600px] px-4">
        <div className="rounded-lg bg-white p-6 shadow-md dark:bg-gray-800">
          <div className="mb-8 flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-indigo-600 text-white">
              <CalendarPlus className="h-6 w-6" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                Haftalık Ders Programları
              </h1>
              <p className="text-gray-600 dark:text-gray-400">
                Bölüm seçerek haftalık ders programını görüntüleyin
              </p>
            </div>
          </div>

          <CourseHierarchyView semesterName={currentSemester} />
        </div>
      </div>
    </div>
  )
}
