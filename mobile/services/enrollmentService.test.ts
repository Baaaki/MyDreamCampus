jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
  },
}));

import api from "./api";
import enrollmentService from "./enrollmentService";

const apiMock = api as unknown as { get: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("enrollmentService.getMyEnrollments", () => {
  it("includes both semester and status when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { programs: [] } });
    await enrollmentService.getMyEnrollments("2026-spring", "approved");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", {
      params: { semester: "2026-spring", status: "approved" },
    });
  });

  it("sends only status when semester omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { programs: [] } });
    await enrollmentService.getMyEnrollments(undefined, "pending");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", {
      params: { status: "pending" },
    });
  });

  it("sends empty params when nothing provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { programs: [] } });
    await enrollmentService.getMyEnrollments();
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", { params: {} });
  });
});
