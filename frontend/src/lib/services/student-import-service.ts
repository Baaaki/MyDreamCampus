import { skipToken, useQuery } from "@tanstack/react-query"
import { studentApi } from "@/lib/api-client"
import type { StudentImportJob, StudentImportStatus } from "@/lib/types"

/** The header row the backend accepts, in this exact order. */
export const STUDENT_IMPORT_HEADER =
  "student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level"

const POLL_INTERVAL_MS = 1000

export function isImportFinished(status: StudentImportStatus | undefined) {
  return status === "completed" || status === "failed"
}

export const studentImportService = {
  async start(file: File): Promise<string> {
    const body = new FormData()
    body.append("file", file)
    // studentApi (ky) attaches cookies + CSRF header; raw fetch would 403
    // on the CSRF-protected admin route.
    const res = await studentApi
      .post("bulk-import", { body })
      .json<{ job_id: string }>()
    return res.job_id
  },

  async get(jobId: string): Promise<StudentImportJob> {
    return studentApi.get(`bulk-import/${jobId}`).json<StudentImportJob>()
  },
}

/** Polls an import job until it completes or fails. */
export function useStudentImportJob(jobId: string | null) {
  return useQuery({
    queryKey: ["students", "import", jobId],
    queryFn: jobId ? () => studentImportService.get(jobId) : skipToken,
    refetchInterval: (query) =>
      isImportFinished(query.state.data?.status) ? false : POLL_INTERVAL_MS,
  })
}
