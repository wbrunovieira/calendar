import { AuthRequiredError, buildLoginUrl, AUTH_BASE } from './api';

export interface SessionState {
  /** The gateway wants a login. Screens must say so instead of showing data. */
  authRequired: boolean;
  /** Where to send the user to get a session back. */
  loginUrl: string;
}

export const ACTIVE_SESSION: SessionState = { authRequired: false, loginUrl: '' };

/**
 * Classifies a failed data load.
 *
 * Only an authentication challenge becomes `authRequired`. A server fault, a
 * dropped connection or a bad payload are all real failures, but none of them
 * are fixed by logging in, and offering a login button for them sends the user
 * on a pointless round trip that ends on the same broken screen.
 */
export function sessionStateFromError(error: unknown): SessionState {
  if (error instanceof AuthRequiredError) {
    return { authRequired: true, loginUrl: error.loginUrl || buildLoginUrl() };
  }
  return ACTIVE_SESSION;
}

/** Logout URL that returns the user to the page they were on. */
export function buildLogoutUrl(returnTo?: string): string {
  const target =
    returnTo ?? (typeof window !== 'undefined' ? window.location.href : undefined);
  return target ? `${AUTH_BASE}/logout?rd=${encodeURIComponent(target)}` : `${AUTH_BASE}/logout`;
}
