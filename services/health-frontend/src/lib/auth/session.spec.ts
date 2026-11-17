// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { createSession, verifySession } from './session';

const SECRET = 'a'.repeat(64);
const NOW = Date.UTC(2026, 10, 17, 12);

describe('session token', () => {
  it('verifies a token it created', async () => {
    const token = await createSession(SECRET, { now: NOW, ttlMs: 60_000 });
    expect(await verifySession(SECRET, token, NOW + 1_000)).toMatchObject({ exp: NOW + 60_000 });
  });

  it('rejects an expired token', async () => {
    const token = await createSession(SECRET, { now: NOW, ttlMs: 60_000 });
    expect(await verifySession(SECRET, token, NOW + 60_001)).toBeNull();
  });

  it('rejects a token signed with another secret', async () => {
    const token = await createSession('b'.repeat(64), { now: NOW, ttlMs: 60_000 });
    expect(await verifySession(SECRET, token, NOW)).toBeNull();
  });

  it('rejects a tampered payload', async () => {
    const token = await createSession(SECRET, { now: NOW, ttlMs: 60_000 });
    const [payload, sig] = token.split('.');
    const forged = Buffer.from(
      JSON.stringify({ ...JSON.parse(Buffer.from(payload, 'base64url').toString()), exp: NOW * 2 }),
    ).toString('base64url');
    expect(await verifySession(SECRET, `${forged}.${sig}`, NOW)).toBeNull();
  });

  it.each([undefined, '', 'x', 'a.b.c', '.', 'abc.', '.abc', '!!!.???'])('rejects malformed %j', async (t) => {
    expect(await verifySession(SECRET, t, NOW)).toBeNull();
  });

  it('refuses to sign or verify with a short secret', async () => {
    await expect(createSession('short', { now: NOW, ttlMs: 1 })).rejects.toThrow();
    expect(await verifySession('short', 'a.b', NOW)).toBeNull();
  });

  it('gives each session a unique id', async () => {
    const a = await createSession(SECRET, { now: NOW, ttlMs: 1 });
    const b = await createSession(SECRET, { now: NOW, ttlMs: 1 });
    expect(a).not.toBe(b);
  });
});
