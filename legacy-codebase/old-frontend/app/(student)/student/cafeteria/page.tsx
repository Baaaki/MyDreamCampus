'use client';

import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  UtensilsCrossed,
  CreditCard,
  Check,
  X,
  MapPin,
  Loader2,
} from 'lucide-react';
import {
  getCafeterias,
  createBatchReservation,
  type CreateReservationRequest,
} from '@/lib/services/meal-service';


// Days of the week
const weekDays = [
  { key: 'monday', label: 'Pazartesi', shortLabel: 'Pzt' },
  { key: 'tuesday', label: 'Salı', shortLabel: 'Sal' },
  { key: 'wednesday', label: 'Çarşamba', shortLabel: 'Çar' },
  { key: 'thursday', label: 'Perşembe', shortLabel: 'Per' },
  { key: 'friday', label: 'Cuma', shortLabel: 'Cum' },
];

// Weekly menu data
const weeklyMenu: Record<string, { normal: string[]; vegan: string[] }> = {
  monday: {
    normal: ['Mercimek Çorbası', 'Etli Nohut', 'Pirinç Pilavı', 'Ayran'],
    vegan: ['Mercimek Çorbası', 'Zeytinyağlı Fasulye', 'Bulgur Pilavı', 'Ayran'],
  },
  tuesday: {
    normal: ['Ezogelin Çorbası', 'Tavuk Sote', 'Makarna', 'Cacık'],
    vegan: ['Ezogelin Çorbası', 'Sebzeli Güveç', 'Makarna', 'Cacık'],
  },
  wednesday: {
    normal: ['Domates Çorbası', 'Köfte', 'Patates Püresi', 'Salata'],
    vegan: ['Domates Çorbası', 'Mercimek Köftesi', 'Patates Püresi', 'Salata'],
  },
  thursday: {
    normal: ['Yayla Çorbası', 'Etli Kuru Fasulye', 'Pirinç Pilavı', 'Turşu'],
    vegan: ['Yayla Çorbası', 'Barbunya Pilaki', 'Bulgur Pilavı', 'Turşu'],
  },
  friday: {
    normal: ['Tarhana Çorbası', 'Balık', 'Patates Kızartması', 'Salata'],
    vegan: ['Tarhana Çorbası', 'Ispanaklı Börek', 'Patates Fırın', 'Salata'],
  },
};

// Meal types
type MealType = 'none' | 'normal' | 'vegan';

interface MealSelection {
  [day: string]: MealType;
}

// Prices
const MEAL_PRICE = 25; // TL

export default function StudentCafeteriaPage() {
  const queryClient = useQueryClient();
  const [selectedCafeteria, setSelectedCafeteria] = useState<string>('');
  const [mealSelections, setMealSelections] = useState<MealSelection>({
    monday: 'none',
    tuesday: 'none',
    wednesday: 'none',
    thursday: 'none',
    friday: 'none',
  });
  const [paymentDialogOpen, setPaymentDialogOpen] = useState(false);
  const [paymentSuccess, setPaymentSuccess] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [paymentUrl, setPaymentUrl] = useState<string | null>(null);

  // Fetch cafeterias from API
  const { data: cafeteriaData, isLoading: isLoadingCafeterias } = useQuery({
    queryKey: ['cafeterias'],
    queryFn: getCafeterias,
  });

  const cafeterias = cafeteriaData?.cafeterias || [];

  // Get current week dates
  const getWeekDates = () => {
    const today = new Date();
    const dayOfWeek = today.getDay();
    const monday = new Date(today);
    monday.setDate(today.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1));
    
    return weekDays.map((day, index) => {
      const date = new Date(monday);
      date.setDate(monday.getDate() + index);
      return {
        ...day,
        date: date.toLocaleDateString('tr-TR', { day: 'numeric', month: 'short' }),
        fullDate: date,
      };
    });
  };

  const weekDates = getWeekDates();

  // Toggle meal selection
  const cycleMealType = (day: string) => {
    setMealSelections(prev => {
      const current = prev[day];
      let next: MealType;
      if (current === 'none') next = 'normal';
      else if (current === 'normal') next = 'vegan';
      else next = 'none';
      return { ...prev, [day]: next };
    });
  };

  // Set specific meal type
  const setMealType = (day: string, type: MealType) => {
    setMealSelections(prev => ({ ...prev, [day]: type }));
  };

  // Calculate total
  const selectedMealsCount = Object.values(mealSelections).filter(m => m !== 'none').length;
  const totalPrice = selectedMealsCount * MEAL_PRICE;

  // Get cafeteria name by id
  const getCafeteriaName = (id: string) => {
    const cafe = cafeterias.find(c => c.id === id);
    return cafe ? cafe.name : 'Bilinmeyen Yemekhane';
  };

  // Handle payment
  const handlePayment = async () => {
    setIsSubmitting(true);
    try {
      // Create reservations for selected meals
      const reservations: CreateReservationRequest[] = weekDates
        .filter(day => mealSelections[day.key] !== 'none')
        .map(day => ({
          cafeteria_id: selectedCafeteria,
          date: day.fullDate.toISOString().split('T')[0],
          meal_time: 'lunch' as const, // Default to lunch, could be made selectable
          menu_type: mealSelections[day.key] as 'normal' | 'vegan',
        }));

      const response = await createBatchReservation({ reservations });

      // Store payment URL for redirect
      setPaymentUrl(response.payment_url);
      setPaymentSuccess(true);

      // Invalidate reservations cache so history page refreshes
      queryClient.invalidateQueries({ queryKey: ['my-reservations'] });
    } catch (error) {
      console.error('Rezervasyon oluşturulurken hata:', error);
      // Could add error handling UI here
    } finally {
      setIsSubmitting(false);
    }
  };

  const resetSelections = () => {
    setMealSelections({
      monday: 'none',
      tuesday: 'none',
      wednesday: 'none',
      thursday: 'none',
      friday: 'none',
    });
    setPaymentSuccess(false);
    setPaymentDialogOpen(false);
    setPaymentUrl(null);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-600 text-white">
          <UtensilsCrossed className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Yemekhane</h1>
          <p className="text-gray-600 dark:text-gray-400">Haftalık yemek seçiminizi yapın</p>
        </div>
      </div>

      {/* Cafeteria Selection */}\n

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
            <Select value={selectedCafeteria} onValueChange={setSelectedCafeteria}>
              <SelectTrigger className="w-full md:w-96">
                <SelectValue placeholder="Yemekhane seçin..." />
              </SelectTrigger>
              <SelectContent>
                {cafeterias.map((cafe) => (
                  <SelectItem key={cafe.id} value={cafe.id}>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{cafe.name}</span>
                      <span className="text-gray-500 text-sm">({cafe.location})</span>
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
            <div className="flex flex-wrap gap-4 mb-6">
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 rounded bg-gray-200 dark:bg-gray-700"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">Seçilmedi</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 rounded bg-orange-500"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">Normal Menü</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 rounded bg-green-500"></div>
                <span className="text-sm text-gray-600 dark:text-gray-400">Vegan Menü</span>
              </div>
            </div>

            {/* Weekly Table */}
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr>
                    {weekDates.map((day) => (
                      <th key={day.key} className="border dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-800 text-center min-w-[120px]">
                        <div className="font-semibold text-gray-900 dark:text-white">{day.label}</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">{day.date}</div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    {weekDates.map((day) => {
                      const selection = mealSelections[day.key];
                      const menuItems = selection === 'vegan' 
                        ? weeklyMenu[day.key].vegan 
                        : weeklyMenu[day.key].normal;
                        
                      return (
                        <td key={day.key} className="border dark:border-gray-700 p-2 text-center min-w-[200px] align-top">
                          <div className="space-y-2 relative">
                            {/* Cancel Button */}
                            {selection !== 'none' && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setMealType(day.key, 'none');
                                }}
                                className="absolute -top-3 -right-3 z-10 bg-red-400 hover:bg-red-500 text-white rounded-full p-1 shadow-md transition-colors"
                                title="İptal Et"
                              >
                                <X className="h-3 w-3" />
                              </button>
                            )}

                            {/* Selection Display */}
                            <div
                              className={`w-full min-h-[160px] p-3 rounded-lg flex flex-col items-center justify-start transition-all relative overflow-hidden ${
                                selection === 'none'
                                  ? 'bg-gray-50 dark:bg-gray-800 border-2 border-dashed border-gray-300 dark:border-gray-600'
                                  : selection === 'normal'
                                  ? 'bg-orange-50 dark:bg-orange-900/20 border-2 border-orange-500 shadow-sm'
                                  : 'bg-green-50 dark:bg-green-900/20 border-2 border-green-500 shadow-sm'
                              }`}
                            >
                              {/* Status Badge */}
                              <div className="mb-3">
                                {selection === 'none' ? (
                                  <Badge variant="outline" className="text-gray-400 border-gray-400">Seçilmedi</Badge>
                                ) : selection === 'normal' ? (
                                  <Badge className="bg-orange-500 hover:bg-orange-600">Normal Menü</Badge>
                                ) : (
                                  <Badge className="bg-green-500 hover:bg-green-600">Vegan Menü</Badge>
                                )}
                              </div>

                              {/* Menu Items List */}
                              <ul className="text-xs space-y-1.5 text-left w-full px-2">
                                {menuItems.map((item, idx) => (
                                  <li key={idx} className={`flex items-start gap-1.5 ${
                                    selection === 'none' 
                                      ? 'text-gray-400 dark:text-gray-500' 
                                      : 'text-gray-700 dark:text-gray-300'
                                  }`}>
                                    <span className={`mt-0.5 w-1 h-1 rounded-full flex-shrink-0 ${
                                      selection === 'none'
                                        ? 'bg-gray-300'
                                        : selection === 'normal'
                                        ? 'bg-orange-400'
                                        : 'bg-green-400'
                                    }`} />
                                    <span>{item}</span>
                                  </li>
                                ))}
                              </ul>
                            </div>

                            {/* Quick Selection Buttons */}
                            <div className="flex gap-1">
                              <button
                                onClick={() => setMealType(day.key, 'normal')}
                                className={`flex-1 py-1.5 px-2 rounded-md text-xs font-medium transition-all ${
                                  selection === 'normal'
                                    ? 'bg-orange-500 text-white shadow-md'
                                    : 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:bg-orange-50 dark:hover:bg-orange-900/20 text-gray-600 dark:text-gray-300'
                                }`}
                              >
                                Normal
                              </button>
                              <button
                                onClick={() => setMealType(day.key, 'vegan')}
                                className={`flex-1 py-1.5 px-2 rounded-md text-xs font-medium transition-all ${
                                  selection === 'vegan'
                                    ? 'bg-green-500 text-white shadow-md'
                                    : 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:bg-green-50 dark:hover:bg-green-900/20 text-gray-600 dark:text-gray-300'
                                }`}
                              >
                                Vegan
                              </button>
                            </div>
                          </div>
                        </td>
                      );
                    })}
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Summary & Payment */}
            <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Seçilen Öğün Sayısı</p>
                  <p className="text-2xl font-bold text-gray-900 dark:text-white">
                    {selectedMealsCount} öğün
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Toplam Tutar</p>
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
                  <CreditCard className="h-5 w-5 mr-2" />
                  Ödeme Yap
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Payment Dialog */}
      <Dialog open={paymentDialogOpen} onOpenChange={setPaymentDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {paymentSuccess ? 'Ödeme Başarılı!' : 'Ödeme Onayı'}
            </DialogTitle>
            <DialogDescription>
              {paymentSuccess
                ? 'Yemek seçimleriniz kaydedildi.'
                : 'Ödemeyi onaylamak için aşağıdaki bilgileri kontrol edin.'}
            </DialogDescription>
          </DialogHeader>

          {!paymentSuccess ? (
            <>
              <div className="space-y-4 py-4">
                <div className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Seçilen Yemekler:</p>
                  <div className="space-y-2">
                    {weekDates.map((day) => {
                      const selection = mealSelections[day.key];
                      if (selection === 'none') return null;
                      return (
                        <div key={day.key} className="flex items-center justify-between">
                          <span className="text-gray-900 dark:text-white">{day.label}</span>
                          <Badge variant={selection === 'vegan' ? 'default' : 'secondary'} className={selection === 'vegan' ? 'bg-green-500' : 'bg-orange-500'}>
                            {selection === 'vegan' ? 'Vegan' : 'Normal'}
                          </Badge>
                        </div>
                      );
                    })}
                  </div>
                </div>
                <div className="flex justify-between items-center p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg">
                  <span className="font-medium text-gray-900 dark:text-white">Toplam Tutar:</span>
                  <span className="text-xl font-bold text-emerald-600">{totalPrice.toFixed(2)} ₺</span>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setPaymentDialogOpen(false)} disabled={isSubmitting}>
                  İptal
                </Button>
                <Button onClick={handlePayment} className="bg-emerald-600 hover:bg-emerald-700" disabled={isSubmitting}>
                  {isSubmitting ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      İşleniyor...
                    </>
                  ) : (
                    <>
                      <CreditCard className="h-4 w-4 mr-2" />
                      Ödemeyi Onayla
                    </>
                  )}
                </Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <div className="flex flex-col items-center py-8">
                <div className="w-16 h-16 bg-emerald-100 dark:bg-emerald-900/50 rounded-full flex items-center justify-center mb-4">
                  <Check className="h-8 w-8 text-emerald-600" />
                </div>
                <p className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  Rezervasyon oluşturuldu!
                </p>
                <p className="text-sm text-gray-500 dark:text-gray-400 text-center">
                  {selectedMealsCount} öğünlük yemek seçiminiz kaydedildi.
                </p>
                {paymentUrl && (
                  <p className="text-xs text-gray-400 mt-2">
                    Ödeme işlemi için yönlendirileceksiniz.
                  </p>
                )}
              </div>
              <DialogFooter className="flex flex-col gap-2 sm:flex-col">
                {paymentUrl && (
                  <Button
                    onClick={() => window.open(paymentUrl, '_blank')}
                    className="w-full bg-emerald-600 hover:bg-emerald-700"
                  >
                    <CreditCard className="h-4 w-4 mr-2" />
                    Ödeme Sayfasına Git
                  </Button>
                )}
                <Button onClick={resetSelections} variant={paymentUrl ? 'outline' : 'default'} className="w-full">
                  Tamam
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
