import { NextResponse, type NextRequest } from 'next/server';
import { authorize } from '@/lib/auth/authorize';
import { authConfigFromEnv, sessionCookieName, warnAboutAuthConfig } from '@/lib/auth/config';

// Every page and API route needs a session, except /login and the login endpoint (see authorize).
export async function middleware(req: NextRequest) {
  const config = authConfigFromEnv();
  warnAboutAuthConfig(config);
  const decision = await authorize({
    pathname: req.nextUrl.pathname,
    search: req.nextUrl.search,
    cookie: req.cookies.get(sessionCookieName(config))?.value,
    now: Date.now(),
    config,
  });

  if (decision.action === 'allow') return NextResponse.next();
  if (decision.action === 'unavailable') {
    return new NextResponse('Login não configurado', { status: 503 });
  }
  if (decision.action === 'unauthorized') {
    return NextResponse.json({ message: 'Não autenticado' }, { status: 401 });
  }
  return NextResponse.redirect(new URL(decision.location, req.url));
}

export const config = {
  // Node runtime: env vars are read when the request comes in, not inlined at build time.
  runtime: 'nodejs',
  // Skips Next's build assets and the exact icon files; anchored so `/icon.svg-x` is still checked.
  matcher: ['/((?!_next/static/|_next/image$|favicon\\.ico$|favicon\\.svg$).*)'],
};
