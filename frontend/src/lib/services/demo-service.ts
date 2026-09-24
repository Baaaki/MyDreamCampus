import { useQuery } from "@tanstack/react-query"
import { authApiSafe } from "@/lib/api-client"
import type { DemoAccount } from "@/lib/types"

export async function fetchDemoAccounts(): Promise<DemoAccount[]> {
  try {
    const res = await authApiSafe.get("demo-accounts").json<DemoAccount[]>()
    return Array.isArray(res) ? res : []
  } catch {
    return []
  }
}

export function useDemoAccounts() {
  return useQuery<DemoAccount[]>({
    queryKey: ["demo-accounts"],
    queryFn: fetchDemoAccounts,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

export function useIsDemoMode(): boolean {
  const { data } = useDemoAccounts()
  return Boolean(data && data.length > 0)
}
