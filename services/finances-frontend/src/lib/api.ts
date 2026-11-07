/**
 * Read lazily rather than through the module-level constant so the value is the
 * one in effect when the call is made. Next.js still inlines the literal at build
 * time, so production behaviour is unchanged.
 */
export function apiBase(): string {
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3335/api/v1';
}

export const API_BASE = apiBase();

/**
 * Where the Authelia gateway lives. The default is the production host so an
 * expired session is handled correctly even if the build argument is ever
 * forgotten — a NEXT_PUBLIC_* variable that fails to reach the build is silent,
 * and this path is exactly where silence hurts most.
 */
export const AUTH_BASE =
  process.env.NEXT_PUBLIC_AUTH_URL || 'https://auth.wbdigitalsolutions.com';

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    message?: string,
  ) {
    super(message || `API error: ${status} ${statusText}`);
    this.name = 'ApiError';
  }
}

/**
 * The session is gone and the gateway wants a login, which is a different fact
 * from "the server refused this request". Callers that cannot tell the two apart
 * end up reporting an empty account to someone who is merely logged out.
 */
export class AuthRequiredError extends ApiError {
  constructor(
    status: number,
    statusText: string,
    public loginUrl: string,
  ) {
    super(status, statusText, 'Sessão expirada');
    this.name = 'AuthRequiredError';
  }
}

/** Login URL that sends the user back where they were once authenticated. */
export function buildLoginUrl(returnTo?: string): string {
  const target =
    returnTo ?? (typeof window !== 'undefined' ? window.location.href : undefined);
  return target ? `${AUTH_BASE}/?rd=${encodeURIComponent(target)}` : `${AUTH_BASE}/`;
}

function sameOrigin(a: string, b: string): boolean {
  try {
    return new URL(a).origin === new URL(b).origin;
  } catch {
    // A relative or unparseable URL never indicates a hop to the auth host.
    return true;
  }
}

/**
 * Decides whether a response is the gateway asking for a login.
 *
 * The status code alone is not enough. Authelia answers an expired session with
 * a 302 to its login page; fetch follows it and hands back HTTP 200 carrying
 * HTML, so `ok` is true and nothing in the status says anything is wrong. The
 * two reliable tells are the response having come from another origin, and JSON
 * having been replaced by a web page.
 */
function isAuthChallenge(res: Response, requestedUrl: string): boolean {
  if (res.status === 401 || res.status === 403) return true;
  if (!sameOrigin(res.url || requestedUrl, requestedUrl)) return true;
  if (res.ok && (res.headers.get('content-type') || '').includes('text/html')) return true;
  return false;
}

export async function apiFetch<T = unknown>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = path.startsWith('http') ? path : `${apiBase()}${path}`;
  const res = await fetch(url, {
    ...options,
    // Carry the Authelia session cookie. The API lives on a sibling subdomain
    // and the session cookie is issued for wbdigitalsolutions.com, so without
    // this the browser sends nothing and the gateway cannot recognise us —
    // which is why the API domain had to be left open to the internet.
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (isAuthChallenge(res, url)) {
    throw new AuthRequiredError(res.status, res.statusText, buildLoginUrl());
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new ApiError(res.status, res.statusText, text || undefined);
  }

  const contentType = res.headers.get('content-type');
  if (contentType?.includes('application/json')) {
    return res.json();
  }
  return undefined as T;
}

export const api = {
  get: <T = unknown>(path: string) => apiFetch<T>(path),

  post: <T = unknown>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: JSON.stringify(body) }),

  put: <T = unknown>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'PUT', body: JSON.stringify(body) }),

  patch: <T = unknown>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),

  delete: <T = unknown>(path: string) =>
    apiFetch<T>(path, { method: 'DELETE' }),
};
