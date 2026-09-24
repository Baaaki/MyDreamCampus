jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

import api from "./api";
import paymentService from "./paymentService";

const apiMock = api as unknown as { get: jest.Mock; post: jest.Mock };

const payment = {
  id: "p-1",
  status: "completed",
  amount: 30,
  currency: "TRY",
  card_brand: "Visa",
  card_last4: "4242",
  failure_reason: null,
  expires_at: "2026-09-24T12:15:00Z",
  completed_at: "2026-09-24T12:01:00Z",
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe("paymentService", () => {
  it("posts the card to /payments/:id/confirm and returns the bare payment", async () => {
    apiMock.post.mockResolvedValueOnce({ data: payment });
    const card = {
      card_number: "4242424242424242",
      exp_month: 12,
      exp_year: 2030,
      cvc: "123",
      cardholder_name: "Zeynep Şahin",
    };

    const res = await paymentService.confirm("p-1", card);

    expect(apiMock.post).toHaveBeenCalledWith("/payments/p-1/confirm", card);
    expect(res.status).toBe("completed");
    expect(res.card_last4).toBe("4242");
  });

  it("reads a payment from /payments/:id", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { ...payment, status: "pending" } });

    const res = await paymentService.get("p-1");

    expect(apiMock.get).toHaveBeenCalledWith("/payments/p-1");
    expect(res.status).toBe("pending");
  });

  it("propagates a rejected card form (422) to the caller", async () => {
    const error = { response: { status: 422, data: { error: "Kart numarası geçersiz" } } };
    apiMock.post.mockRejectedValueOnce(error);

    await expect(
      paymentService.confirm("p-1", {
        card_number: "4242424242424241",
        exp_month: 12,
        exp_year: 2030,
        cvc: "123",
        cardholder_name: "Ali",
      }),
    ).rejects.toBe(error);
  });
});
