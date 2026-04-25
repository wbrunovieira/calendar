import { Injectable } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { GoogleOAuthToken } from '../../domain/entities/google-oauth-token.entity';
import { IGoogleOAuthTokenRepository } from '../../domain/repositories/google-oauth-token.repository.interface';

@Injectable()
export class PrismaGoogleOAuthTokenRepository implements IGoogleOAuthTokenRepository {
  private prisma: PrismaClient;

  constructor() {
    this.prisma = new PrismaClient();
  }

  async findByEmail(email: string): Promise<GoogleOAuthToken | null> {
    const record = await this.prisma.googleOAuthToken.findUnique({
      where: { email },
    });

    return record ? GoogleOAuthToken.create(record) : null;
  }

  async upsert(token: GoogleOAuthToken): Promise<GoogleOAuthToken> {
    const record = await this.prisma.googleOAuthToken.upsert({
      where: { email: token.email },
      update: {
        accessToken: token.accessToken,
        refreshToken: token.refreshToken,
        expiresAt: token.expiresAt,
      },
      create: {
        id: token.id,
        email: token.email,
        accessToken: token.accessToken,
        refreshToken: token.refreshToken,
        expiresAt: token.expiresAt,
      },
    });

    return GoogleOAuthToken.create(record);
  }

  async delete(email: string): Promise<void> {
    await this.prisma.googleOAuthToken.delete({ where: { email } });
  }

  async onModuleDestroy() {
    await this.prisma.$disconnect();
  }
}
