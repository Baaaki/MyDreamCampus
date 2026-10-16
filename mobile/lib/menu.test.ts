import { isWebMenu, menuForDate } from './menu';
import type { MenuWeek, WebMenuData } from '@/types/meal.types';

function day(...items: string[]) {
  return { items, calories: 800 };
}

// September 2026 starts on a Tuesday: week 0 runs Mon 31 Aug - Sun 6 Sep.
const september: WebMenuData = {
  normalMenus: [
    { tuesday: day('Mercimek', 'Kofte', 'Pilav', 'Sutlac', 'Ayran') },
    { monday: day('Ezogelin', 'Tavuk', 'Bulgur', 'Revani', 'Salata') },
  ],
  veganMenus: [
    { tuesday: day('Mercimek', 'Nohut', 'Pilav', 'Meyve', 'Salata') },
    { monday: day('Domates', 'Taze Fasulye', 'Bulgur', 'Ayva', 'Tursu') },
  ],
};

describe('menuForDate', () => {
  it('reads the normal menu of the week format for lunch and dinner alike', () => {
    const menu = menuForDate(september, new Date(2026, 8, 1), 'normal');

    expect(menu.lunch).toEqual(['Mercimek', 'Kofte', 'Pilav', 'Sutlac', 'Ayran']);
    expect(menu.dinner).toEqual(menu.lunch);
  });

  it('reads the vegan menu when the student picks vegan', () => {
    const menu = menuForDate(september, new Date(2026, 8, 7), 'vegan');

    expect(menu.lunch).toEqual(['Domates', 'Taze Fasulye', 'Bulgur', 'Ayva', 'Tursu']);
  });

  it('counts weeks from the Monday-to-Sunday week that holds the 1st', () => {
    // February 2026 starts on a Sunday, so that Sunday alone is week 0 and
    // Monday the 2nd is already week 1.
    const february: WebMenuData = {
      normalMenus: [{ monday: day('Yanlis hafta') }, { monday: day('Dogru hafta') }],
      veganMenus: [],
    };

    expect(menuForDate(february, new Date(2026, 1, 2), 'normal').lunch).toEqual(['Dogru hafta']);
  });

  it('drops the courses the editor left empty', () => {
    const menu: WebMenuData = {
      normalMenus: [{ tuesday: day('Mercimek', '', '  ', 'Sutlac', '') }],
      veganMenus: [],
    };

    expect(menuForDate(menu, new Date(2026, 8, 1), 'normal').lunch).toEqual(['Mercimek', 'Sutlac']);
  });

  it('has nothing for a day whose courses are all empty, or a weekend', () => {
    const blank: WebMenuData = { normalMenus: [{ tuesday: day('', '') }], veganMenus: [] };

    expect(menuForDate(blank, new Date(2026, 8, 1), 'normal')).toEqual({});
    expect(menuForDate(september, new Date(2026, 8, 5), 'normal')).toEqual({});
  });

  it('falls back to the dated format, which only holds the normal menu', () => {
    const dated = { '2026-09-01': { lunch: ['Mercimek'], dinner: ['Kuru Fasulye'] } };

    expect(menuForDate(dated, new Date(2026, 8, 1), 'normal')).toEqual({
      lunch: ['Mercimek'],
      dinner: ['Kuru Fasulye'],
    });
    expect(menuForDate(dated, new Date(2026, 8, 1), 'vegan')).toEqual({});
    expect(menuForDate(dated, new Date(2026, 8, 2), 'normal')).toEqual({});
  });

  it('prefers the week format when a month carries both, as the seed writes it', () => {
    const both = {
      ...september,
      '2026-09-01': { lunch: ['Eski kayit'] },
    };

    expect(menuForDate(both, new Date(2026, 8, 1), 'normal').lunch?.[0]).toBe('Mercimek');
  });

  it('has nothing for a month without a menu', () => {
    expect(menuForDate(null, new Date(2026, 8, 1), 'normal')).toEqual({});
    expect(menuForDate(undefined, new Date(2026, 8, 1), 'vegan')).toEqual({});
  });
});

describe('isWebMenu', () => {
  it('tells the week format from the dated one', () => {
    const emptyWeeks: MenuWeek[] = [];
    expect(isWebMenu({ normalMenus: emptyWeeks, veganMenus: emptyWeeks })).toBe(true);
    expect(isWebMenu({ '2026-09-01': { lunch: ['Mercimek'] } })).toBe(false);
  });
});
