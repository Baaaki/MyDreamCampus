import { describe, it, expect } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  it("joins class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1");
  });

  it("filters out falsy values", () => {
    expect(cn("a", false, null, undefined, "b")).toBe("a b");
  });

  it("respects tailwind-merge precedence (last wins)", () => {
    // tw-merge collapses conflicting tailwind utilities
    expect(cn("px-2", "px-4")).toBe("px-4");
    expect(cn("text-red-500", "text-blue-500")).toBe("text-blue-500");
  });

  it("supports clsx conditional object form", () => {
    expect(cn("base", { active: true, disabled: false })).toBe("base active");
  });

  it("handles arrays", () => {
    expect(cn(["a", "b"], "c")).toBe("a b c");
  });

  it("returns empty string for no args", () => {
    expect(cn()).toBe("");
  });
});
