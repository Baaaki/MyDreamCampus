import { useState } from "react"
import { useParams, useNavigate, useLocation } from "react-router"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { attendanceApiSafe } from "@/lib/api-client"
import { catalogService } from "@/lib/services/catalog-service"
import type {
  SessionRecordsResponse,
  AdminSessionItem,
  SessionDetailsResponse,
} from "@/lib/types"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
import { ChevronLeft } from "lucide-react"

export default function AdminAttendanceSessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()

  // Read session item from router state if available
  const sessionItemState = location.state?.session as
    | AdminSessionItem
    | undefined

  const { data: sessionDetailsApi } = useQuery({
    queryKey: ["admin-session-details", sessionId],
    queryFn: () =>
      attendanceApiSafe
        .get(`sessions/${sessionId}`)
        .json<SessionDetailsResponse>(),
    enabled: !!sessionId && !sessionItemState,
  })

  const sessionItem = sessionItemState || sessionDetailsApi

  // Lookup comprehensive course info from catalog
  const { data: courseInfo } = useQuery({
    queryKey: ["course-by-code", sessionItem?.course_code],
    queryFn: () => catalogService.getCourseByCode(sessionItem!.course_code),
    enabled: !!sessionItem?.course_code,
  })

  const { data: records, isLoading: recordsLoading } = useQuery({
    queryKey: ["admin-session-records", sessionId],
    queryFn: () =>
      attendanceApiSafe
        .get(`sessions/${sessionId}/records`)
        .json<SessionRecordsResponse>(),
    enabled: !!sessionId,
  })

  const presentStudents = records?.records.filter((r) => r.is_present) || []
  const absentStudents = records?.records.filter((r) => !r.is_present) || []

  const [studentToMark, setStudentToMark] = useState<string | null>(null)
  const [isMarking, setIsMarking] = useState(false)

  const confirmMarkPresent = async () => {
    if (!studentToMark || !sessionId) return
    setIsMarking(true)
    try {
      await attendanceApiSafe.post(`sessions/${sessionId}/manual`, {
        json: {
          student_id: studentToMark,
          is_present: true,
          note: "Admin tarafından manuel eklendi",
        },
      })
      await queryClient.invalidateQueries({
        queryKey: ["admin-session-records", sessionId],
      })
    } catch (err) {
      console.error("Manuel yoklama eklenemedi:", err)
    } finally {
      setIsMarking(false)
      setStudentToMark(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="outline"
            size="icon"
            onClick={() => navigate("/attendance")}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              Oturum Detayları
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Bu oturuma katılan öğrencilerin detaylı yoklama kayıtları
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Ders Bilgileri Sidebar */}
        <div className="space-y-6 lg:col-span-1">
          <div className="rounded-lg border bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
            <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
              Ders Bilgileri
            </h2>
            {sessionItem ? (
              <div className="space-y-4">
                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Ders Kodu ve Adı
                  </p>
                  <p className="mt-0.5 font-medium text-gray-900 dark:text-white">
                    {sessionItem.course_code} - {sessionItem.course_name}
                  </p>
                </div>

                {courseInfo && (
                  <>
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Fakülte / Bölüm
                      </p>
                      <p className="mt-0.5 font-medium text-gray-900 dark:text-white">
                        {courseInfo.faculty} / {courseInfo.department}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Ders Sorumlusu (Genel)
                      </p>
                      <p className="mt-0.5 font-medium text-gray-900 dark:text-white">
                        {courseInfo.coordinator?.name || "-"}
                      </p>
                    </div>
                  </>
                )}

                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Dönem
                  </p>
                  <p className="mt-0.5 font-medium text-gray-900 dark:text-white">
                    {sessionItem.semester}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Oturum Tipi / Durumu
                  </p>
                  <div className="mt-1 flex gap-2">
                    <Badge
                      variant={
                        sessionItem.session_type === "theory"
                          ? "default"
                          : "secondary"
                      }
                    >
                      {sessionItem.session_type === "theory"
                        ? "Teorik"
                        : "Uygulama"}
                    </Badge>
                    {sessionItem.is_active ? (
                      <Badge className="border-none bg-green-500">Aktif</Badge>
                    ) : (
                      <Badge variant="secondary">Kapandı</Badge>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="py-4 text-center text-sm text-gray-500">
                Oturum bilgisi yükleniyor veya bulunamadı.
              </div>
            )}
          </div>
        </div>

        {/* Ana Katılım Listesi Alanı */}
        <div className="rounded-lg border bg-white p-6 shadow-sm lg:col-span-2 dark:border-gray-800 dark:bg-gray-900">
          <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
            Katılım Kayıtları
          </h2>
          {recordsLoading ? (
            <div className="py-12 text-center text-sm text-gray-500">
              Kayıtlar yükleniyor...
            </div>
          ) : records ? (
            <div className="space-y-6">
              <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Tarih
                  </p>
                  <p className="mt-1 truncate text-lg font-medium text-gray-900 dark:text-white">
                    {sessionItem?.session_date || "-"}
                  </p>
                </div>
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Hafta
                  </p>
                  <p className="mt-1 text-lg font-medium text-gray-900 dark:text-white">
                    {records.week_number}. Hafta
                  </p>
                </div>
                <div className="rounded-lg border bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-950">
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Toplam Kayıtlı
                  </p>
                  <p className="mt-1 text-lg font-medium text-gray-900 dark:text-white">
                    {records.total_count} Öğrenci
                  </p>
                </div>
                <div className="rounded-lg border bg-emerald-50 p-4 dark:border-emerald-900/20 dark:bg-emerald-950/20">
                  <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
                    Katılım
                  </p>
                  <p className="mt-1 text-xl font-bold text-emerald-700 dark:text-emerald-300">
                    {records.present_count} / {records.total_count}
                  </p>
                </div>
              </div>

              <div className="space-y-8">
                {/* Var / Katılanlar Tablosu */}
                <div>
                  <h3 className="text-md mb-3 flex items-center gap-2 font-semibold text-emerald-700 dark:text-emerald-400">
                    Katılan Öğrenciler (Var)
                    <Badge
                      variant="outline"
                      className="bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-400"
                    >
                      {presentStudents.length}
                    </Badge>
                  </h3>
                  <div className="rounded-md border dark:border-gray-800">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Öğrenci No</TableHead>
                          <TableHead>Ad Soyad</TableHead>
                          <TableHead>Durum</TableHead>
                          <TableHead>Yöntem</TableHead>
                          <TableHead>Yoklama Saati</TableHead>
                          <TableHead>Not / Açıklama</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {presentStudents.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={6}
                              className="h-24 text-center text-sm text-gray-500"
                            >
                              Katılan öğrenci bulunamadı.
                            </TableCell>
                          </TableRow>
                        ) : (
                          presentStudents.map((r) => (
                            <TableRow key={r.id}>
                              <TableCell className="font-medium">
                                {r.student_number}
                              </TableCell>
                              <TableCell>{r.student_name}</TableCell>
                              <TableCell>
                                <Badge className="border-none bg-emerald-500 hover:bg-emerald-600">
                                  Katıldı (Var)
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {r.marked_via === "qr"
                                  ? "QR Okutma"
                                  : r.marked_via === "manual"
                                    ? "Manuel (Elle)"
                                    : "-"}
                              </TableCell>
                              <TableCell>
                                {r.marked_at
                                  ? new Date(r.marked_at).toLocaleTimeString(
                                      "tr-TR",
                                      {
                                        hour: "2-digit",
                                        minute: "2-digit",
                                      }
                                    )
                                  : "-"}
                              </TableCell>
                              <TableCell
                                className="max-w-xs truncate text-sm text-gray-500"
                                title={r.note}
                              >
                                {r.note || "-"}
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>

                {/* Yok / Katılmayanlar Tablosu */}
                <div>
                  <h3 className="text-md mb-3 flex items-center gap-2 font-semibold text-red-700 dark:text-red-400">
                    Katılmayan Öğrenciler (Yok)
                    <Badge
                      variant="outline"
                      className="bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-400"
                    >
                      {absentStudents.length}
                    </Badge>
                  </h3>
                  <div className="rounded-md border dark:border-gray-800">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Öğrenci No</TableHead>
                          <TableHead>Ad Soyad</TableHead>
                          <TableHead>Durum</TableHead>
                          <TableHead className="text-right">İşlem</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {absentStudents.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={4}
                              className="h-24 text-center text-sm text-gray-500"
                            >
                              Tüm öğrenciler katıldı.
                            </TableCell>
                          </TableRow>
                        ) : (
                          absentStudents.map((r) => (
                            <TableRow key={r.id}>
                              <TableCell className="font-medium">
                                {r.student_number}
                              </TableCell>
                              <TableCell>{r.student_name}</TableCell>
                              <TableCell>
                                <Badge variant="destructive">
                                  Katılmadı (Yok)
                                </Badge>
                              </TableCell>
                              <TableCell className="text-right">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => setStudentToMark(r.student_id)}
                                >
                                  Yoklamaya Ekle
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="py-12 text-center text-sm text-gray-500">
              Kayıt bilgisi bulunamadı.
            </div>
          )}
        </div>
      </div>

      <AlertDialog
        open={!!studentToMark}
        onOpenChange={(open) => !open && setStudentToMark(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Öğrenciyi Yoklamaya Ekle</AlertDialogTitle>
            <AlertDialogDescription>
              Bu öğrenciyi yoklamaya eklemek istiyor musunuz? Öğrenci onayınız
              ardından <strong>Var</strong> olarak (manuel) işaretlenecektir.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>İptal</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmMarkPresent}
              disabled={isMarking}
            >
              {isMarking ? "Ekleniyor..." : "Evet, Ekle"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
