import { useEffect, useState } from "react"
import { mealApi } from "@/lib/api-client"
import type { QRResponse, Cafeteria } from "@/lib/types"
import { QRCodeSVG } from "qrcode.react"
import { format } from "date-fns"
import { tr } from "date-fns/locale"

export default function MealAdminQRPage() {
  const [cafeterias, setCafeterias] = useState<Cafeteria[]>([])
  const [selectedCafeteria, setSelectedCafeteria] = useState("")
  const [selectedDate, setSelectedDate] = useState(
    format(new Date(), "yyyy-MM-dd")
  )
  const [lunchQR, setLunchQR] = useState<QRResponse | null>(null)
  const [dinnerQR, setDinnerQR] = useState<QRResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    fetchCafeterias()
  }, [])

  useEffect(() => {
    if (selectedCafeteria && selectedDate) {
      fetchQRCodes()
    }
  }, [selectedCafeteria, selectedDate])

  const fetchCafeterias = async () => {
    try {
      const data = await mealApi.get("cafeterias").json<Cafeteria[]>()
      setCafeterias(data.filter((c) => c.is_active))
    } catch (err: any) {
      setError(err.message || "Yemekhaneler yüklenemedi")
    }
  }

  const fetchQRCodes = async () => {
    try {
      setLoading(true)
      setError("")

      // Backend route: GET cafeterias/:id/qr?date=&meal_time= — payload
      // comes wrapped in SuccessResponse{data}.
      const fetchQR = (mealTime: "lunch" | "dinner") =>
        mealApi
          .get(`cafeterias/${selectedCafeteria}/qr`, {
            searchParams: { date: selectedDate, meal_time: mealTime },
          })
          .json<{ data: QRResponse }>()
          .then((res) => res.data)
          .catch(() => null)

      const [lunch, dinner] = await Promise.all([
        fetchQR("lunch"),
        fetchQR("dinner"),
      ])

      setLunchQR(lunch)
      setDinnerQR(dinner)
    } catch (err: any) {
      setError(err.message || "QR kodları yüklenemedi")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="mx-auto max-w-6xl px-4">
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h1 className="mb-6 text-2xl font-bold text-gray-900">
            Yemekhane QR Kodları
          </h1>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {/* Filters */}
          <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700">
                Yemekhane
              </label>
              <select
                value={selectedCafeteria}
                onChange={(e) => setSelectedCafeteria(e.target.value)}
                className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none"
              >
                <option value="">Yemekhane seçin</option>
                {cafeterias.map((caf) => (
                  <option key={caf.id} value={caf.id}>
                    {caf.name} - {caf.location}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700">
                Tarih
              </label>
              <input
                type="date"
                value={selectedDate}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:ring-indigo-500 focus:outline-none"
              />
            </div>
          </div>

          {loading && <p className="text-gray-600">Yükleniyor...</p>}

          {!loading && selectedCafeteria && selectedDate && (
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
              {/* Lunch QR */}
              <div className="rounded-lg border p-6 text-center">
                <h3 className="mb-4 text-lg font-semibold text-gray-900">
                  Öğle Yemeği QR Kodu
                </h3>
                <p className="mb-4 text-sm text-gray-600">
                  Kullanım Saati: 11:00 - 13:00
                </p>
                {lunchQR ? (
                  <div>
                    <QRCodeSVG
                      value={lunchQR.qr_payload}
                      size={250}
                      className="mx-auto mb-4"
                    />
                    <p className="text-xs text-gray-500">
                      {format(new Date(lunchQR.date), "dd MMMM yyyy", {
                        locale: tr,
                      })}
                    </p>
                  </div>
                ) : (
                  <p className="text-gray-500">QR kod bulunamadı</p>
                )}
              </div>

              {/* Dinner QR */}
              <div className="rounded-lg border p-6 text-center">
                <h3 className="mb-4 text-lg font-semibold text-gray-900">
                  Akşam Yemeği QR Kodu
                </h3>
                <p className="mb-4 text-sm text-gray-600">
                  Kullanım Saati: 16:00 - 19:00
                </p>
                {dinnerQR ? (
                  <div>
                    <QRCodeSVG
                      value={dinnerQR.qr_payload}
                      size={250}
                      className="mx-auto mb-4"
                    />
                    <p className="text-xs text-gray-500">
                      {format(new Date(dinnerQR.date), "dd MMMM yyyy", {
                        locale: tr,
                      })}
                    </p>
                  </div>
                ) : (
                  <p className="text-gray-500">QR kod bulunamadı</p>
                )}
              </div>
            </div>
          )}

          {!selectedCafeteria && !loading && (
            <p className="text-center text-gray-600">
              QR kodlarını görmek için yemekhane ve tarih seçin
            </p>
          )}
        </div>

        <div className="mt-6 rounded-lg bg-blue-50 p-4">
          <h3 className="mb-2 text-sm font-semibold text-blue-900">
            Bilgilendirme
          </h3>
          <ul className="space-y-1 text-sm text-blue-800">
            <li>• Öğle yemeği QR kodu 11:00-13:00 arası kullanılabilir</li>
            <li>• Akşam yemeği QR kodu 16:00-19:00 arası kullanılabilir</li>
            <li>• QR kodlar her gün otomatik olarak güncellenir</li>
            <li>
              • Her öğrenci sadece rezervasyon yaptığı öğün için QR okutabilir
            </li>
          </ul>
        </div>
      </div>
    </div>
  )
}
