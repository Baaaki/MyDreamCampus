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
  it("requests /grades/my/grades and returns the payload", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: {
        student_id: "s-1",
        student_number: "2021510001",
        active_courses: [],
        completed_courses: [],
        cumulative_gpa: 3.2,
        total_credits: 40,
      },
    });

    const res = await gradesService.getMyGrades();

    expect(apiMock.get).toHaveBeenCalledWith("/grades/my/grades");
    expect(res.cumulative_gpa).toBe(3.2);
    expect(res.student_number).toBe("2021510001");
  });

  it("propagates API errors", async () => {
    apiMock.get.mockRejectedValueOnce(new Error("boom"));
    await expect(gradesService.getMyGrades()).rejects.toThrow("boom");
  });
});
