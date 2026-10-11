import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, CheckCircle2, Loader2, Lock } from "lucide-react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { apiErrorMessage } from "@/lib/api-error"
import {
  attendanceFinalizationKey,
  finalizeAttendance,
  useAttendanceFinalization,
} from "@/lib/services/attendance-service"
import type {
  AttendanceFailedStudent,
  AttendanceFinalization,
  SessionTypeAttendance,
} from "@/lib/types"

const FAILED_TYPE_LABELS: Record<
  AttendanceFailedStudent["failed_type"],
  string
> = {
  theory: "Teorik",
  lab: "Uygulama",
  both: "Teorik ve Uygulama",
}

interface AttendanceFinalizeSectionProps {
  courseId: string
  hasLab: boolean
}

/**
 * Finalizes the course's attendance once, after a confirmation, and from then
 * on shows the saved outcome instead of the button.
 */
export function AttendanceFinalizeSection({
  courseId,
  hasLab,
}: AttendanceFinalizeSectionProps) {
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const {
    data: status,
    isLoading,
    isError,
  } = useAttendanceFinalization(courseId)

  const finalize = useMutation({
    mutationFn: () => finalizeAttendance(courseId),
    onSuccess: (result) => {
      queryClient.setQueryData(attendanceFinalizationKey(courseId), {
        finalized: true,
        result,
      })
    },
    // A 409 may mean another tab finalized first; the reread shows its result.
    onError: () => {
      queryClient.invalidateQueries({
        queryKey: attendanceFinalizationKey(courseId),
      })
    },
  })

  if (isLoading) {
    return (
      <div className="flex justify-center rounded-xl border border-gray-200 bg-white p-8 dark:border-gray-800 dark:bg-gray-900">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex items-center gap-2 rounded-xl border border-gray-200 bg-white p-6 text-sm text-destructive dark:border-gray-800 dark:bg-gray-900">
        <AlertCircle className="h-4 w-4 shrink-0" />
        Yoklamanın kesinleştirme durumu alınamadı.
      </div>
    )
  }

  if (status?.finalized && status.result) {
    return <FinalizationResult result={status.result} hasLab={hasLab} />
  }

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Yoklamayı Kesinleştir
          </h2>
          <p className="max-w-2xl text-sm text-gray-500 dark:text-gray-400">
            Dönem sonunda yoklamayı kesinleştirin. Teorik derslerin en az
            10/14&apos;üne
            {hasLab ? ", uygulamaların en az 11/14'üne" : ""} katılmayan
            öğrenciler devamsızlıktan kalır ve not sistemine bildirilir.
          </p>
        </div>
        <Button
          variant="destructive"
          onClick={() => setConfirmOpen(true)}
          disabled={finalize.isPending}
        >
          {finalize.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Lock className="mr-2 h-4 w-4" />
          )}
          Yoklamayı Kesinleştir
        </Button>
      </div>
      {finalize.isError && (
        <p className="mt-4 flex items-center gap-2 text-sm text-destructive">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {apiErrorMessage(finalize.error, "Yoklama kesinleştirilemedi")}
        </p>
      )}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Yoklama kesinleştirilsin mi?</AlertDialogTitle>
            <AlertDialogDescription>
              Eşiğin altında kalan öğrenciler devamsızlıktan kalmış sayılır ve
              not sistemine bildirilir. Bu işlem geri alınamaz; sonrasında bu
              ders için yeni yoklama oturumu açılamaz.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Vazgeç</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => finalize.mutate()}
            >
              Kesinleştir
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function FinalizationResult({
  result,
  hasLab,
}: {
  result: AttendanceFinalization
  hasLab: boolean
}) {
  const { thresholds, finalization_summary: summary } = result
  const finalizedAt = new Date(result.finalized_at).toLocaleString("tr-TR", {
    dateStyle: "long",
    timeStyle: "short",
  })

  return (
    <div className="space-y-4 rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
      <div className="flex items-center gap-2">
        <CheckCircle2 className="h-5 w-5 text-green-600" />
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
          Yoklama kesinleştirildi
        </h2>
      </div>
      <p className="text-sm text-gray-500 dark:text-gray-400">
        {finalizedAt}
        {thresholds.theory_min_required > 0 &&
          ` · Teorik eşik: en az ${thresholds.theory_min_required} oturum`}
        {hasLab &&
          thresholds.lab_min_required > 0 &&
          ` · Uygulama eşiği: en az ${thresholds.lab_min_required} oturum`}
      </p>
      <dl className="grid grid-cols-3 gap-2 text-center text-sm">
        <Count label="Öğrenci" value={result.total_students} />
        <Count label="Geçen" value={summary.passing_count} />
        <Count label="Devamsızlıktan Kalan" value={summary.failing_count} />
      </dl>
      {result.failed_students.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Devamsızlıktan kalan öğrenci yok.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Öğrenci No</TableHead>
                <TableHead>Ad Soyad</TableHead>
                <TableHead>Kaldığı Tür</TableHead>
                <TableHead>Teorik</TableHead>
                {hasLab && <TableHead>Uygulama</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.failed_students.map((student) => (
                <TableRow key={student.student_id}>
                  <TableCell>{student.student_number}</TableCell>
                  <TableCell>{student.student_name}</TableCell>
                  <TableCell>
                    {FAILED_TYPE_LABELS[student.failed_type]}
                  </TableCell>
                  <TableCell>{presence(student.theory)}</TableCell>
                  {hasLab && <TableCell>{presence(student.lab)}</TableCell>}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}

function presence(stats: SessionTypeAttendance | undefined) {
  return stats ? `${stats.present_count} / ${stats.total_sessions}` : "-"
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border p-3">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-2xl font-semibold">{value}</dd>
    </div>
  )
}
