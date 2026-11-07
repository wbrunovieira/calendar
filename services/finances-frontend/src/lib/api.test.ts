import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { apiFetch, ApiError, AuthRequiredError, buildLoginUrl, AUTH_BASE } from './api';

const API_ORIGIN = 'https://calendar-finances.wbdigitalsolutions.com';
const API_URL = `${API_ORIGIN}/api/v1`;

/**
 * Builds the Response shape fetch hands back. `url` matters as much as `status`
 * here: a cross-origin redirect to the login page arrives as a perfectly
 * successful response whose only tell is where it came from.
 */
function response(init: {
  status?: number;
  url?: string;
  redirected?: boolean;
  contentType?: string | null;
  body?: unknown;
}): Response {
  const status = init.status ?? 200;
  const contentType = init.contentType === undefined ? 'application/json' : init.contentType;
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: 'x',
    url: init.url ?? `${API_URL}/profiles`,
    redirected: init.redirected ?? false,
    headers: { get: (k: string) => (k.toLowerCase() === 'content-type' ? contentType : null) },
    json: async () => init.body,
    text: async () => (typeof init.body === 'string' ? init.body : JSON.stringify(init.body ?? '')),
  } as unknown as Response;
}

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.stubEnv('NEXT_PUBLIC_API_URL', API_URL);
  fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe('apiFetch', () => {
  it('returns the parsed body on a normal JSON response', async () => {
    fetchMock.mockResolvedValue(response({ body: { data: [{ id: '1' }] } }));

    await expect(apiFetch('/profiles')).resolves.toEqual({ data: [{ id: '1' }] });
  });

  it('sends the session cookie, since the API sits on a sibling subdomain', async () => {
    fetchMock.mockResolvedValue(response({ body: { data: [] } }));

    await apiFetch('/profiles');

    expect(fetchMock.mock.calls[0][1]).toMatchObject({ credentials: 'include' });
  });

  // The regression this whole change exists to prevent: an account that genuinely
  // has no profiles must keep reaching the caller as an empty list, never as a
  // session problem. Treating "empty" as "logged out" would swap one wrong message
  // for another.
  it('treats an empty list as data, not as a session problem', async () => {
    fetchMock.mockResolvedValue(response({ body: { data: [] } }));

    await expect(apiFetch('/profiles')).resolves.toEqual({ data: [] });
  });

  it('raises AuthRequiredError on 401', async () => {
    fetchMock.mockResolvedValue(response({ status: 401, body: 'unauthorized' }));

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  it('raises AuthRequiredError on 403', async () => {
    fetchMock.mockResolvedValue(response({ status: 403, body: 'forbidden' }));

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  // What actually happened in production on 07/11/2026. Authelia answers the
  // expired session with a 302 to the login page; fetch follows it and returns
  // HTTP 200 carrying HTML. Both `ok` and the status say success, so the old
  // client returned undefined and every screen rendered its empty state.
  it('raises AuthRequiredError when redirected to the auth host with a 200', async () => {
    fetchMock.mockResolvedValue(
      response({
        status: 200,
        redirected: true,
        url: 'https://auth.wbdigitalsolutions.com/?rd=' + encodeURIComponent(`${API_URL}/profiles`),
        contentType: 'text/html; charset=utf-8',
        body: '<html>login</html>',
      }),
    );

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  // The cross-origin hop must stand on its own. The production response also
  // carried HTML, so a test that asserts both leaves the origin check unguarded:
  // deleting it kept the suite green until this case existed.
  it('raises AuthRequiredError on a cross-origin hop even when the body is JSON', async () => {
    fetchMock.mockResolvedValue(
      response({
        status: 200,
        redirected: true,
        url: 'https://auth.wbdigitalsolutions.com/api/verify',
        contentType: 'application/json',
        body: { authenticated: false },
      }),
    );

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  // Same failure without the redirect flag: some gateways rewrite rather than
  // redirect, so HTML where JSON belongs is itself the signal.
  it('raises AuthRequiredError when a 200 arrives as HTML instead of JSON', async () => {
    fetchMock.mockResolvedValue(
      response({ status: 200, contentType: 'text/html', body: '<html>login</html>' }),
    );

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  // A redirect that stays on the API's own origin is ordinary routing.
  it('accepts a same-origin redirect as a normal response', async () => {
    fetchMock.mockResolvedValue(
      response({ redirected: true, url: `${API_URL}/profiles/`, body: { data: [] } }),
    );

    await expect(apiFetch('/profiles')).resolves.toEqual({ data: [] });
  });

  it('still raises a plain ApiError when the server itself fails', async () => {
    fetchMock.mockResolvedValue(response({ status: 500, body: 'boom' }));

    const err = (await apiFetch('/profiles').catch((e: unknown) => e)) as ApiError;
    expect(err).toBeInstanceOf(ApiError);
    expect(err).not.toBeInstanceOf(AuthRequiredError);
    expect(err.status).toBe(500);
  });

  it('carries a login URL that returns to the page the user was on', async () => {
    fetchMock.mockResolvedValue(response({ status: 401, body: '' }));

    const err = (await apiFetch('/profiles').catch((e: unknown) => e)) as AuthRequiredError;
    expect(err.loginUrl).toContain(AUTH_BASE);
  });

  // M9: the only exact-'text/html' case was same-origin, and the realistic
  // charset-bearing one was already caught by the origin rule — so tolerating a
  // charset was never actually asserted. Production sends exactly this.
  it('raises AuthRequiredError on same-origin HTML that carries a charset', async () => {
    fetchMock.mockResolvedValue(
      response({ status: 200, contentType: 'text/html; charset=utf-8', body: '<html/>' }),
    );

    await expect(apiFetch('/profiles')).rejects.toBeInstanceOf(AuthRequiredError);
  });

  // M10: an HTML *error* page is a fault, not a login prompt. Without the res.ok
  // guard a 502 from nginx or Cloudflare would tell the user to sign in again.
  it('treats a 502 HTML error page as a failure, not a login prompt', async () => {
    fetchMock.mockResolvedValue(
      response({ status: 502, contentType: 'text/html', body: '<html>bad gateway</html>' }),
    );

    const err = (await apiFetch('/profiles').catch((e: unknown) => e)) as ApiError;
    expect(err).toBeInstanceOf(ApiError);
    expect(err).not.toBeInstanceOf(AuthRequiredError);
    expect(err.status).toBe(502);
  });

  // M5: an unparseable URL must fail safe towards "this is fine". Flipping the
  // catch would turn every such response into a spurious logout.
  it('does not call a response with an unparseable URL a session problem', async () => {
    fetchMock.mockResolvedValue(response({ url: 'not a url', body: { data: [] } }));

    await expect(apiFetch('/profiles')).resolves.toEqual({ data: [] });
  });

  // A 204 has no body by design and must not be mistaken for a login page.
  it('accepts a 204 with no content type', async () => {
    fetchMock.mockResolvedValue(response({ status: 204, contentType: null, body: undefined }));

    await expect(apiFetch('/transactions/1', { method: 'DELETE' })).resolves.toBeUndefined();
  });
});

describe('AUTH_BASE', () => {
  // M20: every other assertion compares against AUTH_BASE itself, so a wrong
  // default would ship with the suite green. The Dockerfile comment claims this
  // value; this is what holds it to it.
  it('defaults to the production gateway', () => {
    expect(AUTH_BASE).toBe('https://auth.wbdigitalsolutions.com');
  });
});

describe('buildLoginUrl', () => {
  it('points at the auth host and carries the return address', () => {
    const url = buildLoginUrl('https://finances.wbdigitalsolutions.com/contas');

    expect(url.startsWith(AUTH_BASE)).toBe(true);
    expect(url).toContain(encodeURIComponent('https://finances.wbdigitalsolutions.com/contas'));
  });

  // M8: Authelia reads the parameter by name. Asserting only that the encoded
  // address appears somewhere lets a rename through, and a renamed parameter is
  // a login that silently drops the return address.
  it('passes the return address in the parameter Authelia reads', () => {
    expect(buildLoginUrl('https://finances.wbdigitalsolutions.com/')).toBe(
      `${AUTH_BASE}/?rd=${encodeURIComponent('https://finances.wbdigitalsolutions.com/')}`,
    );
  });

  it('still produces a usable URL when there is no return address', () => {
    expect(buildLoginUrl(undefined)).toBe(`${AUTH_BASE}/`);
  });
});
