// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { authorize, safeNextPath } from './authorize';
import { createSession } from './session';
import type { AuthConfig } from './config';

const NOW = Date.UTC(2026, 10, 17, 12);
const config: AuthConfig = {
  enabled: true,
  sessionSecret: 's'.repeat(64),
  passwordHash: 'scrypt:x',
  totpSecret: 'JBSWY3DPEHPK3PXP',
  sessionTtlMs: 60_000,
  secureCookie: true,
  required: false,
};

const valid = () => createSession(config.sessionSecret, { now: NOW, ttlMs: 60_000 });

describe('authorize', () => {
  it('lets everything through when auth is not configured', async () => {
    expect(await authorize({ pathname: '/', cookie: undefined, now: NOW, config: { ...config, enabled: false } }))
      .toEqual({ action: 'allow' });
  });

  it('fails closed (503) when auth is required but not configured', async () => {
    expect(
      await authorize({ pathname: '/', cookie: undefined, now: NOW, config: { ...config, enabled: false, required: true } }),
    ).toEqual({ action: 'unavailable' });
    expect(
      await authorize({ pathname: '/login', cookie: undefined, now: NOW, config: { ...config, enabled: false, required: true } }),
    ).toEqual({ action: 'unavailable' });
  });

  it('fails closed when required and the secret is too short or a piece is missing', async () => {
    for (const broken of [
      { sessionSecret: 's'.repeat(31) },
      { passwordHash: '' },
      { totpSecret: '' },
    ]) {
      const decision = await authorize({
        pathname: '/',
        cookie: await valid(),
        now: NOW,
        config: { ...config, ...broken, required: true },
      });
      expect(decision).toEqual({ action: 'unavailable' });
    }
  });

  it('works normally when required and fully configured', async () => {
    expect(
      await authorize({ pathname: '/', cookie: await valid(), now: NOW, config: { ...config, required: true } }),
    ).toEqual({ action: 'allow' });
  });

  it('allows a valid session', async () => {
    expect(await authorize({ pathname: '/habits', cookie: await valid(), now: NOW, config }))
      .toEqual({ action: 'allow' });
  });

  it('redirects pages without a session to /login, keeping the target', async () => {
    expect(await authorize({ pathname: '/habits', cookie: undefined, now: NOW, config }))
      .toEqual({ action: 'redirect', location: '/login?next=%2Fhabits' });
  });

  it('keeps the query string in the login redirect', async () => {
    expect(await authorize({ pathname: '/habits', search: '?tab=todo', cookie: undefined, now: NOW, config }))
      .toEqual({ action: 'redirect', location: '/login?next=%2Fhabits%3Ftab%3Dtodo' });
  });

  it('answers 401 for API calls without a session', async () => {
    expect(await authorize({ pathname: '/api/backend/events', cookie: undefined, now: NOW, config }))
      .toEqual({ action: 'unauthorized' });
  });

  it('treats an expired or forged session as none', async () => {
    const expired = await createSession(config.sessionSecret, { now: NOW - 120_000, ttlMs: 60_000 });
    expect((await authorize({ pathname: '/', cookie: expired, now: NOW, config })).action).toBe('redirect');
    const forged = await createSession('f'.repeat(64), { now: NOW, ttlMs: 60_000 });
    expect((await authorize({ pathname: '/', cookie: forged, now: NOW, config })).action).toBe('redirect');
  });

  it.each(['/login', '/api/auth/login'])('keeps %s public', async (pathname) => {
    expect(await authorize({ pathname, cookie: undefined, now: NOW, config })).toEqual({ action: 'allow' });
  });

  it.each(['/login/x', '/loginx', '/api/auth/logout', '/api/auth/login/../../backend/events'])(
    'does not treat %s as public',
    async (pathname) => {
      expect((await authorize({ pathname, cookie: undefined, now: NOW, config })).action).not.toBe('allow');
    },
  );
});

describe('safeNextPath', () => {
  it.each([
    ['/habits', '/habits'],
    ['/habits?x=1', '/habits?x=1'],
    [null, '/'],
    ['', '/'],
    ['//evil.com', '/'],
    ['/\\evil.com', '/'],
    ['https://evil.com', '/'],
    ['javascript:alert(1)', '/'],
    ['habits', '/'],
    ['/\t/evil.com', '/'],
    ['/\n/evil.com', '/'],
    ['/\r/evil.com', '/'],
    ['/%09/evil.com', '/%09/evil.com'],
    ['/habits#x', '/habits#x'],
  ])('%j -> %j', (input, expected) => {
    expect(safeNextPath(input)).toBe(expected);
  });
});
