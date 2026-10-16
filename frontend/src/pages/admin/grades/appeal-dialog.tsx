import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, ArrowRight, CheckCircle2, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { apiErrorMessage } from "@/lib/api-error"
import {
  courseResultsKey,
  gradesService,
  toAppealRequest,
  validateAppealForm,
  type AppealForm,
  type AppealFormErrors,
} from "@/lib/services/grades-service"
import type {
  AssessmentStatus,
  CourseResult,
  GradeAppealResponse,
} from "@/lib/types"

const EMPTY_FORM: AppealForm = { slug: "", newScore: "", reason: "" }

interface GradeAppealDialogProps {
  courseId: string
  student: CourseResult | null
  assessments: AssessmentStatus[]
  onOpenChange: (open: boolean) => void
}

/**
 * Processes a grade appeal on a finalized course: one assessment score is
 * corrected and the grade point recomputed against the frozen class
 * statistics. Open while `student` is set.
 */
export function GradeAppealDialog({
  courseId,
  student,
  assessments,
  onOpenChange,
}: GradeAppealDialogProps) {
  const [form, setForm] = useState<AppealForm>(EMPTY_FORM)
  const [errors, setErrors] = useState<AppealFormErrors>({})

  const queryClient = useQueryClient()
  const appeal = useMutation({
    mutationFn: gradesService.processAppeal,
    // The results table reads the same frozen record the appeal changed.
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: courseResultsKey(courseId),
      }),
  })

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setForm(EMPTY_FORM)
      setErrors({})
      appeal.reset()
    }
    onOpenChange(open)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!student) return
    const found = validateAppealForm(form)
    setErrors(found)
    if (Object.keys(found).length > 0) return
    appeal.mutate(toAppealRequest(form, student.student_id, courseId))
  }

  const currentScore = form.slug ? student?.scores[form.slug] : undefined
  const assessmentName = (slug: string) =>
    assessments.find((a) => a.slug === slug)?.name ?? slug

  return (
    <Dialog open={!!student} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Not İtirazı</DialogTitle>
          <DialogDescription>
            {student?.first_name} {student?.last_name} (
            {student?.student_number}) için bir değerlendirme notunu düzeltin.
            Katsayı, ders kesinleştirilirken dondurulan sınıf istatistikleriyle
            yeniden hesaplanır.
          </DialogDescription>
        </DialogHeader>

        {appeal.data ? (
          <AppealResult
            result={appeal.data}
            assessmentName={assessmentName(appeal.data.slug)}
            onClose={() => handleOpenChange(false)}
          />
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            <div className="space-y-2">
              <Label htmlFor="appeal_slug">Değerlendirme</Label>
              <select
                id="appeal_slug"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
                value={form.slug}
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
              >
                <option value="">Seçin</option>
                {assessments.map((a) => (
                  <option key={a.slug} value={a.slug}>
                    {a.name}
                  </option>
                ))}
              </select>
              {errors.slug && <FieldError message={errors.slug} />}
              {currentScore !== undefined && (
                <p className="text-sm text-muted-foreground">
                  Mevcut not: {currentScore}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="appeal_score">Yeni Not (0-100)</Label>
              <Input
                id="appeal_score"
                inputMode="decimal"
                value={form.newScore}
                onChange={(e) => setForm({ ...form, newScore: e.target.value })}
              />
              {errors.newScore && <FieldError message={errors.newScore} />}
            </div>
            <div className="space-y-2">
              <Label htmlFor="appeal_reason">Gerekçe</Label>
              <Textarea
                id="appeal_reason"
                rows={3}
                value={form.reason}
                onChange={(e) => setForm({ ...form, reason: e.target.value })}
              />
              {errors.reason && <FieldError message={errors.reason} />}
            </div>
            {appeal.isError && (
              <FieldError
                message={apiErrorMessage(appeal.error, "İtiraz işlenemedi")}
              />
            )}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
              >
                İptal
              </Button>
              <Button type="submit" disabled={appeal.isPending}>
                {appeal.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                İtirazı İşle
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}

function AppealResult({
  result,
  assessmentName,
  onClose,
}: {
  result: GradeAppealResponse
  assessmentName: string
  onClose: () => void
}) {
  return (
    <div className="space-y-4">
      <p className="flex items-center gap-2 text-sm font-medium">
        <CheckCircle2 className="h-4 w-4 shrink-0 text-green-600" />
        İtiraz işlendi; sonuç öğrencinin transkriptine yansıdı.
      </p>
      <dl className="space-y-2 rounded-md border p-4 text-sm">
        <Change
          label={assessmentName}
          from={result.old_score}
          to={result.new_score}
        />
        <Change
          label="Ağırlıklı ortalama"
          from={result.old_weighted_average.toFixed(2)}
          to={result.new_weighted_average.toFixed(2)}
        />
        <Change
          label="Katsayı"
          from={result.old_grade_point}
          to={result.new_grade_point}
        />
      </dl>
      <DialogFooter>
        <Button type="button" onClick={onClose}>
          Kapat
        </Button>
      </DialogFooter>
    </div>
  )
}

function Change({
  label,
  from,
  to,
}: {
  label: string
  from: string | number
  to: string | number
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="flex items-center gap-2 font-medium">
        <span>{from}</span>
        <ArrowRight
          className="h-3.5 w-3.5 text-muted-foreground"
          aria-label="yeni"
        />
        <span>{to}</span>
      </dd>
    </div>
  )
}

function FieldError({ message }: { message: string }) {
  return (
    <p className="flex items-center gap-2 text-sm text-destructive">
      <AlertCircle className="h-4 w-4 shrink-0" />
      {message}
    </p>
  )
}
