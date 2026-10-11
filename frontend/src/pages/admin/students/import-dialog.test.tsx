import { afterEach, describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { StudentImportJob } from "@/lib/types"
import { studentApi } from "@/lib/api-client"
import { studentImportService } from "@/lib/services/student-import-service"
import { StudentImportDialog } from "./import-dialog"

const JOB_ID = "0190a1b2-0000-7000-8000-0000000000aa"

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function job(overrides: Partial<StudentImportJob>): StudentImportJob {
  return {
    job_id: JOB_ID,
    file_name: "ogrenciler.csv",
    status: "processing",
    total_records: 3,
    processed_records: 0,
    successful_records: 0,
    failed_records: 0,
    created_by: "admin",
    created_at: "2026-10-12T09:00:00Z",
    ...overrides,
  }
}

/** Answers the upload, then serves each job read from `reads` in turn. */
function stubBackend(upload: Response, reads: StudentImportJob[]) {
  // jsdom's File cannot cross into Node's Request, so the upload goes out
  // with the file's name as its body; the multipart body itself is covered
  // by student-import-service.test.ts. Status and error handling stay real.
  vi.spyOn(studentImportService, "start").mockImplementation((file) =>
    studentApi
      .post("bulk-import", { body: file.name })
      .json<{ job_id: string }>()
      .then((res) => res.job_id)
  )
  let read = 0
  const fetchSpy = vi.fn(async (input: RequestInfo | URL) => {
    const request = input as Request
    const path = new URL(request.url).pathname
    if (path === "/api/students/bulk-import" && request.method === "POST") {
      return upload.clone()
    }
    if (path === `/api/students/bulk-import/${JOB_ID}`) {
      return json(reads[Math.min(read++, reads.length - 1)])
    }
    return json({ error: "unexpected" }, 500)
  })
  vi.stubGlobal("fetch", fetchSpy)
  return fetchSpy
}

function renderDialog() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const invalidate = vi.spyOn(client, "invalidateQueries")
  render(
    <QueryClientProvider client={client}>
      <StudentImportDialog />
    </QueryClientProvider>
  )
  return { invalidate }
}

async function uploadFile() {
  await userEvent.click(screen.getByRole("button", { name: "Toplu İçe Aktar" }))
  const file = new File(["x"], "ogrenciler.csv", { type: "text/csv" })
  await userEvent.upload(screen.getByLabelText("CSV Dosyası"), file)
  await userEvent.click(screen.getByRole("button", { name: "İçe Aktar" }))
}

describe("StudentImportDialog", () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it("names the header the backend expects", async () => {
    stubBackend(json({ job_id: JOB_ID }, 202), [])
    renderDialog()

    await userEvent.click(
      screen.getByRole("button", { name: "Toplu İçe Aktar" })
    )

    expect(
      screen.getByText(
        "student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level"
      )
    ).toBeInTheDocument()
  })

  it("polls the job until it completes, then shows counts and row errors and refreshes the list", async () => {
    stubBackend(json({ job_id: JOB_ID, status: "pending" }, 202), [
      job({ status: "processing" }),
      job({
        status: "completed",
        processed_records: 3,
        successful_records: 1,
        failed_records: 2,
        errors: [
          {
            row: 4,
            student_number: "2024003",
            error_code: "STUDENT_NUMBER_EXISTS",
            message: "Bu öğrenci numarası zaten kullanılıyor",
          },
          {
            row: 2,
            student_number: "2024001",
            error_code: "INVALID_CSV_FORMAT",
            message: 'geçersiz e-posta: "x"@univ.edu',
          },
        ],
      }),
    ])
    const { invalidate } = renderDialog()

    await uploadFile()

    expect(await screen.findByText(/Öğrenciler ekleniyor/)).toBeInTheDocument()
    expect(
      await screen.findByText("İçe aktarma tamamlandı.", undefined, {
        timeout: 3000,
      })
    ).toBeInTheDocument()

    expect(
      within(screen.getByText("Başarılı").parentElement!).getByText("1")
    ).toBeInTheDocument()
    expect(
      within(screen.getByText("Başarısız").parentElement!).getByText("2")
    ).toBeInTheDocument()

    const rows = screen.getAllByRole("row").slice(1)
    expect(rows.map((r) => r.textContent)).toEqual([
      '22024001geçersiz e-posta: "x"@univ.edu',
      "42024003Bu öğrenci numarası zaten kullanılıyor",
    ])
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["students", "list"] })
  })

  it("tells the admin when no student could be added", async () => {
    stubBackend(json({ job_id: JOB_ID }, 202), [
      job({
        status: "failed",
        total_records: 1,
        processed_records: 1,
        failed_records: 1,
        errors: [
          {
            row: 2,
            student_number: "2024001",
            error_code: "STUDENT_EMAIL_EXISTS",
            message: "Bu e-posta adresi zaten kullanılıyor",
          },
        ],
      }),
    ])
    renderDialog()

    await uploadFile()

    expect(
      await screen.findByText(
        "İçe aktarma başarısız: hiçbir öğrenci eklenemedi."
      )
    ).toBeInTheDocument()
    expect(
      screen.getByText("Bu e-posta adresi zaten kullanılıyor")
    ).toBeInTheDocument()
  })

  it("shows the backend's message when the file is rejected", async () => {
    const fetchSpy = stubBackend(
      json(
        {
          error:
            "CSV başlığı şu sütunlar olmalı: student_number,first_name,last_name,email,faculty,department,enrollment_year,class_level",
          code: "INVALID_CSV_FORMAT",
        },
        400
      ),
      []
    )
    renderDialog()

    await uploadFile()

    expect(
      await screen.findByText(/CSV başlığı şu sütunlar olmalı/)
    ).toBeInTheDocument()
    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })
})
