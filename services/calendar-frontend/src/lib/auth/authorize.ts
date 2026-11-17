import type { AuthConfig } from './config';
import { verifySession } from './session';

export type AuthDecision = { action: 'allow' } | { action: 'redirect'; location: string } | { action: 'unauthorized' };

/** Exact paths reachable without a session. */
const PUBLIC_PATHS = new Set(['/login', '/api/auth/login']);

export async function authorize(input: {
  pathname: string;
  cookie: string | undefined;
  now: number;
  config: AuthConfig;
}): Promise<AuthDecision> {
  const { pathname, cookie, now, config } = input;
  if (!config.enabled || PUBLIC_PATHS.has(pathname)) return { action: 'allow' };
  if (await verifySession(config.sessionSecret, cookie, now)) return { action: 'allow' };
  if (pathname.startsWith('/api/')) return { action: 'unauthorized' };
  return { action: 'redirect', location: `/login?next=${encodeURIComponent(pathname)}` };
}

/** Only same-app relative paths, so `?next=` can't become an open redirect. */
export function safeNextPath(next: string | null | undefined): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/';
  return next;
}
