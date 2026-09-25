import { useEffect, useState } from "react"
import { apiClient } from "@/lib/api-client"

export interface BaselineStatus {
  mode: "normal" | "editing" | "busy"
  current: string
  versions: string[]
  edit_deadline: string | null
  last_action: string
  last_error: string | null
  updated_at: string
}

let isSystemEditingState = false
const listeners = new Set<(isEditing: boolean) => void>()

export function setSystemEditing(isEditing: boolean): void {
  if (isSystemEditingState !== isEditing) {
    isSystemEditingState = isEditing
    listeners.forEach((listener) => listener(isEditing))
  }
}

export function useIsSystemEditing(): boolean {
  const [isEditing, setIsEditing] = useState(isSystemEditingState)

  useEffect(() => {
    listeners.add(setIsEditing)
    return () => {
      listeners.delete(setIsEditing)
    }
  }, [])

  return isEditing
}

export async function getBaselineStatus(): Promise<BaselineStatus> {
  return apiClient.get("api/catalog/admin/ops/status").json<BaselineStatus>()
}

export async function beginEdit(): Promise<void> {
  await apiClient.post("api/catalog/admin/ops/begin-edit").json()
}

export async function saveBaseline(): Promise<void> {
  await apiClient.post("api/catalog/admin/ops/save").json()
}

export async function cancelEdit(): Promise<void> {
  await apiClient.post("api/catalog/admin/ops/cancel-edit").json()
}

export async function restoreNow(): Promise<void> {
  await apiClient.post("api/catalog/admin/ops/restore-now").json()
}

export async function restoreVersion(version: string): Promise<void> {
  await apiClient.post(`api/catalog/admin/ops/restore/${version}`).json()
}
