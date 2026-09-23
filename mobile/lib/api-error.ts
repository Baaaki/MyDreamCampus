// Backend error bodies come in two shapes: auth answers
// {error: CODE, message: "Türkçe metin"}, the other services
// {error: "Türkçe metin", code: CODE}. Whichever field holds prose is the
// one to show; an UPPER_SNAKE value is a machine code, not a message.
const MACHINE_CODE = /^[A-Z][A-Z0-9_]+$/;

function errorBody(err: unknown): object | null {
  if (typeof err !== 'object' || err === null || !('response' in err)) return null;
  const { response } = err;
  if (typeof response !== 'object' || response === null || !('data' in response)) return null;
  const { data } = response;
  return typeof data === 'object' && data !== null ? data : null;
}

/** The backend's user-facing message for a failed request, or `fallback`. */
export function apiErrorMessage(err: unknown, fallback: string): string {
  const body = errorBody(err);
  if (!body) return fallback;
  const candidates = [
    'message' in body ? body.message : undefined,
    'error' in body ? body.error : undefined,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate !== '' && !MACHINE_CODE.test(candidate)) {
      return candidate;
    }
  }
  return fallback;
}

/** The machine code of a failed request (e.g. RESERVATION_CONFLICTS), if any. */
export function apiErrorCode(err: unknown): string | undefined {
  const body = errorBody(err);
  if (!body) return undefined;
  if ('code' in body && typeof body.code === 'string') return body.code;
  if ('error' in body && typeof body.error === 'string' && MACHINE_CODE.test(body.error)) {
    return body.error;
  }
  return undefined;
}
