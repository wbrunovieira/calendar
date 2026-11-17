// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { base32, hashPassword as scriptHash } from '../../../scripts/auth-setup.mjs';
import { verifyPassword } from './password';
import { base32Decode } from './totp';

describe('scripts/auth-setup.mjs agrees with the app', () => {
  it('produces a hash the app verifies', async () => {
    const hash = await scriptHash('a long enough password');
    expect(await verifyPassword('a long enough password', hash)).toBe(true);
    expect(await verifyPassword('wrong', hash)).toBe(false);
  });

  it('produces base32 the app decodes', () => {
    const bytes = new Uint8Array([1, 2, 3, 4, 5, 250]);
    expect(base32Decode(base32(bytes))).toEqual(bytes);
  });
});
