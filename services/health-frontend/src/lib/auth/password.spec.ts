// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { hashPassword, verifyPassword } from './password';

describe('password hashing (scrypt)', () => {
  it('verifies the right password', async () => {
    const hash = await hashPassword('correct horse battery staple');
    expect(hash.startsWith('scrypt:')).toBe(true);
    expect(await verifyPassword('correct horse battery staple', hash)).toBe(true);
  });

  it('rejects a wrong password', async () => {
    const hash = await hashPassword('correct horse battery staple');
    expect(await verifyPassword('Correct horse battery staple', hash)).toBe(false);
  });

  it('never puts a $ in the hash (compose would interpolate it)', async () => {
    expect(await hashPassword('x')).not.toContain('$');
  });

  it('salts every hash', async () => {
    expect(await hashPassword('same')).not.toBe(await hashPassword('same'));
  });

  it.each(['', 'plain', 'scrypt:x:y', 'bcrypt$12$abc', 'scrypt$32768$8$1$c2FsdA==$aGFzaA=='])('treats malformed hash %j as no match', async (h) => {
    expect(await verifyPassword('anything', h)).toBe(false);
  });
});
