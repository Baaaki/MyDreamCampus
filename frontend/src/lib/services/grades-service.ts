import { gradesApi } from "@/lib/api-client"
import type {
  CourseStatusResponse,
  CourseStudentsResponse,
  SubmitScoreRequest,
  SubmitScoreResponse,
  BulkSubmitScoresRequest,
  BulkSubmitScoresResponse,
  GradeAppealRequest,
  GradeAppealResponse,
  MyGradesResponse,
} from "@/lib/types"

/** The backend's minimum, so a short reason is caught before the request. */
export const APPEAL_REASON_MIN_LENGTH = 10

export interface AppealForm {
  slug: string
  newScore: string
  reason: string
}

export type AppealFormErrors = Partial<Record<keyof AppealForm, string>>

export function validateAppealForm(form: AppealForm): AppealFormErrors {
  const errors: AppealFormErrors = {}
  if (!form.slug) errors.slug = "Değerlendirme seçin"
  const score = Number(form.newScore.replace(",", "."))
  if (form.newScore.trim() === "" || Number.isNaN(score)) {
    errors.newScore = "Yeni notu girin"
  } else if (score < 0 || score > 100) {
    errors.newScore = "Not 0 ile 100 arasında olmalı"
  }
  if (form.reason.trim().length < APPEAL_REASON_MIN_LENGTH) {
    errors.reason = `Gerekçe en az ${APPEAL_REASON_MIN_LENGTH} karakter olmalı`
  }
  return errors
}

export function toAppealRequest(
  form: AppealForm,
  studentId: string,
  courseId: string
): GradeAppealRequest {
  return {
    student_id: studentId,
    course_id: courseId,
    slug: form.slug,
    new_score: Number(form.newScore.replace(",", ".")),
    reason: form.reason.trim(),
  }
}

export const gradesService = {
  /**
   * The logged-in student's courses and GPA. The backend sends null for an
   * empty course list; both lists come back as arrays.
   */
  async getMyGrades(): Promise<MyGradesResponse> {
    const res = await gradesApi.get("my/grades").json<MyGradesResponse>()
    return {
      ...res,
      active_courses: res.active_courses ?? [],
      completed_courses: res.completed_courses ?? [],
    }
  },

  async getCourseStatus(courseId: string): Promise<CourseStatusResponse> {
    try {
      const response = await gradesApi
        .get(`courses/${courseId}/status`)
        .json<CourseStatusResponse>()
      return response
    } catch (error) {
      console.error("Failed to fetch course status:", error)
      throw error
    }
  },

  async getCourseStudents(courseId: string): Promise<CourseStudentsResponse> {
    try {
      const response = await gradesApi
        .get(`courses/${courseId}/students`)
        .json<CourseStudentsResponse>()
      return response
    } catch (error) {
      console.error("Failed to fetch course students:", error)
      throw error
    }
  },

  async submitScore(
    courseId: string,
    data: SubmitScoreRequest
  ): Promise<SubmitScoreResponse> {
    try {
      const response = await gradesApi
        .post(`courses/${courseId}/scores`, {
          json: data,
        })
        .json<SubmitScoreResponse>()
      return response
    } catch (error) {
      console.error("Failed to submit score:", error)
      throw error
    }
  },

  async bulkSubmitScores(
    courseId: string,
    data: BulkSubmitScoresRequest
  ): Promise<BulkSubmitScoresResponse> {
    try {
      const response = await gradesApi
        .post(`courses/${courseId}/scores/bulk`, {
          json: data,
        })
        .json<BulkSubmitScoresResponse>()
      return response
    } catch (error) {
      console.error("Failed to bulk submit scores:", error)
      throw error
    }
  },

  async unlockScore(data: {
    registration_id: string
    slug: string
  }): Promise<void> {
    try {
      await gradesApi
        .post("admin/scores/unlock", {
          json: data,
        })
        .json()
    } catch (error) {
      console.error("Failed to unlock score:", error)
      throw error
    }
  },

  /** Admin: corrects one score of a finalized course and regrades it. */
  async processAppeal(data: GradeAppealRequest): Promise<GradeAppealResponse> {
    return gradesApi
      .post("admin/appeals", { json: data })
      .json<GradeAppealResponse>()
  },

  async lockScore(data: {
    registration_id: string
    slug: string
  }): Promise<void> {
    try {
      await gradesApi
        .post("admin/scores/lock", {
          json: data,
        })
        .json()
    } catch (error) {
      console.error("Failed to lock score:", error)
      throw error
    }
  },
}

export default gradesService
