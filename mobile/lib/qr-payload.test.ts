import { createScanGate, parseQRPayload } from './qr-payload';

// parseQRPayload tests pin the contract every QR-driven UI relies on:
// - returns null for unparseable input rather than throwing,
// - rejects partially-shaped payloads (just sid, just sig, wrong types)
// because a partial match would let bogus QRs through to the API.
describe('parseQRPayload', () => {
  it('returns the typed payload for a well-formed QR', () => {
    const raw = JSON.stringify({ sid: 'session-1', sig: 'sig-1' });
    expect(parseQRPayload(raw)).toEqual({ sid: 'session-1', sig: 'sig-1' });
  });

  it('returns null for invalid JSON without throwing', () => {
    expect(parseQRPayload('not-json')).toBeNull();
    expect(parseQRPayload('')).toBeNull();
    expect(parseQRPayload('{not closed')).toBeNull();
  });

  it('returns null when sid is missing', () => {
    const raw = JSON.stringify({ sig: 'sig-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sig is missing', () => {
    const raw = JSON.stringify({ sid: 'session-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sid is not a string', () => {
    const raw = JSON.stringify({ sid: 12345, sig: 'sig-1' });
    expect(parseQRPayload(raw)).toBeNull();
  });

  it('returns null when sig is not a string', () => {
    const raw = JSON.stringify({ sid: 'session-1', sig: { v: 'x' } });
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
    const raw = JSON.stringify({ sid: 'session-1', sig: 'sig-1', extra: 'ignore-me' });
    const out = parseQRPayload(raw);
    expect(out).toEqual({ sid: 'session-1', sig: 'sig-1' });
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
