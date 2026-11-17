// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { base32Decode, base32Encode, totpAt, verifyTotp } from './totp';

// RFC 6238 appendix B: SHA-1 secret "12345678901234567890"; 8-digit codes, last 6 digits here.
const RFC_SECRET = base32Encode(new TextEncoder().encode('12345678901234567890'));

describe('base32', () => {
  it('round-trips bytes', () => {
    const bytes = new Uint8Array([0, 1, 2, 250, 255, 128, 64]);
    expect(base32Decode(base32Encode(bytes))).toEqual(bytes);
  });

  it('ignores case, spaces and padding', () => {
    expect(base32Decode('mzxw 6ytb oi======')).toEqual(new TextEncoder().encode('foobar'));
  });

  it('rejects characters outside the alphabet', () => {
    expect(() => base32Decode('ABC1')).toThrow();
  });
});

describe('totpAt (RFC 6238 vectors)', () => {
  it.each([
    [59, '287082'],
    [1111111109, '081804'],
    [1111111111, '050471'],
    [1234567890, '005924'],
    [2000000000, '279037'],
  ])('at t=%i gives %s', async (seconds, code) => {
    expect(await totpAt(RFC_SECRET, seconds * 1000)).toBe(code);
  });
});

describe('verifyTotp', () => {
  const now = 1234567890 * 1000;

  it('accepts the current code and returns its time step', async () => {
    const code = await totpAt(RFC_SECRET, now);
    expect(await verifyTotp(RFC_SECRET, code, now)).toBe(Math.floor(now / 30000));
  });

  it('accepts the previous and next step (clock drift)', async () => {
    expect(await verifyTotp(RFC_SECRET, await totpAt(RFC_SECRET, now - 30000), now)).not.toBeNull();
    expect(await verifyTotp(RFC_SECRET, await totpAt(RFC_SECRET, now + 30000), now)).not.toBeNull();
  });

  it('rejects codes two steps away', async () => {
    expect(await verifyTotp(RFC_SECRET, await totpAt(RFC_SECRET, now - 60000), now)).toBeNull();
  });

  it.each(['', '12345', '1234567', 'abcdef', '000000'])('rejects %j', async (code) => {
    expect(await verifyTotp(RFC_SECRET, code, now)).toBeNull();
  });

  it('rejects a step at or before the last one used (replay)', async () => {
    const code = await totpAt(RFC_SECRET, now);
    const step = Math.floor(now / 30000);
    expect(await verifyTotp(RFC_SECRET, code, now, step)).toBeNull();
    expect(await verifyTotp(RFC_SECRET, code, now, step - 1)).toBe(step);
  });
});
