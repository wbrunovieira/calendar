import { describe, it, expect } from 'vitest';
import { ApiError, AuthRequiredError, AUTH_BASE } from './api';
import { sessionStateFromError, buildLogoutUrl, ACTIVE_SESSION } from './session';

describe('sessionStateFromError', () => {
  it('flags an authentication challenge and keeps its login URL', () => {
    const err = new AuthRequiredError(401, 'Unauthorized', 'https://auth.example/?rd=x');

    expect(sessionStateFromError(err)).toEqual({
      authRequired: true,
      loginUrl: 'https://auth.example/?rd=x',
    });
  });

  it('falls back to a built login URL when the error carries none', () => {
    const err = new AuthRequiredError(401, 'Unauthorized', '');

    const state = sessionStateFromError(err);
    expect(state.authRequired).toBe(true);
    expect(state.loginUrl.startsWith(AUTH_BASE)).toBe(true);
  });

  // Offering "log in again" for a 500 sends the user on a round trip that ends
  // on the same broken screen. Only a session problem gets the session message.
  it('does not treat a server fault as a session problem', () => {
    expect(sessionStateFromError(new ApiError(500, 'Server Error'))).toEqual(ACTIVE_SESSION);
  });

  it('does not treat a 404 as a session problem', () => {
    expect(sessionStateFromError(new ApiError(404, 'Not Found'))).toEqual(ACTIVE_SESSION);
  });

  it('does not treat a network failure as a session problem', () => {
    expect(sessionStateFromError(new TypeError('Failed to fetch'))).toEqual(ACTIVE_SESSION);
  });

  it('tolerates a thrown non-error', () => {
    expect(sessionStateFromError('boom')).toEqual(ACTIVE_SESSION);
    expect(sessionStateFromError(undefined)).toEqual(ACTIVE_SESSION);
  });
});

describe('buildLogoutUrl', () => {
  it('points at the gateway logout and carries the return address', () => {
    const url = buildLogoutUrl('https://finances.wbdigitalsolutions.com/contas');

    expect(url.startsWith(`${AUTH_BASE}/logout`)).toBe(true);
    expect(url).toContain(encodeURIComponent('https://finances.wbdigitalsolutions.com/contas'));
  });

  it('still produces a usable URL with no return address', () => {
    expect(buildLogoutUrl(undefined)).toBe(`${AUTH_BASE}/logout`);
  });
});
