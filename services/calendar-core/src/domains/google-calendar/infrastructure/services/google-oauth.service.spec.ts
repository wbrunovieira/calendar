import { describe, it, expect, beforeEach, vi } from 'vitest';
import { NotFoundException } from '@nestjs/common';
import { GoogleOAuthService } from './google-oauth.service';
import { GoogleOAuthToken } from '../../domain/entities/google-oauth-token.entity';

const mockOAuth2Instance = {
  generateAuthUrl: vi.fn().mockReturnValue('https://accounts.google.com/o/oauth2/auth?mock'),
  getToken: vi.fn(),
  setCredentials: vi.fn(),
  refreshAccessToken: vi.fn(),
  revokeToken: vi.fn(),
  request: vi.fn(),
};

vi.mock('google-auth-library', () => ({
  OAuth2Client: function () {
    return mockOAuth2Instance;
  },
}));

const mockTokenRepository = {
  findByEmail: vi.fn(),
  upsert: vi.fn(),
  delete: vi.fn(),
};

describe('GoogleOAuthService', () => {
  let service: GoogleOAuthService;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-01-15T10:00:00Z'));
    service = new GoogleOAuthService(mockTokenRepository as any);
  });

  describe('getAuthorizationUrl', () => {
    it('should return a Google OAuth URL', () => {
      const url = service.getAuthorizationUrl('cal-123');

      expect(url).toContain('accounts.google.com');
    });
  });

  describe('getValidAccessToken', () => {
    it('should return existing access token when not expired', async () => {
      const token = GoogleOAuthToken.create({
        email: 'bruno@wbdigitalsolutions.com',
        accessToken: 'valid-access-token',
        refreshToken: 'refresh-token',
        expiresAt: new Date('2024-01-15T11:00:00Z'), // 1h ahead
      });

      mockTokenRepository.findByEmail.mockResolvedValue(token);

      const result = await service.getValidAccessToken('bruno@wbdigitalsolutions.com');

      expect(result).toBe('valid-access-token');
      expect(mockTokenRepository.upsert).not.toHaveBeenCalled();
    });

    it('should refresh and return new token when expired', async () => {
      const expiredToken = GoogleOAuthToken.create({
        email: 'bruno@wbdigitalsolutions.com',
        accessToken: 'old-access-token',
        refreshToken: 'refresh-token',
        expiresAt: new Date('2024-01-15T09:00:00Z'), // expired
      });

      mockTokenRepository.findByEmail.mockResolvedValue(expiredToken);
      mockTokenRepository.upsert.mockResolvedValue({});

      mockOAuth2Instance.refreshAccessToken.mockResolvedValue({
        credentials: {
          access_token: 'new-access-token',
          expiry_date: new Date('2024-01-15T11:00:00Z').getTime(),
        },
      });

      const result = await service.getValidAccessToken('bruno@wbdigitalsolutions.com');

      expect(result).toBe('new-access-token');
      expect(mockTokenRepository.upsert).toHaveBeenCalledOnce();
    });

    it('should throw NotFoundException when no token exists for email', async () => {
      mockTokenRepository.findByEmail.mockResolvedValue(null);

      await expect(
        service.getValidAccessToken('unknown@gmail.com'),
      ).rejects.toThrow(NotFoundException);
    });
  });

  describe('revokeToken', () => {
    it('should revoke token and delete from repository', async () => {
      const token = GoogleOAuthToken.create({
        email: 'bruno@wbdigitalsolutions.com',
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: new Date('2024-01-15T11:00:00Z'),
      });

      mockTokenRepository.findByEmail.mockResolvedValue(token);
      mockOAuth2Instance.revokeToken.mockResolvedValue({});

      await service.revokeToken('bruno@wbdigitalsolutions.com');

      expect(mockOAuth2Instance.revokeToken).toHaveBeenCalledWith('access-token');
      expect(mockTokenRepository.delete).toHaveBeenCalledWith('bruno@wbdigitalsolutions.com');
    });

    it('should still delete local token when Google revocation fails', async () => {
      const token = GoogleOAuthToken.create({
        email: 'test@example.com',
        accessToken: 'invalid-token',
        refreshToken: 'rt',
        expiresAt: new Date('2024-01-15T11:00:00Z'),
      });

      mockTokenRepository.findByEmail.mockResolvedValue(token);
      mockOAuth2Instance.revokeToken.mockRejectedValue(new Error('Token already revoked'));

      await service.revokeToken('test@example.com');

      expect(mockTokenRepository.delete).toHaveBeenCalledWith('test@example.com');
    });

    it('should do nothing when no token exists for email', async () => {
      mockTokenRepository.findByEmail.mockResolvedValue(null);

      await service.revokeToken('notoken@example.com');

      expect(mockTokenRepository.delete).not.toHaveBeenCalled();
    });
  });
});
