/**
 * Password hashing with scrypt (node:crypto, no dependency). Server-only.
 * Format: scrypt:N:r:p:<salt base64>:<hash base64>. Colons, not `$`, because the value lives in a
 * docker compose .env file, where `$` would be read as variable interpolation.
 */
import { randomBytes, scrypt as scryptCb, timingSafeEqual } from 'node:crypto';
import { promisify } from 'node:util';

const scrypt = promisify(scryptCb) as (
  password: string,
  salt: Buffer,
  keylen: number,
  options: { N: number; r: number; p: number; maxmem: number },
) => Promise<Buffer>;

const N = 2 ** 15;
const R = 8;
const P = 1;
const KEY_LENGTH = 32;
const MAXMEM = 64 * 1024 * 1024;

export async function hashPassword(password: string): Promise<string> {
  const salt = randomBytes(16);
  const hash = await scrypt(password, salt, KEY_LENGTH, { N, r: R, p: P, maxmem: MAXMEM });
  return ['scrypt', N, R, P, salt.toString('base64'), hash.toString('base64')].join(':');
}

export async function verifyPassword(password: string, stored: string): Promise<boolean> {
  const parts = stored.split(':');
  if (parts.length !== 6 || parts[0] !== 'scrypt') return false;
  const [, n, r, p, saltText, hashText] = parts;
  const expected = Buffer.from(hashText, 'base64');
  if (expected.length !== KEY_LENGTH) return false;
  try {
    const actual = await scrypt(password, Buffer.from(saltText, 'base64'), KEY_LENGTH, {
      N: Number(n),
      r: Number(r),
      p: Number(p),
      maxmem: MAXMEM,
    });
    return timingSafeEqual(actual, expected);
  } catch {
    return false;
  }
}
