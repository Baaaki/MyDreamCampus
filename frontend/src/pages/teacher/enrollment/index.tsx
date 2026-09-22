import { useEffect, useState } from "react"
import { enrollmentService } from "@/lib/services/enrollment-service"
import type { EnrollmentProgramResponse } from "@/lib/types"
import { EnrollmentReviewDialog } from "@/components/enrollment/enrollment-review-dialog"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Calendar, User, BookOpen, Clock } from "lucide-react"

export default function EnrollmentPage() {
  const [enrollments, setEnrollments] = useState<EnrollmentProgramResponse[]>(
    []
  )
  const [loading, setLoading] = useState(true)
  const [selectedEnrollment, setSelectedEnrollment] =
    useState<EnrollmentProgramResponse | null>(null)
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  useEffect(() => {
    fetchEnrollments()
  }, [])

  const fetchEnrollments = async () => {
    try {
      setLoading(true)
      const response = await enrollmentService.getPendingEnrollments()
      setEnrollments(response.programs)
    } catch (error) {
      console.error("Failed to fetch enrollments:", error)
    } finally {
      setLoading(false)
    }
  }

  const handleReview = (enrollment: EnrollmentProgramResponse) => {
    setSelectedEnrollment(enrollment)
    setIsDialogOpen(true)
  }

  const handleApprove = async (programId: string) => {
    await enrollmentService.approveEnrollment(programId)
    setEnrollments((prev) => prev.filter((p) => p.id !== programId))
  }

  const handleReject = async (programId: string, reason: string) => {
    await enrollmentService.rejectEnrollment(programId, reason)
    setEnrollments((prev) => prev.filter((p) => p.id !== programId))
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent"></div>
          <p className="mt-2 text-gray-500">Yükleniyor...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto space-y-8 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
          Ders Kaydı Onaylama
        </h1>
        <p className="mt-2 text-gray-500 dark:text-gray-400">
          Danışmanı olduğunuz öğrencilerin ders kayıt taleplerini buradan
          inceleyebilirsiniz.
        </p>
      </div>

      {enrollments.length === 0 ? (
        <Card className="border-dashed bg-gray-50">
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <div className="mb-4 rounded-full bg-white p-4 shadow-sm">
              <CheckAllIcon className="h-8 w-8 text-green-600" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900">
              Bekleyen Talep Yok
            </h3>
            <p className="mt-1 max-w-sm text-gray-500">
              Şu an onayınızı bekleyen herhangi bir ders kaydı talebi
              bulunmamaktadır.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {enrollments.map((enrollment) => (
            <Card
              key={enrollment.id}
              className="transition-shadow hover:shadow-md"
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <CardTitle className="flex items-center gap-2 text-lg font-semibold">
                      {enrollment.student_name || "İsimsiz Öğrenci"}
                    </CardTitle>
                    <CardDescription className="flex items-center gap-1">
                      <User className="h-3 w-3" />
                      {enrollment.student_number}
                    </CardDescription>
                  </div>
                  <Badge
                    variant="outline"
                    className="border-yellow-200 bg-yellow-50 text-yellow-700"
                  >
                    Bekliyor
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="pb-3">
                <div className="space-y-2 text-sm text-gray-600 dark:text-gray-300">
                  <div className="flex items-center gap-2">
                    <BookOpen className="h-4 w-4 text-gray-400" />
                    <span>{enrollment.department}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-gray-400" />
                    <span>
                      {enrollment.class_level}. Sınıf - {enrollment.semester}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Clock className="h-4 w-4 text-gray-400" />
                    <span>
                      {new Date(enrollment.created_at).toLocaleDateString(
                        "tr-TR"
                      )}
                    </span>
                  </div>
                </div>

                <div className="mt-4 flex items-center justify-between border-t pt-4 text-sm">
                  <span className="font-medium text-gray-700">
                    Toplam Ders:
                  </span>
                  <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-700">
                    {enrollment.courses.length}
                  </span>
                </div>
              </CardContent>
              <CardFooter className="pt-3">
                <Button
                  className="w-full bg-blue-600 hover:bg-blue-700"
                  onClick={() => handleReview(enrollment)}
                >
                  İncele
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}

      <EnrollmentReviewDialog
        program={selectedEnrollment}
        isOpen={isDialogOpen}
        onClose={() => setIsDialogOpen(false)}
        onApprove={handleApprove}
        onReject={handleReject}
      />
    </div>
  )
}

function CheckAllIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m9 11 3 3L22 4" />
      <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
    </svg>
  )
}
