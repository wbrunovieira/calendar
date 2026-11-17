// @vitest-environment node
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { calendarBackendFromEnv, healthBackendFromEnv, proxyToBackend } from './backend-proxy';

const config = { baseUrl: 'http://calendar-core:3334', token: 'srv-token' };

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  fetchMock = vi.fn(async () => new Response(JSON.stringify({ ok: true }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  }));
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => vi.unstubAllGlobals());

function lastCall(): { url: string; init: RequestInit & { headers: Headers } } {
  const [url, init] = fetchMock.mock.calls.at(-1)!;
  return { url: String(url), init: { ...init, headers: new Headers(init.headers) } };
}

describe('proxyToBackend', () => {
  it('forwards the path and query string to the backend', async () => {
    const req = new Request('https://calendar.example.com/api/backend/events?calendarId=c1&eventType=TODO');
    await proxyToBackend(req, ['events'], config);
    expect(lastCall().url).toBe('http://calendar-core:3334/events?calendarId=c1&eventType=TODO');
  });

  it('adds the server-side token as a Bearer header', async () => {
    await proxyToBackend(new Request('https://x/api/backend/events'), ['events'], config);
    expect(lastCall().init.headers.get('authorization')).toBe('Bearer srv-token');
  });

  it('sends no Authorization header when no token is configured', async () => {
    await proxyToBackend(new Request('https://x/api/backend/events'), ['events'], { ...config, token: '' });
    expect(lastCall().init.headers.has('authorization')).toBe(false);
  });

  it('never forwards the browser cookies or a client-supplied Authorization', async () => {
    const req = new Request('https://x/api/backend/events', {
      headers: { cookie: 'authelia_session=abc', authorization: 'Bearer evil', 'x-api-key': 'evil' },
    });
    await proxyToBackend(req, ['events'], config);
    const headers = lastCall().init.headers;
    expect(headers.has('cookie')).toBe(false);
    expect(headers.has('x-api-key')).toBe(false);
    expect(headers.get('authorization')).toBe('Bearer srv-token');
  });

  it('forwards method, JSON body and content type', async () => {
    const body = JSON.stringify({ title: 'Academia', startTime: '07:00' });
    const req = new Request('https://x/api/backend/events/e1', {
      method: 'PUT',
      body,
      headers: { 'content-type': 'application/json', 'sec-fetch-site': 'same-origin' },
    });
    await proxyToBackend(req, ['events', 'e1'], config);
    const { init } = lastCall();
    expect(init.method).toBe('PUT');
    expect(init.headers.get('content-type')).toBe('application/json');
    expect(new TextDecoder().decode(init.body as ArrayBuffer)).toBe(body);
  });

  it('sends no body on GET', async () => {
    await proxyToBackend(new Request('https://x/api/backend/events'), ['events'], config);
    expect(lastCall().init.body).toBeUndefined();
  });

  it('encodes path segments', async () => {
    await proxyToBackend(new Request('https://x/api/backend/events/a%20b'), ['events', 'a b'], config);
    expect(lastCall().url).toBe('http://calendar-core:3334/events/a%20b');
  });

  it.each([
    ['a?b', 'a%3Fb'],
    ['x#y', 'x%23y'],
    ['\\evil.com', '%5Cevil.com'],
    ['%2e%2e', '%252e%252e'],
    ['@evil.com', '%40evil.com'],
  ])('encodes %j so it cannot change the query, fragment or host', async (segment, encoded) => {
    await proxyToBackend(new Request('https://x/api/backend/x'), ['events', segment], config);
    expect(lastCall().url).toBe(`http://calendar-core:3334/events/${encoded}`);
  });

  it.each([[['..', 'admin']], [['events', '.']], [['events', '']], [['a/b']]])(
    'rejects path traversal and empty segments (%j)',
    async (path) => {
      const res = await proxyToBackend(new Request('https://x/api/backend/x'), path, config);
      expect(res.status).toBe(400);
      expect(fetchMock).not.toHaveBeenCalled();
    },
  );

  it('passes the backend status and body back', async () => {
    fetchMock.mockResolvedValueOnce(new Response('{"message":"nope"}', {
      status: 404,
      headers: { 'content-type': 'application/json' },
    }));
    const res = await proxyToBackend(new Request('https://x/api/backend/events/zz'), ['events', 'zz'], config);
    expect(res.status).toBe(404);
    expect(res.headers.get('content-type')).toBe('application/json');
    expect(await res.json()).toEqual({ message: 'nope' });
  });

  it('returns 204 with no body', async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
    const req = new Request('https://x/api/backend/events/e1', {
      method: 'DELETE',
      headers: { 'sec-fetch-site': 'same-origin' },
    });
    const res = await proxyToBackend(req, ['events', 'e1'], config);
    expect(res.status).toBe(204);
    expect(await res.text()).toBe('');
  });

  it('answers 502 when the backend is unreachable', async () => {
    fetchMock.mockRejectedValueOnce(new TypeError('fetch failed'));
    const res = await proxyToBackend(new Request('https://x/api/backend/events'), ['events'], config);
    expect(res.status).toBe(502);
  });

  it('does not follow backend redirects', async () => {
    await proxyToBackend(new Request('https://x/api/backend/events'), ['events'], config);
    expect(lastCall().init.redirect).toBe('manual');
  });
});

describe('proxyToBackend write protection (CSRF)', () => {
  const json = (headers: Record<string, string>, method = 'POST') =>
    new Request('https://calendar.example.com/api/backend/events', {
      method,
      body: '{"title":"x"}',
      headers: { 'content-type': 'application/json', host: 'calendar.example.com', ...headers },
    });

  it('allows writes from the same origin (Sec-Fetch-Site)', async () => {
    const res = await proxyToBackend(json({ 'sec-fetch-site': 'same-origin' }), ['events'], config);
    expect(res.status).toBe(200);
  });

  it.each(['same-site', 'cross-site', 'none'])('blocks writes with Sec-Fetch-Site: %s', async (site) => {
    const res = await proxyToBackend(json({ 'sec-fetch-site': site }), ['events'], config);
    expect(res.status).toBe(403);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('falls back to Origin matching the Host when Sec-Fetch-Site is absent', async () => {
    const ok = await proxyToBackend(json({ origin: 'https://calendar.example.com' }), ['events'], config);
    expect(ok.status).toBe(200);
    const sibling = await proxyToBackend(json({ origin: 'https://evil.example.com' }), ['events'], config);
    expect(sibling.status).toBe(403);
  });

  it('blocks writes with neither Sec-Fetch-Site nor Origin', async () => {
    const res = await proxyToBackend(json({}), ['events'], config);
    expect(res.status).toBe(403);
  });

  it.each(['PUT', 'PATCH', 'DELETE'])('applies the same check to %s', async (method) => {
    const res = await proxyToBackend(json({ 'sec-fetch-site': 'same-site' }, method), ['events'], config);
    expect(res.status).toBe(403);
  });

  it('does not restrict reads', async () => {
    const req = new Request('https://x/api/backend/events', { headers: { 'sec-fetch-site': 'cross-site' } });
    expect((await proxyToBackend(req, ['events'], config)).status).toBe(200);
  });

  it.each(['application/x-www-form-urlencoded', 'multipart/form-data; boundary=x', 'text/plain'])(
    'rejects a %s body even from the same origin',
    async (type) => {
      const req = new Request('https://x/api/backend/events', {
        method: 'POST',
        body: 'title=x',
        headers: { 'content-type': type, 'sec-fetch-site': 'same-origin' },
      });
      const res = await proxyToBackend(req, ['events'], config);
      expect(res.status).toBe(415);
      expect(fetchMock).not.toHaveBeenCalled();
    },
  );

  it('accepts a bodyless write without a content type', async () => {
    const req = new Request('https://x/api/backend/google-calendar/sync/c1/pull', {
      method: 'POST',
      headers: { 'sec-fetch-site': 'same-origin' },
    });
    expect((await proxyToBackend(req, ['google-calendar', 'sync', 'c1', 'pull'], config)).status).toBe(200);
  });

  it('rejects bodies over 1 MB', async () => {
    const req = new Request('https://x/api/backend/events', {
      method: 'POST',
      body: JSON.stringify({ blob: 'x'.repeat(1024 * 1024) }),
      headers: { 'content-type': 'application/json', 'sec-fetch-site': 'same-origin' },
    });
    const res = await proxyToBackend(req, ['events'], config);
    expect(res.status).toBe(413);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe('proxyToBackend with a base path (calendar-health /api/v1)', () => {
  const health = { baseUrl: 'http://calendar-health:3336/api/v1/', token: 't' };

  it('keeps the base path', async () => {
    await proxyToBackend(new Request('https://x/api/health-backend/profiles?a=1'), ['profiles'], health);
    expect(lastCall().url).toBe('http://calendar-health:3336/api/v1/profiles?a=1');
  });

  it('cannot climb above the base path', async () => {
    expect((await proxyToBackend(new Request('https://x/y'), ['..', 'admin'], health)).status).toBe(400);
    await proxyToBackend(new Request('https://x/y'), ['%2e%2e', 'admin'], health);
    expect(lastCall().url).toBe('http://calendar-health:3336/api/v1/%252e%252e/admin');
  });
});

describe('backend config from env', () => {
  afterEach(() => vi.unstubAllEnvs());

  it('reads calendar-health URL and trimmed token', () => {
    vi.stubEnv('HEALTH_API_URL', 'http://calendar-health:3336/api/v1');
    vi.stubEnv('HEALTH_API_TOKEN', '  h-tok  ');
    expect(healthBackendFromEnv()).toEqual({ baseUrl: 'http://calendar-health:3336/api/v1', token: 'h-tok' });
  });

  it('reads calendar-core URL and token, defaulting to local dev', () => {
    vi.stubEnv('CALENDAR_API_URL', '');
    vi.stubEnv('CALENDAR_API_TOKEN', '');
    expect(calendarBackendFromEnv()).toEqual({ baseUrl: 'http://localhost:3334', token: '' });
  });
});

