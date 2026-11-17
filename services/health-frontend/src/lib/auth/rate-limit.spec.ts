// @vitest-environment node
import { describe, it, expect } from 'vitest';
import { LoginRateLimiter } from './rate-limit';

describe('LoginRateLimiter', () => {
  it('allows up to the limit of failures per key, then blocks until the window passes', () => {
    const rl = new LoginRateLimiter({ maxFailures: 3, windowMs: 60_000 });
    for (let i = 0; i < 3; i++) {
      expect(rl.isBlocked('ip1', 0)).toBe(false);
      rl.recordFailure('ip1', 0);
    }
    expect(rl.isBlocked('ip1', 1_000)).toBe(true);
    expect(rl.isBlocked('ip2', 1_000)).toBe(false);
    expect(rl.isBlocked('ip1', 60_001)).toBe(false);
  });

  it('clears a key on success', () => {
    const rl = new LoginRateLimiter({ maxFailures: 2, windowMs: 60_000 });
    rl.recordFailure('ip1', 0);
    rl.recordFailure('ip1', 0);
    rl.reset('ip1');
    expect(rl.isBlocked('ip1', 0)).toBe(false);
  });

  it('also caps failures across all keys (distributed guessing)', () => {
    const rl = new LoginRateLimiter({ maxFailures: 100, windowMs: 60_000, maxGlobalFailures: 5 });
    for (let i = 0; i < 5; i++) rl.recordFailure(`ip${i}`, 0);
    expect(rl.isBlocked('fresh-ip', 0)).toBe(true);
  });
});
