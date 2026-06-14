jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

import api from "./api";
import attendanceService from "./attendanceService";

const apiMock = api as unknown as { get: jest.Mock; post: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("attendanceService.scanQR", () => {
  it("posts payload to /attendance/scan and returns response", async () => {
    apiMock.post.mockResolvedValueOnce({
      data: { success: true, session_id: "s-1", marked_at: "2026-04-27T10:00:00Z" },
    });

    const res = await attendanceService.scanQR({ qr_token: "qr-abc", lat: 40.0, lng: 29.0 } as never);

    expect(apiMock.post).toHaveBeenCalledWith("/attendance/scan", {
      qr_token: "qr-abc",
      lat: 40.0,
      lng: 29.0,
    });
    expect(res.success).toBe(true);
    expect(res.session_id).toBe("s-1");
  });

  it("propagates API errors", async () => {
    apiMock.post.mockRejectedValueOnce(new Error("invalid qr"));
    await expect(attendanceService.scanQR({ qr_token: "x" } as never)).rejects.toThrow("invalid qr");
  });
});

describe("attendanceService.getMyAttendance", () => {
  it("passes semester param when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [], summary: {} } });

    await attendanceService.getMyAttendance("2026-spring");

    expect(apiMock.get).toHaveBeenCalledWith("/attendance/my", {
      params: { semester: "2026-spring" },
    });
  });

  it("sends empty params when semester omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [] } });

    await attendanceService.getMyAttendance();

    expect(apiMock.get).toHaveBeenCalledWith("/attendance/my", { params: {} });
  });

  it("returns response data", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { items: [{ session_id: "s-1", status: "present" }] },
    });
    const res = await attendanceService.getMyAttendance();
    expect(res.items).toHaveLength(1);
  });
});
