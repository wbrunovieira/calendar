import { authConfigFromEnv } from '@/lib/auth/config';
import { createLoginHandler } from '@/lib/auth/login';
import { LoginRateLimiter } from '@/lib/auth/rate-limit';

export const dynamic = 'force-dynamic';

// 5 failures per IP and 200 overall per 15 minutes. The global cap is high on purpose: password
// + TOTP already make online guessing hopeless, and a low cap would let anyone lock the owner out.
const limiter = new LoginRateLimiter({ maxFailures: 5, windowMs: 15 * 60_000, maxGlobalFailures: 200 });

export const POST = createLoginHandler({ config: () => authConfigFromEnv(), limiter, now: Date.now });
