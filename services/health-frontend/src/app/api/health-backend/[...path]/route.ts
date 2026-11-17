import { healthBackendFromEnv, proxyToBackend } from '@/lib/server/backend-proxy';

// Every call is a live API request; never cache or prerender it.
export const dynamic = 'force-dynamic';

type Context = { params: Promise<{ path: string[] }> };

async function handle(req: Request, { params }: Context): Promise<Response> {
  const { path } = await params;
  return proxyToBackend(req, path, healthBackendFromEnv());
}

export { handle as GET, handle as POST, handle as PUT, handle as PATCH, handle as DELETE };
