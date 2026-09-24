jest.mock("@/services/api", () => ({
  __esModule: true,
  default: { get: jest.fn(), post: jest.fn(), delete: jest.fn() },
}));

import { checkoutOutcome } from "./useMeals";
import type { Reservation } from "@/types/meal.types";

function reservation(id: string, status: string): Reservation {
  return {
    id,
    date: "2026-09-28",
    meal_time: "lunch",
    menu_type: "normal",
    cafeteria_name: "Merkez Yemekhane",
    status,
    is_used: false,
    created_at: "2026-09-24T12:00:00Z",
  };
}

describe("checkoutOutcome", () => {
  const ids = ["a", "b"];

  it("is confirmed once every reservation of the checkout is", () => {
    expect(
      checkoutOutcome(ids, [
        reservation("a", "confirmed"),
        reservation("b", "confirmed"),
        reservation("other", "pending"),
      ])
    ).toBe("confirmed");
  });

  it("stays pending while meal has not heard from payment", () => {
    expect(
      checkoutOutcome(ids, [reservation("a", "confirmed"), reservation("b", "pending")])
    ).toBe("pending");
  });

  it("is cancelled when a declined card dropped the reservations", () => {
    expect(
      checkoutOutcome(ids, [reservation("a", "cancelled"), reservation("b", "cancelled")])
    ).toBe("cancelled");
  });

  it("stays pending while a reservation is missing from the list", () => {
    expect(checkoutOutcome(ids, [reservation("a", "confirmed")])).toBe("pending");
  });
});
