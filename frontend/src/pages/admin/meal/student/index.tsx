
import { useEffect, useState } from "react";
import { mealApi } from "@/lib/api-client";
import type { MealReservation, Cafeteria } from "@/lib/types";
import { format, addDays, startOfWeek, isBefore, isAfter, setHours, setMinutes } from "date-fns";
import { tr } from "date-fns/locale";
import { MEAL_TIMES, MENU_TYPES } from "@/lib/constants";

export default function StudentMealReservationPage() {
  const [reservations, setReservations] = useState<MealReservation[]>([]);
  const [cafeterias, setCafeterias] = useState<Cafeteria[]>([]);
  const [selectedCafeteria, setSelectedCafeteria] = useState("");
  const [selectedDate, setSelectedDate] = useState("");
  const [selectedMealTime, setSelectedMealTime] = useState<"lunch" | "dinner">("lunch");
  const [selectedMenuType, setSelectedMenuType] = useState<"normal" | "vegan">("normal");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [reservationsData, cafeteriasData] = await Promise.all([
        mealApi.get("reservations/my").json<MealReservation[]>(),
        mealApi.get("cafeterias").json<Cafeteria[]>(),
      ]);

      setReservations(reservationsData);
      setCafeterias(cafeteriasData.filter((c) => c.is_active));
    } catch (err: any) {
      setError(err.message || "Veri yüklenemedi");
    }
  };

  const isReservationWindowOpen = () => {
    const now = new Date();
    const dayOfWeek = now.getDay(); // 0 = Sunday, 1 = Monday, ..., 5 = Friday
    const hour = now.getHours();

    // Monday (1) 08:00 to Friday (5) 13:00
    if (dayOfWeek === 0 || dayOfWeek === 6) return false; // Weekend
    if (dayOfWeek === 1 && hour < 8) return false; // Before Monday 08:00
    if (dayOfWeek === 5 && hour >= 13) return false; // After Friday 13:00

    return true;
  };

  const handleReserve = async () => {
    if (!selectedCafeteria || !selectedDate) {
      setError("Lütfen yemekhane ve tarih seçin");
      return;
    }

    if (!isReservationWindowOpen()) {
      setError("Rezervasyon sadece Pazartesi 08:00 - Cuma 13:00 arası yapılabilir");
      return;
    }

    setError("");
    setSuccess("");
    setLoading(true);

    try {
      await mealApi.post("reservations", {
        json: {
          cafeteria_id: selectedCafeteria,
          reservation_date: selectedDate,
          meal_time: selectedMealTime,
          menu_type: selectedMenuType,
        },
      });

      setSuccess("Rezervasyon başarıyla oluşturuldu. Ödeme sayfasına yönlendiriliyorsunuz...");

      // Refresh reservations
      fetchData();

      // Reset form
      setSelectedCafeteria("");
      setSelectedDate("");
    } catch (err: any) {
      setError(err.message || "Rezervasyon oluşturulamadı");
    } finally {
      setLoading(false);
    }
  };

  const handleCancelReservation = async (reservationId: string) => {
    if (!confirm("Rezervasyonu iptal etmek istediğinize emin misiniz?")) {
      return;
    }

    if (!isReservationWindowOpen()) {
      alert("Rezervasyon iptali sadece Pazartesi 08:00 - Cuma 13:00 arası yapılabilir");
      return;
    }

    try {
      await mealApi.delete(`reservations/${reservationId}`);
      alert("Rezervasyon başarıyla iptal edildi");
      fetchData();
    } catch (err: any) {
      alert(err.message || "Rezervasyon iptal edilemedi");
    }
  };

  // Generate weekday dates for next week
  const getWeekdayDates = () => {
    const monday = startOfWeek(addDays(new Date(), 7), { weekStartsOn: 1 });
    const dates = [];
    for (let i = 0; i < 5; i++) {
      dates.push(format(addDays(monday, i), "yyyy-MM-dd"));
    }
    return dates;
  };

  const weekdayDates = getWeekdayDates();

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        {/* Reservation Window Status */}
        <div className={`mb-6 p-4 rounded-lg ${isReservationWindowOpen() ? "bg-green-50" : "bg-red-50"}`}>
          <p className={`text-sm ${isReservationWindowOpen() ? "text-green-800" : "text-red-800"}`}>
            {isReservationWindowOpen()
              ? "✅ Rezervasyon penceresi açık (Pazartesi 08:00 - Cuma 13:00)"
              : "❌ Rezervasyon penceresi kapalı. Sadece Pazartesi 08:00 - Cuma 13:00 arası işlem yapılabilir."}
          </p>
        </div>

        {/* New Reservation */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Yeni Rezervasyon</h2>

          {error && (
            <div className="rounded-md bg-red-50 p-4 mb-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {success && (
            <div className="rounded-md bg-green-50 p-4 mb-4">
              <p className="text-sm text-green-800">{success}</p>
            </div>
          )}

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Yemekhane</label>
              <select
                value={selectedCafeteria}
                onChange={(e) => setSelectedCafeteria(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                disabled={!isReservationWindowOpen()}
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
              <label className="block text-sm font-medium text-gray-700 mb-2">Tarih (Haftaiçi)</label>
              <select
                value={selectedDate}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                disabled={!isReservationWindowOpen()}
              >
                <option value="">Tarih seçin</option>
                {weekdayDates.map((date) => (
                  <option key={date} value={date}>
                    {format(new Date(date), "dd MMMM yyyy - EEEE", { locale: tr })}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Öğün</label>
              <div className="flex gap-4">
                <label className="flex items-center">
                  <input
                    type="radio"
                    value="lunch"
                    checked={selectedMealTime === "lunch"}
                    onChange={(e) => setSelectedMealTime(e.target.value as "lunch")}
                    className="mr-2"
                    disabled={!isReservationWindowOpen()}
                  />
                  Öğle Yemeği (11:00-13:00)
                </label>
                <label className="flex items-center">
                  <input
                    type="radio"
                    value="dinner"
                    checked={selectedMealTime === "dinner"}
                    onChange={(e) => setSelectedMealTime(e.target.value as "dinner")}
                    className="mr-2"
                    disabled={!isReservationWindowOpen()}
                  />
                  Akşam Yemeği (16:00-19:00)
                </label>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Menü Tipi</label>
              <div className="flex gap-4">
                <label className="flex items-center">
                  <input
                    type="radio"
                    value="normal"
                    checked={selectedMenuType === "normal"}
                    onChange={(e) => setSelectedMenuType(e.target.value as "normal")}
                    className="mr-2"
                    disabled={!isReservationWindowOpen()}
                  />
                  Normal
                </label>
                <label className="flex items-center">
                  <input
                    type="radio"
                    value="vegan"
                    checked={selectedMenuType === "vegan"}
                    onChange={(e) => setSelectedMenuType(e.target.value as "vegan")}
                    className="mr-2"
                    disabled={!isReservationWindowOpen()}
                  />
                  Vegan
                </label>
              </div>
            </div>

            <button
              onClick={handleReserve}
              disabled={loading || !isReservationWindowOpen()}
              className="px-6 py-2 text-white bg-indigo-600 hover:bg-indigo-700 rounded-md font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "İşleniyor..." : "Rezervasyon Yap"}
            </button>
          </div>
        </div>

        {/* My Reservations */}
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Rezervasyonlarım</h2>

          {reservations.length === 0 ? (
            <p className="text-gray-600">Rezervasyon bulunamadı.</p>
          ) : (
            <div className="space-y-3">
              {reservations.map((reservation) => (
                <div key={reservation.id} className="border rounded-lg p-4">
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <h3 className="font-semibold text-gray-900">{reservation.cafeteria_name}</h3>
                      <p className="text-sm text-gray-600">
                        {format(new Date(reservation.reservation_date), "dd MMMM yyyy - EEEE", {
                          locale: tr,
                        })}
                      </p>
                      <p className="text-sm text-gray-600">
                        Öğün: {reservation.meal_time === "lunch" ? "Öğle" : "Akşam"} | Menü:{" "}
                        {reservation.menu_type === "normal" ? "Normal" : "Vegan"}
                      </p>
                      <div className="mt-2 flex gap-2">
                        <span
                          className={`px-2 py-1 text-xs font-medium rounded ${
                            reservation.payment_status === "paid"
                              ? "bg-green-100 text-green-800"
                              : reservation.payment_status === "refunded"
                              ? "bg-gray-100 text-gray-800"
                              : "bg-yellow-100 text-yellow-800"
                          }`}
                        >
                          {reservation.payment_status === "paid"
                            ? "Ödendi"
                            : reservation.payment_status === "refunded"
                            ? "İade Edildi"
                            : "Ödeme Bekliyor"}
                        </span>
                        {reservation.is_used && (
                          <span className="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                            Kullanıldı
                          </span>
                        )}
                      </div>
                    </div>
                    {!reservation.is_used && reservation.payment_status === "paid" && (
                      <button
                        onClick={() => handleCancelReservation(reservation.id)}
                        disabled={!isReservationWindowOpen()}
                        className="ml-4 px-4 py-2 text-sm font-medium text-red-700 bg-red-100 hover:bg-red-200 rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        İptal Et
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
