import { HTTPError } from 'ky';

// Backend error bodies come in two shapes: auth answers
// {error: CODE, message: "Türkçe metin"}, the other services
// {error: "Türkçe metin", code: CODE}. Whichever field holds prose is the
// one to show; an UPPER_SNAKE value is a machine code, not a message.
type ErrorBody = { error?: unknown; message?: unknown };

const MACHINE_CODE = /^[A-Z][A-Z0-9_]+$/;

/** The backend's user-facing message for a failed request, or `fallback`. */
export async function apiErrorMessage(err: unknown, fallback: string): Promise<string> {
  if (!(err instanceof HTTPError)) return fallback;
  try {
    const body = (await err.response.clone().json()) as ErrorBody;
    for (const candidate of [body.message, body.error]) {
      if (typeof candidate === 'string' && candidate !== '' && !MACHINE_CODE.test(candidate)) {
        return candidate;
      }
    }
  } catch {
    // Not JSON — fall through to the generic message.
  }
  return fallback;
}
