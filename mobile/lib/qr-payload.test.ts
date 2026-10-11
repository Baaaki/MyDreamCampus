import { createScanGate, parseQRPayload, parseScannedQR } from './qr-payload';

// parseQRPayload tests pin the contract every QR-driven UI relies on:
// - returns null for unparseable input rather than throwing,
// - rejects partially-shaped payloads (just sid, just sig, wrong types)
// because a partial match would let bogus QRs through to the API.
describe('parseQRPayload', () => {
  it('returns the typed payload for a well-formed QR', () => {
    const raw = JSON.stringify({ sid: 'session-1', win: 7, sig: 'sig-1' });
    expect(parseQRPayload(raw)).toEqual({ sid: 'session-1', win: 7, sig: 'sig-1' });
  });

  it('returns null for invalid JSON without throwing', () => {
    expect(parseQRPayload('not-json')).toBeNull();
    expect(parseQRPayload('')).toBeNull();
    expect(parseQRPayload('{not closed')).toBeNull();
  });

  it('returns null for a pre-rotation code without a window', () => {
    const raw = JSON.stringify({ sid: 'session-1', sig: 'sig-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when win is not an integer', () => {
    expect(parseQRPayload(JSON.stringify({ sid: 'session-1', win: '7', sig: 'sig-1' }))).toBeNull();
    expect(parseQRPayload(JSON.stringify({ sid: 'session-1', win: 7.5, sig: 'sig-1' }))).toBeNull();
  });

  it('returns null when sid is missing', () => {
    const raw = JSON.stringify({ win: 7, sig: 'sig-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sig is missing', () => {
    const raw = JSON.stringify({ sid: 'session-1', win: 7 });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sid is not a string', () => {
    const raw = JSON.stringify({ sid: 12345, win: 7, sig: 'sig-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sig is not a string', () => {
    const raw = JSON.stringify({ sid: 'session-1', win: 7, sig: { v: 'x' } });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null for a bare null payload', () => {
    expect(parseQRPayload('null')).toBeNull();
  });

  it('returns null for a primitive payload (string/number/array)', () => {
    expect(parseQRPayload(JSON.stringify('hello'))).toBeNull();
    expect(parseQRPayload(JSON.stringify(42))).toBeNull();
    expect(parseQRPayload(JSON.stringify(['session-1', 'sig-1']))).toBeNull();
  });

  it('ignores extra unknown fields', () => {
    const raw = JSON.stringify({ sid: 'session-1', win: 7, sig: 'sig-1', extra: 'ignore-me' });
    const out = parseQRPayload(raw);
    expect(out).toEqual({ sid: 'session-1', win: 7, sig: 'sig-1' });
    expect((out as { extra?: string }).extra).toBeUndefined();
  });
});

// createScanGate tests pin the race-condition fix. The component instance
// owns one gate; the camera fires onBarcodeScanned multiple times per second
// for the same QR. Without the gate every frame fires `onScanned`, which
// deduces to N concurrent API calls per scan — the bug this regression
// test is built to never re-introduce.
describe('createScanGate', () => {
  it('starts open', () => {
    const gate = createScanGate();
    expect(gate.isOpen()).toBe(true);
  });

  it('admits a single tryConsume and rejects subsequent attempts', () => {
    const gate = createScanGate();

    expect(gate.tryConsume()).toBe(true);
    expect(gate.tryConsume()).toBe(false);
    expect(gate.tryConsume()).toBe(false);
  });

  it('release re-opens the gate so a re-aim works', () => {
    // The component releases after a 1.5s delay on invalid QR. The gate
    // itself is timing-free: release() is enough.
    const gate = createScanGate();

    expect(gate.tryConsume()).toBe(true);
    expect(gate.tryConsume()).toBe(false);

    gate.release();
    expect(gate.isOpen()).toBe(true);
    expect(gate.tryConsume()).toBe(true);
  });

  it('many parallel scan callbacks see exactly one success', () => {
    // Worst-case simulation: 50 callbacks fire on the same render tick.
    // Without the gate every one calls onScanned. With the gate exactly
    // one wins.
    const gate = createScanGate();
    const results = Array.from({ length: 50 }, () => gate.tryConsume());

    const wins = results.filter(Boolean).length;
    expect(wins).toBe(1);
  });

  it('two independent instances do NOT share state', () => {
    const a = createScanGate();
    const b = createScanGate();

    expect(a.tryConsume()).toBe(true);
    expect(b.tryConsume()).toBe(true);
    expect(b.isOpen()).toBe(false);
  });
});

describe('parseScannedQR', () => {
  const SIG = 'a'.repeat(64);
  const MEAL = `0b1f6c52-9a4e-4d3e-8f10-2c5d7e9a1b34:2026-10-12:lunch:123456:${SIG}`;

  it('recognises an attendance QR', () => {
    const raw = JSON.stringify({ sid: 's-1', win: 7, sig: 'sig-1' });
    expect(parseScannedQR(raw)).toEqual({
      kind: 'attendance',
      payload: { sid: 's-1', win: 7, sig: 'sig-1' },
    });
  });

  it('recognises a cafeteria QR and keeps the raw text', () => {
    expect(parseScannedQR(MEAL)).toEqual({ kind: 'meal', qrPayload: MEAL });
    expect(parseScannedQR(MEAL.replace('lunch', 'dinner'))).toEqual({
      kind: 'meal',
      qrPayload: MEAL.replace('lunch', 'dinner'),
    });
  });

  it('trims surrounding whitespace before matching', () => {
    expect(parseScannedQR(`  ${MEAL}\n`)).toEqual({ kind: 'meal', qrPayload: MEAL });
  });

  it.each([
    ['unknown meal time', MEAL.replace('lunch', 'brunch')],
    ['short signature', MEAL.slice(0, -1)],
    ['non-hex signature', MEAL.slice(0, -1) + 'z'],
    ['non-uuid cafeteria', MEAL.replace('0b1f6c52-9a4e-4d3e-8f10-2c5d7e9a1b34', 'cafe-1')],
    ['bad date', MEAL.replace('2026-10-12', '12.10.2026')],
    ['missing window', `0b1f6c52-9a4e-4d3e-8f10-2c5d7e9a1b34:2026-10-12:lunch:${SIG}`],
  ])('rejects a meal-looking text with %s', (_label, raw) => {
    expect(parseScannedQR(raw)).toBeNull();
  });

  it('rejects text that is neither kind', () => {
    expect(parseScannedQR('https://example.com')).toBeNull();
    expect(parseScannedQR('')).toBeNull();
  });
});
