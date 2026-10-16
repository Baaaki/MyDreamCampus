import { useQuery } from "@tanstack/react-query"
import { AlertCircle, Loader2, Scale } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { courseResultsKey, gradesService } from "@/lib/services/grades-service"
import type { AssessmentStatus, CourseResult } from "@/lib/types"

interface CourseResultsTableProps {
  courseId: string
  assessments: AssessmentStatus[]
  onAppeal: (result: CourseResult) => void
}

/**
 * The frozen results of a finalized course, one row per student, each with
 * the button that opens an appeal for it.
 */
export function CourseResultsTable({
  courseId,
  assessments,
  onAppeal,
}: CourseResultsTableProps) {
  const {
    data: results = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: courseResultsKey(courseId),
    queryFn: () => gradesService.getCourseResults(courseId),
  })
  const columnCount = 5 + assessments.length

  return (
    <div className="overflow-x-auto rounded-lg border bg-white dark:border-gray-800 dark:bg-gray-900">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="whitespace-nowrap">Öğrenci No</TableHead>
            <TableHead className="whitespace-nowrap">Ad Soyad</TableHead>
            {assessments.map((a) => (
              <TableHead key={a.slug} className="text-center whitespace-nowrap">
                {a.name}
              </TableHead>
            ))}
            <TableHead className="text-center whitespace-nowrap">
              Ortalama
            </TableHead>
            <TableHead className="text-center whitespace-nowrap">
              Katsayı
            </TableHead>
            <TableHead className="text-right whitespace-nowrap">
              İtiraz
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={columnCount} className="h-24 text-center">
                <Loader2 className="mx-auto h-6 w-6 animate-spin text-blue-600" />
              </TableCell>
            </TableRow>
          ) : isError ? (
            <TableRow>
              <TableCell
                colSpan={columnCount}
                className="h-24 text-center text-destructive"
              >
                <span className="inline-flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  Ders sonuçları alınamadı.
                </span>
              </TableCell>
            </TableRow>
          ) : results.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={columnCount}
                className="h-24 text-center text-gray-500"
              >
                Öğrenci bulunamadı.
              </TableCell>
            </TableRow>
          ) : (
            results.map((result) => (
              <TableRow key={result.student_id}>
                <TableCell className="font-mono text-sm">
                  {result.student_number}
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  {result.first_name} {result.last_name}
                </TableCell>
                {assessments.map((a) => (
                  <TableCell key={a.slug} className="text-center">
                    {result.scores[a.slug] ?? "-"}
                  </TableCell>
                ))}
                <TableCell className="text-center">
                  {result.weighted_average.toFixed(2)}
                </TableCell>
                <TableCell className="text-center font-semibold">
                  {result.grade_point}
                  {result.is_attendance_failed && (
                    <span className="ml-1 text-xs font-normal text-destructive">
                      (devamsız)
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onAppeal(result)}
                  >
                    <Scale className="mr-1 h-3 w-3" /> Not İtirazı
                  </Button>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  )
}
