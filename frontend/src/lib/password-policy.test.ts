import { describe, it, expect } from "vitest";
import { PASSWORD_POLICY_MESSAGE, validatePasswordPolicy } from "./password-policy";

describe("validatePasswordPolicy", () => {
  it.each([
    ["Aa345678", "valid 8 chars with all classes"],
    ["MyS3cretPassphrase", "long valid password"],
    ["Pa55w0rd", "minimum mix"],
  ])("returns null for valid: %s (%s)", (pw) => {
    expect(validatePasswordPolicy(pw)).toBeNull();
  });

  it.each([
    ["", "empty"],
    ["short1A", "too short (7 chars)"],
    ["abcdefgh1", "no uppercase"],
    ["ABCDEFGH1", "no lowercase"],
    ["Abcdefgh", "no digit"],
    ["12345678", "only digits"],
    ["AAAAAAAA", "only uppercase"],
  ])("returns policy message for invalid: %s (%s)", (pw) => {
    expect(validatePasswordPolicy(pw)).toBe(PASSWORD_POLICY_MESSAGE);
  });

  it("must mirror backend policy: 8+ / upper / lower / digit", () => {
    // Sentinel: this docstring + assertion ensures any change here
    // breaks the test, prompting a sync with backend ValidatePasswordPolicy.
    expect(PASSWORD_POLICY_MESSAGE).toContain("8");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("buyuk");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("kucuk");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("rakam");
  });
});
