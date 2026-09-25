import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  AlertCircle,
  AlertTriangle,
  Clock,
  Database,
  Edit3,
  History,
  Loader2,
  RefreshCw,
  RotateCcw,
  Save,
  XCircle,
} from "lucide-react"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
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
import Toast from "@/components/enrollment/Toast"
import { apiErrorMessage } from "@/lib/api-error"
import {
  beginEdit,
  cancelEdit,
  getBaselineStatus,
  restoreNow,
  restoreVersion,
  saveBaseline,
  type BaselineStatus,
} from "@/lib/services/baseline-service"

function formatVersionDate(version: string): string {
  if (!version) return "Henüz kayıt yok"
  // Format: YYYYMMDD-HHMMSS
  const match = version.match(/^(\d{4})(\d{2})(\d{2})-(\d{2})(\d{2})(\d{2})$/)
  if (!match) return version
  const [, y, m, d, hh, mm, ss] = match
  return `${d}.${m}.${y} ${hh}:${mm}:${ss}`
}

function formatCountdown(diff: number): string {
  if (diff <= 0) return "Süre doldu (otomatik iptal edilecek)"
  const minutes = Math.floor(diff / 60000)
  const seconds = Math.floor((diff % 60000) / 1000)
  return `${minutes} dk ${seconds.toString().padStart(2, "0")} sn`
}

function useCountdown(deadline: string | null | undefined): string | null {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!deadline) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [deadline])

  if (!deadline) return null
  const diff = new Date(deadline).getTime() - now
  return formatCountdown(diff)
}

export default function BaselinePage() {
  const queryClient = useQueryClient()
  const [toast, setToast] = useState<{
    message: string
    type: "success" | "error" | "info"
    isVisible: boolean
  }>({
    message: "",
    type: "info",
    isVisible: false,
  })

  // Confirm dialog state
  const [confirmDialog, setConfirmDialog] = useState<{
    open: boolean
    title: string
    description: string
    actionText: string
    variant?: "default" | "destructive"
    onConfirm: () => void
  }>({
    open: false,
    title: "",
    description: "",
    actionText: "Onayla",
    variant: "default",
    onConfirm: () => {},
  })

  // The last command this page queued. A status written before demo-ops
  // picked it up still shows the old mode, so polling goes on until the
  // status names this command and demo-ops is done with it.
  const [pendingCommand, setPendingCommand] = useState<string | null>(null)
  const isWaitingFor = (s: BaselineStatus | undefined) =>
    pendingCommand !== null &&
    (s?.last_command_id !== pendingCommand || s?.mode === "busy")

  const {
    data: status,
    isLoading,
    error: statusError,
    refetch,
    isFetching,
  } = useQuery<BaselineStatus>({
    queryKey: ["baseline-status"],
    queryFn: getBaselineStatus,
    refetchInterval: (query) => {
      const data = query.state.data
      if (data?.mode === "busy" || isWaitingFor(data)) return 2000
      // demo-ops ends an expired edit on its own; the page follows.
      if (data?.mode === "editing") return 15000
      return false
    },
  })
  const isProcessing = status?.mode === "busy" || isWaitingFor(status)
  // Empty when the backend never answered: behind Cloudflare Access the
  // request is redirected to Access's login page on another origin, which
  // fetch cannot follow. Opening the endpoint itself lets Access ask for the
  // code and set its cookie.
  const statusErrorMessage = statusError ? apiErrorMessage(statusError, "") : ""

  const countdown = useCountdown(status?.edit_deadline)

  const showToast = (
    message: string,
    type: "success" | "error" | "info" = "info"
  ) => {
    setToast({ message, type, isVisible: true })
  }

  const onQueued = (commandId: string) => {
    setPendingCommand(commandId)
    queryClient.invalidateQueries({ queryKey: ["baseline-status"] })
  }

  const mutateBegin = useMutation({
    mutationFn: beginEdit,
    onSuccess: (commandId) => {
      showToast("Düzenleme modu başlatıldı. Veriler hazırlanıyor...", "success")
      onQueued(commandId)
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Düzenleme modu başlatılamadı"), "error"),
  })

  const mutateSave = useMutation({
    mutationFn: saveBaseline,
    onSuccess: (commandId) => {
      showToast("Kalıcı durum anlık görüntüsü alınıyor...", "success")
      onQueued(commandId)
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Kalıcı durum kaydedilemedi"), "error"),
  })

  const mutateCancel = useMutation({
    mutationFn: cancelEdit,
    onSuccess: (commandId) => {
      showToast("Düzenlemeden vazgeçildi. Kalıcı duruma dönülüyor...", "info")
      onQueued(commandId)
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Düzenleme iptal edilemedi"), "error"),
  })

  const mutateRestoreNow = useMutation({
    mutationFn: restoreNow,
    onSuccess: (commandId) => {
      showToast("Geri dönüş işlemi başlatıldı...", "info")
      onQueued(commandId)
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Geri dönüş işlemi başarısız"), "error"),
  })

  const mutateRestoreVersion = useMutation({
    mutationFn: (version: string) => restoreVersion(version),
    onSuccess: (commandId) => {
      showToast("Belirtilen sürüme geri dönüş başlatıldı...", "info")
      onQueued(commandId)
    },
    onError: (err) =>
      showToast(apiErrorMessage(err, "Sürüme geri dönüş başarısız"), "error"),
  })

  const isBusy =
    isProcessing ||
    mutateBegin.isPending ||
    mutateSave.isPending ||
    mutateCancel.isPending ||
    mutateRestoreNow.isPending ||
    mutateRestoreVersion.isPending

  const handleBeginEdit = () => {
    setConfirmDialog({
      open: true,
      title: "Düzenleme Moduna Başla",
      description:
        "Sistem son kalıcı duruma döndürülecek ve siz kaydedene kadar diğer kullanıcılara yazma işlemleri kilitlenecektir. Gün içindeki ziyaretçi kayıtları silinecektir. Devam etmek istiyor musunuz?",
      actionText: "Başla",
      variant: "default",
      onConfirm: () => mutateBegin.mutate(),
    })
  }

  const handleSave = () => {
    setConfirmDialog({
      open: true,
      title: "Kalıcı Durumu Kaydet ve Yayına Al",
      description:
        "Tüm veritabanlarının anlık görüntüsü alınacak ve yeni kalıcı durum yapılacaktır. Yazma kilidi kaldırılacaktır. Devam etmek istiyor musunuz?",
      actionText: "Kaydet ve Yayına Al",
      variant: "default",
      onConfirm: () => mutateSave.mutate(),
    })
  }

  const handleCancel = () => {
    setConfirmDialog({
      open: true,
      title: "Düzenlemeden Vazgeç",
      description:
        "Bu oturumda yaptığınız tüm değişiklikler atılacak ve veritabanı son kalıcı duruma döndürülecektir. Yazma kilidi kaldırılacaktır. Emin misiniz?",
      actionText: "Vazgeç ve Sıfırla",
      variant: "destructive",
      onConfirm: () => mutateCancel.mutate(),
    })
  }

  const handleRestoreNow = () => {
    setConfirmDialog({
      open: true,
      title: "Bugünkü Değişiklikleri Şimdi Geri Al",
      description:
        "Gece 04:00 beklenmeden sistem anında en son kaydedilmiş kalıcı duruma döndürülecektir. Bugün yapılan ziyaretçi değişiklikleri silinecektir. Devam etmek istiyor musunuz?",
      actionText: "Şimdi Geri Al",
      variant: "destructive",
      onConfirm: () => mutateRestoreNow.mutate(),
    })
  }

  const handleRestoreVersion = (version: string) => {
    setConfirmDialog({
      open: true,
      title: "Sürüme Geri Dön",
      description: `Sistem '${formatVersionDate(version)}' (${version}) sürümüne geri döndürülecektir. Bu sürüm yeni aktif kalıcı durum olacaktır. Devam etmek istiyor musunuz?`,
      actionText: "Sürüme Dön",
      variant: "destructive",
      onConfirm: () => mutateRestoreVersion.mutate(version),
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
            Kalıcı Veri ve Canlı Demo Yönetimi
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Süper admin kalıcı veri modunu yönetir; her gece 04:00&apos;te
            sistem burada kaydedilen kalıcı duruma geri döner.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          className="gap-2 self-start"
        >
          <RefreshCw
            className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
          />
          Yenile
        </Button>
      </div>

      {statusError && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200"
        >
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
          <div className="space-y-1 text-sm">
            <p className="font-semibold">Sistem durumu okunamadı</p>
            {statusErrorMessage ? (
              <p>{statusErrorMessage}</p>
            ) : (
              <p>
                Bu uçlar Cloudflare Access arkasındaysa önce doğrulama gerekir:{" "}
                <a
                  href="/api/catalog/admin/ops/status"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium underline"
                >
                  doğrulama sayfasını yeni sekmede aç
                </a>
                , kodu gir, sonra burada &quot;Yenile&quot;ye bas.
              </p>
            )}
          </div>
        </div>
      )}

      {status?.last_error && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-red-900 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200"
        >
          <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-400" />
          <div className="space-y-1">
            <p className="text-sm font-semibold">Son Operasyon Hatası</p>
            <p className="font-mono text-xs">{status.last_error}</p>
          </div>
        </div>
      )}

      {/* Main Status & Controls */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        <Card className="md:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Database className="h-5 w-5 text-indigo-600" />
                Mevcut Sistem Durumu
              </CardTitle>
              {isProcessing ? (
                <Badge className="animate-pulse bg-blue-600 text-white hover:bg-blue-700">
                  İşlem Yapılıyor...
                </Badge>
              ) : status?.mode === "editing" ? (
                <Badge className="bg-amber-500 text-white hover:bg-amber-600">
                  Düzenleme Modu Aktif
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="border-emerald-300 bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30"
                >
                  Normal (Korumalı)
                </Badge>
              )}
            </div>
            <CardDescription>
              Kalıcı veri sürümü ve düzenleme oturumu durumu
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">
                  Aktif Kalıcı Sürüm
                </div>
                <div className="mt-1 text-base font-semibold">
                  {formatVersionDate(status?.current || "")}
                </div>
                <div className="mt-0.5 font-mono text-xs text-muted-foreground">
                  {status?.current || "Henüz kaydedilmedi"}
                </div>
              </div>

              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">
                  Yazma Kilidi / Kalan Süre
                </div>
                <div className="mt-1 flex items-center gap-2 text-base font-semibold">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  {status?.mode === "editing" ? (
                    countdown || "Hesaplanıyor..."
                  ) : (
                    <span className="text-sm font-normal text-muted-foreground">
                      Yazma kilidi kapalı
                    </span>
                  )}
                </div>
                {status?.mode === "editing" && (
                  <div className="mt-0.5 text-xs text-amber-600 dark:text-amber-400">
                    Süre dolunca otomatik olarak iptal edilir
                  </div>
                )}
              </div>
            </div>

            {/* Action buttons */}
            <div className="flex flex-wrap gap-3 pt-2">
              {status?.mode === "editing" ? (
                <>
                  <Button
                    onClick={handleSave}
                    disabled={isBusy}
                    className="gap-2 bg-emerald-600 text-white hover:bg-emerald-700"
                  >
                    {isBusy ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Save className="h-4 w-4" />
                    )}
                    Kaydet ve Yayına Al
                  </Button>
                  <Button
                    variant="outline"
                    onClick={handleCancel}
                    disabled={isBusy}
                    className="gap-2 text-destructive hover:bg-destructive/10"
                  >
                    <XCircle className="h-4 w-4" />
                    Vazgeç ve Sıfırla
                  </Button>
                </>
              ) : (
                <>
                  <Button
                    onClick={handleBeginEdit}
                    disabled={isBusy || isLoading}
                    className="gap-2 bg-indigo-600 text-white hover:bg-indigo-700"
                  >
                    {isBusy ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Edit3 className="h-4 w-4" />
                    )}
                    Düzenlemeye Başla
                  </Button>
                  <Button
                    variant="outline"
                    onClick={handleRestoreNow}
                    disabled={isBusy || isLoading || !status?.current}
                    className="gap-2 border-amber-300 text-amber-700 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400 dark:hover:bg-amber-950/30"
                  >
                    <RotateCcw className="h-4 w-4" />
                    Bugünkü Değişiklikleri Şimdi Geri Al
                  </Button>
                </>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Info card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              Nasıl Çalışır?
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-xs text-muted-foreground">
            <p>
              1. <strong>Düzenlemeye Başla:</strong> Günün ziyaretçi verileri
              silinir, son kalıcı duruma dönülür ve diğer kullanıcılara yazma
              kilitlenir.
            </p>
            <p>
              2. <strong>Kalıcı Veri Düzenleme:</strong> Panelden istediğiniz
              ders, bölüm veya ayarları düzenleyin.
            </p>
            <p>
              3. <strong>Kaydet:</strong> Yeni durum kalıcı hale getirilir ve
              ziyaretçilere açılır.
            </p>
            <p>
              4. <strong>Gece 04:00:</strong> Her gece sistem otomatik olarak bu
              kalıcı duruma geri döner.
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Version History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <History className="h-5 w-5 text-gray-500" />
            Kalıcı Durum Sürüm Geçmişi (Son 7 Sürüm)
          </CardTitle>
          <CardDescription>
            Önceki kalıcı durumlara tek tıkla geri dönebilirsiniz.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {status?.versions && status.versions.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tarih & Saat</TableHead>
                  <TableHead>Sürüm Kodu</TableHead>
                  <TableHead>Durum</TableHead>
                  <TableHead className="text-right">İşlem</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {status.versions.map((version) => {
                  const isCurrent = version === status.current
                  return (
                    <TableRow key={version}>
                      <TableCell className="font-medium">
                        {formatVersionDate(version)}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {version}
                      </TableCell>
                      <TableCell>
                        {isCurrent ? (
                          <Badge className="bg-emerald-600 text-white hover:bg-emerald-700">
                            Aktif Sürüm
                          </Badge>
                        ) : (
                          <span className="text-xs text-muted-foreground">
                            Arşiv
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {!isCurrent && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={isBusy || status.mode === "editing"}
                            onClick={() => handleRestoreVersion(version)}
                            className="gap-1.5 text-xs text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"
                          >
                            <RotateCcw className="h-3.5 w-3.5" />
                            Bu Sürüme Dön
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          ) : (
            <div className="py-6 text-center text-sm text-muted-foreground">
              Henüz kaydedilmiş bir sürüm bulunmuyor.
            </div>
          )}
        </CardContent>
      </Card>

      {/* Confirmation Dialog */}
      <AlertDialog
        open={confirmDialog.open}
        onOpenChange={(open) => setConfirmDialog((prev) => ({ ...prev, open }))}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirmDialog.title}</AlertDialogTitle>
            <AlertDialogDescription>
              {confirmDialog.description}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isBusy}>İptal</AlertDialogCancel>
            <AlertDialogAction
              disabled={isBusy}
              onClick={confirmDialog.onConfirm}
              className={
                confirmDialog.variant === "destructive"
                  ? "text-destructive-foreground bg-destructive hover:bg-destructive/90"
                  : ""
              }
            >
              {confirmDialog.actionText}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Toast
        message={toast.message}
        type={toast.type}
        isVisible={toast.isVisible}
        onClose={() => setToast((prev) => ({ ...prev, isVisible: false }))}
      />
    </div>
  )
}
