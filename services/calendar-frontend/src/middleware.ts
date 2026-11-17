import { NextResponse, type NextRequest } from 'next/server';
import { authorize } from '@/lib/auth/authorize';
import { authConfigFromEnv, sessionCookieName } from '@/lib/auth/config';

// Every page and API route needs a session, except /login and the login endpoint (see authorize).
export async function middleware(req: NextRequest) {
  const config = authConfigFromEnv();
  const decision = await authorize({
    pathname: req.nextUrl.pathname,
    cookie: req.cookies.get(sessionCookieName(config))?.value,
    now: Date.now(),
    config,
  });

  if (decision.action === 'allow') return NextResponse.next();
  if (decision.action === 'unauthorized') {
    return NextResponse.json({ message: 'Não autenticado' }, { status: 401 });
  }
  return NextResponse.redirect(new URL(decision.location, req.url));
}

export const config = {
  // Node runtime: env vars are read when the request comes in, not inlined at build time.
  runtime: 'nodejs',
  matcher: ['/((?!_next/static|_next/image|favicon.ico|icon.svg|apple-icon.svg).*)'],
};
