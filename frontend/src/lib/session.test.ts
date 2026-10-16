import { afterEach, describe, expect, it } from "vitest"
import { QueryClient } from "@tanstack/react-query"
import { clearUserSession } from "./session"

describe("clearUserSession", () => {
  afterEach(() => localStorage.clear())

  it("drops every cached query and the stored user", () => {
    const queryClient = new QueryClient()
    queryClient.setQueryData(["grades", "my"], { gpa: 3.2 })
    queryClient.setQueryData(["auth", "sessions"], [{ id: "s-1" }])
    localStorage.setItem("user", JSON.stringify({ id: "u-1" }))

    clearUserSession(queryClient)

    expect(queryClient.getQueryCache().getAll()).toHaveLength(0)
    expect(queryClient.getQueryData(["grades", "my"])).toBeUndefined()
    expect(localStorage.getItem("user")).toBeNull()
  })
})
