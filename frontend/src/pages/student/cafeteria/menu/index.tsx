
import { useState, useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { mealApi } from '@/lib/api-client';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { CalendarDays, Utensils, Leaf, Loader2 } from 'lucide-react';
import { format, getDay, getDate, getMonth, getYear, getDaysInMonth } from 'date-fns';
import { tr } from 'date-fns/locale';

// Types - Backend'den gelen veri yapısı
interface DayMenu {
  items: [string, string, string, string, string]; // [çorba, ana yemek, yan yemek, tatlı, diğer]
  calories: number;
}

interface WeeklyMenu {
  monday: DayMenu;
  tuesday: DayMenu;
  wednesday: DayMenu;
  thursday: DayMenu;
  friday: DayMenu;
}

interface MonthlyMenuData {
  cafeteria_id: string;
  normalMenus: WeeklyMenu[];
  veganMenus: WeeklyMenu[];
}

interface MonthlyMenuResponse {
  success: boolean;
  data: {
    year: number;
    month: number;
    menu_data: MonthlyMenuData;
    created_at: string;
    updated_at: string;
  };
}

// Görüntüleme için dönüştürülmüş menü yapısı
interface DisplayMenuItem {
  name: string;
  calories: number;
}

interface DisplayDailyMenu {
  normal: {
    soup: DisplayMenuItem;
    main: DisplayMenuItem;
    side: DisplayMenuItem;
    dessert: DisplayMenuItem;
    other: DisplayMenuItem;
  };
  vegan: {
    soup: DisplayMenuItem;
    main: DisplayMenuItem;
    side: DisplayMenuItem;
    dessert: DisplayMenuItem;
    other: DisplayMenuItem;
  };
  totalNormalCalories: number;
  totalVeganCalories: number;
}

// API function - aylık menüyü çek
const fetchMonthlyMenu = async (year: number, month: number): Promise<MonthlyMenuResponse> => {
  return mealApi.get(`menu/monthly?year=${year}&month=${month}`).json<MonthlyMenuResponse>();
};

// Günün haftanın hangi günü olduğunu belirle
const getDayKey = (date: Date): keyof WeeklyMenu | null => {
  const dayOfWeek = getDay(date); // 0 = Pazar, 1 = Pazartesi, ...
  const dayMap: { [key: number]: keyof WeeklyMenu | null } = {
    0: null, // Pazar - menü yok
    1: 'monday',
    2: 'tuesday',
    3: 'wednesday',
    4: 'thursday',
    5: 'friday',
    6: null, // Cumartesi - menü yok
  };
  return dayMap[dayOfWeek];
};

// Tarihin ayın kaçıncı haftasında olduğunu hesapla
const getWeekOfMonth = (date: Date): number => {
  const dayOfMonth = getDate(date);
  const firstDayOfMonth = new Date(getYear(date), getMonth(date), 1);
  const firstDayOfWeek = getDay(firstDayOfMonth);

  // Haftanın Pazartesi'den başladığını varsayarak hesapla
  const adjustedFirstDay = firstDayOfWeek === 0 ? 6 : firstDayOfWeek - 1;
  const adjustedDayOfMonth = dayOfMonth + adjustedFirstDay;

  return Math.floor((adjustedDayOfMonth - 1) / 7);
};

// Boş menü item
const emptyMenuItem: DisplayMenuItem = { name: '-', calories: 0 };

// Boş günlük menü
const emptyDailyMenu: DisplayDailyMenu = {
  normal: {
    soup: emptyMenuItem,
    main: emptyMenuItem,
    side: emptyMenuItem,
    dessert: emptyMenuItem,
    other: emptyMenuItem,
  },
  vegan: {
    soup: emptyMenuItem,
    main: emptyMenuItem,
    side: emptyMenuItem,
    dessert: emptyMenuItem,
    other: emptyMenuItem,
  },
  totalNormalCalories: 0,
  totalVeganCalories: 0,
};

// Backend verisinden günlük menüyü çıkar
const extractDailyMenu = (
  menuData: MonthlyMenuData | null,
  date: Date
): DisplayDailyMenu => {
  if (!menuData) return emptyDailyMenu;

  const dayKey = getDayKey(date);
  if (!dayKey) {
    // Hafta sonu - menü yok
    return emptyDailyMenu;
  }

  const weekIndex = getWeekOfMonth(date);

  // Hafta index'i menü dizisinin dışındaysa boş döndür
  if (weekIndex >= menuData.normalMenus.length) {
    return emptyDailyMenu;
  }

  const normalWeek = menuData.normalMenus[weekIndex];
  const veganWeek = menuData.veganMenus[weekIndex];

  if (!normalWeek || !veganWeek) {
    return emptyDailyMenu;
  }

  const normalDay = normalWeek[dayKey];
  const veganDay = veganWeek[dayKey];

  if (!normalDay || !veganDay) {
    return emptyDailyMenu;
  }

  // Kalori değerlerini hesapla (eğer items boşsa 0)
  const hasNormalItems = normalDay.items.some(item => item && item.trim() !== '');
  const hasVeganItems = veganDay.items.some(item => item && item.trim() !== '');

  return {
    normal: {
      soup: { name: normalDay.items[0] || '-', calories: 0 },
      main: { name: normalDay.items[1] || '-', calories: 0 },
      side: { name: normalDay.items[2] || '-', calories: 0 },
      dessert: { name: normalDay.items[3] || '-', calories: 0 },
      other: { name: normalDay.items[4] || '-', calories: 0 },
    },
    vegan: {
      soup: { name: veganDay.items[0] || '-', calories: 0 },
      main: { name: veganDay.items[1] || '-', calories: 0 },
      side: { name: veganDay.items[2] || '-', calories: 0 },
      dessert: { name: veganDay.items[3] || '-', calories: 0 },
      other: { name: veganDay.items[4] || '-', calories: 0 },
    },
    totalNormalCalories: hasNormalItems ? normalDay.calories : 0,
    totalVeganCalories: hasVeganItems ? veganDay.calories : 0,
  };
};

export default function MealMenuPage() {
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [startIndex, setStartIndex] = useState(0);
  const [visibleCount, setVisibleCount] = useState(5);

  const currentYear = getYear(selectedDate);
  const currentMonth = getMonth(selectedDate) + 1; // 0-indexed to 1-indexed

  // Fetch monthly menu from backend
  const { data: monthlyMenuResponse, isLoading, error } = useQuery({
    queryKey: ['monthlyMenu', currentYear, currentMonth],
    queryFn: () => fetchMonthlyMenu(currentYear, currentMonth),
    retry: 1,
    staleTime: 5 * 60 * 1000, // 5 dakika cache
  });

  // Backend'den veri varsa onu kullan
  const menuData = monthlyMenuResponse?.data?.menu_data || null;
  const isUsingMockData = !menuData || error;

  // Seçilen gün için menüyü hesapla
  const dailyMenu = useMemo(() => {
    return extractDailyMenu(menuData, selectedDate);
  }, [menuData, selectedDate]);

  // Responsive visible days count
  useEffect(() => {
    const updateVisibleCount = () => {
      const width = window.innerWidth;
      if (width < 640) {
        setVisibleCount(3);
      } else if (width < 1024) {
        setVisibleCount(5);
      } else {
        setVisibleCount(7);
      }
    };

    updateVisibleCount();
    window.addEventListener('resize', updateVisibleCount);
    return () => window.removeEventListener('resize', updateVisibleCount);
  }, []);

  // Get current month boundaries
  const now = new Date();
  const displayYear = getYear(now);
  const displayMonth = getMonth(now);
  const daysInMonth = getDaysInMonth(now);

  // Generate all days in current month
  const monthDays = Array.from({ length: daysInMonth }, (_, i) =>
    new Date(displayYear, displayMonth, i + 1)
  );

  // Navigation handlers
  const canGoBack = startIndex > 0;
  const canGoForward = startIndex + visibleCount < daysInMonth;

  const goBack = () => {
    if (canGoBack) setStartIndex(Math.max(0, startIndex - 1));
  };

  const goForward = () => {
    if (canGoForward) setStartIndex(Math.min(daysInMonth - visibleCount, startIndex + 1));
  };

  // Hafta sonu kontrolü
  const isWeekend = getDay(selectedDate) === 0 || getDay(selectedDate) === 6;

  return (
    <div className="container mx-auto py-10">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-orange-600 to-red-600 bg-clip-text text-transparent">
            Yemek Menüsü
          </h1>
          <p className="text-muted-foreground mt-1">
            {format(now, 'MMMM yyyy', { locale: tr })} - Aylık yemek listesi
          </p>
        </div>
        <Button variant="outline" className="hidden md:flex cursor-default" disabled>
          <CalendarDays className="mr-2 h-4 w-4" />
          {format(now, 'MMMM yyyy', { locale: tr })}
        </Button>
      </div>

      {/* Date Selection - Carousel Style */}
      <div className="flex items-center gap-2 mb-8">
        {/* Back Arrow */}
        <button
          onClick={goBack}
          disabled={!canGoBack}
          className={`p-2 rounded-full transition-all duration-300 ${
            canGoBack
              ? 'bg-orange-100 text-orange-600 hover:bg-orange-200 hover:scale-110 active:scale-95'
              : 'bg-gray-100 text-gray-300 cursor-not-allowed'
          }`}
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        {/* Date Cards with sliding animation */}
        <div className="flex-1 overflow-hidden">
          <div
            className="flex gap-2 transition-transform duration-300 ease-out"
            style={{
              transform: `translateX(calc(50% - ${startIndex * 88}px - ${visibleCount * 44}px))`,
            }}
          >
            {monthDays.map((date) => {
              const isSelected = format(date, 'yyyy-MM-dd') === format(selectedDate, 'yyyy-MM-dd');
              const isToday = format(date, 'yyyy-MM-dd') === format(new Date(), 'yyyy-MM-dd');
              const dateIsWeekend = getDay(date) === 0 || getDay(date) === 6;

              return (
                <button
                  key={date.toISOString()}
                  onClick={() => setSelectedDate(date)}
                  className={`
                    flex-shrink-0 flex flex-col items-center w-[80px] p-3 rounded-xl border-2
                    transition-all duration-300 ease-out
                    ${isSelected
                      ? 'border-orange-500 bg-orange-50 text-orange-700 shadow-lg scale-105'
                      : dateIsWeekend
                        ? 'border-gray-200 bg-gray-100 hover:bg-gray-150 text-gray-400'
                        : 'border-gray-200 bg-white hover:bg-gray-50 hover:border-gray-300 hover:shadow-md text-gray-600'}
                  `}
                >
                  <span className="text-xs font-medium uppercase mb-1">
                    {format(date, 'EEE', { locale: tr })}
                  </span>
                  <span className={`text-2xl font-bold transition-colors duration-200 ${isSelected ? 'text-orange-600' : dateIsWeekend ? 'text-gray-400' : 'text-gray-900'}`}>
                    {format(date, 'd')}
                  </span>
                  {isToday && (
                    <span className="text-[8px] font-bold text-white bg-orange-500 px-2 py-0.5 rounded-full mt-1 animate-pulse">
                      BUGÜN
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Forward Arrow */}
        <button
          onClick={goForward}
          disabled={!canGoForward}
          className={`p-2 rounded-full transition-all ${
            canGoForward
              ? 'bg-orange-100 text-orange-600 hover:bg-orange-200'
              : 'bg-gray-100 text-gray-300 cursor-not-allowed'
          }`}
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      {/* Backend bağlantı uyarısı */}
      {isUsingMockData && !isLoading && (
        <div className="rounded-md bg-yellow-50 border border-yellow-200 p-4 mb-4">
          <p className="text-sm text-yellow-800">
            ⚠️ Bu ay için henüz menü oluşturulmamış veya backend'e bağlanılamadı.
          </p>
        </div>
      )}

      {/* Hafta sonu uyarısı */}
      {isWeekend && (
        <div className="rounded-md bg-blue-50 border border-blue-200 p-4 mb-4">
          <p className="text-sm text-blue-800">
            📅 Hafta sonları yemek servisi bulunmamaktadır.
          </p>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <Loader2 className="h-8 w-8 animate-spin text-orange-500" />
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-8">
          {/* Standard Menu */}
          <Card className="border-l-4 border-l-blue-500 shadow-lg hover:shadow-xl transition-shadow">
            <CardHeader className="bg-gradient-to-r from-blue-50 to-transparent pb-4">
              <div className="flex justify-between items-start">
                <CardTitle className="text-xl text-blue-900 flex items-center gap-2">
                  <Utensils className="h-5 w-5" />
                  Standart Menü
                </CardTitle>
                <Badge variant="secondary" className="bg-white text-blue-700 hover:bg-white/80">
                  {dailyMenu.totalNormalCalories} kcal
                </Badge>
              </div>
              <p className="text-sm text-blue-600 mt-1">
                {format(selectedDate, 'd MMMM yyyy, EEEE', { locale: tr })}
              </p>
            </CardHeader>
            <CardContent className="pt-6 space-y-6">
              {isWeekend || dailyMenu.totalNormalCalories === 0 ? (
                <div className="text-center py-8 text-gray-400">
                  <Utensils className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>Bu gün için menü bulunmuyor</p>
                </div>
              ) : (
                <div className="space-y-4">
                  <MenuItemRow label="Çorba" name={dailyMenu.normal.soup.name} color="text-amber-600" />
                  <MenuItemRow label="Ana Yemek" name={dailyMenu.normal.main.name} color="text-red-600" isMain />
                  <MenuItemRow label="Yan Yemek" name={dailyMenu.normal.side.name} color="text-orange-600" />
                  <MenuItemRow label="Tatlı" name={dailyMenu.normal.dessert.name} color="text-purple-600" />
                  <MenuItemRow label="Diğer" name={dailyMenu.normal.other.name} color="text-gray-600" />
                </div>
              )}
            </CardContent>
          </Card>

          {/* Vegan Menu */}
          <Card className="border-l-4 border-l-green-500 shadow-lg hover:shadow-xl transition-shadow">
            <CardHeader className="bg-gradient-to-r from-green-50 to-transparent pb-4">
              <div className="flex justify-between items-start">
                <CardTitle className="text-xl text-green-900 flex items-center gap-2">
                  <Leaf className="h-5 w-5" />
                  Vegan Menü
                </CardTitle>
                <Badge variant="secondary" className="bg-white text-green-700 hover:bg-white/80">
                  {dailyMenu.totalVeganCalories} kcal
                </Badge>
              </div>
              <p className="text-sm text-green-600 mt-1">
                {format(selectedDate, 'd MMMM yyyy, EEEE', { locale: tr })}
              </p>
            </CardHeader>
            <CardContent className="pt-6 space-y-6">
              {isWeekend || dailyMenu.totalVeganCalories === 0 ? (
                <div className="text-center py-8 text-gray-400">
                  <Leaf className="h-12 w-12 mx-auto mb-2 opacity-50" />
                  <p>Bu gün için menü bulunmuyor</p>
                </div>
              ) : (
                <div className="space-y-4">
                  <MenuItemRow label="Çorba" name={dailyMenu.vegan.soup.name} color="text-amber-600" />
                  <MenuItemRow label="Ana Yemek" name={dailyMenu.vegan.main.name} color="text-green-700" isMain />
                  <MenuItemRow label="Yan Yemek" name={dailyMenu.vegan.side.name} color="text-orange-600" />
                  <MenuItemRow label="Tatlı" name={dailyMenu.vegan.dessert.name} color="text-purple-600" />
                  <MenuItemRow label="Diğer" name={dailyMenu.vegan.other.name} color="text-gray-600" />
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Bilgi notu */}
      <Card className="mt-8 bg-gray-50 dark:bg-gray-900">
        <CardContent className="pt-4">
          <p className="text-xs text-muted-foreground">
            *Mücbir sebepler haricinde kesinlikle menü değişimi yapılmayacaktır.
          </p>
          <p className="text-xs text-muted-foreground mt-1">
            *Belirtilen kalori değerlerine 50 gr ekmeğin değeri olan 136 kalori dahildir.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

// Menü satırı komponenti
function MenuItemRow({
  label,
  name,
  color,
  isMain
}: {
  label: string;
  name: string;
  color: string;
  isMain?: boolean;
}) {
  const isEmpty = !name || name === '-' || name.trim() === '';

  return (
    <div className={`group flex items-start justify-between p-3 rounded-lg hover:bg-gray-50 transition-colors ${isMain ? 'bg-orange-50/50' : ''}`}>
      <div className="space-y-1">
        <span className="text-xs font-semibold uppercase text-gray-400 tracking-wider">
          {label}
        </span>
        <h3 className={`font-semibold ${isMain ? 'text-lg' : 'text-base'} ${isEmpty ? 'text-gray-300' : color}`}>
          {isEmpty ? '-' : name}
        </h3>
      </div>
    </div>
  );
}
