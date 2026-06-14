// Mirror of backend shared/utils/password.go ValidatePasswordPolicy.
// Keep these two in sync if you change the rule.

export const PASSWORD_POLICY_MESSAGE =
  "Sifre en az 8 karakter olmali ve en az bir buyuk harf, bir kucuk harf ve bir rakam icermelidir";

export function validatePasswordPolicy(password: string): string | null {
  if (password.length < 8) return PASSWORD_POLICY_MESSAGE;
  if (!/[A-Z]/.test(password)) return PASSWORD_POLICY_MESSAGE;
  if (!/[a-z]/.test(password)) return PASSWORD_POLICY_MESSAGE;
  if (!/[0-9]/.test(password)) return PASSWORD_POLICY_MESSAGE;
  return null;
}
