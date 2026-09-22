import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  History,
  CalendarCheck,
  MapPin,
  ChevronLeft,
  ChevronRight,
  Loader2,
  AlertCircle,
} from "lucide-react"
import {
  getMyReservations,
  getStartOfCurrentWeek,
  getEndOfPreviousWeek,
  formatDateForApi,
  type Reservation,
} from "@/lib/services/meal-service"

const ITEMS_PER_PAGE = 10

// Day name mapping
const DAY_NAMES: Record<number, string> = {
  0: "Pazar",
  1: "Pazartesi",
  2: "Salı",
  3: "Çarşamba",
  4: "Perşembe",
  5: "Cuma",
  6: "Cumartesi",
}

export default function CafeteriaHistoryPage() {
  const [pastPage, setPastPage] = useState(1)

  // Calculate date boundaries
  const weekStartDate = useMemo(
    () => formatDateForApi(getStartOfCurrentWeek()),
    []
  )
  const previousWeekEnd = useMemo(
    () => formatDateForApi(getEndOfPreviousWeek()),
    []
  )

  // Query for current week + future reservations (cached, no pagination)
  const currentQuery = useQuery({
    queryKey: ["current-reservations", weekStartDate],
    queryFn: () =>
      getMyReservations({
        from_date: weekStartDate,
      }),
    staleTime: 5 * 60 * 1000, // 5 minutes cache
  })

  // Query for past reservations (with pagination)
  const pastQuery = useQuery({
    queryKey: ["past-reservations", pastPage, previousWeekEnd],
    queryFn: () =>
      getMyReservations({
        to_date: previousWeekEnd,
        page: pastPage,
        limit: ITEMS_PER_PAGE,
      }),
  })

  // Filter current reservations to show only active ones
  const currentReservations = useMemo(() => {
    const reservations = currentQuery.data?.reservations || []
    return reservations.filter((r) => r.status === "confirmed" && !r.is_used)
  }, [currentQuery.data?.reservations])

  const pastReservations = pastQuery.data?.reservations || []
  const pastPagination = pastQuery.data?.pagination

  const goToNextPage = () => {
    if (pastPagination && pastPage < pastPagination.total_pages) {
      setPastPage((prev) => prev + 1)
    }
  }

  const goToPreviousPage = () => {
    if (pastPage > 1) {
      setPastPage((prev) => prev - 1)
    }
  }

  const getStatusBadge = (reservation: Reservation) => {
    // Simplified status logic: Only "Used" or "Not Used"
    if (reservation.is_used) {
      return (
        <Badge
          variant="outline"
          className="border-green-200 bg-green-50 text-green-700"
        >
          Kullanıldı
        </Badge>
      )
    }
    return (
      <Badge
        variant="outline"
        className="border-orange-200 bg-orange-50 text-orange-700"
      >
        Kullanılmadı
      </Badge>
    )
  }

  const getMealTypeLabel = (type: string) => {
    // Removed emojis
    return type === "vegan" ? (
      <span className="flex items-center gap-1 text-xs font-medium text-green-600">
        <span className="h-2 w-2 rounded-full bg-green-500" /> Vegan
      </span>
    ) : (
      <span className="flex items-center gap-1 text-xs font-medium text-orange-600">
        <span className="h-2 w-2 rounded-full bg-orange-500" /> Normal
      </span>
    )
  }

  const getMealTimeLabel = (mealTime: string) => {
    return mealTime === "lunch" ? "Öğle" : "Akşam"
  }

  const getDayName = (dateString: string) => {
    const date = new Date(dateString)
    return DAY_NAMES[date.getDay()]
  }

  const isLoading = currentQuery.isLoading && pastQuery.isLoading
  const hasError = currentQuery.error || pastQuery.error

  if (isLoading) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-emerald-600" />
      </div>
    )
  }

  if (hasError) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center gap-4">
        <AlertCircle className="h-12 w-12 text-red-500" />
        <p className="text-gray-600 dark:text-gray-400">
          Rezervasyonlar yüklenirken bir hata oluştu.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
          <History className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Geçmiş Randevularım
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Yemekhane randevu geçmişinizi ve gelecek randevularınızı
            görüntüleyin
          </p>
        </div>
      </div>

      {/* Current Week + Future Reservations */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CalendarCheck className="h-5 w-5 text-emerald-600" />
            Aktif Randevular
            {currentQuery.isFetching && (
              <Loader2 className="ml-2 h-4 w-4 animate-spin" />
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {currentReservations.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b dark:border-gray-700">
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                      Tarih
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                      Öğün
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                      Yemekhane
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                      Menü Tipi
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                      Durum
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-gray-700">
                  {currentReservations.map((res) => (
                    <tr
                      key={res.id}
                      className="hover:bg-gray-50 dark:hover:bg-gray-800/50"
                    >
                      <td className="px-4 py-3">
                        <div className="flex flex-col">
                          <span className="font-medium text-gray-900 dark:text-white">
                            {new Date(res.date).toLocaleDateString("tr-TR", {
                              day: "numeric",
                              month: "long",
                              year: "numeric",
                            })}
                          </span>
                          <span className="text-xs text-gray-500">
                            {getDayName(res.date)}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-gray-600 dark:text-gray-300">
                          {getMealTimeLabel(res.meal_time)}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2 text-gray-600 dark:text-gray-300">
                          <MapPin className="h-4 w-4" />
                          {res.cafeteria_name || res.cafeteria?.name}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        {getMealTypeLabel(res.menu_type)}
                      </td>
                      <td className="px-4 py-3">{getStatusBadge(res)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="py-4 text-center text-gray-500">
              Aktif randevunuz bulunmamaktadır.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Past History with Pagination */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5 text-gray-600 dark:text-gray-400" />
            Geçmiş Kayıtlar
            {pastQuery.isFetching && (
              <Loader2 className="ml-2 h-4 w-4 animate-spin" />
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="min-h-[300px] overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b dark:border-gray-700">
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    Tarih
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    Öğün
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    Yemekhane
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    Menü Tipi
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">
                    Durum
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-700">
                {pastReservations.map((res) => (
                  <tr
                    key={res.id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  >
                    <td className="px-4 py-3">
                      <div className="flex flex-col">
                        <span className="font-medium text-gray-900 dark:text-white">
                          {new Date(res.date).toLocaleDateString("tr-TR", {
                            day: "numeric",
                            month: "long",
                            year: "numeric",
                          })}
                        </span>
                        <span className="text-xs text-gray-500">
                          {getDayName(res.date)}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-gray-600 dark:text-gray-300">
                        {getMealTimeLabel(res.meal_time)}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2 text-gray-600 dark:text-gray-300">
                        <MapPin className="h-4 w-4" />
                        {res.cafeteria_name || res.cafeteria?.name}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      {getMealTypeLabel(res.menu_type)}
                    </td>
                    <td className="px-4 py-3">{getStatusBadge(res)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {pastReservations.length === 0 && (
              <p className="py-8 text-center text-gray-500">
                Geçmiş kayıt bulunamadı.
              </p>
            )}
          </div>

          {/* Pagination Controls */}
          {pastPagination && pastPagination.total_pages > 1 && (
            <div className="mt-4 flex items-center justify-between border-t pt-4 dark:border-gray-700">
              <div className="text-sm text-gray-500">
                Toplam {pastPagination.total_items} kayıttan{" "}
                {(pastPage - 1) * ITEMS_PER_PAGE + 1}-
                {Math.min(
                  pastPage * ITEMS_PER_PAGE,
                  pastPagination.total_items
                )}{" "}
                arası gösteriliyor
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={goToPreviousPage}
                  disabled={pastPage === 1}
                  className="h-8 w-8 p-0"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <div className="text-sm font-medium">
                  Sayfa {pastPage} / {pastPagination.total_pages}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={goToNextPage}
                  disabled={pastPage === pastPagination.total_pages}
                  className="h-8 w-8 p-0"
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
