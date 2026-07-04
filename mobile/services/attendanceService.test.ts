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
      data: {
        message: "Yoklama alindi",
        course_code: "CS101",
        course_name: "Intro to CS",
        week_number: 5,
        session_type: "theory",
        marked_at: "2026-04-27T10:00:00Z",
      },
    });

    const res = await attendanceService.scanQR({ qr_payload: { sid: "s-1", sig: "sig-abc" } });

    expect(apiMock.post).toHaveBeenCalledWith("/attendance/scan", {
      qr_payload: { sid: "s-1", sig: "sig-abc" },
    });
    expect(res.course_code).toBe("CS101");
    expect(res.week_number).toBe(5);
  });

  it("propagates API errors", async () => {
    apiMock.post.mockRejectedValueOnce(new Error("invalid qr"));
    await expect(
      attendanceService.scanQR({ qr_payload: { sid: "s-1", sig: "bad" } })
    ).rejects.toThrow("invalid qr");
  });
});

describe("attendanceService.getMyAttendance", () => {
  it("passes semester param when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { courses: [] } });

    await attendanceService.getMyAttendance("2026-spring");

    expect(apiMock.get).toHaveBeenCalledWith("/attendance/my", {
      params: { semester: "2026-spring" },
    });
  });

  it("sends empty params when semester omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { courses: [] } });

    await attendanceService.getMyAttendance();

    expect(apiMock.get).toHaveBeenCalledWith("/attendance/my", { params: {} });
  });

  it("returns response data", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { courses: [{ course_id: "c-1", course_code: "CS101" }] },
    });
    const res = await attendanceService.getMyAttendance();
    expect(res.courses).toHaveLength(1);
  });
});
