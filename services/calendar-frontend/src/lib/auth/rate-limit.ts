/**
 * In-memory login throttle. The frontend runs as a single container, so process memory is enough;
 * a restart resets it, which only helps the legitimate user.
 */

interface Options {
  maxFailures: number;
  windowMs: number;
  /** Cap on failures from all clients together, so rotating IPs doesn't buy more guesses. */
  maxGlobalFailures?: number;
}

export class LoginRateLimiter {
  private readonly failures = new Map<string, number[]>();
  private global: number[] = [];

  constructor(private readonly options: Options) {}

  isBlocked(key: string, now: number): boolean {
    const since = now - this.options.windowMs;
    const own = (this.failures.get(key) ?? []).filter((t) => t > since);
    this.failures.set(key, own);
    this.global = this.global.filter((t) => t > since);
    if (own.length >= this.options.maxFailures) return true;
    return this.options.maxGlobalFailures !== undefined && this.global.length >= this.options.maxGlobalFailures;
  }

  recordFailure(key: string, now: number): void {
    this.failures.set(key, [...(this.failures.get(key) ?? []), now]);
    this.global.push(now);
  }

  reset(key: string): void {
    this.failures.delete(key);
  }
}
