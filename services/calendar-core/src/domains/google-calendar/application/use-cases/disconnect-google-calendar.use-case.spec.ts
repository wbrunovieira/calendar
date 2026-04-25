import { describe, it, expect, beforeEach, vi } from 'vitest';
import { NotFoundException } from '@nestjs/common';
import { DisconnectGoogleCalendarUseCase } from './disconnect-google-calendar.use-case';
import { Calendar } from '@domains/calendars/domain/entities/calendar.entity';

const mockCalendarRepository = {
  findById: vi.fn(),
  update: vi.fn(),
};

const mockGoogleOAuthService = {
  revokeToken: vi.fn(),
};

describe('DisconnectGoogleCalendarUseCase', () => {
  let useCase: DisconnectGoogleCalendarUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new DisconnectGoogleCalendarUseCase(
      mockCalendarRepository as any,
      mockGoogleOAuthService as any,
    );
  });

  it('should revoke token and clear google fields from calendar', async () => {
    const calendar = Calendar.create({
      id: 'cal-1',
      userId: 'user-1',
      name: 'WB Digital Solutions',
      color: '#0077B5',
      type: 'professional',
      email: 'bruno@wbdigitalsolutions.com',
      googleCalendarId: 'primary',
      googleSyncToken: 'sync-token-xyz',
    });

    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockGoogleOAuthService.revokeToken.mockResolvedValue(undefined);

    await useCase.execute('cal-1');

    expect(mockGoogleOAuthService.revokeToken).toHaveBeenCalledWith('bruno@wbdigitalsolutions.com');
    expect(mockCalendarRepository.update).toHaveBeenCalledWith('cal-1', {
      googleCalendarId: null,
      googleSyncToken: null,
      lastSyncAt: null,
    });
  });

  it('should throw NotFoundException when calendar does not exist', async () => {
    mockCalendarRepository.findById.mockResolvedValue(null);

    await expect(useCase.execute('non-existent')).rejects.toThrow(NotFoundException);
    expect(mockGoogleOAuthService.revokeToken).not.toHaveBeenCalled();
  });

  it('should skip token revocation when calendar has no email', async () => {
    const calendar = Calendar.create({
      id: 'cal-1',
      userId: 'user-1',
      name: 'Personal',
      color: '#FF5733',
      type: 'personal',
      googleCalendarId: 'primary',
    });

    mockCalendarRepository.findById.mockResolvedValue(calendar);

    await useCase.execute('cal-1');

    expect(mockGoogleOAuthService.revokeToken).not.toHaveBeenCalled();
    expect(mockCalendarRepository.update).toHaveBeenCalledWith('cal-1', {
      googleCalendarId: null,
      googleSyncToken: null,
      lastSyncAt: null,
    });
  });
});
