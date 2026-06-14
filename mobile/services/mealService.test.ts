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

const apiMock = api as unknown as {
  get: jest.Mock;
  post: jest.Mock;
  delete: jest.Mock;
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe("mealService.getCafeterias", () => {
  it("returns cafeteria list", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { items: [{ id: "caf-1", name: "Main" }] },
    });

    const res = await mealService.getCafeterias();

    expect(apiMock.get).toHaveBeenCalledWith("/meals/cafeterias");
    expect(res.items).toHaveLength(1);
  });
});

describe("mealService.getMonthlyMenu", () => {
  it("passes year and month as query params", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { days: [] } });

    await mealService.getMonthlyMenu(2026, 4);

    expect(apiMock.get).toHaveBeenCalledWith("/meals/menu/monthly", {
      params: { year: 2026, month: 4 },
    });
  });
});

describe("mealService.getMyReservations", () => {
  it("forwards optional params", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [] } });

    await mealService.getMyReservations({ from: "2026-04-01", to: "2026-04-30" } as never);

    expect(apiMock.get).toHaveBeenCalledWith("/meals/reservations/my", {
      params: { from: "2026-04-01", to: "2026-04-30" },
    });
  });

  it("works without params", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [] } });
    await mealService.getMyReservations();
    expect(apiMock.get).toHaveBeenCalledWith("/meals/reservations/my", { params: undefined });
  });
});

describe("mealService.createBatchReservation", () => {
  it("posts batch payload to /reservations/batch", async () => {
    const payload = { dates: ["2026-04-28", "2026-04-29"], meal_time: "lunch" } as never;
    apiMock.post.mockResolvedValueOnce({
      data: { created: 2, conflicts: [] },
    });

    const res = await mealService.createBatchReservation(payload);

    expect(apiMock.post).toHaveBeenCalledWith("/meals/reservations/batch", payload);
    expect(res.created).toBe(2);
  });
});

describe("mealService.cancelReservation", () => {
  it("DELETEs reservation by id", async () => {
    apiMock.delete.mockResolvedValueOnce({ data: { refunded: true } });

    const res = await mealService.cancelReservation("res-1");

    expect(apiMock.delete).toHaveBeenCalledWith("/meals/reservations/res-1");
    expect(res.refunded).toBe(true);
  });
});

describe("mealService.useReservation", () => {
  it("posts QR scan payload to /reservations/use", async () => {
    const payload = { qr_token: "qr-1", cafeteria_id: "caf-1" } as never;
    apiMock.post.mockResolvedValueOnce({
      data: { success: true, used_at: "2026-04-27T12:00:00Z" },
    });

    const res = await mealService.useReservation(payload);

    expect(apiMock.post).toHaveBeenCalledWith("/meals/reservations/use", payload);
    expect(res.success).toBe(true);
  });

  it("propagates errors (e.g. expired QR)", async () => {
    apiMock.post.mockRejectedValueOnce(new Error("expired"));
    await expect(mealService.useReservation({ qr_token: "x" } as never)).rejects.toThrow("expired");
  });
});
