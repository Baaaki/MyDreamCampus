
import { useState, useRef, useEffect, useMemo } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { mealApi } from '@/lib/api-client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import {
  Save,
  Printer,
  Pencil,
  Lock,
  AlertTriangle,
  X,
  Plus,
  Calendar,
  Eye,
  Loader2,
} from 'lucide-react';
import Toast from '@/components/enrollment/Toast';

// Autocomplete Combobox Component
interface MealAutocompleteProps {
  value: string;
  onChange: (value: string, calories: number) => void;
  placeholder: string;
  categoryFilter?: 'soup' | 'main' | 'side' | 'dessert' | 'other';
  disabled?: boolean;
}

function MealAutocomplete({ value, onChange, placeholder, categoryFilter, disabled = false }: MealAutocompleteProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState(value);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSearchTerm(value);
  }, [value]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Filtreleme - kategori ve arama terimine göre
  const filteredMeals = mealDatabase.filter(meal => {
    const matchesSearch = meal.name.includes(searchTerm.toUpperCase());
    const matchesCategory = !categoryFilter || meal.category === categoryFilter;
    return matchesSearch && matchesCategory;
  }).slice(0, 8); // Max 8 sonuç göster

  const handleSelect = (meal: MealItem) => {
    setSearchTerm(meal.name);
    onChange(meal.name, meal.calories);
    setIsOpen(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value.toUpperCase();
    setSearchTerm(newValue);
    setIsOpen(true);
    // Manuel giriş durumunda kalori 0 olur (listeden seçilmezse)
    onChange(newValue, getMealCalories(newValue));
  };

  const handleClear = () => {
    setSearchTerm('');
    onChange('', 0);
  };

  return (
    <div ref={containerRef} className="relative">
      <div className="flex items-center">
        <Input
          value={searchTerm}
          onChange={handleInputChange}
          onFocus={() => !disabled && setIsOpen(true)}
          placeholder={placeholder}
          disabled={disabled}
          className={`h-8 text-xs text-center uppercase border-0 bg-transparent focus:bg-white dark:focus:bg-gray-800 pr-6 ${disabled ? 'cursor-not-allowed opacity-60' : ''}`}
        />
        {searchTerm && !disabled && (
          <button
            onClick={handleClear}
            className="absolute right-1 p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            type="button"
          >
            <span className="text-xs">✕</span>
          </button>
        )}
      </div>
      {isOpen && !disabled && filteredMeals.length > 0 && (
        <div className="absolute z-50 w-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-48 overflow-y-auto">
          {filteredMeals.map((meal, idx) => (
            <button
              key={idx}
              type="button"
              onClick={() => handleSelect(meal)}
              className="w-full px-2 py-1.5 text-left text-xs hover:bg-gray-100 dark:hover:bg-gray-700 flex justify-between items-center"
            >
              <span className="uppercase">{meal.name}</span>
              <span className="text-gray-400 text-[10px]">{meal.calories} kcal</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// Vegan Autocomplete Combobox Component
function VeganMealAutocomplete({ value, onChange, placeholder, categoryFilter, disabled = false }: MealAutocompleteProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState(value);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSearchTerm(value);
  }, [value]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Vegan veritabanından filtreleme
  const filteredMeals = veganMealDatabase.filter(meal => {
    const matchesSearch = meal.name.includes(searchTerm.toUpperCase());
    const matchesCategory = !categoryFilter || meal.category === categoryFilter;
    return matchesSearch && matchesCategory;
  }).slice(0, 8);

  const handleSelect = (meal: MealItem) => {
    setSearchTerm(meal.name);
    onChange(meal.name, meal.calories);
    setIsOpen(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value.toUpperCase();
    setSearchTerm(newValue);
    setIsOpen(true);
    onChange(newValue, getVeganMealCalories(newValue));
  };

  const handleClear = () => {
    setSearchTerm('');
    onChange('', 0);
  };

  return (
    <div ref={containerRef} className="relative">
      <div className="flex items-center">
        <Input
          value={searchTerm}
          onChange={handleInputChange}
          onFocus={() => !disabled && setIsOpen(true)}
          placeholder={placeholder}
          disabled={disabled}
          className={`h-8 text-xs text-center uppercase border-0 bg-transparent focus:bg-white dark:focus:bg-gray-800 pr-6 ${disabled ? 'cursor-not-allowed opacity-60' : ''}`}
        />
        {searchTerm && !disabled && (
          <button
            onClick={handleClear}
            className="absolute right-1 p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            type="button"
          >
            <span className="text-xs">✕</span>
          </button>
        )}
      </div>
      {isOpen && !disabled && filteredMeals.length > 0 && (
        <div className="absolute z-50 w-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-48 overflow-y-auto">
          {filteredMeals.map((meal, idx) => (
            <button
              key={idx}
              type="button"
              onClick={() => handleSelect(meal)}
              className="w-full px-2 py-1.5 text-left text-xs hover:bg-gray-100 dark:hover:bg-gray-700 flex justify-between items-center"
            >
              <span className="uppercase">{meal.name}</span>
              <span className="text-gray-400 text-[10px]">{meal.calories} kcal</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// Kategori index'inden kategori tipine dönüşüm
const getCategoryByIndex = (index: number): 'soup' | 'main' | 'side' | 'dessert' | 'other' => {
  const categories: ('soup' | 'main' | 'side' | 'dessert' | 'other')[] = ['soup', 'main', 'side', 'dessert', 'other'];
  return categories[index];
};

// API functions
interface SaveMenuPayload {
  year: number;
  month: number;
  menu_data: {
    normalMenus: WeeklyMenu[];
    veganMenus: WeeklyMenu[];
  };
}

interface MonthlyMenuResponse {
  id: string;
  year: number;
  month: number;
  menu_data: {
    normalMenus: WeeklyMenu[];
    veganMenus: WeeklyMenu[];
  };
  created_at: string;
  updated_at: string;
}

const saveMonthlyMenu = async (payload: SaveMenuPayload): Promise<void> => {
  await mealApi.post('menu/monthly', { json: payload });
};

interface ApiSuccessResponse<T> {
  success: boolean;
  data: T;
}

const fetchMonthlyMenu = async (year: number, month: number): Promise<MonthlyMenuResponse | null> => {
  try {
    const response = await mealApi.get('menu/monthly', {
      searchParams: { year, month }
    }).json<ApiSuccessResponse<MonthlyMenuResponse>>();
    return response.data;
  } catch {
    return null;
  }
};

// Sabit 5 yemek kategorisi
const mealCategories = ['Çorba', 'Ana Yemek', 'Yan Yemek', 'Tatlı', 'Diğer'];

// Yemek veritabanı - Türk yemekhanelerinde yaygın yemekler ve kalorileri
interface MealItem {
  name: string;
  calories: number;
  category: 'soup' | 'main' | 'side' | 'dessert' | 'other';
}

const mealDatabase: MealItem[] = [
  // ÇORBALAR (soup) - ortalama 80-150 kcal
  { name: 'MERCİMEK ÇORBASI', calories: 120, category: 'soup' },
  { name: 'YEŞİL MERCİMEK ÇORBASI', calories: 125, category: 'soup' },
  { name: 'KIRMIZI MERCİMEK ÇORBASI', calories: 118, category: 'soup' },
  { name: 'EZOGELİN ÇORBASI', calories: 130, category: 'soup' },
  { name: 'DOMATES ÇORBASI', calories: 85, category: 'soup' },
  { name: 'TARHANA ÇORBASI', calories: 110, category: 'soup' },
  { name: 'YAYLA ÇORBASI', calories: 95, category: 'soup' },
  { name: 'TUTMAÇ ÇORBASI', calories: 140, category: 'soup' },
  { name: 'DÜĞÜN ÇORBASI', calories: 150, category: 'soup' },
  { name: 'ŞEHRİYE ÇORBASI', calories: 90, category: 'soup' },
  { name: 'PİRİNÇ ÇORBASI', calories: 100, category: 'soup' },
  { name: 'SEBZE ÇORBASI', calories: 75, category: 'soup' },
  { name: 'TAVUK SUYU ÇORBASI', calories: 80, category: 'soup' },
  { name: 'İŞKEMBE ÇORBASI', calories: 160, category: 'soup' },
  { name: 'PAÇA ÇORBASI', calories: 170, category: 'soup' },
  { name: 'KREMALI MANTAR ÇORBASI', calories: 145, category: 'soup' },
  { name: 'KREMALI BROKOLI ÇORBASI', calories: 130, category: 'soup' },
  { name: 'BALKABAĞI ÇORBASI', calories: 95, category: 'soup' },
  { name: 'PATATES ÇORBASI', calories: 115, category: 'soup' },
  { name: 'UN ÇORBASI', calories: 105, category: 'soup' },

  // ANA YEMEKLER (main) - ortalama 250-450 kcal
  { name: 'KURU FASULYE', calories: 280, category: 'main' },
  { name: 'NOHUT YEMEĞİ', calories: 260, category: 'main' },
  { name: 'ETLİ NOHUT', calories: 320, category: 'main' },
  { name: 'ETLİ KURU FASULYE', calories: 340, category: 'main' },
  { name: 'BARBUNYA PİLAKİ', calories: 245, category: 'main' },
  { name: 'KARNIYARIK', calories: 380, category: 'main' },
  { name: 'İMAM BAYILDI', calories: 290, category: 'main' },
  { name: 'MUSAKKA', calories: 350, category: 'main' },
  { name: 'TÜRLÜ', calories: 220, category: 'main' },
  { name: 'GÜVEÇ', calories: 310, category: 'main' },
  { name: 'ETLİ GÜVEÇ', calories: 380, category: 'main' },
  { name: 'TAVUK GÜVEÇ', calories: 340, category: 'main' },
  { name: 'TAVUK SOTE', calories: 320, category: 'main' },
  { name: 'ET SOTE', calories: 380, category: 'main' },
  { name: 'MANTARLI ET SOTE', calories: 395, category: 'main' },
  { name: 'ANKARA TAVASI', calories: 420, category: 'main' },
  { name: 'İZMİR KÖFTE', calories: 400, category: 'main' },
  { name: 'KÖFTE', calories: 350, category: 'main' },
  { name: 'KADINBUDU KÖFTE', calories: 380, category: 'main' },
  { name: 'PATATES OTURTMA', calories: 320, category: 'main' },
  { name: 'BEĞENDI', calories: 340, category: 'main' },
  { name: 'HÜNKARBEĞENDİ', calories: 420, category: 'main' },
  { name: 'ALİ NAZİK', calories: 400, category: 'main' },
  { name: 'TAVUKLU PİLAV', calories: 380, category: 'main' },
  { name: 'PİLİÇ FIRINDA', calories: 360, category: 'main' },
  { name: 'TAVUK ŞNİTZEL', calories: 420, category: 'main' },
  { name: 'ET ŞNİTZEL', calories: 450, category: 'main' },
  { name: 'BALIK TAVA', calories: 380, category: 'main' },
  { name: 'BALIK IZGARA', calories: 320, category: 'main' },
  { name: 'HAMSI TAVA', calories: 340, category: 'main' },
  { name: 'PALAMUT TAVA', calories: 360, category: 'main' },
  { name: 'SULU KÖFTE', calories: 290, category: 'main' },
  { name: 'ANA YEMEK', calories: 350, category: 'main' },
  { name: 'MANTARLI TAVUK', calories: 340, category: 'main' },
  { name: 'SEBZELI TAVUK', calories: 310, category: 'main' },
  { name: 'BAMYA YEMEĞİ', calories: 180, category: 'main' },
  { name: 'ETLİ BAMYA', calories: 280, category: 'main' },
  { name: 'PATLICAN KEBABI', calories: 390, category: 'main' },
  { name: 'ADANA KEBAP', calories: 420, category: 'main' },
  { name: 'URFA KEBAP', calories: 400, category: 'main' },
  { name: 'ÇOP ŞİŞ', calories: 380, category: 'main' },
  { name: 'TAVUK ŞİŞ', calories: 320, category: 'main' },
  { name: 'DÖNER', calories: 400, category: 'main' },
  { name: 'TAVUK DÖNER', calories: 350, category: 'main' },

  // YAN YEMEKLER / PİLAVLAR / MAKARNALAR (side) - ortalama 150-280 kcal
  { name: 'PİRİNÇ PİLAVI', calories: 200, category: 'side' },
  { name: 'BULGUR PİLAVI', calories: 180, category: 'side' },
  { name: 'ŞEHRİYELİ PİRİNÇ PİLAVI', calories: 220, category: 'side' },
  { name: 'NOHUTlu PİLAV', calories: 240, category: 'side' },
  { name: 'İÇ PİLAV', calories: 260, category: 'side' },
  { name: 'MAKARNA', calories: 220, category: 'side' },
  { name: 'SOSLU MAKARNA', calories: 280, category: 'side' },
  { name: 'DOMATES SOSLU MAKARNA', calories: 260, category: 'side' },
  { name: 'KIYMALIMAKARNA', calories: 340, category: 'side' },
  { name: 'FIRIN MAKARNA', calories: 320, category: 'side' },
  { name: 'ERİŞTE', calories: 230, category: 'side' },
  { name: 'YOĞURTLU ERİŞTE', calories: 280, category: 'side' },
  { name: 'MANTI', calories: 350, category: 'side' },
  { name: 'ISPANAK', calories: 80, category: 'side' },
  { name: 'PİRİNÇLİ ISPANAK', calories: 140, category: 'side' },
  { name: 'YEŞİL FASULYE', calories: 90, category: 'side' },
  { name: 'ZEYTİNYAĞLI YEŞİL FASULYE', calories: 120, category: 'side' },
  { name: 'BEZELYE', calories: 100, category: 'side' },
  { name: 'ETLİ BEZELYE', calories: 200, category: 'side' },
  { name: 'HAVUÇLU BEZELYE', calories: 130, category: 'side' },
  { name: 'PATATES PÜRESİ', calories: 180, category: 'side' },
  { name: 'PATATES KIZARTMASI', calories: 280, category: 'side' },
  { name: 'FIRINDA PATATES', calories: 200, category: 'side' },
  { name: 'KABAK MÜCVER', calories: 220, category: 'side' },
  { name: 'KIZARTMA', calories: 250, category: 'side' },
  { name: 'SALÇALI PATATES', calories: 190, category: 'side' },
  { name: 'ZEYTINYAĞLI PATLICAN', calories: 150, category: 'side' },
  { name: 'ZEYTINYAĞLI BİBER DOLMASI', calories: 170, category: 'side' },
  { name: 'ZEYTİNYAĞLI YAPRAK SARMASI', calories: 180, category: 'side' },
  { name: 'ETLİ YAPRAK SARMASI', calories: 280, category: 'side' },
  { name: 'LAHANA SARMASI', calories: 250, category: 'side' },

  // TATLILAR (dessert) - ortalama 150-350 kcal
  { name: 'TATLI', calories: 200, category: 'dessert' },
  { name: 'SÜTLAÇ', calories: 180, category: 'dessert' },
  { name: 'KAZANDIBI', calories: 220, category: 'dessert' },
  { name: 'TAVUK GÖĞSÜ', calories: 210, category: 'dessert' },
  { name: 'MUHALLEBI', calories: 170, category: 'dessert' },
  { name: 'PUDING', calories: 190, category: 'dessert' },
  { name: 'ÇİKOLATALI PUDING', calories: 220, category: 'dessert' },
  { name: 'KESMEBİR', calories: 200, category: 'dessert' },
  { name: 'BAKLAVA', calories: 350, category: 'dessert' },
  { name: 'KADAYIF', calories: 320, category: 'dessert' },
  { name: 'KÜNEFE', calories: 380, category: 'dessert' },
  { name: 'REVANI', calories: 280, category: 'dessert' },
  { name: 'ŞAMBALI', calories: 290, category: 'dessert' },
  { name: 'TULUMBA TATLISI', calories: 300, category: 'dessert' },
  { name: 'LOKMA TATLISI', calories: 280, category: 'dessert' },
  { name: 'KEMALPAŞA TATLISI', calories: 260, category: 'dessert' },
  { name: 'UN HELVASI', calories: 320, category: 'dessert' },
  { name: 'İRMİK HELVASI', calories: 300, category: 'dessert' },
  { name: 'AŞURE', calories: 250, category: 'dessert' },
  { name: 'GÜLLAÇ', calories: 200, category: 'dessert' },
  { name: 'KOMPOSTO', calories: 80, category: 'dessert' },
  { name: 'MEYVE', calories: 60, category: 'dessert' },
  { name: 'MEVSIM MEYVE', calories: 65, category: 'dessert' },
  { name: 'KARPUZ', calories: 45, category: 'dessert' },
  { name: 'KAVUN', calories: 50, category: 'dessert' },
  { name: 'PORTAKAL', calories: 55, category: 'dessert' },
  { name: 'ELMA', calories: 60, category: 'dessert' },
  { name: 'MUZ', calories: 90, category: 'dessert' },
  { name: 'DONDURMA', calories: 180, category: 'dessert' },
  { name: 'PASTA', calories: 350, category: 'dessert' },
  { name: 'KEK', calories: 280, category: 'dessert' },

  // DİĞER - Salatalar, içecekler, ek malzemeler (other) - ortalama 30-150 kcal
  { name: 'CACIK', calories: 80, category: 'other' },
  { name: 'YOĞURT', calories: 90, category: 'other' },
  { name: 'AYRAN', calories: 60, category: 'other' },
  { name: 'ÇOBAN SALATA', calories: 70, category: 'other' },
  { name: 'MEVSİM SALATA', calories: 50, category: 'other' },
  { name: 'KARIŞIK SALATA', calories: 60, category: 'other' },
  { name: 'AKDENIZ SALATASI', calories: 90, category: 'other' },
  { name: 'ROKA SALATA', calories: 45, category: 'other' },
  { name: 'HAVUÇ SALATA', calories: 55, category: 'other' },
  { name: 'LAHANA SALATA', calories: 40, category: 'other' },
  { name: 'TURŞU', calories: 25, category: 'other' },
  { name: 'ZEYTİN', calories: 50, category: 'other' },
  { name: 'BEYAZ PEYNİR', calories: 100, category: 'other' },
  { name: 'KAŞAR PEYNİR', calories: 120, category: 'other' },
  { name: 'HAYDARİ', calories: 110, category: 'other' },
  { name: 'ATOM', calories: 90, category: 'other' },
  { name: 'ACILI EZME', calories: 70, category: 'other' },
  { name: 'HUMUS', calories: 130, category: 'other' },
  { name: 'FAVA', calories: 120, category: 'other' },
  { name: 'PATLICAN SALATASI', calories: 100, category: 'other' },
  { name: 'KÖZLENMIŞ BİBER', calories: 40, category: 'other' },
  { name: 'SÖGÜŞ', calories: 30, category: 'other' },
  { name: 'EKMEK', calories: 136, category: 'other' },
];

// Vegan yemek veritabanı
const veganMealDatabase: MealItem[] = [
  // VEGAN ÇORBALAR
  { name: 'MERCİMEK ÇORBASI', calories: 120, category: 'soup' },
  { name: 'SEBZE ÇORBASI', calories: 75, category: 'soup' },
  { name: 'DOMATES ÇORBASI', calories: 85, category: 'soup' },
  { name: 'EZOGELİN ÇORBASI', calories: 130, category: 'soup' },
  { name: 'TARHANA ÇORBASI (VEGAN)', calories: 100, category: 'soup' },
  { name: 'BALKABAGI ÇORBASI', calories: 95, category: 'soup' },
  { name: 'BROKOLI ÇORBASI', calories: 80, category: 'soup' },
  { name: 'ISPANAK ÇORBASI', calories: 70, category: 'soup' },
  { name: 'KARNABAHAR ÇORBASI', calories: 65, category: 'soup' },
  { name: 'PATATES ÇORBASI', calories: 115, category: 'soup' },
  { name: 'HAVUÇ ÇORBASI', calories: 60, category: 'soup' },
  { name: 'MANTAR ÇORBASI (VEGAN)', calories: 90, category: 'soup' },
  // VEGAN ANA YEMEKLER
  { name: 'NOHUT YEMEĞİ', calories: 260, category: 'main' },
  { name: 'KURU FASULYE', calories: 280, category: 'main' },
  { name: 'BARBUNYA PİLAKİ', calories: 245, category: 'main' },
  { name: 'MERCİMEK KÖFTE', calories: 180, category: 'main' },
  { name: 'ZEYTİNYAĞLI FASULYE', calories: 150, category: 'main' },
  { name: 'ZEYTİNYAĞLI ENGINAR', calories: 140, category: 'main' },
  { name: 'ZEYTİNYAĞLI BAKLA', calories: 160, category: 'main' },
  { name: 'ZEYTİNYAĞLI PAZI', calories: 100, category: 'main' },
  { name: 'İMAM BAYILDI', calories: 290, category: 'main' },
  { name: 'TÜRLÜ', calories: 220, category: 'main' },
  { name: 'SEBZE GÜVEÇ', calories: 180, category: 'main' },
  { name: 'PATLICAN MUSAKKA (VEGAN)', calories: 200, category: 'main' },
  { name: 'KABAK DOLMASI', calories: 170, category: 'main' },
  { name: 'BİBER DOLMASI', calories: 180, category: 'main' },
  { name: 'YAPRAK SARMASI', calories: 190, category: 'main' },
  { name: 'MANTARLI SEBZE SOTE', calories: 150, category: 'main' },
  { name: 'SEBZELI NOHUT', calories: 240, category: 'main' },
  { name: 'VEGAN KÖFTE', calories: 200, category: 'main' },
  // VEGAN YAN YEMEKLER
  { name: 'PİRİNÇ PİLAVI', calories: 200, category: 'side' },
  { name: 'BULGUR PİLAVI', calories: 180, category: 'side' },
  { name: 'SOSLU MAKARNA', calories: 240, category: 'side' },
  { name: 'ISPANAK', calories: 80, category: 'side' },
  { name: 'ZEYTİNYAĞLI YEŞİL FASULYE', calories: 120, category: 'side' },
  { name: 'HAVUÇLU BEZELYE', calories: 130, category: 'side' },
  { name: 'PATATES PÜRESİ (VEGAN)', calories: 160, category: 'side' },
  { name: 'FIRINDA PATATES', calories: 200, category: 'side' },
  { name: 'KABAK MÜCVER (VEGAN)', calories: 180, category: 'side' },
  { name: 'ZEYTİNYAĞLI PATLICAN', calories: 150, category: 'side' },
  { name: 'KINOA SALATASI', calories: 170, category: 'side' },
  // VEGAN TATLILAR
  { name: 'MEYVE', calories: 60, category: 'dessert' },
  { name: 'MEVSİM MEYVE', calories: 65, category: 'dessert' },
  { name: 'KOMPOSTO', calories: 80, category: 'dessert' },
  { name: 'AŞURE', calories: 250, category: 'dessert' },
  { name: 'KABAK TATLISI', calories: 180, category: 'dessert' },
  { name: 'AYVA TATLISI', calories: 160, category: 'dessert' },
  { name: 'İNCİR TATLISI', calories: 170, category: 'dessert' },
  { name: 'HURMA', calories: 100, category: 'dessert' },
  { name: 'CEVİZLİ İNCİR', calories: 150, category: 'dessert' },
  { name: 'MEYVE SALATASI', calories: 90, category: 'dessert' },
  // VEGAN DİĞER
  { name: 'ÇOBAN SALATA', calories: 70, category: 'other' },
  { name: 'MEVSİM SALATA', calories: 50, category: 'other' },
  { name: 'AKDENIZ SALATASI', calories: 90, category: 'other' },
  { name: 'ROKA SALATA', calories: 45, category: 'other' },
  { name: 'HUMUS', calories: 130, category: 'other' },
  { name: 'FAVA', calories: 120, category: 'other' },
  { name: 'TURŞU', calories: 25, category: 'other' },
  { name: 'ZEYTİN', calories: 50, category: 'other' },
  { name: 'ACILI EZME', calories: 70, category: 'other' },
  { name: 'PATLICAN SALATASI', calories: 100, category: 'other' },
  { name: 'EKMEK', calories: 136, category: 'other' },
];

// Yemek adından kalori bul
const getMealCalories = (mealName: string): number => {
  const meal = mealDatabase.find(m => m.name === mealName.toUpperCase());
  return meal?.calories || 0;
};

// Vegan yemek adından kalori bul
const getVeganMealCalories = (mealName: string): number => {
  const meal = veganMealDatabase.find(m => m.name === mealName.toUpperCase());
  return meal?.calories || 0;
};

interface DayMenu {
  items: [string, string, string, string, string]; // 5 sabit alan
  calories: number;
}

interface WeeklyMenu {
  [key: string]: DayMenu;
}

const dayNames = [
  { key: 'monday', label: 'Pazartesi' },
  { key: 'tuesday', label: 'Salı' },
  { key: 'wednesday', label: 'Çarşamba' },
  { key: 'thursday', label: 'Perşembe' },
  { key: 'friday', label: 'Cuma' },
];

const getEmptyWeeklyMenu = (): WeeklyMenu => ({
  monday: { items: ['', '', '', '', ''], calories: 0 },
  tuesday: { items: ['', '', '', '', ''], calories: 0 },
  wednesday: { items: ['', '', '', '', ''], calories: 0 },
  thursday: { items: ['', '', '', '', ''], calories: 0 },
  friday: { items: ['', '', '', '', ''], calories: 0 },
});

// Kalori hesaplama helper - örnek veri için
const calcCalories = (items: string[]): number => {
  return items.reduce((total, item) => total + getMealCalories(item), 0);
};

const calcVeganCalories = (items: string[]): number => {
  return items.reduce((total, item) => total + getVeganMealCalories(item), 0);
};

// Örnek veri - DEU formatında (kaloriler otomatik hesaplanır)
const sampleMenuItems = {
  monday: ['YEŞİL MERCİMEK ÇORBASI', 'KURU FASULYE', 'PİRİNÇ PİLAVI', 'SÜTLAÇ', 'TURŞU'] as [string, string, string, string, string],
  tuesday: ['DOMATES ÇORBASI', 'PATATES OTURTMA', 'MAKARNA', 'MEYVE', 'HAYDARİ'] as [string, string, string, string, string],
  wednesday: ['TUTMAÇ ÇORBASI', 'ANKARA TAVASI', 'PİRİNÇLİ ISPANAK', 'REVANI', 'YOĞURT'] as [string, string, string, string, string],
  thursday: ['', '', '', '', ''] as [string, string, string, string, string],
  friday: ['EZOGELİN ÇORBASI', 'MANTARLI ET SOTE', 'BULGUR PİLAVI', 'KOMPOSTO', 'ÇOBAN SALATA'] as [string, string, string, string, string],
};

const sampleWeeklyMenu: WeeklyMenu = {
  monday: { items: sampleMenuItems.monday, calories: calcCalories(sampleMenuItems.monday) },
  tuesday: { items: sampleMenuItems.tuesday, calories: calcCalories(sampleMenuItems.tuesday) },
  wednesday: { items: sampleMenuItems.wednesday, calories: calcCalories(sampleMenuItems.wednesday) },
  thursday: { items: sampleMenuItems.thursday, calories: calcCalories(sampleMenuItems.thursday) },
  friday: { items: sampleMenuItems.friday, calories: calcCalories(sampleMenuItems.friday) },
};

// Vegan örnek menü
const sampleVeganMenuItems = {
  monday: ['MERCİMEK ÇORBASI', 'NOHUT YEMEĞİ', 'BULGUR PİLAVI', 'MEYVE', 'HUMUS'] as [string, string, string, string, string],
  tuesday: ['SEBZE ÇORBASI', 'ZEYTİNYAĞLI FASULYE', 'PİRİNÇ PİLAVI', 'KOMPOSTO', 'ÇOBAN SALATA'] as [string, string, string, string, string],
  wednesday: ['DOMATES ÇORBASI', 'İMAM BAYILDI', 'SOSLU MAKARNA', 'KABAK TATLISI', 'TURŞU'] as [string, string, string, string, string],
  thursday: ['', '', '', '', ''] as [string, string, string, string, string],
  friday: ['EZOGELİN ÇORBASI', 'SEBZE GÜVEÇ', 'ZEYTİNYAĞLI YEŞİL FASULYE', 'MEVSİM MEYVE', 'FAVA'] as [string, string, string, string, string],
};

const sampleVeganWeeklyMenu: WeeklyMenu = {
  monday: { items: sampleVeganMenuItems.monday, calories: calcVeganCalories(sampleVeganMenuItems.monday) },
  tuesday: { items: sampleVeganMenuItems.tuesday, calories: calcVeganCalories(sampleVeganMenuItems.tuesday) },
  wednesday: { items: sampleVeganMenuItems.wednesday, calories: calcVeganCalories(sampleVeganMenuItems.wednesday) },
  thursday: { items: sampleVeganMenuItems.thursday, calories: calcVeganCalories(sampleVeganMenuItems.thursday) },
  friday: { items: sampleVeganMenuItems.friday, calories: calcVeganCalories(sampleVeganMenuItems.friday) },
};

// Değişiklik bilgisi tipi
interface MenuChange {
  weekIndex: number;
  day: string;
  dayLabel: string;
  dateString: string; // "25 Ocak 2026" formatında tarih
  itemIndex: number;
  category: string;
  oldValue: string;
  newValue: string;
  isVegan: boolean;
  timestamp: Date;
}

export default function MenusPage() {
  const [activeTab, setActiveTab] = useState<string>('normal');

  // Bilgisayar tarihinden ay ve yılı otomatik belirle
  const currentDate = new Date();
  const systemMonth = String(currentDate.getMonth() + 1).padStart(2, '0');
  const systemYear = String(currentDate.getFullYear());

  // Seçilen ay ve yıl (kullanıcı tarafından değiştirilebilir)
  const [selectedMonth, setSelectedMonth] = useState(systemMonth);
  const [selectedYear, setSelectedYear] = useState(systemYear);

  const [weeklyMenus, setWeeklyMenus] = useState<WeeklyMenu[]>([
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
  ]);
  const [veganWeeklyMenus, setVeganWeeklyMenus] = useState<WeeklyMenu[]>([
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
    getEmptyWeeklyMenu(),
  ]);

  // Menü durumu: 'empty' (boş), 'saved' (kaydedildi), 'editing' (düzenleniyor)
  const [menuStatus, setMenuStatus] = useState<'empty' | 'saved' | 'editing'>('empty');

  // Düzenleme modu aktif mi?
  const [isEditMode, setIsEditMode] = useState<boolean>(true); // Başlangıçta düzenlenebilir (yeni menü oluşturma)

  // Kaydedilen menünün kopyası (değişiklikleri karşılaştırmak için)
  const [savedWeeklyMenus, setSavedWeeklyMenus] = useState<WeeklyMenu[] | null>(null);
  const [savedVeganWeeklyMenus, setSavedVeganWeeklyMenus] = useState<WeeklyMenu[] | null>(null);

  // Yeni menü oluşturma modunda mı? (tablo gösterilsin mi)
  const [isCreatingNew, setIsCreatingNew] = useState<boolean>(false);

  // Değişiklikler listesi
  const [pendingChanges, setPendingChanges] = useState<MenuChange[]>([]);

  // Toast state
  const [toast, setToast] = useState<{ message: string; type: 'error' | 'warning' | 'success' | 'info'; isVisible: boolean }>({
    message: '',
    type: 'success',
    isVisible: false,
  });

  const showToast = (message: string, type: 'error' | 'warning' | 'success' | 'info') => {
    setToast({ message, type, isVisible: true });
  };

  const hideToast = () => {
    setToast(prev => ({ ...prev, isVisible: false }));
  };

  // Yıl seçenekleri (mevcut yıl -1'den +3 yıla kadar)
  const yearOptions = useMemo(() => {
    const years = [];
    const currentYear = currentDate.getFullYear();
    for (let y = currentYear - 1; y <= currentYear + 3; y++) {
      years.push(String(y));
    }
    return years;
  }, [currentDate]);

  // Seçilen ay/yılın geçmişte olup olmadığını kontrol et
  const isPastMonth = useMemo(() => {
    const selectedDate = new Date(parseInt(selectedYear), parseInt(selectedMonth) - 1, 1);
    const currentFirstDay = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1);
    return selectedDate < currentFirstDay;
  }, [selectedYear, selectedMonth, currentDate]);

  // Seçilen ay/yılın çok ileride olup olmadığını kontrol et (sadece mevcut ay ve sonraki ay düzenlenebilir)
  const isFutureMonth = useMemo(() => {
    const selectedDate = new Date(parseInt(selectedYear), parseInt(selectedMonth) - 1, 1);
    // Sonraki ayın 1'i (maksimum düzenlenebilir ay)
    const maxEditableDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1);
    return selectedDate > maxEditableDate;
  }, [selectedYear, selectedMonth, currentDate]);

  // Düzenleme yapılabilir mi? (sadece mevcut ay ve sonraki ay)
  const canEdit = !isPastMonth && !isFutureMonth;

  // Seçilen ay/yıl için menü sorgulama
  const { data: existingMenu, isLoading: isLoadingMenu, refetch: refetchMenu } = useQuery({
    queryKey: ['monthlyMenu', selectedYear, selectedMonth],
    queryFn: () => fetchMonthlyMenu(parseInt(selectedYear), parseInt(selectedMonth)),
  });

  // Menü verisi geldiğinde state'leri güncelle
  useEffect(() => {
    if (existingMenu) {
      // Veritabanından gelen menüyü form state'ine yükle
      setWeeklyMenus(existingMenu.menu_data.normalMenus);
      setVeganWeeklyMenus(existingMenu.menu_data.veganMenus);
      setSavedWeeklyMenus(JSON.parse(JSON.stringify(existingMenu.menu_data.normalMenus)));
      setSavedVeganWeeklyMenus(JSON.parse(JSON.stringify(existingMenu.menu_data.veganMenus)));
      setMenuStatus('saved');
      setIsEditMode(false);
      setIsCreatingNew(false);
      setPendingChanges([]);
    } else {
      // Menü yoksa boş form göster
      setWeeklyMenus([
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
      ]);
      setVeganWeeklyMenus([
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
        getEmptyWeeklyMenu(),
      ]);
      setSavedWeeklyMenus(null);
      setSavedVeganWeeklyMenus(null);
      setMenuStatus('empty');
      setIsEditMode(false); // Başlangıçta düzenleme kapalı
      setIsCreatingNew(false); // Oluştur butonuna basılana kadar tablo gizli
      setPendingChanges([]);
    }
  }, [existingMenu, isPastMonth, selectedYear, selectedMonth]);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: saveMonthlyMenu,
    onSuccess: () => {
      showToast('Menü başarıyla kaydedildi!', 'success');
    },
    onError: (error: Error) => {
      console.error('Menü kaydedilirken hata oluştu:', error);
      showToast('Menü kaydedilirken bir hata oluştu!', 'error');
    },
  });

  const isSubmitting = saveMutation.isPending;

  // Ay ve yıl seçenekleri
  const months = [
    { value: '01', label: 'Ocak' },
    { value: '02', label: 'Şubat' },
    { value: '03', label: 'Mart' },
    { value: '04', label: 'Nisan' },
    { value: '05', label: 'Mayıs' },
    { value: '06', label: 'Haziran' },
    { value: '07', label: 'Temmuz' },
    { value: '08', label: 'Ağustos' },
    { value: '09', label: 'Eylül' },
    { value: '10', label: 'Ekim' },
    { value: '11', label: 'Kasım' },
    { value: '12', label: 'Aralık' },
  ];

  // Haftanın tarihlerini hesapla
  const getWeekDates = (weekIndex: number) => {
    const year = parseInt(selectedYear);
    const month = parseInt(selectedMonth) - 1;
    const firstDay = new Date(year, month, 1);
    const dayOfWeek = firstDay.getDay();
    const startOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;

    const weekStart = new Date(year, month, 1 + startOffset + (weekIndex * 7));

    return dayNames.map((_, idx) => {
      const date = new Date(weekStart);
      date.setDate(weekStart.getDate() + idx);
      return `${date.getDate()} ${months[date.getMonth()]?.label || ''} ${date.getFullYear()}`;
    });
  };

  // Günün toplam kalorisini hesapla
  const calculateDayCalories = (items: string[], isVegan = false): number => {
    return items.reduce((total, item) => {
      if (!item) return total;
      return total + (isVegan ? getVeganMealCalories(item) : getMealCalories(item));
    }, 0);
  };

  // Gün adını bul
  const getDayLabel = (dayKey: string): string => {
    const dayObj = dayNames.find(d => d.key === dayKey);
    return dayObj?.label || dayKey;
  };

  // Belirli bir hafta ve gün için tarih string'i hesapla
  const getDateStringForDay = (weekIndex: number, dayKey: string): string => {
    const year = parseInt(selectedYear);
    const month = parseInt(selectedMonth) - 1;
    const firstDay = new Date(year, month, 1);
    const dayOfWeek = firstDay.getDay();
    const startOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
    const weekStart = new Date(year, month, 1 + startOffset + (weekIndex * 7));

    // Gün index'ini bul (monday=0, tuesday=1, ...)
    const dayIndex = dayNames.findIndex(d => d.key === dayKey);
    const targetDate = new Date(weekStart);
    targetDate.setDate(weekStart.getDate() + dayIndex);

    const dayNum = targetDate.getDate();
    const monthLabel = months[targetDate.getMonth()]?.label || '';
    const yearNum = targetDate.getFullYear();

    return `${dayNum} ${monthLabel} ${yearNum}`;
  };

  // Değişiklik ekle - oldValue her zaman kaydedilmiş (orijinal) değerden alınır
  const addChange = (
    weekIndex: number,
    day: string,
    itemIndex: number,
    newValue: string,
    isVegan: boolean
  ) => {
    // Sadece kaydedilmiş menü varsa değişiklik izle
    if (menuStatus !== 'saved') return;

    const savedMenus = isVegan ? savedVeganWeeklyMenus : savedWeeklyMenus;
    if (!savedMenus) return;

    // Orijinal (kaydedilmiş) değeri al - bu her zaman sabit kalır
    const originalValue = savedMenus[weekIndex]?.[day]?.items[itemIndex] || '';

    // Eğer yeni değer orijinal değerle aynıysa, bu hücre için değişiklik kaldır
    if (newValue === originalValue) {
      setPendingChanges(prev => 
        prev.filter(c => !(c.weekIndex === weekIndex && c.day === day && c.itemIndex === itemIndex && c.isVegan === isVegan))
      );
      return;
    }

    // Değişiklik oluştur - oldValue her zaman orijinal değer
    const change: MenuChange = {
      weekIndex,
      day,
      dayLabel: getDayLabel(day),
      dateString: getDateStringForDay(weekIndex, day),
      itemIndex,
      category: mealCategories[itemIndex],
      oldValue: originalValue || '(boş)',
      newValue: newValue || '(boş)',
      isVegan,
      timestamp: new Date(),
    };

    setPendingChanges(prev => {
      // Aynı hücre için önceki değişikliği kaldır ve yenisini ekle
      const filtered = prev.filter(
        c => !(c.weekIndex === weekIndex && c.day === day && c.itemIndex === itemIndex && c.isVegan === isVegan)
      );
      return [...filtered, change];
    });
  };

  // Menü öğesi güncelle ve kaloriyi otomatik hesapla
  const updateMenuItem = (weekIndex: number, day: string, itemIndex: number, value: string) => {
    // Düzenleme modu kapalıysa güncelleme yapma
    if (!isEditMode) return;

    const newValue = value.toUpperCase();

    // Değişikliği kaydet (eğer kaydedilmiş menü varsa)
    addChange(weekIndex, day, itemIndex, newValue, false);

    setWeeklyMenus(prev => {
      const updated = [...prev];
      const newItems = [...updated[weekIndex][day].items] as [string, string, string, string, string];
      newItems[itemIndex] = newValue;

      // Otomatik kalori hesapla
      const totalCalories = calculateDayCalories(newItems);

      updated[weekIndex] = {
        ...updated[weekIndex],
        [day]: {
          ...updated[weekIndex][day],
          items: newItems,
          calories: totalCalories,
        },
      };
      return updated;
    });
  };

  // Vegan menü öğesi güncelle
  const updateVeganMenuItem = (weekIndex: number, day: string, itemIndex: number, value: string) => {
    // Düzenleme modu kapalıysa güncelleme yapma
    if (!isEditMode) return;

    const newValue = value.toUpperCase();

    // Değişikliği kaydet (eğer kaydedilmiş menü varsa)
    addChange(weekIndex, day, itemIndex, newValue, true);

    setVeganWeeklyMenus(prev => {
      const updated = [...prev];
      const newItems = [...updated[weekIndex][day].items] as [string, string, string, string, string];
      newItems[itemIndex] = newValue;

      // Otomatik kalori hesapla
      const totalCalories = calculateDayCalories(newItems, true);

      updated[weekIndex] = {
        ...updated[weekIndex],
        [day]: {
          ...updated[weekIndex][day],
          items: newItems,
          calories: totalCalories,
        },
      };
      return updated;
    });
  };

  // Değişikliği kaldır
  const removeChange = (index: number) => {
    setPendingChanges(prev => prev.filter((_, i) => i !== index));
  };

  // Tüm değişiklikleri temizle
  const clearAllChanges = () => {
    setPendingChanges([]);
  };

  // Düzenleme modunu aç
  const enableEditMode = () => {
    setIsEditMode(true);
  };

  // Düzenleme modunu kapat
  const disableEditMode = () => {
    setIsEditMode(false);
  };

  // Kaydet (ilk kez)
  const handleSave = async () => {
    saveMutation.mutate({
      year: parseInt(selectedYear),
      month: parseInt(selectedMonth),
      menu_data: {
        normalMenus: weeklyMenus,
        veganMenus: veganWeeklyMenus,
      },
    }, {
      onSuccess: () => {
        // Kaydedilen menüleri sakla
        setSavedWeeklyMenus(JSON.parse(JSON.stringify(weeklyMenus)));
        setSavedVeganWeeklyMenus(JSON.parse(JSON.stringify(veganWeeklyMenus)));
        setMenuStatus('saved');
        setIsEditMode(false);
        setPendingChanges([]);
        refetchMenu();
      }
    });
  };

  // Güncelle (değişiklikleri kaydet)
  const handleUpdate = async () => {
    saveMutation.mutate({
      year: parseInt(selectedYear),
      month: parseInt(selectedMonth),
      menu_data: {
        normalMenus: weeklyMenus,
        veganMenus: veganWeeklyMenus,
      },
    }, {
      onSuccess: () => {
        // Kaydedilen menüleri güncelle
        setSavedWeeklyMenus(JSON.parse(JSON.stringify(weeklyMenus)));
        setSavedVeganWeeklyMenus(JSON.parse(JSON.stringify(veganWeeklyMenus)));
        setPendingChanges([]);
        setIsEditMode(false);
        refetchMenu();
      }
    });
  };

  const selectedMonthLabel = months.find(m => m.value === selectedMonth)?.label || '';
  const hasUnsavedChanges = pendingChanges.length > 0;

  // Yazdırma fonksiyonu
  const handlePrint = () => {
    const currentMenus = activeTab === 'vegan' ? veganWeeklyMenus : weeklyMenus;
    const menuType = activeTab === 'vegan' ? 'VEGAN MENÜ' : 'YEMEK LİSTESİ';
    const headerColor = activeTab === 'vegan' ? '#166534' : '#1e3a8a';
    const subHeaderColor = activeTab === 'vegan' ? '#dcfce7' : '#fef3c7';

    const printContent = `
      <!DOCTYPE html>
      <html>
      <head>
        <meta charset="utf-8">
        <title>${selectedYear} ${selectedMonthLabel} - ${menuType}</title>
        <style>
          @page {
            size: A4 landscape;
            margin: 5mm;
          }
          * { margin: 0; padding: 0; box-sizing: border-box; }
          html, body {
            width: 100%;
            height: 100%;
            font-family: Arial, sans-serif;
          }
          body {
            padding: 5px;
            display: flex;
            flex-direction: column;
          }
          .container {
            display: flex;
            flex-direction: column;
            height: 100%;
          }
          .header {
            background-color: ${headerColor};
            color: white;
            text-align: center;
            padding: 6px;
            margin-bottom: 5px;
            flex-shrink: 0;
          }
          .header h1 { font-size: 12px; margin-bottom: 2px; }
          .header h2 { font-size: 9px; font-weight: normal; margin-bottom: 2px; }
          .header h3 { font-size: 10px; }
          .weeks-container {
            display: flex;
            flex-direction: column;
            flex: 1;
            gap: 3px;
          }
          .week-container {
            flex: 1;
            display: flex;
            flex-direction: column;
          }
          table {
            width: 100%;
            height: 100%;
            border-collapse: collapse;
            font-size: 7px;
            table-layout: fixed;
          }
          th {
            background-color: ${subHeaderColor};
            padding: 2px 1px;
            border: 1px solid #999;
            text-align: center;
            height: 20px;
          }
          th .date { font-weight: bold; color: ${headerColor}; font-size: 6px; }
          th .day { font-weight: normal; color: #666; font-size: 5px; }
          td {
            border: 1px solid #999;
            padding: 1px;
            text-align: center;
            vertical-align: middle;
            font-size: 6px;
            line-height: 1.1;
            overflow: hidden;
          }
          .meal-item { text-transform: uppercase; }
          .calories-row { background-color: #f0f0f0; }
          .calories { font-weight: bold; color: ${headerColor}; }
          .notes {
            margin-top: 3px;
            font-size: 6px;
            color: #666;
            flex-shrink: 0;
          }
          .notes p { margin-bottom: 1px; }
          @media print {
            html, body {
              width: 297mm;
              height: 210mm;
              overflow: hidden;
              -webkit-print-color-adjust: exact;
              print-color-adjust: exact;
            }
            @page {
              margin: 0;
            }
            /* Tarayıcı header/footer gizle */
            title { display: none; }
          }
        </style>
      </head>
      <body>
        <div class="container">
          <div class="header">
            <h1>DOKUZ EYLÜL ÜNİVERSİTESİ</h1>
            <h2>SAĞLIK KÜLTÜR VE SPOR DAİRE BAŞKANLIĞI</h2>
            <h3>${selectedYear} ${selectedMonthLabel.toUpperCase()} AYI - ${menuType}</h3>
          </div>
          <div class="weeks-container">
            ${currentMenus.map((weekMenu, weekIndex) => {
              const weekDates = getWeekDates(weekIndex);
              return `
                <div class="week-container">
                  <table>
                    <thead>
                      <tr>
                        ${dayNames.map((day, idx) => `
                          <th>
                            <div class="date">${weekDates[idx]}</div>
                            <div class="day">${day.label}</div>
                          </th>
                        `).join('')}
                      </tr>
                    </thead>
                    <tbody>
                      ${mealCategories.map((_, rowIdx) => `
                        <tr>
                          ${dayNames.map(day => `
                            <td class="meal-item">${weekMenu[day.key]?.items[rowIdx] || '-'}</td>
                          `).join('')}
                        </tr>
                      `).join('')}
                      <tr class="calories-row">
                        ${dayNames.map(day => `
                          <td><span class="calories">${weekMenu[day.key]?.calories || 0}</span> kcal</td>
                        `).join('')}
                      </tr>
                    </tbody>
                  </table>
                </div>
              `;
            }).join('')}
          </div>
          <div class="notes">
            <p>*Mücbir sebepler haricinde kesinlikle menü değişimi yapılmayacaktır.</p>
            <p>*Yukarıda belirtilen 1 öğünlük toplam kalori değerlerine, 50 gr ekmeğin değeri olan 136 kalori ilave edilmiştir.</p>
          </div>
        </div>
      </body>
      </html>
    `;

    const printWindow = window.open('', '_blank');
    if (printWindow) {
      printWindow.document.write(printContent);
      printWindow.document.close();
      printWindow.onload = () => {
        printWindow.print();
      };
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-bold dark:text-white">Aylık Menü Yönetimi</h1>
          <p className="text-muted-foreground">
            {selectedMonthLabel} {selectedYear} - {menuStatus === 'saved' ? (isEditMode ? 'Düzenleme modu' : 'Kaydedildi') : 'Yeni menü'}
            {isPastMonth && <span className="ml-2 text-amber-600">(Geçmiş ay - salt okunur)</span>}
            {isFutureMonth && <span className="ml-2 text-purple-600">(İleri tarih - henüz düzenlenemez)</span>}
          </p>
        </div>
        
        {/* Yıl ve Ay Seçimi */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Label htmlFor="year-select" className="text-sm font-medium">Yıl:</Label>
            <Select value={selectedYear} onValueChange={setSelectedYear}>
              <SelectTrigger id="year-select" className="w-24">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {yearOptions.map(year => (
                  <SelectItem key={year} value={year}>{year}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <Label htmlFor="month-select" className="text-sm font-medium">Ay:</Label>
            <Select value={selectedMonth} onValueChange={setSelectedMonth}>
              <SelectTrigger id="month-select" className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {months.map(month => (
                  <SelectItem key={month.value} value={month.value}>{month.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex gap-2">
          <Button variant="outline" onClick={handlePrint}>
            <Printer className="h-4 w-4 mr-2" />
            Yazdır
          </Button>

          {/* Menü kaydedilmişse ve düzenleme modu kapalıysa ve geçmiş ay değilse Düzenle butonu göster */}
          {menuStatus === 'saved' && !isEditMode && canEdit && (
            <Button variant="outline" onClick={enableEditMode}>
              <Pencil className="h-4 w-4 mr-2" />
              Düzenle
            </Button>
          )}

          {/* Menü kaydedilmişse ve düzenleme modundaysa Kilitle butonu göster */}
          {menuStatus === 'saved' && isEditMode && !hasUnsavedChanges && (
            <Button variant="outline" onClick={disableEditMode}>
              <Lock className="h-4 w-4 mr-2" />
              Kilitle
            </Button>
          )}

          {/* Menü yoksa ve henüz oluşturma başlamadıysa - Oluştur butonu (tabloyu gösterir) */}
          {menuStatus === 'empty' && !isCreatingNew && canEdit && (
            <Button onClick={() => { setIsCreatingNew(true); setIsEditMode(true); }}>
              <Plus className="h-4 w-4 mr-2" />
              Oluştur
            </Button>
          )}

          {/* Menü oluşturma modundaysa - Kaydet butonu */}
          {menuStatus === 'empty' && isCreatingNew && canEdit && (
            <Button onClick={handleSave} disabled={isSubmitting}>
              <Save className="h-4 w-4 mr-2" />
              {isSubmitting ? 'Kaydediliyor...' : 'Kaydet'}
            </Button>
          )}

          {/* Değişiklik varsa Güncelle butonu */}
          {hasUnsavedChanges && (
            <Button onClick={handleUpdate} disabled={isSubmitting} className="bg-orange-600 hover:bg-orange-700">
              <Save className="h-4 w-4 mr-2" />
              {isSubmitting ? 'Güncelleniyor...' : 'Güncelle'}
            </Button>
          )}
        </div>
      </div>

      {/* Loading durumu */}
      {isLoadingMenu && (
        <div className="rounded-lg border border-blue-300 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-800 p-4">
          <div className="flex items-center gap-3">
            <Loader2 className="h-5 w-5 text-blue-600 dark:text-blue-400 animate-spin" />
            <p className="text-blue-800 dark:text-blue-300">
              {selectedMonthLabel} {selectedYear} için menü yükleniyor...
            </p>
          </div>
        </div>
      )}

      {/* Geçmiş ay uyarısı */}
      {isPastMonth && (
        <div className="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-900/20 dark:border-amber-800 p-4">
          <div className="flex items-center gap-3">
            <Eye className="h-5 w-5 text-amber-600 dark:text-amber-400" />
            <p className="text-amber-800 dark:text-amber-300">
              Bu menü geçmiş bir aya ait olduğu için yalnızca görüntülenebilir. Düzenleme yapılamaz.
            </p>
          </div>
        </div>
      )}

      {/* Menü yoksa ve henüz oluşturma başlamadıysa ve düzenlenebilir ay ise bilgi */}
      {!isLoadingMenu && menuStatus === 'empty' && !isCreatingNew && canEdit && (
        <div className="rounded-lg border border-green-300 bg-green-50 dark:bg-green-900/20 dark:border-green-800 p-4">
          <div className="flex items-center gap-3">
            <Plus className="h-5 w-5 text-green-600 dark:text-green-400" />
            <p className="text-green-800 dark:text-green-300">
              {selectedMonthLabel} {selectedYear} için henüz menü oluşturulmamış. <strong>Oluştur</strong> butonuna tıklayarak yeni menü oluşturmaya başlayabilirsiniz.
            </p>
          </div>
        </div>
      )}

      {/* Menü yoksa ve geçmiş ay ise bilgi */}
      {!isLoadingMenu && menuStatus === 'empty' && isPastMonth && (
        <div className="rounded-lg border border-gray-300 bg-gray-50 dark:bg-gray-900/20 dark:border-gray-700 p-4">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-gray-600 dark:text-gray-400" />
            <p className="text-gray-800 dark:text-gray-300">
              {selectedMonthLabel} {selectedYear} için menü oluşturulmamış. Geçmiş aylar için yeni menü oluşturulamaz.
            </p>
          </div>
        </div>
      )}

      {/* Menü yoksa ve ileri tarih ise bilgi */}
      {!isLoadingMenu && menuStatus === 'empty' && isFutureMonth && (
        <div className="rounded-lg border border-purple-300 bg-purple-50 dark:bg-purple-900/20 dark:border-purple-700 p-4">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            <p className="text-purple-800 dark:text-purple-300">
              {selectedMonthLabel} {selectedYear} için henüz menü oluşturulamaz. Sadece mevcut ay ve sonraki ay için menü düzenleyebilirsiniz.
            </p>
          </div>
        </div>
      )}

      {/* Değişiklik Uyarısı */}
      {hasUnsavedChanges && (
        <div className="rounded-lg border border-orange-300 bg-orange-50 dark:bg-orange-900/20 dark:border-orange-800 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <h3 className="font-semibold text-orange-800 dark:text-orange-300 mb-2">
                Kaydedilmemiş Değişiklikler ({pendingChanges.length})
              </h3>
              <div className="space-y-2 max-h-40 overflow-y-auto">
                {pendingChanges.map((change, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between text-sm bg-white dark:bg-gray-800 rounded p-2 border border-orange-200 dark:border-orange-700"
                  >
                    <div className="flex-1">
                      <span className="font-medium text-orange-700 dark:text-orange-300">
                        {change.isVegan ? '🥗 Vegan' : '🍖 Normal'} - {change.dateString} ({change.dayLabel})
                      </span>
                      <span className="mx-2 text-gray-500">|</span>
                      <span className="text-gray-600 dark:text-gray-400">{change.category}:</span>
                      <span className="ml-2">
                        <span className="line-through text-red-500">{change.oldValue}</span>
                        <span className="mx-1">→</span>
                        <span className="text-green-600 font-medium">{change.newValue}</span>
                      </span>
                    </div>
                    <button
                      onClick={() => removeChange(idx)}
                      className="ml-2 p-1 hover:bg-orange-100 dark:hover:bg-orange-800 rounded"
                      title="Değişikliği geri al"
                    >
                      <X className="h-4 w-4 text-orange-600" />
                    </button>
                  </div>
                ))}
              </div>
              <div className="mt-3 flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={clearAllChanges}
                  className="text-orange-700 border-orange-300 hover:bg-orange-100"
                >
                  Tüm Değişiklikleri Geri Al
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Düzenleme modu kapalı uyarısı */}
      {menuStatus === 'saved' && !isEditMode && (
        <div className="rounded-lg border border-blue-300 bg-blue-50 dark:bg-blue-900/20 dark:border-blue-800 p-4">
          <div className="flex items-center gap-3">
            <Lock className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            <p className="text-blue-800 dark:text-blue-300">
              Menü kilitli. Değişiklik yapmak için <strong>Düzenle</strong> butonuna tıklayın.
            </p>
          </div>
        </div>
      )}

      {/* Tabs for Normal and Vegan Menu - sadece menü varsa veya oluşturma modundaysa göster */}
      {(menuStatus === 'saved' || isCreatingNew) && (
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-2 mb-4">
          <TabsTrigger value="normal" className="text-base">
            🍖 Normal Menü
          </TabsTrigger>
          <TabsTrigger value="vegan" className="text-base">
            🥗 Vegan Menü
          </TabsTrigger>
        </TabsList>

        {/* Normal Menü Tab */}
        <TabsContent value="normal">
          <Card className="dark:bg-gray-900 dark:border-gray-800 overflow-hidden">
            <CardHeader className="bg-blue-900 text-white text-center py-4">
              <div className="flex items-center justify-center gap-4">
                <img src="/deu-logo.png" alt="DEU" className="h-12 w-12 hidden" />
                <div>
                  <h2 className="text-lg font-bold">DOKUZ EYLÜL ÜNİVERSİTESİ</h2>
                  <h3 className="text-sm">SAĞLIK KÜLTÜR VE SPOR DAİRE BAŞKANLIĞI</h3>
                  <h4 className="text-base font-semibold mt-1">
                    {selectedYear} {selectedMonthLabel.toUpperCase()} AYI - YEMEK LİSTESİ
                  </h4>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {weeklyMenus.map((weekMenu, weekIndex) => {
                const weekDates = getWeekDates(weekIndex);
                return (
                  <div key={weekIndex} className="border-b dark:border-gray-700 last:border-b-0">
                    <div className="overflow-x-auto">
                      <table className="w-full border-collapse">
                        <thead>
                          <tr className="bg-yellow-100 dark:bg-yellow-900/30">
                            {dayNames.map((day, idx) => (
                              <th
                                key={day.key}
                                className="border border-gray-300 dark:border-gray-600 p-2 text-center text-sm font-semibold text-blue-900 dark:text-blue-300 min-w-[180px]"
                              >
                                <div>{weekDates[idx]}</div>
                                <div className="text-xs font-normal text-gray-600 dark:text-gray-400">
                                  {day.label}
                                </div>
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {mealCategories.map((category, rowIdx) => (
                            <tr key={rowIdx}>
                              {dayNames.map(day => (
                                <td
                                  key={day.key}
                                  className="border border-gray-300 dark:border-gray-600 p-1"
                                >
                                  <MealAutocomplete
                                    value={weekMenu[day.key]?.items[rowIdx] || ''}
                                    onChange={(value) => updateMenuItem(weekIndex, day.key, rowIdx, value)}
                                    placeholder={category}
                                    categoryFilter={getCategoryByIndex(rowIdx)}
                                    disabled={!isEditMode || isPastMonth}
                                  />
                                </td>
                              ))}
                            </tr>
                          ))}
                          <tr className="bg-gray-50 dark:bg-gray-800">
                            {dayNames.map(day => (
                              <td
                                key={day.key}
                                className="border border-gray-300 dark:border-gray-600 p-2 text-center"
                              >
                                <div className="flex items-center justify-center gap-1">
                                  <span className="text-sm font-semibold text-blue-600 dark:text-blue-400">
                                    {weekMenu[day.key]?.calories || 0}
                                  </span>
                                  <span className="text-xs text-gray-500">kcal</span>
                                </div>
                              </td>
                            ))}
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                );
              })}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Vegan Menü Tab */}
        <TabsContent value="vegan">
          <Card className="dark:bg-gray-900 dark:border-gray-800 overflow-hidden">
            <CardHeader className="bg-green-800 text-white text-center py-4">
              <div className="flex items-center justify-center gap-4">
                <img src="/deu-logo.png" alt="DEU" className="h-12 w-12 hidden" />
                <div>
                  <h2 className="text-lg font-bold">DOKUZ EYLÜL ÜNİVERSİTESİ</h2>
                  <h3 className="text-sm">SAĞLIK KÜLTÜR VE SPOR DAİRE BAŞKANLIĞI</h3>
                  <h4 className="text-base font-semibold mt-1">
                    {selectedYear} {selectedMonthLabel.toUpperCase()} AYI - VEGAN MENÜ
                  </h4>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {veganWeeklyMenus.map((weekMenu, weekIndex) => {
                const weekDates = getWeekDates(weekIndex);
                return (
                  <div key={weekIndex} className="border-b dark:border-gray-700 last:border-b-0">
                    <div className="overflow-x-auto">
                      <table className="w-full border-collapse">
                        <thead>
                          <tr className="bg-green-100 dark:bg-green-900/30">
                            {dayNames.map((day, idx) => (
                              <th
                                key={day.key}
                                className="border border-gray-300 dark:border-gray-600 p-2 text-center text-sm font-semibold text-green-900 dark:text-green-300 min-w-[180px]"
                              >
                                <div>{weekDates[idx]}</div>
                                <div className="text-xs font-normal text-gray-600 dark:text-gray-400">
                                  {day.label}
                                </div>
                              </th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {mealCategories.map((category, rowIdx) => (
                            <tr key={rowIdx}>
                              {dayNames.map(day => (
                                <td
                                  key={day.key}
                                  className="border border-gray-300 dark:border-gray-600 p-1"
                                >
                                  <VeganMealAutocomplete
                                    value={weekMenu[day.key]?.items[rowIdx] || ''}
                                    onChange={(value) => updateVeganMenuItem(weekIndex, day.key, rowIdx, value)}
                                    placeholder={category}
                                    categoryFilter={getCategoryByIndex(rowIdx)}
                                    disabled={!isEditMode || isPastMonth}
                                  />
                                </td>
                              ))}
                            </tr>
                          ))}
                          <tr className="bg-gray-50 dark:bg-gray-800">
                            {dayNames.map(day => (
                              <td
                                key={day.key}
                                className="border border-gray-300 dark:border-gray-600 p-2 text-center"
                              >
                                <div className="flex items-center justify-center gap-1">
                                  <span className="text-sm font-semibold text-green-600 dark:text-green-400">
                                    {weekMenu[day.key]?.calories || 0}
                                  </span>
                                  <span className="text-xs text-gray-500">kcal</span>
                                </div>
                              </td>
                            ))}
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                );
              })}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
      )}
      {/* Notlar */}
      <Card className="dark:bg-gray-900 dark:border-gray-800">
        <CardContent className="pt-4">
          <p className="text-xs text-muted-foreground">
            *Mücbir sebepler haricinde kesinlikle menü değişimi yapılmayacaktır.
          </p>
          <p className="text-xs text-muted-foreground mt-1">
            *Yukarıda belirtilen 1 öğünlük toplam kalori değerlerine, 50 gr ekmeğin değeri olan 136 kalori ilave edilmiştir.
          </p>
        </CardContent>
      </Card>

      {/* Toast bildirimi */}
      <div className="fixed top-4 left-1/2 -translate-x-1/2 z-50">
        <Toast
          message={toast.message}
          type={toast.type}
          isVisible={toast.isVisible}
          onClose={hideToast}
        />
      </div>
    </div>
  );
}
