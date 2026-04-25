import { Injectable, Inject, NotFoundException } from '@nestjs/common';
import { OAuth2Client } from 'google-auth-library';
import {
  IGoogleOAuthTokenRepository,
  GOOGLE_OAUTH_TOKEN_REPOSITORY,
} from '../../domain/repositories/google-oauth-token.repository.interface';
import { GoogleOAuthToken } from '../../domain/entities/google-oauth-token.entity';

export interface TokenData {
  email: string;
  accessToken: string;
  refreshToken: string;
  expiresAt: Date;
}

@Injectable()
export class GoogleOAuthService {
  private oauth2Client: OAuth2Client;

  constructor(
    @Inject(GOOGLE_OAUTH_TOKEN_REPOSITORY)
    private readonly tokenRepository: IGoogleOAuthTokenRepository,
  ) {
    this.oauth2Client = new OAuth2Client(
      process.env.GOOGLE_CLIENT_ID,
      process.env.GOOGLE_CLIENT_SECRET,
      process.env.GOOGLE_REDIRECT_URI,
    );
  }

  getAuthorizationUrl(state: string): string {
    return this.oauth2Client.generateAuthUrl({
      access_type: 'offline',
      prompt: 'consent',
      scope: [
        'https://www.googleapis.com/auth/calendar',
        'https://www.googleapis.com/auth/calendar.events',
        'https://www.googleapis.com/auth/userinfo.email',
      ],
      state,
    });
  }

  async exchangeCode(code: string): Promise<TokenData> {
    const { tokens } = await this.oauth2Client.getToken(code);

    const userInfoClient = new OAuth2Client();
    userInfoClient.setCredentials(tokens);
    const userInfoResponse = await userInfoClient.request<{ email: string }>({
      url: 'https://www.googleapis.com/oauth2/v2/userinfo',
    });

    const email = userInfoResponse.data.email;
    const expiresAt = new Date(tokens.expiry_date!);

    return {
      email,
      accessToken: tokens.access_token!,
      refreshToken: tokens.refresh_token!,
      expiresAt,
    };
  }

  async getValidAccessToken(email: string): Promise<string> {
    const token = await this.tokenRepository.findByEmail(email);

    if (!token) {
      throw new NotFoundException(`No Google token found for ${email}`);
    }

    if (!token.isExpired()) {
      return token.accessToken;
    }

    return this.refreshToken(token);
  }

  async saveToken(tokenData: TokenData): Promise<GoogleOAuthToken> {
    const token = GoogleOAuthToken.create(tokenData);
    return this.tokenRepository.upsert(token);
  }

  async revokeToken(email: string): Promise<void> {
    const token = await this.tokenRepository.findByEmail(email);

    if (token) {
      try {
        await this.oauth2Client.revokeToken(token.accessToken);
      } catch {
        // Token may already be invalid — continue with local cleanup
      }
      await this.tokenRepository.delete(email);
    }
  }

  private async refreshToken(token: GoogleOAuthToken): Promise<string> {
    this.oauth2Client.setCredentials({ refresh_token: token.refreshToken });
    const { credentials } = await this.oauth2Client.refreshAccessToken();

    const updated = GoogleOAuthToken.create({
      ...token,
      accessToken: credentials.access_token!,
      expiresAt: new Date(credentials.expiry_date!),
    });

    await this.tokenRepository.upsert(updated);
    return updated.accessToken;
  }
}
