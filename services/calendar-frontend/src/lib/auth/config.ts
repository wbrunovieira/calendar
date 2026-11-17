export interface AuthConfig {
  /** False when no session secret is set: the app is left open (Authelia still guards it). */
  enabled: boolean;
  sessionSecret: string;
  /** scrypt hash from `npm run auth:setup`. */
  passwordHash: string;
  /** Base32 TOTP secret from `npm run auth:setup`. */
  totpSecret: string;
  sessionTtlMs: number;
  /** Secure, `__Host-` cookie. Off only for plain-http local dev. */
  secureCookie: boolean;
  /** AUTH_REQUIRED=true: a missing login config closes the app (503) instead of opening it. */
  required: boolean;
}

const DAY_MS = 24 * 60 * 60 * 1000;

export function authConfigFromEnv(env: Record<string, string | undefined> = process.env): AuthConfig {
  const sessionSecret = (env.AUTH_SESSION_SECRET ?? '').trim();
  const days = Number(env.AUTH_SESSION_DAYS ?? 30);
  return {
    enabled: sessionSecret !== '',
    sessionSecret,
    passwordHash: (env.AUTH_PASSWORD_HASH ?? '').trim(),
    totpSecret: (env.AUTH_TOTP_SECRET ?? '').trim(),
    sessionTtlMs: (Number.isFinite(days) && days > 0 ? days : 30) * DAY_MS,
    secureCookie: env.AUTH_INSECURE_COOKIE !== 'true',
    required: (env.AUTH_REQUIRED ?? '').trim().toLowerCase() === 'true',
  };
}

export const MIN_SESSION_SECRET_LENGTH = 32;

/** Everything a login needs. A half-filled config keeps the app locked (middleware on) but says why. */
export function isLoginConfigured(config: AuthConfig): boolean {
  return (
    config.enabled &&
    config.sessionSecret.length >= MIN_SESSION_SECRET_LENGTH &&
    config.passwordHash !== '' &&
    config.totpSecret !== ''
  );
}

let warned = false;

/** Logs once per process when the login is off or misconfigured, so it doesn't go unnoticed. */
export function warnAboutAuthConfig(config: AuthConfig): void {
  if (warned) return;
  warned = true;
  if (!config.enabled && config.required) {
    console.error('[auth] AUTH_REQUIRED=true but AUTH_SESSION_SECRET is not set: every request gets 503');
  } else if (!config.enabled) {
    console.warn('[auth] AUTH_SESSION_SECRET not set: login is OFF and the app is open');
  } else if (!isLoginConfigured(config)) {
    console.error(
      '[auth] login misconfigured (need AUTH_SESSION_SECRET >= 32 chars, AUTH_PASSWORD_HASH, AUTH_TOTP_SECRET): nobody can log in',
    );
  }
}

export function sessionCookieName(config: Pick<AuthConfig, 'secureCookie'>): string {
  // `__Host-` makes the browser enforce Secure, Path=/ and no Domain: the cookie stays on this
  // exact host and never rides along to sibling subdomains.
  return config.secureCookie ? '__Host-session' : 'session';
}
