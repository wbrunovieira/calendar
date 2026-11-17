/**
 * Stateless session token: base64url(JSON payload) + "." + base64url(HMAC-SHA256).
 * Web Crypto only, so the middleware (edge runtime) can verify it.
 */

export interface SessionPayload {
  sid: string;
  iat: number;
  exp: number;
}

const MIN_SECRET_LENGTH = 32;
const encoder = new TextEncoder();

function toBase64Url(bytes: Uint8Array): string {
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function fromBase64Url(text: string): Uint8Array<ArrayBuffer> | null {
  if (!/^[A-Za-z0-9_-]+$/.test(text)) return null;
  try {
    const binary = atob(text.replace(/-/g, '+').replace(/_/g, '/'));
    return Uint8Array.from(binary, (c) => c.charCodeAt(0));
  } catch {
    return null;
  }
}

async function hmacKey(secret: string): Promise<CryptoKey> {
  return crypto.subtle.importKey('raw', encoder.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, [
    'sign',
    'verify',
  ]);
}

export async function createSession(secret: string, opts: { now: number; ttlMs: number }): Promise<string> {
  if (secret.length < MIN_SECRET_LENGTH) throw new Error('Session secret too short');
  const payload: SessionPayload = {
    sid: toBase64Url(crypto.getRandomValues(new Uint8Array(16))),
    iat: opts.now,
    exp: opts.now + opts.ttlMs,
  };
  const body = toBase64Url(encoder.encode(JSON.stringify(payload)));
  const sig = new Uint8Array(await crypto.subtle.sign('HMAC', await hmacKey(secret), encoder.encode(body)));
  return `${body}.${toBase64Url(sig)}`;
}

export async function verifySession(
  secret: string,
  token: string | undefined,
  now: number,
): Promise<SessionPayload | null> {
  if (!token || secret.length < MIN_SECRET_LENGTH) return null;
  const parts = token.split('.');
  if (parts.length !== 2) return null;
  const [body, sigText] = parts;
  const sig = fromBase64Url(sigText);
  if (!body || !sig) return null;

  // crypto.subtle.verify compares in constant time.
  const valid = await crypto.subtle.verify('HMAC', await hmacKey(secret), sig, encoder.encode(body));
  if (!valid) return null;

  const raw = fromBase64Url(body);
  if (!raw) return null;
  try {
    const payload = JSON.parse(new TextDecoder().decode(raw)) as SessionPayload;
    if (typeof payload.exp !== 'number' || payload.exp < now) return null;
    return payload;
  } catch {
    return null;
  }
}
