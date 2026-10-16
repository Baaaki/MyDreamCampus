import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { gradesService } from "@/lib/services/grades-service"

/**
 * The student's cumulative GPA from their transcript. A student with no
 * completed credit has no GPA to show yet, which is not the same as 0.00.
 */
export function GpaCard({ studentId }: { studentId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["grades", "transcript", studentId],
    queryFn: () => gradesService.getStudentTranscript(studentId),
  })

  const value = isLoading
    ? "—"
    : isError || !data || data.summary.total_credits === 0
      ? "Veri yok"
      : data.summary.cumulative_gpa.toFixed(2)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium text-gray-500">
          Genel Ortalaması (GNO)
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold text-[#005a87]">{value}</div>
      </CardContent>
    </Card>
  )
}
