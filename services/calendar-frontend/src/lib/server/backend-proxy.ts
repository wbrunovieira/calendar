/**
 * Server-side proxy from the browser to calendar-core.
 *
 * The browser calls `/api/backend/...` on this app; the route handler forwards the request over
 * the internal network and adds the API token, so the token never ships in the JS bundle.
 * Only an allowlist of request headers is forwarded: the browser's cookies (including the
 * session cookie) and any client-supplied credentials stay on this side.
 *
 * Because the proxy attaches a credential the browser doesn't hold, a write must provably come
 * from this app's own pages, or another site could make the browser post here with its session
 * cookie (CSRF). Sibling subdomains are same-site, so SameSite=Lax alone doesn't stop them. Writes therefore need
 * `Sec-Fetch-Site: same-origin` (or, from browsers that don't send it, an Origin matching the
 * host) and, when they carry a body, a JSON content type, which no plain HTML form can produce.
 */

import { isSameOriginRequest } from './same-origin';

export interface BackendConfig {
  baseUrl: string;
  token: string;
}

const FORWARDED_REQUEST_HEADERS = ['content-type', 'accept', 'accept-language'];
const FORWARDED_RESPONSE_HEADERS = ['content-type', 'content-disposition', 'cache-control'];
const MAX_BODY_BYTES = 1024 * 1024;
const READ_METHODS = new Set(['GET', 'HEAD']);

export function backendConfigFromEnv(): BackendConfig {
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

export async function proxyToBackend(
  req: Request,
  path: string[],
  config: BackendConfig = backendConfigFromEnv(),
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
    if (!isSameOriginRequest(req)) {
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
    return json(502, { message: 'Calendar API unreachable' });
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
