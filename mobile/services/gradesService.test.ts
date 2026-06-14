jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
  },
}));

import api from "./api";
import gradesService from "./gradesService";

const apiMock = api as unknown as { get: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("gradesService.getMyGrades", () => {
  it("hits /grades/student/my and returns data", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { items: [{ course_code: "CS101", grade: "AA" }], gpa: 3.5 },
    });

    const res = await gradesService.getMyGrades();

    expect(apiMock.get).toHaveBeenCalledWith("/grades/student/my");
    expect(res.gpa).toBe(3.5);
    expect(res.items).toHaveLength(1);
  });

  it("propagates API errors", async () => {
    apiMock.get.mockRejectedValueOnce(new Error("forbidden"));
    await expect(gradesService.getMyGrades()).rejects.toThrow("forbidden");
  });
});

describe("gradesService.getTranscript", () => {
  it("interpolates studentId into URL", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { semesters: [] } });

    await gradesService.getTranscript("stu-42");

    expect(apiMock.get).toHaveBeenCalledWith("/grades/transcript/stu-42");
  });

  it("returns response payload", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: { semesters: [{ code: "2025F", gpa: 3.2 }] },
    });
    const res = await gradesService.getTranscript("stu-1");
    expect(res.semesters).toHaveLength(1);
  });
});
