import { GoogleOAuthToken } from '../entities/google-oauth-token.entity';

export interface IGoogleOAuthTokenRepository {
  findByEmail(email: string): Promise<GoogleOAuthToken | null>;
  upsert(token: GoogleOAuthToken): Promise<GoogleOAuthToken>;
  delete(email: string): Promise<void>;
}

export const GOOGLE_OAUTH_TOKEN_REPOSITORY = Symbol('GOOGLE_OAUTH_TOKEN_REPOSITORY');
