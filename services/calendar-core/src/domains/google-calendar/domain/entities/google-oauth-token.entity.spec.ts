import { describe, it, expect, beforeEach, vi } from 'vitest';
import { GoogleOAuthToken } from './google-oauth-token.entity';

describe('GoogleOAuthToken Entity', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-01-15T10:00:00Z'));
  });

  describe('create', () => {
    it('should create token with required fields', () => {
      const expiresAt = new Date('2024-01-15T11:00:00Z');

      const token = GoogleOAuthToken.create({
        email: 'bruno@wbdigitalsolutions.com',
        accessToken: 'access-token-abc',
        refreshToken: 'refresh-token-xyz',
        expiresAt,
      });

      expect(token.email).toBe('bruno@wbdigitalsolutions.com');
      expect(token.accessToken).toBe('access-token-abc');
      expect(token.refreshToken).toBe('refresh-token-xyz');
      expect(token.expiresAt).toEqual(expiresAt);
      expect(token.id).toBeDefined();
    });

    it('should use provided id when given', () => {
      const token = GoogleOAuthToken.create({
        id: 'token-123',
        email: 'test@example.com',
        accessToken: 'at',
        refreshToken: 'rt',
        expiresAt: new Date(),
      });

      expect(token.id).toBe('token-123');
    });
  });

  describe('isExpired', () => {
    it('should return false when token expires in more than 5 minutes', () => {
      const token = GoogleOAuthToken.create({
        email: 'test@example.com',
        accessToken: 'at',
        refreshToken: 'rt',
        expiresAt: new Date('2024-01-15T10:10:00Z'), // 10 min ahead
      });

      expect(token.isExpired()).toBe(false);
    });

    it('should return true when token expires in less than 5 minutes', () => {
      const token = GoogleOAuthToken.create({
        email: 'test@example.com',
        accessToken: 'at',
        refreshToken: 'rt',
        expiresAt: new Date('2024-01-15T10:04:00Z'), // 4 min ahead
      });

      expect(token.isExpired()).toBe(true);
    });

    it('should return true when token is already expired', () => {
      const token = GoogleOAuthToken.create({
        email: 'test@example.com',
        accessToken: 'at',
        refreshToken: 'rt',
        expiresAt: new Date('2024-01-15T09:00:00Z'), // 1h ago
      });

      expect(token.isExpired()).toBe(true);
    });

    it('should use custom buffer when provided', () => {
      const token = GoogleOAuthToken.create({
        email: 'test@example.com',
        accessToken: 'at',
        refreshToken: 'rt',
        expiresAt: new Date('2024-01-15T10:02:00Z'), // 2 min ahead
      });

      expect(token.isExpired(60_000)).toBe(false);  // 1 min buffer
      expect(token.isExpired(300_000)).toBe(true);   // 5 min buffer
    });
  });
});
