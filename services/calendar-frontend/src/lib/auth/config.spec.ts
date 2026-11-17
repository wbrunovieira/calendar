// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { authConfigFromEnv, isLoginConfigured } from './config';

describe('authConfigFromEnv', () => {
  it('is off and not required by default', () => {
    const config = authConfigFromEnv({});
    expect(config.enabled).toBe(false);
    expect(config.required).toBe(false);
    expect(config.secureCookie).toBe(true);
  });

  it.each(['true', ' TRUE ', 'True'])('reads AUTH_REQUIRED=%j as required', (value) => {
    expect(authConfigFromEnv({ AUTH_REQUIRED: value }).required).toBe(true);
  });

  it.each(['false', '', '1', 'yes'])('reads AUTH_REQUIRED=%j as not required', (value) => {
    expect(authConfigFromEnv({ AUTH_REQUIRED: value }).required).toBe(false);
  });

  it('needs every piece to call the login configured', () => {
    const full = authConfigFromEnv({
      AUTH_SESSION_SECRET: 's'.repeat(32),
      AUTH_PASSWORD_HASH: 'scrypt:x',
      AUTH_TOTP_SECRET: 'JBSWY3DPEHPK3PXP',
    });
    expect(isLoginConfigured(full)).toBe(true);
    expect(isLoginConfigured({ ...full, sessionSecret: 's'.repeat(31) })).toBe(false);
    expect(isLoginConfigured({ ...full, passwordHash: '' })).toBe(false);
    expect(isLoginConfigured({ ...full, totpSecret: '' })).toBe(false);
  });

  it('falls back to 30 days for an invalid AUTH_SESSION_DAYS', () => {
    expect(authConfigFromEnv({ AUTH_SESSION_DAYS: 'abc' }).sessionTtlMs).toBe(30 * 24 * 3600 * 1000);
    expect(authConfigFromEnv({ AUTH_SESSION_DAYS: '7' }).sessionTtlMs).toBe(7 * 24 * 3600 * 1000);
  });
});
