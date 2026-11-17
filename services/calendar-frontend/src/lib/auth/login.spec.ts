// @vitest-environment node
import { describe, it, expect, beforeAll, beforeEach } from 'vitest';
import { createLoginHandler } from './login';
import { hashPassword } from './password';
import { totpAt } from './totp';
import { verifySession } from './session';
import { LoginRateLimiter } from './rate-limit';
import type { AuthConfig } from './config';

const NOW = Date.UTC(2026, 10, 17, 12);
const SECRET_B32 = 'JBSWY3DPEHPK3PXP';
let config: AuthConfig;

beforeAll(async () => {
  config = {
    enabled: true,
    sessionSecret: 's'.repeat(64),
    passwordHash: await hashPassword('hunter2-but-longer'),
    totpSecret: SECRET_B32,
    sessionTtlMs: 30 * 24 * 3600 * 1000,
    secureCookie: true,
  };
});

let limiter: LoginRateLimiter;
let handler: (req: Request) => Promise<Response>;
let clock: number;
beforeEach(() => {
  clock = NOW;
  limiter = new LoginRateLimiter({ maxFailures: 3, windowMs: 15 * 60_000 });
  handler = createLoginHandler({ config: () => config, limiter, now: () => clock });
});

async function login(body: unknown, headers: Record<string, string> = {}) {
  return handler(
    new Request('https://calendar.example.com/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(body),
      headers: {
        'content-type': 'application/json',
        'sec-fetch-site': 'same-origin',
        'x-real-ip': '1.2.3.4',
        ...headers,
      },
    }),
  );
}

const cookieValue = (res: Response) => /__Host-session=([^;]+)/.exec(res.headers.get('set-cookie') ?? '')?.[1];

describe('login handler', () => {
  it('logs in with the right password and code, setting a host-only secure session cookie', async () => {
    const res = await login({ password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, NOW) });
    expect(res.status).toBe(200);
    const setCookie = res.headers.get('set-cookie') ?? '';
    expect(setCookie).toMatch(/^__Host-session=/);
    expect(setCookie).toMatch(/HttpOnly/i);
    expect(setCookie).toMatch(/Secure/i);
    expect(setCookie).toMatch(/SameSite=Lax/i);
    expect(setCookie).toMatch(/Path=\//);
    expect(setCookie).not.toMatch(/Domain=/i);
    expect(setCookie).toMatch(/Max-Age=2592000/);
    expect(await verifySession(config.sessionSecret, cookieValue(res), NOW)).not.toBeNull();
  });

  it('rejects a wrong password with a generic message', async () => {
    const res = await login({ password: 'nope', code: await totpAt(SECRET_B32, NOW) });
    expect(res.status).toBe(401);
    expect(res.headers.get('set-cookie')).toBeNull();
    expect(await res.json()).toEqual({ message: 'Senha ou código inválido' });
  });

  it('rejects a wrong code with the same generic message', async () => {
    const res = await login({ password: 'hunter2-but-longer', code: '000001' });
    expect(res.status).toBe(401);
    expect(await res.json()).toEqual({ message: 'Senha ou código inválido' });
  });

  it('refuses to reuse a code that already logged in', async () => {
    const code = await totpAt(SECRET_B32, NOW);
    expect((await login({ password: 'hunter2-but-longer', code })).status).toBe(200);
    expect((await login({ password: 'hunter2-but-longer', code })).status).toBe(401);
  });

  it('blocks after repeated failures, even with the right credentials', async () => {
    for (let i = 0; i < 3; i++) await login({ password: 'nope', code: '123456' });
    const res = await login({ password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, NOW) });
    expect(res.status).toBe(429);
  });

  it('rate-limits per client IP', async () => {
    for (let i = 0; i < 3; i++) await login({ password: 'nope', code: '123456' });
    clock = NOW + 31_000; // fresh TOTP step
    const res = await login(
      { password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, clock) },
      { 'x-real-ip': '5.6.7.8' },
    );
    expect(res.status).toBe(200);
  });

  it.each([
    ['same-site', { 'sec-fetch-site': 'same-site' }],
    ['cross-site', { 'sec-fetch-site': 'cross-site' }],
  ])('rejects a %s login (login CSRF)', async (_label, headers) => {
    const res = await login({ password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, NOW) }, headers);
    expect(res.status).toBe(403);
  });

  it.each([{}, { password: 'x' }, { code: '123456' }, { password: 1, code: 2 }, 'not json'])(
    'rejects malformed body %j',
    async (body) => {
      expect((await login(body)).status).toBe(400);
    },
  );

  it('refuses to log in when auth is not configured', async () => {
    handler = createLoginHandler({ config: () => ({ ...config, enabled: false }), limiter, now: () => clock });
    expect((await login({ password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, NOW) })).status).toBe(503);
  });

  it('uses a plain cookie name when the cookie is not secure (local http dev)', async () => {
    handler = createLoginHandler({ config: () => ({ ...config, secureCookie: false }), limiter, now: () => clock });
    const res = await login({ password: 'hunter2-but-longer', code: await totpAt(SECRET_B32, NOW) });
    const setCookie = res.headers.get('set-cookie') ?? '';
    expect(setCookie).toMatch(/^session=/);
    expect(setCookie).not.toMatch(/Secure/i);
  });
});
