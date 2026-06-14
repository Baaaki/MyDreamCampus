import { PASSWORD_POLICY_MESSAGE, validatePasswordPolicy } from "./password-policy";

describe("validatePasswordPolicy (mobile)", () => {
  it.each([
    ["Aa345678", "8-char minimum mix"],
    ["MyS3cretPass", "longer mix"],
    ["P4ssword", "exactly 8 chars"],
  ])("accepts valid: %s (%s)", (pw) => {
    expect(validatePasswordPolicy(pw)).toBeNull();
  });

  it.each([
    ["", "empty"],
    ["short1A", "7 chars"],
    ["abcdefgh1", "no uppercase"],
    ["ABCDEFGH1", "no lowercase"],
    ["Abcdefgh", "no digit"],
    ["12345678", "only digits"],
  ])("rejects invalid: %s (%s)", (pw) => {
    expect(validatePasswordPolicy(pw)).toBe(PASSWORD_POLICY_MESSAGE);
  });

  it("policy message must mention 8 chars + casing + digit", () => {
    expect(PASSWORD_POLICY_MESSAGE).toContain("8");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("buyuk");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("kucuk");
    expect(PASSWORD_POLICY_MESSAGE.toLowerCase()).toContain("rakam");
  });

  it("matches frontend rule (cross-platform contract)", () => {
    // Same input must yield same result across web + mobile clients.
    // If this test breaks, sync with frontend/src/lib/password-policy.ts.
    expect(validatePasswordPolicy("Pa55word")).toBeNull();
    expect(validatePasswordPolicy("password")).toBe(PASSWORD_POLICY_MESSAGE);
  });
});
