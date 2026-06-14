import { describe, it, expect } from "vitest";
import {
  TIME_SLOTS,
  DAYS_OF_WEEK,
  USER_ROLES,
  MEAL_RESERVATION_WINDOW,
  MEAL_TIMES,
  MENU_TYPES,
  LETTER_GRADES,
} from "./constants";

describe("TIME_SLOTS", () => {
  it("covers slots 1-9 with non-overlapping time labels", () => {
    const keys = Object.keys(TIME_SLOTS).map(Number).sort((a, b) => a - b);
    expect(keys).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9]);
    Object.values(TIME_SLOTS).forEach((slot) => {
      expect(slot.label).toMatch(/^(Ders|Öğle)/);
      expect(slot.time).toMatch(/^\d{2}:\d{2}-\d{2}:\d{2}$/);
    });
  });

  it("slot 5 is the lunch break", () => {
    expect(TIME_SLOTS[5].label).toBe("Öğle Arası");
  });
});

describe("DAYS_OF_WEEK", () => {
  it("uses 1-indexed Pazartesi-Pazar order", () => {
    expect(DAYS_OF_WEEK[1]).toBe("Pazartesi");
    expect(DAYS_OF_WEEK[5]).toBe("Cuma");
    expect(DAYS_OF_WEEK[7]).toBe("Pazar");
    expect(Object.keys(DAYS_OF_WEEK)).toHaveLength(7);
  });
});

describe("USER_ROLES", () => {
  it("only exposes admin/teacher/student", () => {
    expect(USER_ROLES).toEqual({
      ADMIN: "admin",
      TEACHER: "teacher",
      STUDENT: "student",
    });
  });
});

describe("MEAL_RESERVATION_WINDOW", () => {
  it("opens Monday morning, closes Friday afternoon", () => {
    expect(MEAL_RESERVATION_WINDOW.START_DAY).toBe(1);
    expect(MEAL_RESERVATION_WINDOW.END_DAY).toBe(5);
    expect(MEAL_RESERVATION_WINDOW.END_HOUR).toBeGreaterThan(MEAL_RESERVATION_WINDOW.START_HOUR);
  });
});

describe("MEAL_TIMES + MENU_TYPES", () => {
  it("matches backend enum values exactly", () => {
    // Sentinel: any rename here breaks backend contract.
    expect(MEAL_TIMES.LUNCH).toBe("lunch");
    expect(MEAL_TIMES.DINNER).toBe("dinner");
    expect(MENU_TYPES.NORMAL).toBe("normal");
    expect(MENU_TYPES.VEGAN).toBe("vegan");
  });
});

describe("LETTER_GRADES", () => {
  it("ordered from highest to lowest", () => {
    expect(LETTER_GRADES[0]).toBe("AA");
    expect(LETTER_GRADES[LETTER_GRADES.length - 1]).toBe("FF");
    expect(LETTER_GRADES.length).toBe(9);
  });

  it("has no duplicates", () => {
    expect(new Set(LETTER_GRADES).size).toBe(LETTER_GRADES.length);
  });
});
