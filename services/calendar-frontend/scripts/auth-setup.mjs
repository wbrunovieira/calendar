#!/usr/bin/env node
/**
 * Generates the login secrets for the frontend: a scrypt password hash, a TOTP secret (plus the
 * otpauth:// URI to add to an authenticator app) and a session signing secret.
 *
 *   npm run auth:setup
 *
 * The hash format must match src/lib/auth/password.ts (a test checks they agree).
 */
import { randomBytes, scrypt as scryptCb } from 'node:crypto';
import { promisify } from 'node:util';
import { createInterface } from 'node:readline';
import { pathToFileURL } from 'node:url';

const scrypt = promisify(scryptCb);
const ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';

export async function hashPassword(password) {
  const N = 2 ** 15, r = 8, p = 1;
  const salt = randomBytes(16);
  const hash = await scrypt(password, salt, 32, { N, r, p, maxmem: 64 * 1024 * 1024 });
  return ['scrypt', N, r, p, salt.toString('base64'), hash.toString('base64')].join(':');
}

export function base32(bytes) {
  let bits = 0, value = 0, out = '';
  for (const byte of bytes) {
    value = (value << 8) | byte;
    bits += 8;
    while (bits >= 5) {
      out += ALPHABET[(value >>> (bits - 5)) & 31];
      bits -= 5;
    }
  }
  if (bits > 0) out += ALPHABET[(value << (5 - bits)) & 31];
  return out;
}

function askHidden(question) {
  return new Promise((resolve) => {
    const rl = createInterface({ input: process.stdin, output: process.stdout, terminal: true });
    rl._writeToOutput = (s) => { if (s.includes(question)) rl.output.write(s); };
    rl.question(question, (answer) => {
      rl.close();
      process.stdout.write('\n');
      resolve(answer);
    });
  });
}

async function main() {
  const password = await askHidden('Password (min 12 chars): ');
  if (password.length < 12) {
    console.error('Too short.');
    process.exit(1);
  }
  if ((await askHidden('Repeat: ')) !== password) {
    console.error('Passwords differ.');
    process.exit(1);
  }

  const totpSecret = base32(randomBytes(20));
  const label = encodeURIComponent(process.argv[2] ?? 'WB Calendar');
  console.log('\nAdd to the server .env:\n');
  console.log(`AUTH_PASSWORD_HASH=${await hashPassword(password)}`);
  console.log(`AUTH_TOTP_SECRET=${totpSecret}`);
  console.log(`AUTH_SESSION_SECRET=${randomBytes(32).toString('hex')}`);
  console.log('\nAuthenticator app (paste the URI or type the secret):');
  console.log(`otpauth://totp/${label}?secret=${totpSecret}&issuer=WB%20Digital&algorithm=SHA1&digits=6&period=30`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();
