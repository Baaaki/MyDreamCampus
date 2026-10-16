import type { QueryClient } from "@tanstack/react-query"

/**
 * Forgets everything the signed-in user loaded. Login and logout happen
 * inside the SPA without a page load, so without this the next account in
 * the same tab is served the previous account's cached data.
 */
export function clearUserSession(queryClient: QueryClient) {
  queryClient.clear()
  localStorage.removeItem("user")
}
