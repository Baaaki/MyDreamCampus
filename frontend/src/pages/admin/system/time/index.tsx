import { useCallback, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { format } from "date-fns"
import { tr } from "date-fns/locale"
import { Clock, Play, RotateCcw, RefreshCw, Loader2 } from "lucide-react"

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

import {
  getAllTimeStatuses,
  simulateTimeAll,
  resetTimeAll,
} from "@/lib/services/system-service"

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
    isFetching: loading,
    isError,
    refetch: fetchStatuses,
  } = useQuery({
    queryKey: ["system", "time-statuses"],
    queryFn: getAllTimeStatuses,
  })
  const [simulateTime, setSimulateTime] = useState("")
  const [actionLoading, setActionLoading] = useState(false)

  const handleSimulate = async () => {
    if (!simulateTime) {
      showToast("Lütfen bir tarih-saat seçin", "warning")
      return
    }
    setActionLoading(true)
    try {
      const isoTime = new Date(simulateTime).toISOString()
      const result = await simulateTimeAll(isoTime)
      if (result.failed.length === 0) {
        showToast("Tüm servisler simüle moduna geçirildi", "success")
      } else if (result.success.length > 0) {
        showToast(
          `Başarılı: ${result.success.join(", ")} | Hata: ${result.failed.join(", ")}`,
          "warning"
        )
      } else {
        showToast("Hiçbir servis simüle edilemedi", "error")
      }
      await fetchStatuses()
    } catch {
      showToast("Simülasyon başlatılamadı", "error")
    } finally {
      setActionLoading(false)
    }
  }

  const handleReset = async () => {
    setActionLoading(true)
    try {
      const result = await resetTimeAll()
      if (result.failed.length === 0) {
        showToast("Tüm servisler gerçek zamana döndürüldü", "success")
      } else {
        showToast(
          `Başarılı: ${result.success.join(", ")} | Hata: ${result.failed.join(", ")}`,
          "warning"
        )
      }
      setSimulateTime("")
      await fetchStatuses()
    } catch {
      showToast("Sıfırlama başarısız", "error")
    } finally {
      setActionLoading(false)
    }
  }

  const isAnySimulated = statuses.some((s) => s.status?.mode === "simulated")

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Zaman Makinesi
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Servis saatlerini simüle et — demo ve test amaçlıdır
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-indigo-600" />
              <CardTitle>Servis Saatleri</CardTitle>
            </div>
            <div className="flex items-center gap-2">
              {isAnySimulated ? (
                <Badge
                  variant="outline"
                  className="border-amber-500 text-amber-600 dark:text-amber-400"
                >
                  Simüle Edilmiş
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="border-green-500 text-green-600 dark:text-green-400"
                >
                  Gerçek Zaman
                </Badge>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => fetchStatuses()}
                disabled={loading}
              >
                <RefreshCw
                  className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
                />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-lg border border-gray-200 dark:border-gray-700">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Servis</TableHead>
                  <TableHead>Mod</TableHead>
                  <TableHead>Anlık Saat</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={3} className="py-4 text-center">
                      <Loader2 className="mx-auto h-5 w-5 animate-spin text-gray-400" />
                    </TableCell>
                  </TableRow>
                ) : isError ? (
                  <TableRow>
                    <TableCell
                      colSpan={3}
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
                        {s.error ? (
                          <Badge variant="destructive">Hata</Badge>
                        ) : s.status?.mode === "simulated" ? (
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
                      <TableCell className="text-sm text-gray-600 dark:text-gray-400">
                        {s.error
                          ? s.error
                          : s.status
                            ? format(
                                new Date(s.status.current_time),
                                "dd MMM yyyy HH:mm:ss",
                                { locale: tr }
                              )
                            : "—"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-[250px] flex-1">
              <Label htmlFor="simulate-time">Hedef Tarih-Saat</Label>
              <Input
                id="simulate-time"
                type="datetime-local"
                value={simulateTime}
                onChange={(e) => setSimulateTime(e.target.value)}
                className="mt-1"
              />
            </div>
            <Button
              onClick={handleSimulate}
              disabled={actionLoading || !simulateTime}
              className="bg-indigo-600 text-white hover:bg-indigo-700"
            >
              {actionLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Play className="mr-2 h-4 w-4" />
              )}
              Simüle Et
            </Button>
            <Button
              variant="outline"
              onClick={handleReset}
              disabled={actionLoading}
            >
              {actionLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 h-4 w-4" />
              )}
              Sıfırla
            </Button>
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
