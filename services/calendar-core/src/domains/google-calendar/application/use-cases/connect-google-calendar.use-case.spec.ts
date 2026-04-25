import { describe, it, expect, beforeEach, vi } from 'vitest';
import { NotFoundException } from '@nestjs/common';
import { ConnectGoogleCalendarUseCase } from './connect-google-calendar.use-case';
import { Calendar } from '@domains/calendars/domain/entities/calendar.entity';

const mockCalendarRepository = {
  findById: vi.fn(),
  update: vi.fn(),
};

const mockGoogleOAuthService = {
  exchangeCode: vi.fn(),
  saveToken: vi.fn(),
};

describe('ConnectGoogleCalendarUseCase', () => {
  let useCase: ConnectGoogleCalendarUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new ConnectGoogleCalendarUseCase(
      mockCalendarRepository as any,
      mockGoogleOAuthService as any,
    );
  });

  it('should connect Google Calendar and save token', async () => {
    const calendar = Calendar.create({
      id: 'cal-1',
      userId: 'user-1',
      name: 'WB Digital Solutions',
      color: '#0077B5',
      type: 'professional',
    });

    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockGoogleOAuthService.exchangeCode.mockResolvedValue({
      email: 'bruno@wbdigitalsolutions.com',
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: new Date(),
    });
    mockGoogleOAuthService.saveToken.mockResolvedValue({});

    const result = await useCase.execute('auth-code-123', 'cal-1');

    expect(result.email).toBe('bruno@wbdigitalsolutions.com');
    expect(result.googleCalendarId).toBe('primary');
    expect(mockCalendarRepository.update).toHaveBeenCalledWith('cal-1', {
      email: 'bruno@wbdigitalsolutions.com',
      googleCalendarId: 'primary',
    });
  });

  it('should throw NotFoundException when calendar does not exist', async () => {
    mockCalendarRepository.findById.mockResolvedValue(null);

    await expect(useCase.execute('code', 'non-existent')).rejects.toThrow(NotFoundException);
    expect(mockGoogleOAuthService.exchangeCode).not.toHaveBeenCalled();
  });

  it('should save token before updating calendar', async () => {
    const calendar = Calendar.create({
      id: 'cal-1',
      userId: 'user-1',
      name: 'Personal',
      color: '#FF5733',
      type: 'personal',
    });

    const callOrder: string[] = [];
    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockGoogleOAuthService.exchangeCode.mockResolvedValue({
      email: 'wbrunovieira77@gmail.com',
      accessToken: 'at',
      refreshToken: 'rt',
      expiresAt: new Date(),
    });
    mockGoogleOAuthService.saveToken.mockImplementation(() => {
      callOrder.push('saveToken');
      return Promise.resolve({});
    });
    mockCalendarRepository.update.mockImplementation(() => {
      callOrder.push('updateCalendar');
      return Promise.resolve({});
    });

    await useCase.execute('code', 'cal-1');

    expect(callOrder).toEqual(['saveToken', 'updateCalendar']);
  });
});
