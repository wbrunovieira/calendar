import { authConfigFromEnv, sessionCookieName } from '@/lib/auth/config';
import { isSameOriginRequest } from '@/lib/server/same-origin';

export const dynamic = 'force-dynamic';

export async function POST(req: Request): Promise<Response> {
  if (!isSameOriginRequest(req)) return new Response(null, { status: 403 });
  const config = authConfigFromEnv();
  const cookie = [
    `${sessionCookieName(config)}=`,
    'Path=/',
    'HttpOnly',
    'SameSite=Lax',
    'Max-Age=0',
    ...(config.secureCookie ? ['Secure'] : []),
  ].join('; ');
  return new Response(null, { status: 204, headers: { 'set-cookie': cookie } });
}
