jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
    delete: jest.fn(),
  },
}));

import api from "./api";
import enrollmentService from "./enrollmentService";

const apiMock = api as unknown as {
  get: jest.Mock;
  post: jest.Mock;
  delete: jest.Mock;
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe("enrollmentService.getAvailableCourses", () => {
  it("passes semester as query param", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { courses: [] } });
    await enrollmentService.getAvailableCourses("2026-spring");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/available-courses", {
      params: { semester: "2026-spring" },
    });
  });
});

describe("enrollmentService.createEnrollment", () => {
  it("posts the enrollment payload and returns the program", async () => {
    const payload = {
      semester: "2026-spring",
      course_ids: ["c1", "c2"],
    } as never;
    apiMock.post.mockResolvedValueOnce({
      data: { id: "p-1", status: "pending" },
    });

    const res = await enrollmentService.createEnrollment(payload);

    expect(apiMock.post).toHaveBeenCalledWith("/enrollment/programs", payload);
    expect(res.id).toBe("p-1");
  });
});

describe("enrollmentService.getMyEnrollments", () => {
  it("includes both semester and status when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { enrollments: [] } });
    await enrollmentService.getMyEnrollments("2026-spring", "approved");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", {
      params: { semester: "2026-spring", status: "approved" },
    });
  });

  it("sends only status when semester omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { enrollments: [] } });
    await enrollmentService.getMyEnrollments(undefined, "pending");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", {
      params: { status: "pending" },
    });
  });

  it("sends empty params when nothing provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { enrollments: [] } });
    await enrollmentService.getMyEnrollments();
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-enrollments", { params: {} });
  });
});

describe("enrollmentService.cancelEnrollment", () => {
  it("DELETEs with semester query param", async () => {
    apiMock.delete.mockResolvedValueOnce({ data: undefined });
    await enrollmentService.cancelEnrollment("2026-spring");
    expect(apiMock.delete).toHaveBeenCalledWith("/enrollment/programs", {
      params: { semester: "2026-spring" },
    });
  });
});

describe("enrollmentService.getLatestRejection", () => {
  it("requests latest rejection for semester", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { reason: "schedule_conflict" } });
    const res = await enrollmentService.getLatestRejection("2026-spring");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/latest-rejection", {
      params: { semester: "2026-spring" },
    });
    expect(res.reason).toBe("schedule_conflict");
  });
});

describe("enrollmentService.getMyRejections", () => {
  it("forwards semester filter when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { rejections: [] } });
    await enrollmentService.getMyRejections("2026-spring");
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-rejections", {
      params: { semester: "2026-spring" },
    });
  });

  it("sends empty params when omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { rejections: [] } });
    await enrollmentService.getMyRejections();
    expect(apiMock.get).toHaveBeenCalledWith("/enrollment/my-rejections", { params: {} });
  });
});
