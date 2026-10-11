import type { QRPayload } from '@/types/attendance.types';

// parseQRPayload turns the raw text inside a QR code into a typed payload, or
// returns null on any structural failure. It is intentionally tolerant: the
// caller decides how to surface "unparseable" (toast vs. close vs. silent).
//
// Rejecting both bad JSON and partial shapes here keeps the QRScannerModal
// component handler small — it only has to branch on null vs. payload.
export function parseQRPayload(raw: string): QRPayload | null {
  try {
    const parsed = JSON.parse(raw);
    if (
      parsed &&
      typeof parsed === 'object' &&
      typeof parsed.sid === 'string' &&
      Number.isInteger(parsed.win) &&
      typeof parsed.sig === 'string'
    ) {
      return { sid: parsed.sid, win: parsed.win, sig: parsed.sig };
    }
  } catch {
    // fall through to null
  }
  return null;
}

// Cafeteria QR is plain text, not JSON (meal-service generateQRPayload):
// "<cafeteria uuid>:<YYYY-MM-DD>:<lunch|dinner>:<window>:<hmac-sha256 hex>".
// The signature is the backend's to verify; this only tells the two QR kinds
// apart so a meal code is not sent to the attendance endpoint (or vice versa).
const MEAL_QR =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}:\d{4}-\d{2}-\d{2}:(?:lunch|dinner):\d+:[0-9a-f]{64}$/i;

export type ScannedQR =
  | { kind: 'attendance'; payload: QRPayload }
  | { kind: 'meal'; qrPayload: string };

export function parseScannedQR(raw: string): ScannedQR | null {
  const text = raw.trim();
  if (MEAL_QR.test(text)) return { kind: 'meal', qrPayload: text };
  const payload = parseQRPayload(text);
  return payload ? { kind: 'attendance', payload } : null;
}

// ScanGate guards against the camera firing onBarcodeScanned multiple times
// per second for the same physical QR. The component fix (handledRef.current)
// implements the same state machine inline; pulling it out lets us pin the
// behaviour with unit tests, which is important because every regression
// here causes duplicate API calls (and silent dedup quotas).
//
// States:
//   - idle → tryConsume(payload) returns true and transitions to "consumed"
//   - consumed → tryConsume(...) returns false, even for different payloads
//   - release() returns to idle
//
// Concurrency: a single component instance owns one ScanGate. Calls are
// serialised via the React render loop, so we don't need atomicity beyond
// "first call wins on a single scheduler tick".
export type ScanGate = {
  tryConsume: () => boolean;
  release: () => void;
  isOpen: () => boolean;
};

export function createScanGate(): ScanGate {
  let consumed = false;
  return {
    tryConsume() {
      if (consumed) return false;
      consumed = true;
      return true;
    },
    release() {
      consumed = false;
    },
    isOpen() {
      return !consumed;
    },
  };
}
