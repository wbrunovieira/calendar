import { AuthRequiredError, buildLoginUrl, setRequestObservers, AUTH_BASE } from './api';

export interface SessionState {
  /** The gateway wants a login. Screens must say so instead of showing data. */
  authRequired: boolean;
  /** Where to send the user to get a session back. */
  loginUrl: string;
}

export const ACTIVE_SESSION: SessionState = { authRequired: false, loginUrl: '' };

/**
 * Classifies a failed request.
 *
 * Only an authentication challenge becomes `authRequired`. A server fault, a
 * dropped connection or a bad payload are all real failures, but none of them
 * are fixed by logging in, and offering a login button for them sends the user
 * on a pointless round trip that ends on the same broken screen.
 *
 * A `TypeError` from fetch is deliberately NOT treated as a session problem. It
 * is what an offline laptop produces, and it is indistinguishable from one. The
 * gateway was making it the only possible outcome by answering the API with a
 * cross-origin redirect; that is fixed where it belongs, in nginx, so that a
 * lapsed session arrives here as the 401 it always was.
 */
export function sessionStateFromError(error: unknown): SessionState {
  if (error instanceof AuthRequiredError) {
    return { authRequired: true, loginUrl: error.loginUrl || buildLoginUrl() };
  }
  return ACTIVE_SESSION;
}

/* ─── Application-wide session state ──────────────────────────────────────
 *
 * The session belongs to the application, not to whichever request happened to
 * notice it had ended. Deriving it from one component's mount effect means a
 * session that lapses while the tab is open is never detected: the effect has
 * already run, and every later request fails into a local catch that renders an
 * empty table. Keeping it here lets any call — a profile load, a transaction
 * being saved, an invoice being paid — report the fact once, to everyone.
 */

let current: SessionState = ACTIVE_SESSION;
const listeners = new Set<(state: SessionState) => void>();

export function getSessionState(): SessionState {
  return current;
}

export function subscribeSession(listener: (state: SessionState) => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/**
 * Records the outcome of a failed request and tells subscribers when it changed
 * the session. Returns whether the failure was an authentication challenge, so
 * a caller can skip its own error toast for one.
 */
export function reportRequestFailure(error: unknown): boolean {
  const next = sessionStateFromError(error);
  if (next.authRequired === current.authRequired) return next.authRequired;
  current = next;
  listeners.forEach((l) => l(current));
  return next.authRequired;
}

/** Called once a login succeeds, or when a request proves the session works. */
export function reportRequestSuccess(): void {
  if (!current.authRequired) return;
  current = ACTIVE_SESSION;
  listeners.forEach((l) => l(current));
}

/** Test seam: drops subscribers and returns the module to a signed-in state. */
export function resetSessionState(): void {
  current = ACTIVE_SESSION;
  listeners.clear();
}

/** Logout URL that returns the user to the page they were on. */
export function buildLogoutUrl(returnTo?: string): string {
  const target =
    returnTo ?? (typeof window !== 'undefined' ? window.location.href : undefined);
  return target ? `${AUTH_BASE}/logout?rd=${encodeURIComponent(target)}` : `${AUTH_BASE}/logout`;
}

// Registered on import so every apiFetch reports into the shared state without
// any call site having to remember to. ProfileContext imports this module, so
// the registration happens before the first request goes out.
setRequestObservers({ onFailure: reportRequestFailure, onSuccess: reportRequestSuccess });
