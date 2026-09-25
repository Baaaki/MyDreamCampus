import { useEffect, useState } from "react"
import { apiClient } from "@/lib/api-client"

export interface BaselineStatus {
  mode: "normal" | "editing" | "busy"
  current: string
  versions: string[]
  edit_deadline: string | null
  last_action: string
  // The command the status reflects; absent before demo-ops wrote one.
  last_command_id?: string | null
  last_error: string | null
  updated_at: string
}

interface QueuedCommand {
  command_id: string
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

// The ops endpoints only queue the command for demo-ops; the returned id
// shows up in the status as last_command_id once demo-ops has handled it.
async function queueCommand(path: string): Promise<string> {
  const res = await apiClient.post(path).json<QueuedCommand>()
  return res.command_id
}

export async function getBaselineStatus(): Promise<BaselineStatus> {
  return apiClient.get("api/catalog/admin/ops/status").json<BaselineStatus>()
}

export async function beginEdit(): Promise<string> {
  return queueCommand("api/catalog/admin/ops/begin-edit")
}

export async function saveBaseline(): Promise<string> {
  return queueCommand("api/catalog/admin/ops/save")
}

export async function cancelEdit(): Promise<string> {
  return queueCommand("api/catalog/admin/ops/cancel-edit")
}

export async function restoreNow(): Promise<string> {
  return queueCommand("api/catalog/admin/ops/restore-now")
}

export async function restoreVersion(version: string): Promise<string> {
  return queueCommand(`api/catalog/admin/ops/restore/${version}`)
}
