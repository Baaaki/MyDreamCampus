export interface DayMenu {
  items: [string, string, string, string, string]
  calories: number
}

export interface WeeklyMenu {
  [key: string]: DayMenu
}

// Sample weekly menu for demonstration / test data
export const sampleWeeklyMenu: WeeklyMenu = {
  monday: {
    items: [
      "YEŞİL MERCİMEK ÇORBASI",
      "KURU FASULYE",
      "PİRİNÇ PİLAVI",
      "SÜTLAÇ",
      "TURŞU",
    ],
    calories: 1040,
  },
  tuesday: {
    items: [
      "DOMATES ÇORBASI",
      "PATATES OTURTMA",
      "MAKARNA",
      "MEYVE",
      "HAYDARİ",
    ],
    calories: 840,
  },
  wednesday: {
    items: [
      "TUTMAÇ ÇORBASI",
      "ANKARA TAVASI",
      "PİRİNÇLİ ISPANAK",
      "REVANI",
      "YOĞURT",
    ],
    calories: 1080,
  },
  thursday: {
    items: ["", "", "", "", ""],
    calories: 0,
  },
  friday: {
    items: [
      "EZOGELİN ÇORBASI",
      "MANTARLI ET SOTE",
      "BULGUR PİLAVI",
      "KOMPOSTO",
      "ÇOBAN SALATA",
    ],
    calories: 870,
  },
}

// Vegan sample weekly menu for demonstration / test data
export const sampleVeganWeeklyMenu: WeeklyMenu = {
  monday: {
    items: [
      "MERCİMEK ÇORBASI",
      "NOHUT YEMEĞİ",
      "BULGUR PİLAVI",
      "MEYVE",
      "HUMUS",
    ],
    calories: 870,
  },
  tuesday: {
    items: [
      "SEBZE ÇORBASI",
      "ZEYTİNYAĞLI FASULYE",
      "PİRİNÇ PİLAVI",
      "KOMPOSTO",
      "ÇOBAN SALATA",
    ],
    calories: 785,
  },
  wednesday: {
    items: [
      "DOMATES ÇORBASI",
      "İMAM BAYILDI",
      "SOSLU MAKARNA",
      "KABAK TATLISI",
      "TURŞU",
    ],
    calories: 795,
  },
  thursday: {
    items: ["", "", "", "", ""],
    calories: 0,
  },
  friday: {
    items: [
      "EZOGELİN ÇORBASI",
      "SEBZE GÜVEÇ",
      "ZEYTİNYAĞLI YEŞİL FASULYE",
      "MEVSİM MEYVE",
      "FAVA",
    ],
    calories: 765,
  },
}
