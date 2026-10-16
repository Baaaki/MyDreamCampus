import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { AlertCircle, CalendarPlus, Loader2 } from "lucide-react"
import CourseHierarchyView from "@/components/course-hierarchy-view"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { parseSemester } from "@/lib/semester"
import { listSemesters } from "@/lib/services/system-service"
import type { Semester } from "@/lib/types"

/** Newest term first; semesters with the same term by creation time. */
function newestFirst(semesters: Semester[]): Semester[] {
  return [...semesters].sort(
    (a, b) =>
      parseSemester(b.name).sortKey - parseSemester(a.name).sortKey ||
      b.created_at.localeCompare(a.created_at)
  )
}

export default function OpenedCoursesPage() {
  const [pickedSemester, setPickedSemester] = useState("")
  const {
    data: semesters = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["system", "semesters"],
    queryFn: listSemesters,
    select: newestFirst,
  })

  const defaultSemester =
    (semesters.find((s) => s.status === "active") ?? semesters[0])?.name ?? ""
  const currentSemester = pickedSemester || defaultSemester

  return (
    <div className="min-h-screen bg-gray-50 py-8 dark:bg-gray-900">
      <div className="mx-auto max-w-[1600px] px-4">
        <div className="rounded-lg bg-white p-4 shadow-md sm:p-6 dark:bg-gray-800">
          <div className="mb-8 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-4">
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

            {semesters.length > 0 && (
              <div className="flex items-center gap-3">
                <label
                  htmlFor="semester-select"
                  className="text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  Dönem
                </label>
                <Select
                  value={currentSemester}
                  onValueChange={setPickedSemester}
                >
                  <SelectTrigger id="semester-select" className="min-w-[220px]">
                    <SelectValue placeholder="Dönem seçin" />
                  </SelectTrigger>
                  <SelectContent>
                    {semesters.map((s) => (
                      <SelectItem key={s.id} value={s.name}>
                        {parseSemester(s.name).label}
                        {s.status === "active" ? " (Aktif)" : ""}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-16 text-gray-500 dark:text-gray-400">
              <Loader2 className="h-5 w-5 animate-spin" />
              Dönemler yükleniyor...
            </div>
          ) : isError ? (
            <div className="flex items-center justify-center gap-2 py-16 text-red-600 dark:text-red-400">
              <AlertCircle className="h-5 w-5" />
              Dönemler yüklenemedi
            </div>
          ) : semesters.length === 0 ? (
            <p className="py-16 text-center text-gray-500 dark:text-gray-400">
              Henüz tanımlı bir dönem yok
            </p>
          ) : (
            <CourseHierarchyView semesterName={currentSemester} />
          )}
        </div>
      </div>
    </div>
  )
}
