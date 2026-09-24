import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { UtensilsCrossed, CreditCard, X, MapPin, Loader2 } from "lucide-react"
import {
  getCafeterias,
  type CreateReservationRequest,
} from "@/lib/services/meal-service"
import { CheckoutDialog, type CheckoutItem } from "./checkout-dialog"

// Days of the week
const weekDays = [
  { key: "monday", label: "Pazartesi", shortLabel: "Pzt" },
  { key: "tuesday", label: "Salı", shortLabel: "Sal" },
  { key: "wednesday", label: "Çarşamba", shortLabel: "Çar" },
  { key: "thursday", label: "Perşembe", shortLabel: "Per" },
  { key: "friday", label: "Cuma", shortLabel: "Cum" },
]

// Weekly menu data
const weeklyMenu: Record<string, { normal: string[]; vegan: string[] }> = {
  monday: {
    normal: ["Mercimek Çorbası", "Etli Nohut", "Pirinç Pilavı", "Ayran"],
    vegan: [
      "Mercimek Çorbası",
      "Zeytinyağlı Fasulye",
      "Bulgur Pilavı",
      "Ayran",
    ],
  },
  tuesday: {
    normal: ["Ezogelin Çorbası", "Tavuk Sote", "Makarna", "Cacık"],
    vegan: ["Ezogelin Çorbası", "Sebzeli Güveç", "Makarna", "Cacık"],
  },
  wednesday: {
    normal: ["Domates Çorbası", "Köfte", "Patates Püresi", "Salata"],
    vegan: ["Domates Çorbası", "Mercimek Köftesi", "Patates Püresi", "Salata"],
  },
  thursday: {
    normal: ["Yayla Çorbası", "Etli Kuru Fasulye", "Pirinç Pilavı", "Turşu"],
    vegan: ["Yayla Çorbası", "Barbunya Pilaki", "Bulgur Pilavı", "Turşu"],
  },
  friday: {
    normal: ["Tarhana Çorbası", "Balık", "Patates Kızartması", "Salata"],
    vegan: ["Tarhana Çorbası", "Ispanaklı Börek", "Patates Fırın", "Salata"],
  },
}

// Meal types
type MealType = "none" | "normal" | "vegan"

interface MealSelection {
  [day: string]: MealType
}

// Prices
const MEAL_PRICE = 25 // TL

export default function StudentCafeteriaPage() {
  const [selectedCafeteria, setSelectedCafeteria] = useState<string>("")
  const [mealSelections, setMealSelections] = useState<MealSelection>({
    monday: "none",
    tuesday: "none",
    wednesday: "none",
    thursday: "none",
    friday: "none",
  })
  const [paymentDialogOpen, setPaymentDialogOpen] = useState(false)

  // Fetch cafeterias from API
  const { data: cafeteriaData, isLoading: isLoadingCafeterias } = useQuery({
    queryKey: ["cafeterias"],
    queryFn: getCafeterias,
  })

  const cafeterias = cafeteriaData?.cafeterias || []

  // Get current week dates
  const getWeekDates = () => {
    const today = new Date()
    const dayOfWeek = today.getDay()
    const monday = new Date(today)
    monday.setDate(today.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1))

    return weekDays.map((day, index) => {
      const date = new Date(monday)
      date.setDate(monday.getDate() + index)
      return {
        ...day,
        date: date.toLocaleDateString("tr-TR", {
          day: "numeric",
          month: "short",
        }),
        fullDate: date,
      }
    })
  }

  const weekDates = getWeekDates()

  // Set specific meal type
  const setMealType = (day: string, type: MealType) => {
    setMealSelections((prev) => ({ ...prev, [day]: type }))
  }

  // Calculate total
  const selectedMealsCount = Object.values(mealSelections).filter(
    (m) => m !== "none"
  ).length
  const totalPrice = selectedMealsCount * MEAL_PRICE

  const checkoutItems: CheckoutItem[] = weekDates
    .filter((day) => mealSelections[day.key] !== "none")
    .map((day) => ({
      key: day.key,
      label: day.label,
      menuType: mealSelections[day.key] as "normal" | "vegan",
    }))

  const buildReservations = (): CreateReservationRequest[] =>
    weekDates
      .filter((day) => mealSelections[day.key] !== "none")
      .map((day) => ({
        cafeteria_id: selectedCafeteria,
        date: day.fullDate.toISOString().split("T")[0],
        meal_time: "lunch" as const, // Default to lunch, could be made selectable
        menu_type: mealSelections[day.key] as "normal" | "vegan",
      }))

  const resetSelections = () => {
    setMealSelections({
      monday: "none",
      tuesday: "none",
      wednesday: "none",
      thursday: "none",
      friday: "none",
    })
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
          <UtensilsCrossed className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Yemekhane
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Haftalık yemek seçiminizi yapın
          </p>
        </div>
      </div>
      {/* Cafeteria Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MapPin className="h-5 w-5 text-emerald-600" />
            Yemekhane Seçimi
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoadingCafeterias ? (
            <div className="flex items-center gap-2 text-gray-500">
              <Loader2 className="h-4 w-4 animate-spin" />
              Yemekhaneler yükleniyor...
            </div>
          ) : (
            <Select
              value={selectedCafeteria}
              onValueChange={setSelectedCafeteria}
            >
              <SelectTrigger className="w-full md:w-96">
                <SelectValue placeholder="Yemekhane seçin..." />
              </SelectTrigger>
              <SelectContent>
                {cafeterias.map((cafe) => (
                  <SelectItem key={cafe.id} value={cafe.id}>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{cafe.name}</span>
                      <span className="text-sm text-gray-500">
                        ({cafe.location})
                      </span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </CardContent>
      </Card>
      {/* Weekly Meal Selection */}
      {selectedCafeteria && (
        <Card>
          <CardHeader>
            <CardTitle>Haftalık Yemek Seçimi</CardTitle>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Yemek almak istediğiniz günleri ve menü tipini seçin
            </p>
          </CardHeader>
          <CardContent>
            {/* Legend */}
            <div className="mb-6 flex flex-wrap gap-4">
              <div className="flex items-center gap-2">
                <div className="h-4 w-4 rounded bg-gray-200 dark:bg-gray-700"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  Seçilmedi
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-4 w-4 rounded bg-orange-500"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  Normal Menü
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-4 w-4 rounded bg-green-500"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  Vegan Menü
                </span>
              </div>
            </div>

            {/* Weekly Table */}
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr>
                    {weekDates.map((day) => (
                      <th
                        key={day.key}
                        className="min-w-[120px] border bg-gray-50 p-3 text-center dark:border-gray-700 dark:bg-gray-800"
                      >
                        <div className="font-semibold text-gray-900 dark:text-white">
                          {day.label}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {day.date}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    {weekDates.map((day) => {
                      const selection = mealSelections[day.key]
                      const menuItems =
                        selection === "vegan"
                          ? weeklyMenu[day.key].vegan
                          : weeklyMenu[day.key].normal

                      return (
                        <td
                          key={day.key}
                          className="min-w-[200px] border p-2 text-center align-top dark:border-gray-700"
                        >
                          <div className="relative space-y-2">
                            {/* Cancel Button */}
                            {selection !== "none" && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation()
                                  setMealType(day.key, "none")
                                }}
                                className="absolute -top-3 -right-3 z-10 rounded-full bg-red-400 p-1 text-white shadow-md transition-colors hover:bg-red-500"
                                title="İptal Et"
                              >
                                <X className="h-3 w-3" />
                              </button>
                            )}

                            {/* Selection Display */}
                            <div
                              className={`relative flex min-h-[160px] w-full flex-col items-center justify-start overflow-hidden rounded-lg p-3 transition-all ${
                                selection === "none"
                                  ? "border-2 border-dashed border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-800"
                                  : selection === "normal"
                                    ? "border-2 border-orange-500 bg-orange-50 shadow-sm dark:bg-orange-900/20"
                                    : "border-2 border-green-500 bg-green-50 shadow-sm dark:bg-green-900/20"
                              }`}
                            >
                              {/* Status Badge */}
                              <div className="mb-3">
                                {selection === "none" ? (
                                  <Badge
                                    variant="outline"
                                    className="border-gray-400 text-gray-400"
                                  >
                                    Seçilmedi
                                  </Badge>
                                ) : selection === "normal" ? (
                                  <Badge className="bg-orange-500 hover:bg-orange-600">
                                    Normal Menü
                                  </Badge>
                                ) : (
                                  <Badge className="bg-green-500 hover:bg-green-600">
                                    Vegan Menü
                                  </Badge>
                                )}
                              </div>

                              {/* Menu Items List */}
                              <ul className="w-full space-y-1.5 px-2 text-left text-xs">
                                {menuItems.map((item, idx) => (
                                  <li
                                    key={idx}
                                    className={`flex items-start gap-1.5 ${
                                      selection === "none"
                                        ? "text-gray-400 dark:text-gray-500"
                                        : "text-gray-700 dark:text-gray-300"
                                    }`}
                                  >
                                    <span
                                      className={`mt-0.5 h-1 w-1 flex-shrink-0 rounded-full ${
                                        selection === "none"
                                          ? "bg-gray-300"
                                          : selection === "normal"
                                            ? "bg-orange-400"
                                            : "bg-green-400"
                                      }`}
                                    />
                                    <span>{item}</span>
                                  </li>
                                ))}
                              </ul>
                            </div>

                            {/* Quick Selection Buttons */}
                            <div className="flex gap-1">
                              <button
                                onClick={() => setMealType(day.key, "normal")}
                                className={`flex-1 rounded-md px-2 py-1.5 text-xs font-medium transition-all ${
                                  selection === "normal"
                                    ? "bg-orange-500 text-white shadow-md"
                                    : "border border-gray-200 bg-white text-gray-600 hover:bg-orange-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-orange-900/20"
                                }`}
                              >
                                Normal
                              </button>
                              <button
                                onClick={() => setMealType(day.key, "vegan")}
                                className={`flex-1 rounded-md px-2 py-1.5 text-xs font-medium transition-all ${
                                  selection === "vegan"
                                    ? "bg-green-500 text-white shadow-md"
                                    : "border border-gray-200 bg-white text-gray-600 hover:bg-green-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-green-900/20"
                                }`}
                              >
                                Vegan
                              </button>
                            </div>
                          </div>
                        </td>
                      )
                    })}
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Summary & Payment */}
            <div className="mt-6 rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
              <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Seçilen Öğün Sayısı
                  </p>
                  <p className="text-2xl font-bold text-gray-900 dark:text-white">
                    {selectedMealsCount} öğün
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Toplam Tutar
                  </p>
                  <p className="text-2xl font-bold text-emerald-600">
                    {totalPrice.toFixed(2)} ₺
                  </p>
                </div>
                <Button
                  size="lg"
                  disabled={selectedMealsCount === 0}
                  onClick={() => setPaymentDialogOpen(true)}
                  className="bg-emerald-600 hover:bg-emerald-700"
                >
                  <CreditCard className="mr-2 h-5 w-5" />
                  Ödeme Yap
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
      <CheckoutDialog
        open={paymentDialogOpen}
        onOpenChange={setPaymentDialogOpen}
        items={checkoutItems}
        estimatedTotal={totalPrice}
        buildRequest={buildReservations}
        onPaid={resetSelections}
      />
    </div>
  )
}
