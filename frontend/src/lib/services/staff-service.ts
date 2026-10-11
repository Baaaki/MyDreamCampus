import { staffApi } from "@/lib/api-client"
import type { Staff } from "@/lib/types"

/**
 * What PUT /api/staff/:id accepts. Name, e-mail and status are not
 * editable there; anything else in the body is ignored.
 */
export interface StaffUpdate {
  department: string
  phone: string
  office_location: string
}

export function toStaffUpdate(staff: Staff): StaffUpdate {
  return {
    department: staff.department ?? "",
    phone: staff.phone ?? "",
    office_location: staff.office_location ?? "",
  }
}

export const staffService = {
  update(id: string, data: StaffUpdate): Promise<Staff> {
    return staffApi.put(id, { json: data }).json<Staff>()
  },

  /**
   * Deactivates the account: auth closes its login and student drops it as
   * advisor. A protected account answers 403.
   */
  async remove(id: string): Promise<void> {
    await staffApi.delete(id)
  },
}
