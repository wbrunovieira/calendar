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
  };
}

export function sessionCookieName(config: Pick<AuthConfig, 'secureCookie'>): string {
  // `__Host-` makes the browser enforce Secure, Path=/ and no Domain: the cookie stays on this
  // exact host and never rides along to sibling subdomains.
  return config.secureCookie ? '__Host-session' : 'session';
}
