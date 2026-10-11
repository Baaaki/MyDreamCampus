import { afterEach, describe, expect, it, vi } from "vitest"

const sentBodies: string[] = []

function stubFetch(body: unknown, status = 200) {
  sentBodies.length = 0
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const request = input as Request
    // ky consumes the request after fetch resolves; read the body now.
    sentBodies.push(await request.clone().text())
    return new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    })
  })
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

async function loadService() {
  vi.resetModules()
  return await import("./grades-service")
}

const validForm = {
  slug: "midterm",
  newScore: "72,5",
  reason: "Sınav kağıdında toplama hatası",
}

describe("grade appeals", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("accepts a complete form, reading a comma as the decimal point", async () => {
    const { validateAppealForm, toAppealRequest } = await loadService()

    expect(validateAppealForm(validForm)).toEqual({})
    expect(toAppealRequest(validForm, "st-1", "c-1")).toEqual({
      student_id: "st-1",
      course_id: "c-1",
      slug: "midterm",
      new_score: 72.5,
      reason: "Sınav kağıdında toplama hatası",
    })
  })

  it("rejects what the backend would reject", async () => {
    const { validateAppealForm } = await loadService()

    expect(
      validateAppealForm({ slug: "", newScore: "", reason: "kısa" })
    ).toEqual({
      slug: "Değerlendirme seçin",
      newScore: "Yeni notu girin",
      reason: "Gerekçe en az 10 karakter olmalı",
    })
    expect(validateAppealForm({ ...validForm, newScore: "101" }).newScore).toBe(
      "Not 0 ile 100 arasında olmalı"
    )
    expect(validateAppealForm({ ...validForm, newScore: "abc" }).newScore).toBe(
      "Yeni notu girin"
    )
    // Zero is a real score.
    expect(validateAppealForm({ ...validForm, newScore: "0" })).toEqual({})
    // Padding does not count toward the reason's length.
    expect(
      validateAppealForm({ ...validForm, reason: "   kısa   gerekçe" }).reason
    ).toBeUndefined()
    expect(
      validateAppealForm({ ...validForm, reason: "  123456789  " }).reason
    ).toBe("Gerekçe en az 10 karakter olmalı")
  })

  it("posts the appeal to /api/grades/admin/appeals", async () => {
    const fetchSpy = stubFetch({ new_grade_point: "BB" })
    const { gradesService, toAppealRequest } = await loadService()

    const result = await gradesService.processAppeal(
      toAppealRequest(validForm, "st-1", "c-1")
    )

    const req = fetchSpy.mock.calls[0]![0] as Request
    expect(req.method).toBe("POST")
    expect(new URL(req.url).pathname).toBe("/api/grades/admin/appeals")
    expect(JSON.parse(sentBodies[0]!)).toMatchObject({
      slug: "midterm",
      new_score: 72.5,
    })
    expect(result.new_grade_point).toBe("BB")
  })
})
