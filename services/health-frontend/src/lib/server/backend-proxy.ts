/**
 * Server-side proxy from the browser to the backend APIs (calendar-health and calendar-core).
 * Same module as calendar-frontend's src/lib/server/backend-proxy.ts; keep the two in step.
 *
 * The browser calls `/api/health-backend/...` or `/api/calendar-backend/...` on this app; the
 * route handler forwards the request over the internal network and adds that API's token, so no
 * token ships in the JS bundle.
 * Only an allowlist of request headers is forwarded: the browser's cookies (including the
 * session cookie) and any client-supplied credentials stay on this side.
 *
 * Because the proxy attaches a credential the browser doesn't hold, a write must provably come
 * from this app's own pages: the session cookie is scoped to the parent domain, so a page on any
 * sibling subdomain would otherwise be able to post here with it (CSRF). Writes therefore need
 * `Sec-Fetch-Site: same-origin` (or, from browsers that don't send it, an Origin matching the
 * host) and, when they carry a body, a JSON content type, which no plain HTML form can produce.
 */

export interface BackendConfig {
  baseUrl: string;
  token: string;
}

const FORWARDED_REQUEST_HEADERS = ['content-type', 'accept', 'accept-language'];
const FORWARDED_RESPONSE_HEADERS = ['content-type', 'content-disposition', 'cache-control'];
const MAX_BODY_BYTES = 1024 * 1024;
const READ_METHODS = new Set(['GET', 'HEAD']);

export function healthBackendFromEnv(): BackendConfig {
  return {
    baseUrl: process.env.HEALTH_API_URL || 'http://localhost:3336/api/v1',
    token: (process.env.HEALTH_API_TOKEN ?? '').trim(),
  };
}

export function calendarBackendFromEnv(): BackendConfig {
  return {
    baseUrl: process.env.CALENDAR_API_URL || 'http://localhost:3334',
    token: (process.env.CALENDAR_API_TOKEN ?? '').trim(),
  };
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

/** Rejects `.`/`..`/empty segments and slashes inside a segment, so the target stays under baseUrl. */
function isSafePath(path: string[]): boolean {
  return path.length > 0 && path.every((s) => s !== '' && s !== '.' && s !== '..' && !s.includes('/'));
}

function isSameOriginWrite(req: Request): boolean {
  const site = req.headers.get('sec-fetch-site');
  if (site) return site === 'same-origin';

  const origin = req.headers.get('origin');
  if (!origin) return false;
  const host = req.headers.get('host') ?? new URL(req.url).host;
  try {
    return new URL(origin).host === host;
  } catch {
    return false;
  }
}

export async function proxyToBackend(
  req: Request,
  path: string[],
  config: BackendConfig,
): Promise<Response> {
  if (!isSafePath(path)) {
    return json(400, { message: 'Invalid path' });
  }

  const target = new URL(
    `${config.baseUrl.replace(/\/+$/, '')}/${path.map(encodeURIComponent).join('/')}`,
  );
  target.search = new URL(req.url).search;

  const headers = new Headers();
  for (const name of FORWARDED_REQUEST_HEADERS) {
    const value = req.headers.get(name);
    if (value) headers.set(name, value);
  }
  if (config.token) headers.set('authorization', `Bearer ${config.token}`);

  const isWrite = !READ_METHODS.has(req.method);
  let body: ArrayBuffer | undefined;
  if (isWrite) {
    if (!isSameOriginWrite(req)) {
      return json(403, { message: 'Cross-origin write rejected' });
    }
    if (Number(req.headers.get('content-length') ?? 0) > MAX_BODY_BYTES) {
      return json(413, { message: 'Request body too large' });
    }
    body = await req.arrayBuffer();
    if (body.byteLength > MAX_BODY_BYTES) {
      return json(413, { message: 'Request body too large' });
    }
    const contentType = req.headers.get('content-type') ?? '';
    if (body.byteLength > 0 && !contentType.toLowerCase().startsWith('application/json')) {
      return json(415, { message: 'Request body must be JSON' });
    }
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: req.method,
      headers,
      body: body && body.byteLength > 0 ? body : undefined,
      redirect: 'manual',
      cache: 'no-store',
    });
  } catch (error) {
    // Log the cause (host down, DNS, refused) but never the headers: they carry the token.
    const cause = error instanceof Error ? (error.cause as Error | undefined)?.message ?? error.message : String(error);
    console.error(`[backend-proxy] ${req.method} ${target.pathname} failed: ${cause}`);
    return json(502, { message: 'Backend API unreachable' });
  }

  const responseHeaders = new Headers();
  for (const name of FORWARDED_RESPONSE_HEADERS) {
    const value = upstream.headers.get(name);
    if (value) responseHeaders.set(name, value);
  }

  const noBody = upstream.status === 204 || upstream.status === 304 || req.method === 'HEAD';
  return new Response(noBody ? null : await upstream.arrayBuffer(), {
    status: upstream.status,
    headers: responseHeaders,
  });
}
