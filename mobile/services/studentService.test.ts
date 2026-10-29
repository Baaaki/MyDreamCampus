jest.mock("./api", () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
  },
}));

import api from "./api";
import studentService from "./studentService";

const apiMock = api as unknown as { get: jest.Mock };

beforeEach(() => {
  jest.clearAllMocks();
});

describe("studentService.getById", () => {
  it("requests the student's own record and returns it", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: {
        id: "s-1",
        student_number: "2021510001",
        first_name: "Zeynep",
        last_name: "Şahin",
        department: "Bilgisayar Mühendisliği",
        class_level: 4,
      },
    });

    const res = await studentService.getById("s-1");

    expect(apiMock.get).toHaveBeenCalledWith("/students/s-1");
    expect(res.first_name).toBe("Zeynep");
    expect(res.class_level).toBe(4);
  });

  it("propagates API errors", async () => {
    apiMock.get.mockRejectedValueOnce(new Error("boom"));
    await expect(studentService.getById("s-1")).rejects.toThrow("boom");
  });
});
