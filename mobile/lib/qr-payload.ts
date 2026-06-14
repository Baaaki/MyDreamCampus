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
      typeof parsed.sig === 'string'
    ) {
      return { sid: parsed.sid, sig: parsed.sig };
    }
  } catch {
    // fall through to null
  }
  return null;
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
