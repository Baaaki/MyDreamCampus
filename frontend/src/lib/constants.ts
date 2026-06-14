// Time slots mapping for course catalog
export const TIME_SLOTS = {
  1: { label: "Ders 1", time: "08:30-09:15" },
  2: { label: "Ders 2", time: "09:25-10:10" },
  3: { label: "Ders 3", time: "10:20-11:05" },
  4: { label: "Ders 4", time: "11:15-12:00" },
  5: { label: "Öğle Arası", time: "12:10-12:55" },
  6: { label: "Ders 5", time: "13:00-13:45" },
  7: { label: "Ders 6", time: "13:55-14:40" },
  8: { label: "Ders 7", time: "14:50-15:35" },
  9: { label: "Ders 8", time: "15:45-16:30" }
} as const;

// Days of week mapping
export const DAYS_OF_WEEK = {
  1: "Pazartesi",
  2: "Salı",
  3: "Çarşamba",
  4: "Perşembe",
  5: "Cuma",
  6: "Cumartesi",
  7: "Pazar"
} as const;

// User roles
export const USER_ROLES = {
  ADMIN: 'admin',
  TEACHER: 'teacher',
  STUDENT: 'student'
} as const;

// Attendance absence limit
export const ATTENDANCE_ABSENCE_LIMIT = 3;

// Meal service reservation window
export const MEAL_RESERVATION_WINDOW = {
  START_DAY: 1, // Monday
  START_HOUR: 8,
  END_DAY: 5, // Friday
  END_HOUR: 13
} as const;

// Meal time slots
export const MEAL_TIMES = {
  LUNCH: 'lunch',
  DINNER: 'dinner'
} as const;

// Menu types
export const MENU_TYPES = {
  NORMAL: 'normal',
  VEGAN: 'vegan'
} as const;

// Letter grades
export const LETTER_GRADES = ['AA', 'BA', 'BB', 'CB', 'CC', 'DC', 'DD', 'FD', 'FF'] as const;
