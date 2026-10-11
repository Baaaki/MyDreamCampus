jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
    delete: jest.fn(),
  },
}));

import api from "./api";
import mealService from "./mealService";

const apiMock = api as unknown as { get: jest.Mock; post: jest.Mock; delete: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("mealService envelope unwrapping", () => {
  it("unwraps {success, data} for cafeterias", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: {
        success: true,
        data: { cafeterias: [{ id: "c-1", name: "Merkez" }], meal_price: 15, currency: "TRY" },
      },
    });

    const res = await mealService.getCafeterias();

    expect(apiMock.get).toHaveBeenCalledWith("/meals/cafeterias");
    expect(res.cafeterias).toHaveLength(1);
    expect(res.cafeterias[0].id).toBe("c-1");
    expect(res.meal_price).toBe(15);
  });

  it("passes year/month params to monthly menu and unwraps data", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { success: true, data: { year: 2026, month: 7, menu_data: {} } },
    });

    const res = await mealService.getMonthlyMenu(2026, 7);

    expect(apiMock.get).toHaveBeenCalledWith("/meals/menu/monthly", {
      params: { year: 2026, month: 7 },
    });
    expect(res.year).toBe(2026);
  });

  it("posts a reservation and returns the unwrapped result", async () => {
    apiMock.post.mockResolvedValueOnce({
      data: {
        success: true,
        data: { reservation_id: "r-1", payment_id: "p-1", amount: 0, currency: "TRY" },
      },
    });

    const res = await mealService.createReservation({
      cafeteria_id: "c-1",
      date: "2026-07-20",
      meal_time: "lunch",
      menu_type: "normal",
    });

    expect(apiMock.post).toHaveBeenCalledWith("/meals/reservations", {
      cafeteria_id: "c-1",
      date: "2026-07-20",
      meal_time: "lunch",
      menu_type: "normal",
    });
    expect(res.reservation_id).toBe("r-1");
  });

  it("cancels a reservation by id", async () => {
    apiMock.delete.mockResolvedValueOnce({ data: { success: true } });
    await mealService.cancelReservation("r-1");
    expect(apiMock.delete).toHaveBeenCalledWith("/meals/reservations/r-1");
  });
});

describe("mealService.useReservation", () => {
  it("posts the raw QR text and unwraps the envelope", async () => {
    apiMock.post.mockResolvedValueOnce({
      data: {
        success: true,
        data: {
          message: "ok",
          reservation_id: "r-1",
          cafeteria_name: "Merkez",
          meal_time: "lunch",
          menu_type: "vegan",
        },
      },
    });

    const res = await mealService.useReservation({ qr_payload: "c:2026-10-12:lunch:1:sig" });

    expect(apiMock.post).toHaveBeenCalledWith("/meals/reservations/use", {
      qr_payload: "c:2026-10-12:lunch:1:sig",
    });
    expect(res.cafeteria_name).toBe("Merkez");
    expect(res.menu_type).toBe("vegan");
  });

  it("propagates a backend rejection so the screen can show its message", async () => {
    const rejection = { response: { data: { error: "Rezervasyon zaten kullanildi" } } };
    apiMock.post.mockRejectedValueOnce(rejection);

    await expect(mealService.useReservation({ qr_payload: "x" })).rejects.toBe(rejection);
  });
});

