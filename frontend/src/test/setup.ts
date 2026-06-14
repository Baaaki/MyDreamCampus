import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

// Tear down DOM mounts and reset matchers between tests so suite ordering
// doesn't leak state.
afterEach(() => {
  cleanup();
});
