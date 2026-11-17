import type { AuthConfig } from './config';
import { sessionCookieName } from './config';
import { verifyPassword } from './password';
import type { LoginRateLimiter } from './rate-limit';
import { createSession } from './session';
import { verifyTotp } from './totp';
import { isSameOriginRequest } from '../server/same-origin';

interface Deps {
  config: () => AuthConfig;
  limiter: LoginRateLimiter;
  now: () => number;
}

const INVALID = 'Senha ou código inválido';

function json(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json', ...headers },
  });
}

/** nginx sets X-Real-IP; anything else is a fallback for local runs. */
function clientKey(req: Request): string {
  return (
    req.headers.get('x-real-ip') ??
    req.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ??
    'unknown'
  );
}

export function createLoginHandler(deps: Deps) {
  // Last TOTP step that logged in: a code works once, even inside its 30 s window.
  let lastUsedStep = -1;

  return async function handleLogin(req: Request): Promise<Response> {
    const config = deps.config();
    if (!config.enabled || !config.passwordHash || !config.totpSecret) {
      return json(503, { message: 'Login não configurado' });
    }
    if (!isSameOriginRequest(req)) return json(403, { message: 'Origem não permitida' });

    const key = clientKey(req);
    const now = deps.now();
    if (deps.limiter.isBlocked(key, now)) {
      return json(429, { message: 'Muitas tentativas. Tente de novo em alguns minutos.' });
    }

    let body: unknown;
    try {
      body = await req.json();
    } catch {
      return json(400, { message: 'Requisição inválida' });
    }
    const { password, code } = (body ?? {}) as { password?: unknown; code?: unknown };
    if (typeof password !== 'string' || typeof code !== 'string') {
      return json(400, { message: 'Requisição inválida' });
    }

    // Check both even when the first fails, so timing doesn't reveal which one was wrong.
    const [passwordOk, step] = await Promise.all([
      verifyPassword(password, config.passwordHash),
      verifyTotp(config.totpSecret, code.trim(), now, lastUsedStep),
    ]);
    if (!passwordOk || step === null) {
      deps.limiter.recordFailure(key, now);
      return json(401, { message: INVALID });
    }

    lastUsedStep = step;
    deps.limiter.reset(key);
    const token = await createSession(config.sessionSecret, { now, ttlMs: config.sessionTtlMs });
    const cookie = [
      `${sessionCookieName(config)}=${token}`,
      'Path=/',
      'HttpOnly',
      'SameSite=Lax',
      `Max-Age=${Math.floor(config.sessionTtlMs / 1000)}`,
      ...(config.secureCookie ? ['Secure'] : []),
    ].join('; ');
    return json(200, { ok: true }, { 'set-cookie': cookie });
  };
}
