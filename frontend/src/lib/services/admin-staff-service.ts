import { adminStaffApi } from "@/lib/api-client"
import type { AdminStaffProfile, AdminStaffRecord } from "@/lib/types"

export function toAdminStaffProfile(r: AdminStaffRecord): AdminStaffProfile {
  return {
    id: r.id,
    title: r.title,
    firstName: r.first_name,
    lastName: r.last_name,
    faculty: r.faculty,
    department: r.department || undefined,
    email: r.email,
    phone: r.phone,
    profileImage: r.profile_image_url || undefined,
    position: r.position,
    jobDescription: r.job_description,
    responsibilities: r.responsibilities ?? [],
    workingHours: r.working_hours,
    officeLocation: r.office_location,
    startDate: r.start_date ?? "",
  }
}

export function toAdminStaffPayload(p: AdminStaffProfile) {
  return {
    email: p.email,
    title: p.title,
    first_name: p.firstName,
    last_name: p.lastName,
    faculty: p.faculty,
    department: p.department ?? "",
    phone: p.phone,
    profile_image_url: p.profileImage ?? "",
    position: p.position,
    job_description: p.jobDescription,
    // "Add" in the editor appends an empty row; an unfilled one is not a
    // responsibility.
    responsibilities: p.responsibilities
      .map((r) => r.trim())
      .filter((r) => r !== ""),
    working_hours: p.workingHours,
    office_location: p.officeLocation,
    start_date: p.startDate,
  }
}

export const adminStaffService = {
  /** Every active record of a faculty; all of them when faculty is empty. */
  async list(faculty: string): Promise<AdminStaffRecord[]> {
    const res = await adminStaffApi
      .get("", { searchParams: { faculty } })
      .json<{ data: AdminStaffRecord[] }>()
    return res.data
  },

  async get(id: string): Promise<AdminStaffProfile> {
    return toAdminStaffProfile(
      await adminStaffApi.get(id).json<AdminStaffRecord>()
    )
  },

  /** Replaces the whole record — the editor always holds every field. */
  async update(
    id: string,
    profile: AdminStaffProfile
  ): Promise<AdminStaffProfile> {
    const res = await adminStaffApi
      .put(id, { json: toAdminStaffPayload(profile) })
      .json<AdminStaffRecord>()
    return toAdminStaffProfile(res)
  },
}
