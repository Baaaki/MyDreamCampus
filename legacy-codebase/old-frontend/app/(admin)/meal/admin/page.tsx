"use client";

import { useEffect, useState } from "react";
import { mealApi } from "@/lib/api-client";
import type { DailyQRCode, Cafeteria } from "@/lib/types";
import { QRCodeSVG } from "qrcode.react";
import { format } from "date-fns";
import { tr } from "date-fns/locale";

export default function MealAdminQRPage() {
  const [cafeterias, setCafeterias] = useState<Cafeteria[]>([]);
  const [selectedCafeteria, setSelectedCafeteria] = useState("");
  const [selectedDate, setSelectedDate] = useState(format(new Date(), "yyyy-MM-dd"));
  const [lunchQR, setLunchQR] = useState<DailyQRCode | null>(null);
  const [dinnerQR, setDinnerQR] = useState<DailyQRCode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchCafeterias();
  }, []);

  useEffect(() => {
    if (selectedCafeteria && selectedDate) {
      fetchQRCodes();
    }
  }, [selectedCafeteria, selectedDate]);

  const fetchCafeterias = async () => {
    try {
      const data = await mealApi.get("cafeterias").json<Cafeteria[]>();
      setCafeterias(data.filter((c) => c.is_active));
    } catch (err: any) {
      setError(err.message || "Yemekhaneler yüklenemedi");
    }
  };

  const fetchQRCodes = async () => {
    try {
      setLoading(true);
      setError("");

      const [lunch, dinner] = await Promise.all([
        mealApi
          .get(`qr/${selectedCafeteria}/${selectedDate}/lunch`)
          .json<DailyQRCode>()
          .catch(() => null),
        mealApi
          .get(`qr/${selectedCafeteria}/${selectedDate}/dinner`)
          .json<DailyQRCode>()
          .catch(() => null),
      ]);

      setLunchQR(lunch);
      setDinnerQR(dinner);
    } catch (err: any) {
      setError(err.message || "QR kodları yüklenemedi");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-6xl mx-auto px-4">
        <div className="bg-white rounded-lg shadow-md p-6">
          <h1 className="text-2xl font-bold text-gray-900 mb-6">Yemekhane QR Kodları</h1>

          {error && (
            <div className="rounded-md bg-red-50 p-4 mb-4">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {/* Filters */}
          <div className="mb-6 grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Yemekhane</label>
              <select
                value={selectedCafeteria}
                onChange={(e) => setSelectedCafeteria(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
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
              <label className="block text-sm font-medium text-gray-700 mb-2">Tarih</label>
              <input
                type="date"
                value={selectedDate}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
              />
            </div>
          </div>

          {loading && <p className="text-gray-600">Yükleniyor...</p>}

          {!loading && selectedCafeteria && selectedDate && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Lunch QR */}
              <div className="border rounded-lg p-6 text-center">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                  Öğle Yemeği QR Kodu
                </h3>
                <p className="text-sm text-gray-600 mb-4">
                  Kullanım Saati: 11:00 - 13:00
                </p>
                {lunchQR ? (
                  <div>
                    <QRCodeSVG value={lunchQR.qr_data} size={250} className="mx-auto mb-4" />
                    <p className="text-xs text-gray-500">
                      {format(new Date(lunchQR.date), "dd MMMM yyyy", { locale: tr })}
                    </p>
                  </div>
                ) : (
                  <p className="text-gray-500">QR kod bulunamadı</p>
                )}
              </div>

              {/* Dinner QR */}
              <div className="border rounded-lg p-6 text-center">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                  Akşam Yemeği QR Kodu
                </h3>
                <p className="text-sm text-gray-600 mb-4">
                  Kullanım Saati: 16:00 - 19:00
                </p>
                {dinnerQR ? (
                  <div>
                    <QRCodeSVG value={dinnerQR.qr_data} size={250} className="mx-auto mb-4" />
                    <p className="text-xs text-gray-500">
                      {format(new Date(dinnerQR.date), "dd MMMM yyyy", { locale: tr })}
                    </p>
                  </div>
                ) : (
                  <p className="text-gray-500">QR kod bulunamadı</p>
                )}
              </div>
            </div>
          )}

          {!selectedCafeteria && !loading && (
            <p className="text-gray-600 text-center">QR kodlarını görmek için yemekhane ve tarih seçin</p>
          )}
        </div>

        <div className="mt-6 bg-blue-50 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-blue-900 mb-2">Bilgilendirme</h3>
          <ul className="text-sm text-blue-800 space-y-1">
            <li>• Öğle yemeği QR kodu 11:00-13:00 arası kullanılabilir</li>
            <li>• Akşam yemeği QR kodu 16:00-19:00 arası kullanılabilir</li>
            <li>• QR kodlar her gün otomatik olarak güncellenir</li>
            <li>• Her öğrenci sadece rezervasyon yaptığı öğün için QR okutabilir</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
