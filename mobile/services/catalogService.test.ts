jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
  },
}));

import api from "./api";
import catalogService from "./catalogService";

const apiMock = api as unknown as { get: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("catalogService.getActiveSemester", () => {
  it("requests the active semester and returns it", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { id: "sem-1", name: "2026-spring", status: "active" },
    });

    const res = await catalogService.getActiveSemester();

    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/active");
    expect(res.name).toBe("2026-spring");
  });

  it("propagates API errors", async () => {
    apiMock.get.mockRejectedValueOnce(new Error("network"));
    await expect(catalogService.getActiveSemester()).rejects.toThrow("network");
  });
});
