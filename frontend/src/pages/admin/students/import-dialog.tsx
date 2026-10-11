import { useEffect, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { AlertCircle, CheckCircle2, Loader2, Upload } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  STUDENT_IMPORT_HEADER,
  isImportFinished,
  studentImportService,
  useStudentImportJob,
} from "@/lib/services/student-import-service"
import type { StudentImportJob } from "@/lib/types"

/**
 * Uploads a student CSV and follows the background job to its result. The
 * job keeps polling while the dialog is closed, so closing it mid-run and
 * reopening shows where the import got to.
 */
export function StudentImportDialog() {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [jobId, setJobId] = useState<string | null>(null)

  const upload = useMutation({
    mutationFn: studentImportService.start,
    onSuccess: setJobId,
  })
  const { data: job, isError: isPollError } = useStudentImportJob(jobId)
  const finished = isImportFinished(job?.status)

  useEffect(() => {
    if (finished) {
      queryClient.invalidateQueries({ queryKey: ["students", "list"] })
    }
  }, [finished, jobId, queryClient])

  const reset = () => {
    setFile(null)
    setJobId(null)
    upload.reset()
  }

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next && (finished || !jobId)) reset()
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (file) upload.mutate(file)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Upload className="mr-2 h-4 w-4" /> Toplu İçe Aktar
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Öğrencileri Toplu İçe Aktar</DialogTitle>
          <DialogDescription>
            Birden fazla öğrenciyi tek seferde eklemek için bir CSV dosyası
            yükleyin.
          </DialogDescription>
        </DialogHeader>

        {!jobId ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="csv_file">CSV Dosyası</Label>
              <Input
                id="csv_file"
                type="file"
                accept=".csv"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              <p className="text-sm text-muted-foreground">
                İlk satır tam olarak şu başlık olmalı:
              </p>
              <code className="block overflow-x-auto rounded-md bg-muted px-3 py-2 text-xs whitespace-nowrap">
                {STUDENT_IMPORT_HEADER}
              </code>
              <p className="text-sm text-muted-foreground">
                Eklenen öğrenciler e-posta adresleriyle giriş yapar; ilk
                şifreleri de e-posta adresleridir ve ilk girişte değiştirmeleri
                istenir.
              </p>
            </div>
            {upload.isError && (
              <p className="flex items-center gap-2 text-sm text-destructive">
                <AlertCircle className="h-4 w-4 shrink-0" />
                {apiErrorMessage(upload.error, "İçe aktarma başlatılamadı")}
              </p>
            )}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
              >
                İptal
              </Button>
              <Button type="submit" disabled={upload.isPending || !file}>
                {upload.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                İçe Aktar
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <div className="space-y-4">
            {isPollError && !job ? (
              <p className="flex items-center gap-2 text-sm text-destructive">
                <AlertCircle className="h-4 w-4 shrink-0" />
                İçe aktarma durumu alınamadı.
              </p>
            ) : !job || !finished ? (
              <ImportProgress job={job} />
            ) : (
              <ImportResult job={job} />
            )}
            <DialogFooter>
              {finished && (
                <Button type="button" variant="outline" onClick={reset}>
                  Yeni Dosya Yükle
                </Button>
              )}
              <Button type="button" onClick={() => handleOpenChange(false)}>
                Kapat
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function ImportProgress({ job }: { job: StudentImportJob | undefined }) {
  return (
    <div className="flex items-center gap-3 rounded-md border p-4 text-sm">
      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      <span>
        Öğrenciler ekleniyor
        {job && job.total_records > 0
          ? `: ${job.processed_records} / ${job.total_records}`
          : "..."}
      </span>
    </div>
  )
}

function ImportResult({ job }: { job: StudentImportJob }) {
  const rowErrors = [...(job.errors ?? [])].sort((a, b) => a.row - b.row)
  const failed = job.status === "failed"

  return (
    <div className="space-y-4">
      <div
        className={
          failed
            ? "flex items-center gap-2 text-sm font-medium text-destructive"
            : "flex items-center gap-2 text-sm font-medium"
        }
      >
        {failed ? (
          <AlertCircle className="h-4 w-4 shrink-0" />
        ) : (
          <CheckCircle2 className="h-4 w-4 shrink-0" />
        )}
        {failed
          ? "İçe aktarma başarısız: hiçbir öğrenci eklenemedi."
          : "İçe aktarma tamamlandı."}
      </div>
      <dl className="grid grid-cols-3 gap-2 text-center text-sm">
        <ImportCount label="Toplam" value={job.total_records} />
        <ImportCount label="Başarılı" value={job.successful_records} />
        <ImportCount label="Başarısız" value={job.failed_records} />
      </dl>
      {rowErrors.length > 0 && (
        <div className="max-h-64 overflow-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">Satır</TableHead>
                <TableHead className="w-32">Öğrenci No</TableHead>
                <TableHead>Hata</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rowErrors.map((rowError) => (
                <TableRow key={rowError.row}>
                  <TableCell>{rowError.row}</TableCell>
                  <TableCell>{rowError.student_number || "-"}</TableCell>
                  <TableCell className="whitespace-normal">
                    {rowError.message}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}

function ImportCount({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border p-3">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-2xl font-semibold">{value}</dd>
    </div>
  )
}
