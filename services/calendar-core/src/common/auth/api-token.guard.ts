import { CanActivate, ExecutionContext, Injectable, UnauthorizedException } from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { timingSafeEqual } from 'crypto';
import { IS_PUBLIC_KEY } from './public.decorator';

/**
 * Requires a shared API token on every request — EXCEPT:
 *  - routes marked @Public() (health checks), and
 *  - when API_TOKEN is unset/blank, in which case the API stays open.
 *
 * The "open when unconfigured" behaviour is deliberate: it lets the guard ship without breaking
 * local dev or any environment that hasn't set the token yet. Enforcement turns on the moment
 * API_TOKEN is populated.
 *
 * Accepted forms: `Authorization: Bearer <token>` or `X-API-Key: <token>`.
 */
@Injectable()
export class ApiTokenGuard implements CanActivate {
  constructor(private readonly reflector: Reflector) {}

  canActivate(context: ExecutionContext): boolean {
    const isPublic = this.reflector.getAllAndOverride<boolean>(IS_PUBLIC_KEY, [
      context.getHandler(),
      context.getClass(),
    ]);
    if (isPublic) return true;

    const expected = process.env.API_TOKEN?.trim();
    if (!expected) return true; // not configured → API stays open

    const provided = this.extractToken(context.switchToHttp().getRequest());
    if (!provided || !this.safeEqual(provided, expected)) {
      throw new UnauthorizedException('Invalid or missing API token');
    }
    return true;
  }

  private extractToken(req: { headers?: Record<string, unknown> }): string | null {
    const headers = req.headers ?? {};
    const auth = headers['authorization'];
    if (typeof auth === 'string') {
      const [scheme, value] = auth.split(' ');
      if (scheme === 'Bearer' && value) return value;
    }
    const apiKey = headers['x-api-key'];
    if (typeof apiKey === 'string' && apiKey) return apiKey;
    return null;
  }

  /** Constant-time comparison so a wrong token can't be guessed byte-by-byte from timing. */
  private safeEqual(a: string, b: string): boolean {
    const ab = Buffer.from(a);
    const bb = Buffer.from(b);
    if (ab.length !== bb.length) return false;
    return timingSafeEqual(ab, bb);
  }
}
