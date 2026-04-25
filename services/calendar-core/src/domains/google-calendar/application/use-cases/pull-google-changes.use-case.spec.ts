import { describe, it, expect, beforeEach, vi } from 'vitest';
import { PullGoogleChangesUseCase } from './pull-google-changes.use-case';
import { Calendar } from '@domains/calendars/domain/entities/calendar.entity';
import { Event } from '@domains/events/domain/entities/event.entity';

const mockCalendarRepository = {
  findById: vi.fn(),
  update: vi.fn(),
  updateSyncState: vi.fn(),
};

const mockEventRepository = {
  findByGoogleEventId: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};

const mockApiClient = {
  fetchChanges: vi.fn(),
};

function makeCalendar(overrides: Partial<Calendar> = {}): Calendar {
  return Calendar.create({
    id: 'cal-1',
    userId: 'user-1',
    name: 'WB Digital Solutions',
    color: '#0077B5',
    type: 'professional',
    email: 'bruno@wbdigitalsolutions.com',
    googleCalendarId: 'primary',
    ...overrides,
  });
}

function makeGoogleEvent(overrides = {}) {
  return {
    id: 'g-evt-1',
    summary: 'Google Meeting',
    start: { dateTime: '2024-01-15T10:00:00-03:00' },
    end: { dateTime: '2024-01-15T11:00:00-03:00' },
    status: 'confirmed',
    ...overrides,
  };
}

describe('PullGoogleChangesUseCase', () => {
  let useCase: PullGoogleChangesUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new PullGoogleChangesUseCase(
      mockCalendarRepository as any,
      mockEventRepository as any,
      mockApiClient as any,
    );
  });

  it('should create local event for new Google event', async () => {
    const calendar = makeCalendar();
    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent()],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(null);
    mockEventRepository.create.mockResolvedValue({});

    const result = await useCase.execute('cal-1');

    expect(result.created).toBe(1);
    expect(result.updated).toBe(0);
    expect(result.deleted).toBe(0);
    expect(mockEventRepository.create).toHaveBeenCalledOnce();
  });

  it('should update local event when Google event already exists', async () => {
    const calendar = makeCalendar();
    const existingEvent = Object.assign(new Event({}), {
      id: 'local-evt-1',
      googleEventId: 'g-evt-1',
      calendarId: 'cal-1',
    });

    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ summary: 'Updated Meeting' })],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(existingEvent);
    mockEventRepository.update.mockResolvedValue({});

    const result = await useCase.execute('cal-1');

    expect(result.updated).toBe(1);
    expect(mockEventRepository.update).toHaveBeenCalledWith(
      'local-evt-1',
      expect.objectContaining({ title: 'Updated Meeting' }),
    );
  });

  it('should delete local event when Google event is cancelled', async () => {
    const calendar = makeCalendar();
    const existingEvent = Object.assign(new Event({}), {
      id: 'local-evt-1',
      googleEventId: 'g-evt-1',
      calendarId: 'cal-1',
    });

    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ status: 'cancelled' })],
      nextSyncToken: 'new-sync-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(existingEvent);
    mockEventRepository.delete.mockResolvedValue(undefined);

    const result = await useCase.execute('cal-1');

    expect(result.deleted).toBe(1);
    expect(mockEventRepository.delete).toHaveBeenCalledWith('local-evt-1');
  });

  it('should persist nextSyncToken after successful sync', async () => {
    const calendar = makeCalendar();
    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [],
      nextSyncToken: 'sync-token-xyz',
    });

    await useCase.execute('cal-1');

    expect(mockCalendarRepository.updateSyncState).toHaveBeenCalledWith(
      'cal-1',
      'sync-token-xyz',
      expect.any(Date),
    );
  });

  it('should return zeros and skip when calendar has no Google integration', async () => {
    const calendar = makeCalendar({ googleCalendarId: null });
    mockCalendarRepository.findById.mockResolvedValue(calendar);

    const result = await useCase.execute('cal-1');

    expect(result).toEqual({ created: 0, updated: 0, deleted: 0 });
    expect(mockApiClient.fetchChanges).not.toHaveBeenCalled();
  });

  it('should reset syncToken when Google returns 410 Gone (token expired)', async () => {
    const calendar = makeCalendar({ googleSyncToken: 'expired-token' });
    mockCalendarRepository.findById.mockResolvedValue(calendar);

    const goneError = Object.assign(new Error('Gone'), { code: 410 });
    mockApiClient.fetchChanges.mockRejectedValue(goneError);

    const result = await useCase.execute('cal-1');

    expect(mockCalendarRepository.update).toHaveBeenCalledWith('cal-1', { googleSyncToken: null });
    expect(result).toEqual({ created: 0, updated: 0, deleted: 0 });
  });

  it('should skip cancelled event that does not exist locally', async () => {
    const calendar = makeCalendar();
    mockCalendarRepository.findById.mockResolvedValue(calendar);
    mockApiClient.fetchChanges.mockResolvedValue({
      events: [makeGoogleEvent({ status: 'cancelled' })],
      nextSyncToken: 'new-token',
    });
    mockEventRepository.findByGoogleEventId.mockResolvedValue(null);

    const result = await useCase.execute('cal-1');

    expect(result.deleted).toBe(0);
    expect(mockEventRepository.delete).not.toHaveBeenCalled();
  });
});
