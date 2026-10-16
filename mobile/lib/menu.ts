import { toISODate } from '@/constants/datetime';
import type {
  DailyMenu,
  MenuData,
  MenuDay,
  MenuType,
  MenuWeekday,
  WebMenuData,
} from '@/types/meal.types';

const MENU_WEEKDAYS: (MenuWeekday | null)[] = [
  null,
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  null,
];

/** Whether the month's menu is in the week format the web admin saves. */
export function isWebMenu(menu: MenuData): menu is WebMenuData {
  return (
    ('normalMenus' in menu && Array.isArray(menu.normalMenus)) ||
    ('veganMenus' in menu && Array.isArray(menu.veganMenus))
  );
}

// The same lookup as the web's menuForDay (frontend meal-service): week i is
// the i-th Monday-to-Sunday week touching the month.
function webMenuForDay(menu: WebMenuData, date: Date, menuType: MenuType): MenuDay | null {
  const weekday = MENU_WEEKDAYS[date.getDay()];
  if (!weekday) return null;
  const firstOfMonth = new Date(date.getFullYear(), date.getMonth(), 1);
  const sinceMonday = (firstOfMonth.getDay() + 6) % 7;
  const week = Math.floor((date.getDate() + sinceMonday - 1) / 7);
  const weeks = menuType === 'vegan' ? menu.veganMenus : menu.normalMenus;
  return weeks?.[week]?.[weekday] ?? null;
}

/**
 * The dishes served on `date` for the student's menu type. `menu` must be the
 * month `date` falls in: week-format menus of two months cannot be merged.
 */
export function menuForDate(
  menu: MenuData | null | undefined,
  date: Date,
  menuType: MenuType
): DailyMenu {
  if (!menu) return {};

  if (isWebMenu(menu)) {
    const items = webMenuForDay(menu, date, menuType)?.items ?? [];
    // The editor leaves an unused course as an empty string.
    const dishes = items.filter((item) => item.trim() !== '');
    if (dishes.length === 0) return {};
    // The week format has no lunch/dinner split; the web shows the day's
    // menu for both meals, and so does the app.
    return { lunch: dishes, dinner: dishes };
  }

  // The dated format only ever held the normal menu.
  if (menuType === 'vegan') return {};
  return menu[toISODate(date)] ?? {};
}
