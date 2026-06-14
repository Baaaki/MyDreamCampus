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

describe("catalogService.getSemesterCourses", () => {
  it("hits the correct semester URL with params", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [], total: 0 } });

    await catalogService.getSemesterCourses("sem-1", {
      page: 2,
      limit: 50,
      department: "CS",
    });

    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/sem-1/courses", {
      params: { page: 2, limit: 50, department: "CS" },
    });
  });

  it("works without optional params", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { items: [], total: 0 } });
    await catalogService.getSemesterCourses("sem-2");
    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/sem-2/courses", {
      params: undefined,
    });
  });
});

describe("catalogService.getSemesterCourse", () => {
  it("builds nested URL with both ids", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { id: "c-1", code: "CS101" } });

    const res = await catalogService.getSemesterCourse("sem-1", "c-1");

    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/sem-1/courses/c-1");
    expect(res.id).toBe("c-1");
  });
});

describe("catalogService.getTeacherCourses", () => {
  it("includes semester param when provided", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { courses: [] } });
    await catalogService.getTeacherCourses("2026-spring");
    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/teacher/courses", {
      params: { semester: "2026-spring" },
    });
  });

  it("sends empty params when semester omitted", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { courses: [] } });
    await catalogService.getTeacherCourses();
    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/teacher/courses", {
      params: {},
    });
  });
});

describe("catalogService.getActiveSemester", () => {
  it("hits the active endpoint", async () => {
    apiMock.get.mockResolvedValueOnce({ data: { id: "sem-active", code: "2026S" } });
    const res = await catalogService.getActiveSemester();
    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters/active");
    expect(res.code).toBe("2026S");
  });
});

describe("catalogService.getSemesters", () => {
  it("returns the semester array", async () => {
    apiMock.get.mockResolvedValueOnce({
      data: [{ id: "s1" }, { id: "s2" }],
    });
    const list = await catalogService.getSemesters();
    expect(apiMock.get).toHaveBeenCalledWith("/catalog/semesters");
    expect(list).toHaveLength(2);
  });
});
