import { useCallback, useEffect, useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import {
  addYears,
  format,
  formatDistanceToNowStrict,
  formatDuration,
  intervalToDuration,
} from "date-fns"
import { tr } from "date-fns/locale"
import {
  AlertTriangle,
  Clock,
  Loader2,
  Play,
  RefreshCw,
  RotateCcw,
} from "lucide-react"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
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
import Toast from "@/components/enrollment/Toast"
import { apiErrorMessage } from "@/lib/api-error"
import type { TimeStatus } from "@/lib/types"

import {
  clockOffsetMs,
  clockOffsetSpreadMs,
  getAllTimeStatuses,
  resetTimeAll,
  simulateTimeAll,
} from "@/lib/services/system-service"

// Services re-read the shared setting every 10 seconds, so a spread above
// a second that outlives one refresh means a service is out of step.
const DRIFT_TOLERANCE_MS = 1000
const REFRESH_MS = 10_000
const MAX_YEARS = 2
const INPUT_FORMAT = "yyyy-MM-dd'T'HH:mm"
const CLOCK_FORMAT = "dd MMM yyyy HH:mm:ss"

/** "45 sn", "3 saat 2 dakika", "1 yıl 2 gün" — sign dropped. */
function formatSpan(seconds: number): string {
  const abs = Math.abs(Math.round(seconds))
  if (abs < 60) return `${abs} sn`
  // The target is picked to the minute, so seconds only add noise.
  const minutes = Math.round(abs / 60)
  const duration = intervalToDuration({ start: 0, end: minutes * 60_000 })
  return formatDuration(duration, {
    format: ["years", "months", "days", "hours", "minutes"],
    locale: tr,
  })
}

/** "1 yıl 2 gün ileri" / "3 saat geri" */
function describeOffset(seconds: number): string {
  if (Math.abs(seconds) < 60) return "gerçek saatle aynı"
  return `${formatSpan(seconds)} ${seconds > 0 ? "ileri" : "geri"}`
}

/** "+2 sn" / "−1 yıl" relative to the reference offset. */
function describeDiff(ms: number): string {
  const seconds = Math.round(ms / 1000)
  if (seconds === 0) return "—"
  return `${seconds > 0 ? "+" : "−"}${formatSpan(seconds)}`
}

/** Ticks once a second so the displayed clocks keep moving between fetches. */
function useNow(): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(id)
  }, [])
  return now
}

export default function TimeMachinePage() {
  const [toast, setToast] = useState<{
    message: string
    type: "error" | "warning" | "success" | "info"
    isVisible: boolean
  }>({ message: "", type: "info", isVisible: false })

  const showToast = useCallback(
    (message: string, type: "error" | "warning" | "success" | "info") => {
      setToast({ message, type, isVisible: true })
    },
    []
  )

  const {
    data: statuses = [],
    dataUpdatedAt,
    isLoading,
    isFetching,
    isError,
    refetch,
  } = useQuery({
    queryKey: ["system", "time-statuses"],
    queryFn: getAllTimeStatuses,
    refetchInterval: REFRESH_MS,
  })
  const now = useNow()
  const [simulateTime, setSimulateTime] = useState("")

  const simulate = useMutation({
    mutationFn: (time: string) => simulateTimeAll(time),
    onSuccess: () => {
      showToast("Tüm servislerin saati ayarlandı", "success")
      void refetch()
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Simülasyon başlatılamadı"), "error"),
  })
  const reset = useMutation({
    mutationFn: resetTimeAll,
    onSuccess: () => {
      showToast("Tüm servisler gerçek saate döndü", "success")
      setSimulateTime("")
      void refetch()
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Sıfırlama başarısız"), "error"),
  })
  const busy = simulate.isPending || reset.isPending

  const handleSimulate = () => {
    if (!simulateTime) {
      showToast("Lütfen bir tarih-saat seçin", "warning")
      return
    }
    simulate.mutate(new Date(simulateTime).toISOString())
  }

  // Catalog owns the setting; fall back to any service that answered.
  const reference: TimeStatus | undefined =
    statuses.find((s) => s.service === "catalog")?.status ??
    statuses.find((s) => s.status)?.status ??
    undefined
  const referenceOffset = reference ? clockOffsetMs(reference) : 0
  const spread = clockOffsetSpreadMs(statuses)
  const unreachable = statuses.filter((s) => s.error)

  // Advances a fetched clock by the time elapsed since the fetch.
  const running = (iso: string) =>
    new Date(Date.parse(iso) + Math.max(0, now - dataUpdatedAt))

  const realNow = new Date(now)
  const inputMin = format(addYears(realNow, -MAX_YEARS), INPUT_FORMAT)
  const inputMax = format(addYears(realNow, MAX_YEARS), INPUT_FORMAT)

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Zaman Makinesi
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Tüm servislerin saatini birlikte ileri veya geri al. Saat akmaya devam
          eder; oturumlar ve güvenlik süreleri gerçek saatte kalır.
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-indigo-600" />
              <CardTitle>Sistem Saati</CardTitle>
            </div>
            {reference &&
              (reference.active ? (
                <Badge
                  variant="outline"
                  className="border-amber-500 text-amber-600 dark:text-amber-400"
                >
                  Simüle Ediliyor
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="border-green-500 text-green-600 dark:text-green-400"
                >
                  Gerçek Zaman
                </Badge>
              ))}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <Loader2 className="mx-auto h-6 w-6 animate-spin text-gray-400" />
          ) : !reference ? (
            <p className="text-destructive">Hiçbir servisin saati alınamadı</p>
          ) : (
            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Sistem saati
                </p>
                <p className="text-lg font-semibold tabular-nums">
                  {format(running(reference.current_time), CLOCK_FORMAT, {
                    locale: tr,
                  })}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Gerçek saate göre
                </p>
                <p className="text-lg font-semibold">
                  {reference.active
                    ? describeOffset(reference.offset_seconds)
                    : "gerçek saat"}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Gerçek saate dönüş
                </p>
                <p className="text-lg font-semibold">
                  {!reference.active
                    ? "—"
                    : reference.until
                      ? `${formatDistanceToNowStrict(new Date(reference.until), { locale: tr })} sonra`
                      : "Sıfırlanana kadar açık"}
                </p>
              </div>
            </div>
          )}

          {spread !== null && spread > DRIFT_TOLERANCE_MS && (
            <div className="flex items-start gap-2 rounded-lg border border-amber-500 bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950 dark:text-amber-300">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>
                {`Servislerin saatleri arasında ${formatSpan(spread / 1000)} fark var.`}{" "}
                Servisler ayarı en geç 10 sn içinde alır; fark sürüyorsa ilgili
                servisin Redis bağlantısını kontrol edin.
              </span>
            </div>
          )}
          {unreachable.length > 0 && (
            <div className="flex items-start gap-2 rounded-lg border border-destructive p-3 text-sm text-destructive">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>
                Saati alınamayan servisler:{" "}
                {unreachable.map((s) => s.label).join(", ")}
              </span>
            </div>
          )}

          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-[250px] flex-1">
              <Label htmlFor="simulate-time">Hedef Tarih-Saat</Label>
              <Input
                id="simulate-time"
                type="datetime-local"
                min={inputMin}
                max={inputMax}
                value={simulateTime}
                onChange={(e) => setSimulateTime(e.target.value)}
                className="mt-1"
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Gerçek saatten en fazla {MAX_YEARS} yıl ileri veya geri.
              </p>
            </div>
            <Button
              onClick={handleSimulate}
              disabled={busy || !simulateTime}
              className="bg-indigo-600 text-white hover:bg-indigo-700"
            >
              {simulate.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-2 h-4 w-4" />
              )}
              Simüle Et
            </Button>
            <Button
              variant="outline"
              onClick={() => reset.mutate()}
              disabled={busy}
            >
              {reset.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 h-4 w-4" />
              )}
              Gerçek Saate Dön
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Servis Saatleri</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void refetch()}
              disabled={isFetching}
              aria-label="Yenile"
            >
              <RefreshCw
                className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
              />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Servis</TableHead>
                  <TableHead>Durum</TableHead>
                  <TableHead>Saat</TableHead>
                  <TableHead>Sapma</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-4 text-center">
                      <Loader2 className="mx-auto h-5 w-5 animate-spin text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : isError ? (
                  <TableRow>
                    <TableCell
                      colSpan={4}
                      className="py-4 text-center text-destructive"
                    >
                      Servis durumları alınamadı
                    </TableCell>
                  </TableRow>
                ) : (
                  statuses.map((s) => (
                    <TableRow key={s.service}>
                      <TableCell className="font-medium">{s.label}</TableCell>
                      <TableCell>
                        {!s.status ? (
                          <Badge variant="destructive">Hata</Badge>
                        ) : s.status.active ? (
                          <Badge
                            variant="outline"
                            className="border-amber-500 text-amber-600 dark:text-amber-400"
                          >
                            Simüle
                          </Badge>
                        ) : (
                          <Badge
                            variant="outline"
                            className="border-green-500 text-green-600 dark:text-green-400"
                          >
                            Gerçek
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-gray-600 tabular-nums dark:text-gray-400">
                        {s.status
                          ? format(
                              running(s.status.current_time),
                              CLOCK_FORMAT,
                              {
                                locale: tr,
                              }
                            )
                          : s.error}
                      </TableCell>
                      <TableCell className="text-sm tabular-nums">
                        {s.status
                          ? describeDiff(
                              clockOffsetMs(s.status) - referenceOffset
                            )
                          : "—"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Toast
        message={toast.message}
        type={toast.type}
        isVisible={toast.isVisible}
        onClose={() => setToast((prev) => ({ ...prev, isVisible: false }))}
        duration={5000}
      />
    </div>
  )
}
