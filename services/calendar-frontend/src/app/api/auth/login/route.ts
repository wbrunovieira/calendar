import { authConfigFromEnv } from '@/lib/auth/config';
import { createLoginHandler } from '@/lib/auth/login';
import { LoginRateLimiter } from '@/lib/auth/rate-limit';

export const dynamic = 'force-dynamic';

// 5 failures per IP and 20 overall per 15 minutes.
const limiter = new LoginRateLimiter({ maxFailures: 5, windowMs: 15 * 60_000, maxGlobalFailures: 20 });

export const POST = createLoginHandler({ config: () => authConfigFromEnv(), limiter, now: Date.now });
