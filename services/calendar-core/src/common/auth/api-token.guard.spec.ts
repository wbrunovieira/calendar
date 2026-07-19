import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { ExecutionContext, UnauthorizedException } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { ApiTokenGuard } from './api-token.guard';
import { IS_PUBLIC_KEY } from './public.decorator';

function makeContext(headers: Record<string, string> = {}, url = '/events'): ExecutionContext {
  // Header lookup is case-insensitive in Node/Express; normalize keys to lower-case.
  const lower = Object.fromEntries(Object.entries(headers).map(([k, v]) => [k.toLowerCase(), v]));
  return {
    switchToHttp: () => ({ getRequest: () => ({ headers: lower, url }) }),
    getHandler: () => () => undefined,
    getClass: () => class {},
  } as unknown as ExecutionContext;
}

describe('ApiTokenGuard', () => {
  let reflector: Reflector;
  let guard: ApiTokenGuard;

  const TOKEN = 'super-secret-token-value';

  beforeEach(() => {
    reflector = new Reflector();
    vi.spyOn(reflector, 'getAllAndOverride').mockReturnValue(false); // not @Public by default
    guard = new ApiTokenGuard(reflector);
    delete process.env.API_TOKEN;
  });

  afterEach(() => {
    delete process.env.API_TOKEN;
    vi.restoreAllMocks();
  });

  describe('when API_TOKEN is not configured (open mode / local dev / safe rollout)', () => {
    it('allows any request, with or without a token', () => {
      expect(guard.canActivate(makeContext())).toBe(true);
      expect(guard.canActivate(makeContext({ Authorization: 'Bearer whatever' }))).toBe(true);
    });

    it('treats a blank/whitespace API_TOKEN as not configured', () => {
      process.env.API_TOKEN = '   ';
      expect(guard.canActivate(makeContext())).toBe(true);
    });
  });

  describe('when API_TOKEN is configured', () => {
    beforeEach(() => {
      process.env.API_TOKEN = TOKEN;
    });

    it('allows a request with the correct Bearer token', () => {
      expect(guard.canActivate(makeContext({ Authorization: `Bearer ${TOKEN}` }))).toBe(true);
    });

    it('allows a request with the correct X-API-Key header', () => {
      expect(guard.canActivate(makeContext({ 'X-API-Key': TOKEN }))).toBe(true);
    });

    it('rejects a request with no token', () => {
      expect(() => guard.canActivate(makeContext())).toThrow(UnauthorizedException);
    });

    it('rejects a request with the wrong token', () => {
      expect(() => guard.canActivate(makeContext({ Authorization: 'Bearer nope' }))).toThrow(
        UnauthorizedException,
      );
    });

    it('rejects a malformed Authorization header (missing Bearer scheme)', () => {
      expect(() => guard.canActivate(makeContext({ Authorization: TOKEN }))).toThrow(
        UnauthorizedException,
      );
    });

    it('rejects a token that is a prefix of the real one (no partial match)', () => {
      expect(() =>
        guard.canActivate(makeContext({ Authorization: `Bearer ${TOKEN.slice(0, -1)}` })),
      ).toThrow(UnauthorizedException);
    });

    it('lets a @Public() route through even without a token', () => {
      vi.spyOn(reflector, 'getAllAndOverride').mockImplementation((key) =>
        key === IS_PUBLIC_KEY ? true : false,
      );
      expect(guard.canActivate(makeContext())).toBe(true);
    });

    it('leaves the Swagger docs open without a token (UI, assets and spec)', () => {
      // Swagger routes are not Nest controllers, so @Public() can't attach — the guard exempts
      // them by path so the docs stay readable by link even when the token is enforced.
      expect(guard.canActivate(makeContext({}, '/docs'))).toBe(true);
      expect(guard.canActivate(makeContext({}, '/docs-json'))).toBe(true);
      expect(guard.canActivate(makeContext({}, '/docs/swagger-ui.css'))).toBe(true);
    });
  });
});
