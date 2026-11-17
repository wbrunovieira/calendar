/** True when AUTH_REQUIRED says so; read loosely so a "TRUE" still fails closed. */
export function isAuthRequired(env: Record<string, string | undefined>): boolean {
  return (env.AUTH_REQUIRED ?? '').trim().toLowerCase() === 'true';
}

/**
 * The ApiTokenGuard leaves the API open when API_TOKEN is blank, so it could ship before the token
 * existed. With AUTH_REQUIRED=true (production, once nothing else guards the API) a blank token
 * is a misconfiguration: refuse to start instead of serving an open API.
 */
export function assertApiTokenConfigured(env: Record<string, string | undefined> = process.env): void {
  if (isAuthRequired(env) && !(env.API_TOKEN ?? '').trim()) {
    throw new Error('AUTH_REQUIRED=true but API_TOKEN is empty: refusing to start with an open API');
  }
}
